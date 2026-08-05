package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	kproducer "github.com/nusiss-capstone-project/asset-mservice/server/kafka/producer"
	producermocks "github.com/nusiss-capstone-project/asset-mservice/server/kafka/producer/mocks"
	"github.com/nusiss-capstone-project/asset-mservice/server/proxy"
	proxymocks "github.com/nusiss-capstone-project/asset-mservice/server/proxy/mocks"
	daomocks "github.com/nusiss-capstone-project/asset-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	cacheredis "github.com/nusiss-capstone-project/asset-mservice/server/repository/redis"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOrderServiceForTest(
	orderDao *daomocks.AssetOrderDao,
	assetDao *daomocks.AssetDao,
	holdingDao *daomocks.UserAssetHoldingDao,
	paymentProxy *proxymocks.PaymentProxy,
	resultProducer *producermocks.OrderPaymentResultProducer,
) *OrderServiceImpl {
	return &OrderServiceImpl{
		orderDao:                   orderDao,
		assetDao:                   assetDao,
		holdingDao:                 holdingDao,
		paymentProxy:               paymentProxy,
		orderPaymentResultProducer: resultProducer,
	}
}

func saveTestQuote(t *testing.T, quoteID string, userID, assetID int64) {
	t.Helper()
	require.NoError(t, cacheredis.SaveQuote(context.Background(), &cacheredis.QuoteCache{
		QuoteID:   quoteID,
		AssetID:   assetID,
		Currency:  "USD",
		UnitPrice: "100.00",
		UserID:    userID,
		ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
	}))
}

func TestCreateOrder_ValidationErrors(t *testing.T) {
	initServiceTestEnv(t)
	svc := newOrderServiceForTest(new(daomocks.AssetOrderDao), new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))

	_, err := svc.CreateOrder(context.Background(), 1, &data.CreateOrderRequest{})
	require.ErrorContains(t, err, "idempotency_key")

	_, err = svc.CreateOrder(context.Background(), 1, &data.CreateOrderRequest{IdempotencyKey: "k"})
	require.ErrorContains(t, err, "payment_method_id")

	_, err = svc.CreateOrder(context.Background(), 1, &data.CreateOrderRequest{
		IdempotencyKey: "k", PaymentMethodID: 1, Quantity: "-1",
	})
	require.ErrorContains(t, err, "quantity")
}

func TestCreateOrder_IdempotentHit(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	existing := sampleOrder(10, 42, model.OrderStatusPaySucceed)
	orderDao.On("GetByIdempotencyKey", mock.Anything, int64(42), "idem-1").Return(existing, nil)

	vo, err := svc.CreateOrder(context.Background(), 42, &data.CreateOrderRequest{
		IdempotencyKey: "idem-1", PaymentMethodID: 1, Quantity: "0.5", QuoteID: "q",
	})
	require.NoError(t, err)
	require.Equal(t, "10", vo.OrderID)
	require.Equal(t, model.OrderStatusPaySucceed, vo.Status)
}

func TestCreateOrder_QuoteNotFound(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByIdempotencyKey", mock.Anything, int64(1), "k").Return(nil, nil)

	_, err := svc.CreateOrder(context.Background(), 1, &data.CreateOrderRequest{
		IdempotencyKey: "k", PaymentMethodID: 1, Quantity: "0.5", QuoteID: "missing",
	})
	require.ErrorContains(t, err, "quote not found")
}

func TestCreateOrder_QuoteWrongUser(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	saveTestQuote(t, "q1", 99, 1)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByIdempotencyKey", mock.Anything, int64(1), "k").Return(nil, nil)

	_, err := svc.CreateOrder(context.Background(), 1, &data.CreateOrderRequest{
		IdempotencyKey: "k", PaymentMethodID: 1, Quantity: "0.5", QuoteID: "q1",
	})
	require.ErrorContains(t, err, "does not belong")
}

func TestCreateOrder_QuoteExpired(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	require.NoError(t, cacheredis.SaveQuote(context.Background(), &cacheredis.QuoteCache{
		QuoteID: "qexp", AssetID: 1, Currency: "USD", UnitPrice: "100.00", UserID: 1, ExpiresAt: time.Now().Add(-time.Minute).Unix(),
	}))
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByIdempotencyKey", mock.Anything, int64(1), "k").Return(nil, nil)

	_, err := svc.CreateOrder(context.Background(), 1, &data.CreateOrderRequest{
		IdempotencyKey: "k", PaymentMethodID: 1, Quantity: "0.5", QuoteID: "qexp",
	})
	require.ErrorContains(t, err, "expired")
}

