package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	kproducer "github.com/nusiss-capstone-project/asset-mservice/server/kafka/producer"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/proxy"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	cacheredis "github.com/nusiss-capstone-project/asset-mservice/server/repository/redis"
	"github.com/nusiss-capstone-project/asset-mservice/server/util"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	paymentEventSucceeded = "payment-succeeded"
	paymentEventFailed    = "payment-failed"
)

type PaymentResultInput struct {
	BizID     string
	PaymentID string
	EventType string
	Status    string
}

type createOrderContext struct {
	userID          int64
	idempotencyKey  string
	paymentMethodID int64
	quantity        decimal.Decimal
	quote           *cacheredis.QuoteCache
	payAmount       decimal.Decimal
	minorAmount     int64
}

type OrderService interface {
	CreateOrder(ctx context.Context, userID int64, req *data.CreateOrderRequest) (*data.OrderVO, error)
	GetOrderDetail(ctx context.Context, userID, orderID int64) (*data.OrderDetailVO, error)
	ListOrders(ctx context.Context, userID int64, status string, assetID, createdFrom, createdTo, cursor int64, limit int) (*data.OrderListVO, error)
	HandlePaymentResult(ctx context.Context, in PaymentResultInput) error
}

type OrderServiceImpl struct {
	orderDao                  dao.AssetOrderDao
	assetDao                  dao.AssetDao
	holdingDao                dao.UserAssetHoldingDao
	paymentProxy              proxy.PaymentProxy
	orderPaymentResultProducer kproducer.OrderPaymentResultProducer
}

var (
	orderServiceOnce sync.Once
	orderServiceInst OrderService
)

func GetOrderService() OrderService {
	orderServiceOnce.Do(func() {
		orderServiceInst = &OrderServiceImpl{
			orderDao:                   dao.GetAssetOrderDao(),
			assetDao:                   dao.GetAssetDao(),
			holdingDao:                 dao.GetUserAssetHoldingDao(),
			paymentProxy:               proxy.GetPaymentProxy(),
			orderPaymentResultProducer: kproducer.GetOrderPaymentResultProducer(),
		}
	})
	return orderServiceInst
}

func (s *OrderServiceImpl) CreateOrder(ctx context.Context, userID int64, req *data.CreateOrderRequest) (*data.OrderVO, error) {
	prepared, existing, err := s.validateCreateOrder(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return toOrderVO(existing), nil
	}

	order, err := s.buildAndCreateOrder(ctx, prepared)
	if err != nil {
		return nil, err
	}

	payResult, err := s.callCreatePayment(ctx, prepared, order)
	if err != nil {
		_ = s.HandlePaymentResult(ctx, PaymentResultInput{
			BizID:     order.OrderNo,
			EventType: paymentEventFailed,
			Status:    "FAILED",
		})
		return nil, err
	}

	if err := s.HandlePaymentResult(ctx, paymentResultFromCreate(order.OrderNo, payResult)); err != nil {
		return nil, err
	}

	latest, err := s.orderDao.GetByID(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return toOrderVO(order), nil
	}
	return toOrderVO(latest), nil
}

