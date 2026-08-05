package service

import (
	"context"
	"strconv"
	"sync"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/shopspring/decimal"
)

type HoldingService interface {
	ListHoldings(ctx context.Context, userID, assetID int64) (*data.HoldingListVO, error)
}

type HoldingServiceImpl struct {
	holdingDao dao.UserAssetHoldingDao
	assetDao   dao.AssetDao
}

var (
	holdingServiceOnce sync.Once
	holdingServiceInst HoldingService
)

func GetHoldingService() HoldingService {
	holdingServiceOnce.Do(func() {
		holdingServiceInst = &HoldingServiceImpl{
			holdingDao: dao.GetUserAssetHoldingDao(),
			assetDao:   dao.GetAssetDao(),
		}
	})
	return holdingServiceInst
}

func (s *HoldingServiceImpl) ListHoldings(ctx context.Context, userID, assetID int64) (*data.HoldingListVO, error) {
	holdings, err := s.holdingDao.ListByUser(ctx, userID, assetID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(holdings))
	for _, h := range holdings {
		ids = append(ids, h.AssetID)
	}
	assets, err := s.assetDao.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	assetMap := make(map[int64]*model.Asset, len(assets))
	for _, a := range assets {
		assetMap[a.ID] = a
	}

	items := make([]*data.HoldingVO, 0, len(holdings))
	for _, h := range holdings {
		asset, ok := assetMap[h.AssetID]
		if !ok {
			continue
		}
		unitPrice, err := decimal.NewFromString(asset.CurrentPrice)
		if err != nil {
			unitPrice = decimal.Zero
		}
		amount := unitPrice.Mul(h.Quantity)
		items = append(items, &data.HoldingVO{
			AssetID:   strconv.FormatInt(h.AssetID, 10),
			Quantity:  h.Quantity.String(),
			UpdatedAt: h.UpdatedAt.Unix(),
			Asset: data.HoldingAssetVO{
				Name:    asset.Name,
				Symbol:  asset.Symbol,
				IconURL: asset.IconURL,
			},
			Valuation: data.HoldingValuationVO{
				Currency:  asset.Currency,
				UnitPrice: asset.CurrentPrice,
				Amount:    amount.String(),
			},
		})
	}
	return &data.HoldingListVO{Items: items}, nil
}