func TestCreateOrder_InvalidUnitPrice(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	require.NoError(t, cacheredis.SaveQuote(context.Background(), &cacheredis.QuoteCache{
		QuoteID: "qbad", AssetID: 1, Currency: "USD", UnitPrice: "bad", UserID: 1,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
	}))
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByIdempotencyKey", mock.Anything, int64(1), "k").Return(nil, nil)
	_, err := svc.CreateOrder(context.Background(), 1, &data.CreateOrderRequest{
		IdempotencyKey: "k", PaymentMethodID: 1, Quantity: "1", QuoteID: "qbad",
	})
	require.ErrorContains(t, err, "unit_price")
}

func TestCreateOrder_Success_PaySucceed(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	saveTestQuote(t, "qok", 42, 1)

	orderDao := new(daomocks.AssetOrderDao)
	assetDao := new(daomocks.AssetDao)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	paymentProxy := new(proxymocks.PaymentProxy)
	resultProducer := new(producermocks.OrderPaymentResultProducer)
	svc := newOrderServiceForTest(orderDao, assetDao, holdingDao, paymentProxy, resultProducer)

	pendingOrder := sampleOrder(100, 42, model.OrderStatusPending)
	pendingOrder.QuoteID = "qok"

	orderDao.On("GetByIdempotencyKey", mock.Anything, int64(42), "idem-ok").Return(nil, nil)
	orderDao.On("Create", mock.Anything, (*gorm.DB)(nil), mock.AnythingOfType("*model.AssetOrder")).
		Run(func(args mock.Arguments) {
			o := args.Get(2).(*model.AssetOrder)
			o.ID = 100
			pendingOrder.OrderNo = o.OrderNo
			pendingOrder.IdempotencyKey = o.IdempotencyKey
		}).Return(nil)
	paymentProxy.On("CreatePayment", mock.Anything, mock.MatchedBy(func(req proxy.CreatePaymentRequest) bool {
		return req.UserID == 42 && req.Amount == 5000 && req.PaymentMethodID == 7
	})).Return(&proxy.CreatePaymentResult{PaymentID: "pay_1", Status: "SUCCEEDED"}, nil)

	orderDao.On("GetByOrderNo", mock.Anything, mock.AnythingOfType("string")).Return(pendingOrder, nil)
	orderDao.On("UpdatePaymentResult", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(100), "pay_1", model.OrderStatusPending, model.OrderStatusPaySucceed).
		Return(int64(1), nil)
	holdingDao.On("UpsertAddQuantity", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(42), int64(1), mock.Anything).Return(nil)
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(sampleAsset(1), nil)
	resultProducer.On("PublishOrderPaymentResult", mock.Anything, mock.MatchedBy(func(e kproducer.OrderPaymentResultEvent) bool {
		return e.UserID == 42 && e.PaymentID == "pay_1" && e.AssetSymbol == "BTC" && e.Status == model.OrderStatusPaySucceed
	})).Return(nil)

	succeeded := sampleOrder(100, 42, model.OrderStatusPaySucceed)
	succeeded.PaymentID = "pay_1"
	orderDao.On("GetByID", mock.Anything, int64(100)).Return(succeeded, nil)

	vo, err := svc.CreateOrder(context.Background(), 42, &data.CreateOrderRequest{
		QuoteID: "qok", Quantity: "0.5", PaymentMethodID: 7, IdempotencyKey: "idem-ok",
	})
	require.NoError(t, err)
	require.Equal(t, "100", vo.OrderID)
	require.Equal(t, model.OrderStatusPaySucceed, vo.Status)
	require.Equal(t, "pay_1", vo.PaymentID)
	orderDao.AssertExpectations(t)
	paymentProxy.AssertExpectations(t)
	holdingDao.AssertExpectations(t)
	resultProducer.AssertExpectations(t)
}