func (s *OrderServiceImpl) validateCreateOrder(
	ctx context.Context,
	userID int64,
	req *data.CreateOrderRequest,
) (*createOrderContext, *model.AssetOrder, error) {
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey == "" {
		return nil, nil, fmt.Errorf("idempotency_key is required")
	}
	if req.PaymentMethodID <= 0 {
		return nil, nil, fmt.Errorf("payment_method_id is required")
	}
	quantity, err := decimal.NewFromString(strings.TrimSpace(req.Quantity))
	if err != nil || !quantity.IsPositive() {
		return nil, nil, fmt.Errorf("invalid quantity")
	}

	existing, err := s.orderDao.GetByIdempotencyKey(ctx, userID, idempotencyKey)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		log.WithContext(ctx).Infow("create order idempotent hit",
			"order_id", existing.ID,
			"order_no", existing.OrderNo,
			"user_id", userID,
		)
		return nil, existing, nil
	}

	quote, err := cacheredis.GetQuote(ctx, strings.TrimSpace(req.QuoteID))
	if err != nil {
		if err == redis.Nil {
			return nil, nil, fmt.Errorf("quote not found or expired")
		}
		return nil, nil, err
	}
	if quote.UserID != userID {
		return nil, nil, fmt.Errorf("quote does not belong to user")
	}
	if quote.ExpiresAt > 0 && time.Now().Unix() > quote.ExpiresAt {
		return nil, nil, fmt.Errorf("quote expired")
	}

	unitPrice, err := decimal.NewFromString(quote.UnitPrice)
	if err != nil || !unitPrice.IsPositive() {
		return nil, nil, fmt.Errorf("invalid quote unit_price")
	}
	payAmount := unitPrice.Mul(quantity)
	minorAmount, err := util.ToMinorUnits(payAmount.StringFixed(2), quote.Currency)
	if err != nil {
		return nil, nil, err
	}

	return &createOrderContext{
		userID:          userID,
		idempotencyKey:  idempotencyKey,
		paymentMethodID: req.PaymentMethodID,
		quantity:        quantity,
		quote:           quote,
		payAmount:       payAmount,
		minorAmount:     minorAmount,
	}, nil, nil
}

func (s *OrderServiceImpl) buildAndCreateOrder(ctx context.Context, prepared *createOrderContext) (*model.AssetOrder, error) {
	orderNo, err := util.NewOrderNo()
	if err != nil {
		return nil, err
	}
	order := &model.AssetOrder{
		UserID:         prepared.userID,
		OrderNo:        orderNo,
		AssetID:        prepared.quote.AssetID,
		QuoteID:        prepared.quote.QuoteID,
		UnitPrice:      prepared.quote.UnitPrice,
		Quantity:       prepared.quantity,
		PayCurrency:    prepared.quote.Currency,
		PayAmount:      prepared.payAmount,
		Status:         model.OrderStatusPending,
		IdempotencyKey: prepared.idempotencyKey,
	}
	if err := s.orderDao.Create(ctx, nil, order); err != nil {
		return nil, err
	}
	return order, nil
}

func (s *OrderServiceImpl) callCreatePayment(
	ctx context.Context,
	prepared *createOrderContext,
	order *model.AssetOrder,
) (*proxy.CreatePaymentResult, error) {
	payResult, err := s.paymentProxy.CreatePayment(ctx, proxy.CreatePaymentRequest{
		BizID:           order.OrderNo,
		UserID:          prepared.userID,
		Amount:          prepared.minorAmount,
		Currency:        strings.ToLower(prepared.quote.Currency),
		PaymentMethodID: prepared.paymentMethodID,
	})
	if err != nil {
		log.WithContext(ctx).Errorw("create payment failed",
			"order_id", order.ID,
			"order_no", order.OrderNo,
			"error", err,
		)
		return nil, err
	}
	return payResult, nil
}

func (s *OrderServiceImpl) GetOrderDetail(ctx context.Context, userID, orderID int64) (*data.OrderDetailVO, error) {
	order, err := s.orderDao.GetByID(ctx, orderID)
	if err != nil {
		log.WithContext(ctx).Errorw("get order by id failed", "order_id", orderID, "error", err)
		return nil, err
	}
	if order == nil || order.UserID != userID {
		return nil, fmt.Errorf("order not found")
	}
	asset, err := s.assetDao.GetByID(ctx, order.AssetID)
	if err != nil {
		log.WithContext(ctx).Errorw("get asset by id failed", "order_id", order.ID, "error", err)
		return nil, err
	}
	vo := &data.OrderDetailVO{
		OrderID:     strconv.FormatInt(order.ID, 10),
		OrderNo:     order.OrderNo,
		UnitPrice:   order.UnitPrice,
		Quantity:    order.Quantity.String(),
		PayCurrency: order.PayCurrency,
		PayAmount:   order.PayAmount.String(),
		PaymentID:   order.PaymentID,
		Status:      order.Status,
		CreatedAt:   order.CreatedAt.Unix(),
		UpdatedAt:   order.UpdatedAt.Unix(),
	}
	if asset != nil {
		vo.Asset = data.OrderAssetVO{
			AssetID: strconv.FormatInt(asset.ID, 10),
			Name:    asset.Name,
			Symbol:  asset.Symbol,
			IconURL: asset.IconURL,
		}
	}
	return vo, nil
}

