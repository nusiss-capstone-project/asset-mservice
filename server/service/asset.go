package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
)

type AssetService interface {
	ListAssets(ctx context.Context, currency, status string) ([]*data.AssetVO, error)
	GetAsset(ctx context.Context, assetID int64, currency string) (*data.AssetVO, error)
}

type AssetServiceImpl struct {
	assetDao dao.AssetDao
}

var (
	assetServiceOnce sync.Once
	assetServiceInst AssetService
)

func GetAssetService() AssetService {
	assetServiceOnce.Do(func() {
		assetServiceInst = &AssetServiceImpl{
			assetDao: dao.GetAssetDao(),
		}
	})
	return assetServiceInst
}

func (s *AssetServiceImpl) ListAssets(ctx context.Context, currency, status string) ([]*data.AssetVO, error) {
	currency = normalizeCurrency(currency)
	assets, err := s.assetDao.List(ctx, dao.AssetListFilter{
		Currency: currency,
		Status:   strings.TrimSpace(status),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*data.AssetVO, 0, len(assets))
	for _, a := range assets {
		out = append(out, toAssetVO(a))
	}
	return out, nil
}

func (s *AssetServiceImpl) GetAsset(ctx context.Context, assetID int64, currency string) (*data.AssetVO, error) {
	currency = normalizeCurrency(currency)
	asset, err := s.assetDao.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, fmt.Errorf("asset not found")
	}
	if currency != "" && !strings.EqualFold(asset.Currency, currency) {
		return nil, fmt.Errorf("asset currency mismatch")
	}
	return toAssetVO(asset), nil
}

func toAssetVO(a *model.Asset) *data.AssetVO {
	return &data.AssetVO{
		ID:           strconv.FormatInt(a.ID, 10),
		Name:         a.Name,
		Symbol:       a.Symbol,
		IconURL:      a.IconURL,
		Currency:     a.Currency,
		CurrentPrice: a.CurrentPrice,
		Status:       a.Status,
		CreatedAt:    a.CreatedAt.Unix(),
		UpdatedAt:    a.UpdatedAt.Unix(),
	}
}

func normalizeCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return "USD"
	}
	return currency
}
