package service

import (
	"context"
	"errors"
	"testing"
	"time"

	daomocks "github.com/nusiss-capstone-project/asset-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListHoldings_Success(t *testing.T) {
	initServiceTestEnv(t)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	assetDao := new(daomocks.AssetDao)
	svc := &HoldingServiceImpl{holdingDao: holdingDao, assetDao: assetDao}

	now := time.Now()
	holdingDao.On("ListByUser", mock.Anything, int64(42), int64(0)).
		Return([]*model.UserAssetHolding{{
			ID: 1, UserID: 42, AssetID: 1,
			Quantity: decimal.RequireFromString("0.5"),
			UpdatedAt: now,
		}}, nil)
	assetDao.On("GetByIDs", mock.Anything, []int64{1}).
		Return([]*model.Asset{sampleAsset(1)}, nil)

	list, err := svc.ListHoldings(context.Background(), 42, 0)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, "0.5", list.Items[0].Quantity)
	require.Equal(t, "USD", list.Items[0].Valuation.Currency)
	require.Equal(t, "50", list.Items[0].Valuation.Amount)
}

func TestListHoldings_SkipMissingAsset(t *testing.T) {
	initServiceTestEnv(t)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	assetDao := new(daomocks.AssetDao)
	svc := &HoldingServiceImpl{holdingDao: holdingDao, assetDao: assetDao}

	holdingDao.On("ListByUser", mock.Anything, int64(42), int64(0)).
		Return([]*model.UserAssetHolding{{
			UserID: 42, AssetID: 99, Quantity: decimal.RequireFromString("1"),
		}}, nil)
	assetDao.On("GetByIDs", mock.Anything, []int64{99}).Return([]*model.Asset{}, nil)

	list, err := svc.ListHoldings(context.Background(), 42, 0)
	require.NoError(t, err)
	require.Empty(t, list.Items)
}

func TestListHoldings_InvalidPriceUsesZero(t *testing.T) {
	initServiceTestEnv(t)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	assetDao := new(daomocks.AssetDao)
	svc := &HoldingServiceImpl{holdingDao: holdingDao, assetDao: assetDao}

	asset := sampleAsset(1)
	asset.CurrentPrice = "bad"
	holdingDao.On("ListByUser", mock.Anything, int64(1), int64(1)).
		Return([]*model.UserAssetHolding{{
			UserID: 1, AssetID: 1, Quantity: decimal.RequireFromString("2"),
		}}, nil)
	assetDao.On("GetByIDs", mock.Anything, []int64{1}).Return([]*model.Asset{asset}, nil)

	list, err := svc.ListHoldings(context.Background(), 1, 1)
	require.NoError(t, err)
	require.Equal(t, "0", list.Items[0].Valuation.Amount)
}

func TestListHoldings_HoldingDAOError(t *testing.T) {
	initServiceTestEnv(t)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	assetDao := new(daomocks.AssetDao)
	svc := &HoldingServiceImpl{holdingDao: holdingDao, assetDao: assetDao}
	holdingDao.On("ListByUser", mock.Anything, int64(1), int64(0)).Return(nil, errors.New("db"))

	_, err := svc.ListHoldings(context.Background(), 1, 0)
	require.Error(t, err)
}

func TestListHoldings_AssetDAOError(t *testing.T) {
	initServiceTestEnv(t)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	assetDao := new(daomocks.AssetDao)
	svc := &HoldingServiceImpl{holdingDao: holdingDao, assetDao: assetDao}
	holdingDao.On("ListByUser", mock.Anything, int64(1), int64(0)).
		Return([]*model.UserAssetHolding{{UserID: 1, AssetID: 1}}, nil)
	assetDao.On("GetByIDs", mock.Anything, []int64{1}).Return(nil, errors.New("db"))

	_, err := svc.ListHoldings(context.Background(), 1, 0)
	require.Error(t, err)
}