func (s *OrderServiceImpl) ListOrders(
	ctx context.Context,
	userID int64,
	status string,
	assetID, createdFrom, createdTo, cursor int64,
	limit int,
) (*data.OrderListVO, error) {
	if limit <= 0 {
		limit = 20
	}
	orders, err := s.orderDao.ListByCursor(ctx, dao.AssetOrderListFilter{
		UserID:      userID,
		Status:      strings.TrimSpace(status),
		AssetID:     assetID,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
		Cursor:      cursor,
		Limit:       limit + 1,
	})
	if err != nil {
		return nil, err
	}
	nextCursor := ""
	if len(orders) > limit {
		nextCursor = strconv.FormatInt(orders[limit-1].ID, 10)
		orders = orders[:limit]
	}
	items := make([]*data.OrderVO, 0, len(orders))
	for _, o := range orders {
		items = append(items, toOrderVO(o))
	}
	return &data.OrderListVO{NextCursor: nextCursor, Items: items}, nil
}

// HandlePaymentResult is shared by CreateOrder (sync gRPC result) and Kafka payment.transaction.updated.
func (s *OrderServiceImpl) HandlePaymentResult(ctx context.Context, in PaymentResultInput) error {
	if strings.TrimSpace(in.BizID) == "" {
		return fmt.Errorf("biz_id is required")
	}
	order, err := s.orderDao.GetByOrderNo(ctx, in.BizID)
	if err != nil {
		log.WithContext(ctx).Errorw("get order by order no failed", "biz_id", in.BizID, "error", err)
		return err
	}
	if order == nil {
		log.WithContext(ctx).Warnw("payment result order not found", "biz_id", in.BizID, "payment_id", in.PaymentID)
		return nil
	}

	targetStatus := mapPaymentEvent(in.EventType, in.Status)
	if order.Status == targetStatus {
		log.WithContext(ctx).Infow("payment result already applied",
			"order_id", order.ID,
			"status", order.Status,
		)
		return nil
	}
	// only process pending orders
	if order.Status != model.OrderStatusPending {
		log.WithContext(ctx).Warnw("order already settled, skip payment result application",
			"order_id", order.ID,
			"order_no", order.OrderNo,
			"payment_id", in.PaymentID,
			"status", order.Status,
		)
		return nil
	}
	return s.applyPaymentResult(ctx, order, in.PaymentID, targetStatus)
}

func (s *OrderServiceImpl) applyPaymentResult(ctx context.Context, order *model.AssetOrder, paymentID, status string) error {
	if status != model.OrderStatusPaySucceed {
		rows, err := s.orderDao.UpdatePaymentResult(ctx, nil, order.ID, paymentID, model.OrderStatusPending, status)
		if err != nil {
			return err
		}
		if rows == 0 {
			return nil
		}
		return s.publishOrderPaymentResult(ctx, order, paymentID, status)
	}

	err := repository.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, err := s.orderDao.UpdatePaymentResult(ctx, tx, order.ID, paymentID, model.OrderStatusPending, model.OrderStatusPaySucceed)
		if err != nil {
			return err
		}
		if rows == 0 {
			log.WithContext(ctx).Infow("order already settled, skip holding update",
				"order_id", order.ID,
				"order_no", order.OrderNo,
				"payment_id", paymentID,
			)
			return errOrderAlreadySettled
		}
		if err := s.holdingDao.UpsertAddQuantity(ctx, tx, order.UserID, order.AssetID, order.Quantity); err != nil {
			return err
		}
		log.WithContext(ctx).Infow("order settled successfully",
			"order_id", order.ID,
			"order_no", order.OrderNo,
			"payment_id", paymentID,
			"asset_id", order.AssetID,
			"quantity", order.Quantity.String(),
		)
		return nil
	})
	if err != nil {
		if errors.Is(err, errOrderAlreadySettled) {
			return nil
		}
		return err
	}
	return s.publishOrderPaymentResult(ctx, order, paymentID, model.OrderStatusPaySucceed)
}

