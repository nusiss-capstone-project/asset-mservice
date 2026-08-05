package service

import (
	"context"
	"errors"
	"testing"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	daomocks "github.com/nusiss-capstone-project/asset-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	cacheredis "github.com/nusiss-capstone-project/asset-mservice/server/repository/redis"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGenQuote_RedisDisabled(t *testing.T) {
	initServiceTestEnv(t)
	cacheredis.Client = nil
	svc := &QuoteServiceImpl{assetDao: new(daomocks.AssetDao)}
	_, err := svc.GenQuote(context.Background(), 1, &data.GenQuoteRequest{AssetID: "1"})
	require.ErrorContains(t, err, "redis disabled")
}

func TestGenQuote_InvalidAssetID(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	svc := &QuoteServiceImpl{assetDao: new(daomocks.AssetDao)}
	_, err := svc.GenQuote(context.Background(), 1, &data.GenQuoteRequest{AssetID: "x"})
	require.ErrorContains(t, err, "invalid asset_id")
}

func TestGenQuote_Success(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	assetDao := new(daomocks.AssetDao)
	svc := &QuoteServiceImpl{assetDao: assetDao}
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(sampleAsset(1), nil)

	quote, err := svc.GenQuote(context.Background(), 42, &data.GenQuoteRequest{AssetID: "1", Currency: "USD"})
	require.NoError(t, err)
	require.NotEmpty(t, quote.QuoteID)
	require.Equal(t, "100.00", quote.UnitPrice)
	require.Greater(t, quote.ExpiresAt, int64(0))

	cached, err := cacheredis.GetQuote(context.Background(), quote.QuoteID)
	require.NoError(t, err)
	require.Equal(t, int64(42), cached.UserID)
}

func TestGenQuote_AssetNotFound(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	assetDao := new(daomocks.AssetDao)
	svc := &QuoteServiceImpl{assetDao: assetDao}
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(nil, nil)

	_, err := svc.GenQuote(context.Background(), 1, &data.GenQuoteRequest{AssetID: "1"})
	require.ErrorContains(t, err, "not found")
}

func TestGenQuote_InactiveAsset(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	assetDao := new(daomocks.AssetDao)
	svc := &QuoteServiceImpl{assetDao: assetDao}
	asset := sampleAsset(1)
	asset.Status = model.AssetStatusInactive
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(asset, nil)

	_, err := svc.GenQuote(context.Background(), 1, &data.GenQuoteRequest{AssetID: "1"})
	require.ErrorContains(t, err, "not active")
}

func TestGenQuote_CurrencyMismatch(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	assetDao := new(daomocks.AssetDao)
	svc := &QuoteServiceImpl{assetDao: assetDao}
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(sampleAsset(1), nil)

	_, err := svc.GenQuote(context.Background(), 1, &data.GenQuoteRequest{AssetID: "1", Currency: "SGD"})
	require.ErrorContains(t, err, "mismatch")
}

func TestGenQuote_DAOError(t *testing.T) {
	initServiceTestEnv(t)
	startTestRedis(t)
	assetDao := new(daomocks.AssetDao)
	svc := &QuoteServiceImpl{assetDao: assetDao}
	assetDao.On("GetByID", mock.Anything, int64(1)).Return(nil, errors.New("db"))

	_, err := svc.GenQuote(context.Background(), 1, &data.GenQuoteRequest{AssetID: "1"})
	require.Error(t, err)
}
