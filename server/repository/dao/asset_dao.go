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

type AssetListFilter struct {
	Currency string
	Status   string
}

type AssetDao interface {
	List(ctx context.Context, filter AssetListFilter) ([]*model.Asset, error)
	GetByID(ctx context.Context, id int64) (*model.Asset, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*model.Asset, error)
	GetBySymbol(ctx context.Context, symbol string) (*model.Asset, error)
}

type AssetDaoImpl struct {
	db *gorm.DB
}

var (
	assetOnce sync.Once
	assetDao  *AssetDaoImpl
)

func GetAssetDao() *AssetDaoImpl {
	assetOnce.Do(func() {
		assetDao = &AssetDaoImpl{db: repository.DB}
	})
	return assetDao
}

func (dao *AssetDaoImpl) List(ctx context.Context, filter AssetListFilter) ([]*model.Asset, error) {
	var assets []*model.Asset
	q := dao.db.WithContext(ctx).Model(&model.Asset{})
	if filter.Currency != "" {
		q = q.Where("currency = ?", filter.Currency)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if err := q.Order("id asc").Find(&assets).Error; err != nil {
		log.WithContext(ctx).Errorw("list assets failed", "error", err)
		return nil, err
	}
	return assets, nil
}

func (dao *AssetDaoImpl) GetByID(ctx context.Context, id int64) (*model.Asset, error) {
	var asset model.Asset
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&asset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get asset by id failed", "asset_id", id, "error", err)
		return nil, err
	}
	return &asset, nil
}

func (dao *AssetDaoImpl) GetByIDs(ctx context.Context, ids []int64) ([]*model.Asset, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var assets []*model.Asset
	if err := dao.db.WithContext(ctx).Where("id IN ?", ids).Find(&assets).Error; err != nil {
		log.WithContext(ctx).Errorw("get assets by ids failed", "error", err)
		return nil, err
	}
	return assets, nil
}

func (dao *AssetDaoImpl) GetBySymbol(ctx context.Context, symbol string) (*model.Asset, error) {
	var asset model.Asset
	err := dao.db.WithContext(ctx).Where("symbol = ?", symbol).Order("id asc").First(&asset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get asset by symbol failed", "symbol", symbol, "error", err)
		return nil, err
	}
	return &asset, nil
}