var errOrderAlreadySettled = errors.New("order already settled")

func (s *OrderServiceImpl) publishOrderPaymentResult(
	ctx context.Context,
	order *model.AssetOrder,
	paymentID, status string,
) error {
	assetSymbol := ""
	asset, err := s.assetDao.GetByID(ctx, order.AssetID)
	if err != nil {
		log.WithContext(ctx).Errorw("load asset for payment result event failed",
			"order_id", order.ID,
			"asset_id", order.AssetID,
			"error", err,
		)
		return err
	}
	if asset != nil {
		assetSymbol = asset.Symbol
	}

	event := kproducer.OrderPaymentResultEvent{
		UserID:        order.UserID,
		OrderID:       order.ID,
		OrderNo:       order.OrderNo,
		PaymentID:     paymentID,
		AssetID:       order.AssetID,
		AssetSymbol:   assetSymbol,
		Quantity:      order.Quantity.String(),
		UnitPrice:     order.UnitPrice,
		PaymentAmount: order.PayAmount.String(),
		PayCurrency:   order.PayCurrency,
		Status:        status,
		EventTime:     time.Now().Unix(),
	}
	if err := s.orderPaymentResultProducer.PublishOrderPaymentResult(ctx, event); err != nil {
		log.WithContext(ctx).Errorw("publish order payment result failed",
			"order_id", order.ID,
			"order_no", order.OrderNo,
			"payment_id", paymentID,
			"status", status,
			"error", err,
		)
		return err
	}
	log.WithContext(ctx).Infow("order payment result published",
		"order_id", order.ID,
		"order_no", order.OrderNo,
		"payment_id", paymentID,
		"status", status,
		"asset_symbol", assetSymbol,
	)
	return nil
}

func paymentResultFromCreate(orderNo string, result *proxy.CreatePaymentResult) PaymentResultInput {
	eventType := ""
	switch mapPaymentStatus(result.Status) {
	case model.OrderStatusPaySucceed:
		eventType = paymentEventSucceeded
	case model.OrderStatusPayFail:
		eventType = paymentEventFailed
	}
	return PaymentResultInput{
		BizID:     orderNo,
		PaymentID: result.PaymentID,
		EventType: eventType,
		Status:    result.Status,
	}
}

func toOrderVO(o *model.AssetOrder) *data.OrderVO {
	return &data.OrderVO{
		OrderID:     strconv.FormatInt(o.ID, 10),
		OrderNo:     o.OrderNo,
		AssetID:     strconv.FormatInt(o.AssetID, 10),
		QuoteID:     o.QuoteID,
		UnitPrice:   o.UnitPrice,
		Quantity:    o.Quantity.String(),
		PayCurrency: o.PayCurrency,
		PayAmount:   o.PayAmount.String(),
		PaymentID:   o.PaymentID,
		Status:      o.Status,
		CreatedAt:   o.CreatedAt.Unix(),
		UpdatedAt:   o.UpdatedAt.Unix(),
	}
}

func mapPaymentStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCEEDED":
		return model.OrderStatusPaySucceed
	case "FAILED":
		return model.OrderStatusPayFail
	default:
		return model.OrderStatusPending
	}
}

func mapPaymentEvent(eventType, status string) string {
	switch strings.TrimSpace(eventType) {
	case paymentEventSucceeded:
		return model.OrderStatusPaySucceed
	case paymentEventFailed:
		return model.OrderStatusPayFail
	default:
		return mapPaymentStatus(status)
	}
}
