package service

import (
	"context"
	"errors"
	"testing"

	"github.com/nusiss-capstone-project/asset-mservice/server/repository/dao"
	daomocks "github.com/nusiss-capstone-project/asset-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListAssets_Success(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	svc := &AssetServiceImpl{assetDao: assetDao}

	assetDao.On("List", mock.Anything, dao.AssetListFilter{Currency: "USD", Status: model.AssetStatusActive}).
		Return([]*model.Asset{sampleAsset(1)}, nil)

	items, err := svc.ListAssets(context.Background(), "", model.AssetStatusActive)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "1", items[0].ID)
	require.Equal(t, "BTC", items[0].Symbol)
	require.Greater(t, items[0].CreatedAt, int64(0))
	assetDao.AssertExpectations(t)
}

func TestListAssets_DAOError(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	svc := &AssetServiceImpl{assetDao: assetDao}
	assetDao.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))

	items, err := svc.ListAssets(context.Background(), "usd", "")
	require.Error(t, err)
	require.Nil(t, items)
}

func TestGetAsset_Success(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	svc := &AssetServiceImpl{assetDao: assetDao}
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(sampleAsset(1), nil)

	item, err := svc.GetAsset(context.Background(), 1, "USD")
	require.NoError(t, err)
	require.Equal(t, "Bitcoin", item.Name)
}

func TestGetAsset_NotFound(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	svc := &AssetServiceImpl{assetDao: assetDao}
	assetDao.On("GetByID", mock.Anything, int64(9)).Return(nil, nil)

	_, err := svc.GetAsset(context.Background(), 9, "")
	require.ErrorContains(t, err, "not found")
}

func TestGetAsset_CurrencyMismatch(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	svc := &AssetServiceImpl{assetDao: assetDao}
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(sampleAsset(1), nil)

	_, err := svc.GetAsset(context.Background(), 1, "SGD")
	require.ErrorContains(t, err, "mismatch")
}

func TestGetAsset_DAOError(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	svc := &AssetServiceImpl{assetDao: assetDao}
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(nil, errors.New("db"))

	_, err := svc.GetAsset(context.Background(), 1, "USD")
	require.Error(t, err)
}

func TestNormalizeCurrency(t *testing.T) {
	require.Equal(t, "USD", normalizeCurrency(""))
	require.Equal(t, "SGD", normalizeCurrency(" sgd "))
}
