package dao

import (
	"context"
	"errors"
	"sync"

	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserAssetHoldingDao interface {
	GetByUserAndAsset(ctx context.Context, userID, assetID int64) (*model.UserAssetHolding, error)
	ListByUser(ctx context.Context, userID, assetID int64) ([]*model.UserAssetHolding, error)
	UpsertAddQuantity(ctx context.Context, tx *gorm.DB, userID, assetID int64, quantity decimal.Decimal) error
}

type UserAssetHoldingDaoImpl struct {
	db *gorm.DB
}

var (
	holdingOnce sync.Once
	holdingDao  *UserAssetHoldingDaoImpl
)

func GetUserAssetHoldingDao() *UserAssetHoldingDaoImpl {
	holdingOnce.Do(func() {
		holdingDao = &UserAssetHoldingDaoImpl{db: repository.DB}
	})
	return holdingDao
}

func (dao *UserAssetHoldingDaoImpl) GetByUserAndAsset(ctx context.Context, userID, assetID int64) (*model.UserAssetHolding, error) {
	var holding model.UserAssetHolding
	err := dao.db.WithContext(ctx).
		Where("user_id = ? AND asset_id = ?", userID, assetID).
		First(&holding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get user asset holding failed",
			"user_id", userID,
			"asset_id", assetID,
			"error", err,
		)
		return nil, err
	}
	return &holding, nil
}

func (dao *UserAssetHoldingDaoImpl) ListByUser(ctx context.Context, userID, assetID int64) ([]*model.UserAssetHolding, error) {
	var holdings []*model.UserAssetHolding
	q := dao.db.WithContext(ctx).Where("user_id = ?", userID)
	if assetID > 0 {
		q = q.Where("asset_id = ?", assetID)
	}
	if err := q.Order("id asc").Find(&holdings).Error; err != nil {
		log.WithContext(ctx).Errorw("list user asset holdings failed", "user_id", userID, "error", err)
		return nil, err
	}
	return holdings, nil
}

func (dao *UserAssetHoldingDaoImpl) dbOr(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return dao.db
}

func (dao *UserAssetHoldingDaoImpl) UpsertAddQuantity(ctx context.Context, tx *gorm.DB, userID, assetID int64, quantity decimal.Decimal) error {
	holding := &model.UserAssetHolding{
		UserID:   userID,
		AssetID:  assetID,
		Quantity: quantity,
	}
	err := dao.dbOr(tx).WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "asset_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quantity":   gorm.Expr("quantity + ?", quantity),
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(holding).Error
	if err != nil {
		log.WithContext(ctx).Errorw("upsert user asset holding failed",
			"user_id", userID,
			"asset_id", assetID,
			"quantity", quantity.String(),
			"error", err,
		)
		return err
	}
	log.WithContext(ctx).Infow("user asset holding upserted",
		"user_id", userID,
		"asset_id", assetID,
		"quantity_added", quantity.String(),
	)
	return nil
}