func TestCreateOrder_PaymentRPCFailedMarksPayFail(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	saveTestQuote(t, "qfail", 1, 1)

	orderDao := new(daomocks.AssetOrderDao)
	assetDao := new(daomocks.AssetDao)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	paymentProxy := new(proxymocks.PaymentProxy)
	resultProducer := new(producermocks.OrderPaymentResultProducer)
	svc := newOrderServiceForTest(orderDao, assetDao, holdingDao, paymentProxy, resultProducer)

	pendingOrder := sampleOrder(11, 1, model.OrderStatusPending)
	orderDao.On("GetByIdempotencyKey", mock.Anything, int64(1), "idem-fail").Return(nil, nil)
	orderDao.On("Create", mock.Anything, (*gorm.DB)(nil), mock.AnythingOfType("*model.AssetOrder")).
		Run(func(args mock.Arguments) {
			o := args.Get(2).(*model.AssetOrder)
			o.ID = 11
			pendingOrder.OrderNo = o.OrderNo
		}).Return(nil)
	paymentProxy.On("CreatePayment", mock.Anything, mock.Anything).Return(nil, errors.New("grpc down"))
	orderDao.On("GetByOrderNo", mock.Anything, mock.AnythingOfType("string")).Return(pendingOrder, nil)
	orderDao.On("UpdatePaymentResult", mock.Anything, (*gorm.DB)(nil), int64(11), "", model.OrderStatusPending, model.OrderStatusPayFail).
		Return(int64(1), nil)
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(sampleAsset(1), nil)
	resultProducer.On("PublishOrderPaymentResult", mock.Anything, mock.MatchedBy(func(e kproducer.OrderPaymentResultEvent) bool {
		return e.Status == model.OrderStatusPayFail
	})).Return(nil)

	_, err := svc.CreateOrder(context.Background(), 1, &data.CreateOrderRequest{
		QuoteID: "qfail", Quantity: "0.5", PaymentMethodID: 1, IdempotencyKey: "idem-fail",
	})
	require.ErrorContains(t, err, "grpc down")
}

func TestHandlePaymentResult_BizIDRequired(t *testing.T) {
	initServiceTestEnv(t)
	svc := newOrderServiceForTest(new(daomocks.AssetOrderDao), new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	require.ErrorContains(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{}), "biz_id")
}

func TestHandlePaymentResult_OrderNotFound(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByOrderNo", mock.Anything, "AOX").Return(nil, nil)
	require.NoError(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{BizID: "AOX"}))
}

func TestHandlePaymentResult_OrderLookupError(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByOrderNo", mock.Anything, "AOX").Return(nil, errors.New("db"))
	require.Error(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{BizID: "AOX"}))
}

func TestHandlePaymentResult_AlreadyApplied(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByOrderNo", mock.Anything, "AO1").Return(sampleOrder(1, 1, model.OrderStatusPaySucceed), nil)
	require.NoError(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{
		BizID: "AO1", EventType: paymentEventSucceeded, Status: "SUCCEEDED",
	}))
}

func TestHandlePaymentResult_SkipNonPending(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByOrderNo", mock.Anything, "AO1").Return(sampleOrder(1, 1, model.OrderStatusPayFail), nil)
	require.NoError(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{
		BizID: "AO1", EventType: paymentEventSucceeded,
	}))
}

func TestHandlePaymentResult_FailPath_RowsZero(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByOrderNo", mock.Anything, "AO1").Return(sampleOrder(1, 1, model.OrderStatusPending), nil)
	orderDao.On("UpdatePaymentResult", mock.Anything, (*gorm.DB)(nil), int64(1), "pay", model.OrderStatusPending, model.OrderStatusPayFail).
		Return(int64(0), nil)
	require.NoError(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{
		BizID: "AO1", PaymentID: "pay", EventType: paymentEventFailed,
	}))
}

func TestHandlePaymentResult_FailUpdateError(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByOrderNo", mock.Anything, "AO1").Return(sampleOrder(1, 1, model.OrderStatusPending), nil)
	orderDao.On("UpdatePaymentResult", mock.Anything, (*gorm.DB)(nil), int64(1), "pay", model.OrderStatusPending, model.OrderStatusPayFail).
		Return(int64(0), errors.New("update failed"))
	require.Error(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{
		BizID: "AO1", PaymentID: "pay", EventType: paymentEventFailed,
	}))
}

