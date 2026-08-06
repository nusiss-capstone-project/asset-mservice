package service

import (
	"context"
	"errors"
	"testing"

	daomocks "github.com/nusiss-capstone-project/asset-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newRewardServiceForTest(
	assetDao *daomocks.AssetDao,
	holdingDao *daomocks.UserAssetHoldingDao,
	ledgerDao *daomocks.AccountLedgerDao,
) *RewardServiceImpl {
	return &RewardServiceImpl{
		assetDao:   assetDao,
		holdingDao: holdingDao,
		ledgerDao:  ledgerDao,
	}
}

func TestReward_ValidationErrors(t *testing.T) {
	initServiceTestEnv(t)
	svc := newRewardServiceForTest(new(daomocks.AssetDao), new(daomocks.UserAssetHoldingDao), new(daomocks.AccountLedgerDao))

	_, err := svc.Reward(context.Background(), RewardInput{})
	require.ErrorIs(t, err, ErrRewardInvalidArgument)

	_, err = svc.Reward(context.Background(), RewardInput{BizID: "b1", UserID: 0, AssetCode: "BTC", Amount: "1"})
	require.ErrorIs(t, err, ErrRewardInvalidArgument)

	_, err = svc.Reward(context.Background(), RewardInput{BizID: "b1", UserID: 1, AssetCode: "", Amount: "1"})
	require.ErrorIs(t, err, ErrRewardInvalidArgument)

	_, err = svc.Reward(context.Background(), RewardInput{BizID: "b1", UserID: 1, AssetCode: "BTC", Amount: "-1"})
	require.ErrorIs(t, err, ErrRewardInvalidArgument)
}

func TestReward_IdempotentHit(t *testing.T) {
	initServiceTestEnv(t)
	ledgerDao := new(daomocks.AccountLedgerDao)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	svc := newRewardServiceForTest(new(daomocks.AssetDao), holdingDao, ledgerDao)
	ledgerDao.On("GetByBusinessKey", mock.Anything, (*gorm.DB)(nil), model.LedgerBusinessTypeReward, "reward-1", "BTC").
		Return(&model.AccountLedger{LedgerNo: "AL_EXISTING"}, nil)

	result, err := svc.Reward(context.Background(), RewardInput{
		BizID: "reward-1", UserID: 42, AssetCode: "btc", Amount: "0.1",
	})
	require.NoError(t, err)
	require.Equal(t, "AL_EXISTING", result.TransactionID)
	holdingDao.AssertNotCalled(t, "UpsertAddQuantity", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReward_AssetNotFound(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	ledgerDao := new(daomocks.AccountLedgerDao)
	svc := newRewardServiceForTest(assetDao, new(daomocks.UserAssetHoldingDao), ledgerDao)
	ledgerDao.On("GetByBusinessKey", mock.Anything, (*gorm.DB)(nil), model.LedgerBusinessTypeReward, "reward-1", "BTC").
		Return(nil, nil)
	assetDao.On("GetBySymbol", mock.Anything, "BTC").Return(nil, nil)

	_, err := svc.Reward(context.Background(), RewardInput{
		BizID: "reward-1", UserID: 42, AssetCode: "BTC", Amount: "0.1",
	})
	require.ErrorIs(t, err, ErrRewardAssetNotFound)
}

func TestReward_Success(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	ledgerDao := new(daomocks.AccountLedgerDao)
	svc := newRewardServiceForTest(assetDao, holdingDao, ledgerDao)

	ledgerDao.On("GetByBusinessKey", mock.Anything, (*gorm.DB)(nil), model.LedgerBusinessTypeReward, "reward-1", "BTC").
		Return(nil, nil)
	assetDao.On("GetBySymbol", mock.Anything, "BTC").Return(sampleAsset(1), nil)
	holdingDao.On("UpsertAddQuantity", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(42), int64(1), mock.Anything).
		Return(decimal.RequireFromString("1.1"), nil)
	ledgerDao.On("Create", mock.Anything, mock.AnythingOfType("*gorm.DB"), mock.MatchedBy(func(l *model.AccountLedger) bool {
		return l.BusinessType == model.LedgerBusinessTypeReward &&
			l.BusinessID == "reward-1" &&
			l.AssetCode == "BTC" &&
			l.UserID == 42 &&
			l.ChangeAmount.Equal(decimal.RequireFromString("0.1")) &&
			l.BalanceAfter.Equal(decimal.RequireFromString("1.1")) &&
			l.LedgerNo != ""
	})).Return(nil)

	result, err := svc.Reward(context.Background(), RewardInput{
		BizID: "reward-1", UserID: 42, AssetCode: "BTC", Amount: "0.1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.TransactionID)
	ledgerDao.AssertExpectations(t)
	holdingDao.AssertExpectations(t)
}

func TestReward_HoldingError(t *testing.T) {
	initServiceTestEnv(t)
	assetDao := new(daomocks.AssetDao)
	holdingDao := new(daomocks.UserAssetHoldingDao)
	ledgerDao := new(daomocks.AccountLedgerDao)
	svc := newRewardServiceForTest(assetDao, holdingDao, ledgerDao)

	ledgerDao.On("GetByBusinessKey", mock.Anything, (*gorm.DB)(nil), model.LedgerBusinessTypeReward, "reward-1", "BTC").
		Return(nil, nil)
	assetDao.On("GetBySymbol", mock.Anything, "BTC").Return(sampleAsset(1), nil)
	holdingDao.On("UpsertAddQuantity", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(42), int64(1), mock.Anything).
		Return(decimal.Zero, errors.New("holding failed"))

	_, err := svc.Reward(context.Background(), RewardInput{
		BizID: "reward-1", UserID: 42, AssetCode: "BTC", Amount: "0.1",
	})
	require.ErrorContains(t, err, "holding failed")
	ledgerDao.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}
