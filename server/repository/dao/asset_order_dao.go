package dao

import (
	"context"
	"errors"
	"sync"

	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"gorm.io/gorm"
)

type AssetOrderListFilter struct {
	UserID      int64
	Status      string
	AssetID     int64
	CreatedFrom int64
	CreatedTo   int64
	Cursor      int64
	Limit       int
}

type AssetOrderDao interface {
	Create(ctx context.Context, tx *gorm.DB, order *model.AssetOrder) error
	GetByID(ctx context.Context, id int64) (*model.AssetOrder, error)
	GetByOrderNo(ctx context.Context, orderNo string) (*model.AssetOrder, error)
	GetByIdempotencyKey(ctx context.Context, userID int64, key string) (*model.AssetOrder, error)
	UpdatePaymentResult(ctx context.Context, tx *gorm.DB, id int64, paymentID, fromStatus, toStatus string) (int64, error)
	ListByCursor(ctx context.Context, filter AssetOrderListFilter) ([]*model.AssetOrder, error)
}

type AssetOrderDaoImpl struct {
	db *gorm.DB
}

var (
	assetOrderOnce sync.Once
	assetOrderDao  *AssetOrderDaoImpl
)

func GetAssetOrderDao() *AssetOrderDaoImpl {
	assetOrderOnce.Do(func() {
		assetOrderDao = &AssetOrderDaoImpl{db: repository.DB}
	})
	return assetOrderDao
}

func (dao *AssetOrderDaoImpl) dbOr(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return dao.db
}

func (dao *AssetOrderDaoImpl) Create(ctx context.Context, tx *gorm.DB, order *model.AssetOrder) error {
	if err := dao.dbOr(tx).WithContext(ctx).Create(order).Error; err != nil {
		log.WithContext(ctx).Errorw("create asset order failed",
			"user_id", order.UserID,
			"order_no", order.OrderNo,
			"error", err,
		)
		return err
	}
	log.WithContext(ctx).Infow("asset order created",
		"order_id", order.ID,
		"order_no", order.OrderNo,
		"user_id", order.UserID,
		"status", order.Status,
	)
	return nil
}

func (dao *AssetOrderDaoImpl) GetByID(ctx context.Context, id int64) (*model.AssetOrder, error) {
	var order model.AssetOrder
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get asset order by id failed", "order_id", id, "error", err)
		return nil, err
	}
	return &order, nil
}

func (dao *AssetOrderDaoImpl) GetByOrderNo(ctx context.Context, orderNo string) (*model.AssetOrder, error) {
	var order model.AssetOrder
	err := dao.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get asset order by order_no failed", "order_no", orderNo, "error", err)
		return nil, err
	}
	return &order, nil
}

func (dao *AssetOrderDaoImpl) GetByIdempotencyKey(ctx context.Context, userID int64, key string) (*model.AssetOrder, error) {
	var order model.AssetOrder
	err := dao.db.WithContext(ctx).
		Where("user_id = ? AND idempotency_key = ?", userID, key).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get asset order by idempotency_key failed",
			"user_id", userID,
			"idempotency_key", key,
			"error", err,
		)
		return nil, err
	}
	return &order, nil
}

func (dao *AssetOrderDaoImpl) UpdatePaymentResult(ctx context.Context, tx *gorm.DB, id int64, paymentID, fromStatus, toStatus string) (int64, error) {
	updates := map[string]interface{}{
		"status": toStatus,
	}
	if paymentID != "" {
		updates["payment_id"] = paymentID
	}
	ret := dao.dbOr(tx).WithContext(ctx).Model(&model.AssetOrder{}).
		Where("id = ? AND status = ?", id, fromStatus).
		Updates(updates)
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("update asset order payment result failed",
			"order_id", id,
			"payment_id", paymentID,
			"from_status", fromStatus,
			"to_status", toStatus,
			"error", ret.Error,
		)
		return 0, ret.Error
	}
	log.WithContext(ctx).Infow("asset order payment result updated",
		"order_id", id,
		"payment_id", paymentID,
		"from_status", fromStatus,
		"to_status", toStatus,
		"rows", ret.RowsAffected,
	)
	return ret.RowsAffected, nil
}

func (dao *AssetOrderDaoImpl) ListByCursor(ctx context.Context, filter AssetOrderListFilter) ([]*model.AssetOrder, error) {
	var orders []*model.AssetOrder
	q := dao.db.WithContext(ctx).Model(&model.AssetOrder{}).Where("user_id = ?", filter.UserID)
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.AssetID > 0 {
		q = q.Where("asset_id = ?", filter.AssetID)
	}
	if filter.CreatedFrom > 0 {
		q = q.Where("UNIX_TIMESTAMP(created_at) >= ?", filter.CreatedFrom)
	}
	if filter.CreatedTo > 0 {
		q = q.Where("UNIX_TIMESTAMP(created_at) <= ?", filter.CreatedTo)
	}
	if filter.Cursor > 0 {
		q = q.Where("id < ?", filter.Cursor)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if err := q.Order("id desc").Limit(limit).Find(&orders).Error; err != nil {
		log.WithContext(ctx).Errorw("list asset orders failed", "user_id", filter.UserID, "error", err)
		return nil, err
	}
	return orders, nil
}