func TestHandlePaymentResult_SucceedAlreadySettledInTx(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByOrderNo", mock.Anything, "AO1").Return(sampleOrder(1, 1, model.OrderStatusPending), nil)
	orderDao.On("UpdatePaymentResult", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(1), "pay", model.OrderStatusPending, model.OrderStatusPaySucceed).
		Return(int64(0), nil)
	require.NoError(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{
		BizID: "AO1", PaymentID: "pay", EventType: paymentEventSucceeded,
	}))
}

func TestHandlePaymentResult_SucceedHoldingError(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), holdingDao, new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByOrderNo", mock.Anything, "AO1").Return(sampleOrder(1, 1, model.OrderStatusPending), nil)
	orderDao.On("UpdatePaymentResult", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(1), "pay", model.OrderStatusPending, model.OrderStatusPaySucceed).
		Return(int64(1), nil)
	holdingDao.On("UpsertAddQuantity", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(1), int64(1), mock.Anything).
		Return(errors.New("holding failed"))
	require.ErrorContains(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{
		BizID: "AO1", PaymentID: "pay", EventType: paymentEventSucceeded,
	}), "holding failed")
}

func TestHandlePaymentResult_SucceedPublish(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	assetDao := new(daomocks.AssetDao)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	resultProducer := new(producermocks.OrderPaymentResultProducer)
	svc := newOrderServiceForTest(orderDao, assetDao, holdingDao, new(proxymocks.PaymentProxy), resultProducer)
	orderDao.On("GetByOrderNo", mock.Anything, "AO1").Return(sampleOrder(1, 1, model.OrderStatusPending), nil)
	orderDao.On("UpdatePaymentResult", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(1), "pay", model.OrderStatusPending, model.OrderStatusPaySucceed).
		Return(int64(1), nil)
	holdingDao.On("UpsertAddQuantity", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(1), int64(1), mock.Anything).Return(nil)
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(sampleAsset(1), nil)
	resultProducer.On("PublishOrderPaymentResult", mock.Anything, mock.Anything).Return(nil)
	require.NoError(t, svc.HandlePaymentResult(context.Background(), PaymentResultInput{
		BizID: "AO1", PaymentID: "pay", EventType: paymentEventSucceeded,
	}))
}

func TestGetOrderDetail_Success(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	assetDao := new(daomocks.AssetDao)
	svc := newOrderServiceForTest(orderDao, assetDao, new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByID", mock.Anything, int64(5)).Return(sampleOrder(5, 42, model.OrderStatusPending), nil)
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(sampleAsset(1), nil)

	vo, err := svc.GetOrderDetail(context.Background(), 42, 5)
	require.NoError(t, err)
	require.Equal(t, "5", vo.OrderID)
	require.Equal(t, "BTC", vo.Asset.Symbol)
}

func TestGetOrderDetail_NotOwned(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByID", mock.Anything, int64(5)).Return(sampleOrder(5, 99, model.OrderStatusPending), nil)
	_, err := svc.GetOrderDetail(context.Background(), 42, 5)
	require.ErrorContains(t, err, "not found")
}

func TestGetOrderDetail_NilOrder(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByID", mock.Anything, int64(1)).Return(nil, nil)
	_, err := svc.GetOrderDetail(context.Background(), 1, 1)
	require.ErrorContains(t, err, "not found")
}

func TestGetOrderDetail_DAOError(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("GetByID", mock.Anything, int64(1)).Return(nil, errors.New("db"))
	_, err := svc.GetOrderDetail(context.Background(), 1, 1)
	require.Error(t, err)
}

func TestListOrders_WithCursor(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	o1 := sampleOrder(3, 1, model.OrderStatusPending)
	o2 := sampleOrder(2, 1, model.OrderStatusPending)
	o3 := sampleOrder(1, 1, model.OrderStatusPending)
	orderDao.On("ListByCursor", mock.Anything, mock.Anything).Return([]*model.AssetOrder{o1, o2, o3}, nil)

	list, err := svc.ListOrders(context.Background(), ListOrdersQuery{UserID: 1, Limit: 2})
	require.NoError(t, err)
	require.Len(t, list.Items, 2)
	require.Equal(t, "2", list.NextCursor)
}

func TestListOrders_DefaultLimit(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("ListByCursor", mock.Anything, mock.Anything).Return([]*model.AssetOrder{}, nil)
	list, err := svc.ListOrders(context.Background(), ListOrdersQuery{UserID: 1})
	require.NoError(t, err)
	require.Empty(t, list.Items)
}

func TestListOrders_DAOError(t *testing.T) {
	initServiceTestEnv(t)
	orderDao := new(daomocks.AssetOrderDao)
	svc := newOrderServiceForTest(orderDao, new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	orderDao.On("ListByCursor", mock.Anything, mock.Anything).Return(nil, errors.New("db"))
	_, err := svc.ListOrders(context.Background(), ListOrdersQuery{UserID: 1, Limit: 10})
	require.Error(t, err)
}

func TestMapPaymentHelpers(t *testing.T) {
	require.Equal(t, model.OrderStatusPaySucceed, mapPaymentStatus("SUCCEEDED"))
	require.Equal(t, model.OrderStatusPayFail, mapPaymentStatus("FAILED"))
	require.Equal(t, model.OrderStatusPending, mapPaymentStatus("PENDING"))
	require.Equal(t, model.OrderStatusPaySucceed, mapPaymentEvent(paymentEventSucceeded, ""))
	require.Equal(t, model.OrderStatusPayFail, mapPaymentEvent(paymentEventFailed, ""))
	require.Equal(t, model.OrderStatusPending, mapPaymentEvent("", "PENDING"))

	in := paymentResultFromCreate("AO1", &proxy.CreatePaymentResult{PaymentID: "p", Status: "SUCCEEDED"})
	require.Equal(t, paymentEventSucceeded, in.EventType)
	in = paymentResultFromCreate("AO1", &proxy.CreatePaymentResult{PaymentID: "p", Status: "FAILED"})
	require.Equal(t, paymentEventFailed, in.EventType)
	in = paymentResultFromCreate("AO1", &proxy.CreatePaymentResult{PaymentID: "p", Status: "PENDING"})
	require.Empty(t, in.EventType)
}

func TestPublishOrderPaymentResult_AssetLoadError(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	svc := newOrderServiceForTest(new(daomocks.AssetOrderDao), assetDao, new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), new(producermocks.OrderPaymentResultProducer))
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(nil, errors.New("db"))
	err := svc.publishOrderPaymentResult(context.Background(), sampleOrder(1, 1, model.OrderStatusPayFail), "pay", model.OrderStatusPayFail)
	require.Error(t, err)
}

func TestPublishOrderPaymentResult_PublishError(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	resultProducer := new(producermocks.OrderPaymentResultProducer)
	svc := newOrderServiceForTest(new(daomocks.AssetOrderDao), assetDao, new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), resultProducer)
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(sampleAsset(1), nil)
	resultProducer.On("PublishOrderPaymentResult", mock.Anything, mock.Anything).Return(errors.New("kafka"))
	err := svc.publishOrderPaymentResult(context.Background(), sampleOrder(1, 1, model.OrderStatusPayFail), "pay", model.OrderStatusPayFail)
	require.ErrorContains(t, err, "kafka")
}

func TestPublishOrderPaymentResult_NilAsset(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	resultProducer := new(producermocks.OrderPaymentResultProducer)
	svc := newOrderServiceForTest(new(daomocks.AssetOrderDao), assetDao, new(daomocks.UserAssetHoldingDao), new(proxymocks.PaymentProxy), resultProducer)
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(nil, nil)
	resultProducer.On("PublishOrderPaymentResult", mock.Anything, mock.MatchedBy(func(e kproducer.OrderPaymentResultEvent) bool {
		return e.AssetSymbol == ""
	})).Return(nil)
	require.NoError(t, svc.publishOrderPaymentResult(context.Background(), sampleOrder(1, 1, model.OrderStatusPayFail), "pay", model.OrderStatusPayFail))
}

func TestToOrderVO(t *testing.T) {
	vo := toOrderVO(sampleOrder(1, 1, model.OrderStatusPending))
	require.Equal(t, "0.5", vo.Quantity)
	require.Equal(t, "1", vo.OrderID)
}
