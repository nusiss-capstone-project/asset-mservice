package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/nusiss-capstone-project/asset-mservice/server/util"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrRewardInvalidArgument = errors.New("invalid argument")
	ErrRewardAssetNotFound   = errors.New("asset not found")
)

type RewardInput struct {
	BizID     string
	UserID    int64
	AssetCode string
	Amount    string
}

type RewardResult struct {
	TransactionID string // ledger_no
}

// Rewarder credits asset holdings for reward business events.
type Rewarder interface {
	Reward(ctx context.Context, in RewardInput) (*RewardResult, error)
}

type RewardServiceImpl struct {
	assetDao   dao.AssetDao
	holdingDao dao.UserAssetHoldingDao
	ledgerDao  dao.AccountLedgerDao
}

var (
	rewardServiceOnce sync.Once
	rewardServiceInst Rewarder
)

func GetRewardService() Rewarder {
	rewardServiceOnce.Do(func() {
		rewardServiceInst = &RewardServiceImpl{
			assetDao:   dao.GetAssetDao(),
			holdingDao: dao.GetUserAssetHoldingDao(),
			ledgerDao:  dao.GetAccountLedgerDao(),
		}
	})
	return rewardServiceInst
}

func (s *RewardServiceImpl) Reward(ctx context.Context, in RewardInput) (*RewardResult, error) {
	bizID, assetCode, amount, err := parseRewardInput(in)
	if err != nil {
		return nil, err
	}

	existing, err := s.ledgerDao.GetByBusinessKey(ctx, nil, model.LedgerBusinessTypeReward, bizID, assetCode)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		log.WithContext(ctx).Infow("reward idempotent hit",
			"biz_id", bizID,
			"asset_code", assetCode,
			"ledger_no", existing.LedgerNo,
		)
		return &RewardResult{TransactionID: existing.LedgerNo}, nil
	}

	asset, err := s.loadActiveRewardAsset(ctx, assetCode)
	if err != nil {
		return nil, err
	}

	ledgerNo, err := s.creditReward(ctx, in.UserID, bizID, assetCode, asset.ID, amount)
	if err != nil {
		return s.rewardOnDuplicate(ctx, bizID, assetCode, err)
	}
	return &RewardResult{TransactionID: ledgerNo}, nil
}

func parseRewardInput(in RewardInput) (bizID, assetCode string, amount decimal.Decimal, err error) {
	bizID = strings.TrimSpace(in.BizID)
	assetCode = strings.ToUpper(strings.TrimSpace(in.AssetCode))
	if bizID == "" {
		return "", "", decimal.Zero, fmt.Errorf("%w: biz_id is required", ErrRewardInvalidArgument)
	}
	if in.UserID <= 0 {
		return "", "", decimal.Zero, fmt.Errorf("%w: user_id is required", ErrRewardInvalidArgument)
	}
	if assetCode == "" {
		return "", "", decimal.Zero, fmt.Errorf("%w: asset_code is required", ErrRewardInvalidArgument)
	}
	amount, err = decimal.NewFromString(strings.TrimSpace(in.Amount))
	if err != nil || !amount.IsPositive() {
		return "", "", decimal.Zero, fmt.Errorf("%w: invalid amount", ErrRewardInvalidArgument)
	}
	return bizID, assetCode, amount, nil
}

func (s *RewardServiceImpl) loadActiveRewardAsset(ctx context.Context, assetCode string) (*model.Asset, error) {
	asset, err := s.assetDao.GetBySymbol(ctx, assetCode)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, fmt.Errorf("%w: symbol %s", ErrRewardAssetNotFound, assetCode)
	}
	if asset.Status != model.AssetStatusActive {
		return nil, fmt.Errorf("%w: asset %s is not active", ErrRewardInvalidArgument, assetCode)
	}
	return asset, nil
}

func (s *RewardServiceImpl) creditReward(
	ctx context.Context, userID int64, bizID, assetCode string, assetID int64, amount decimal.Decimal,
) (string, error) {
	var ledgerNo string
	err := repository.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		balanceAfter, err := s.holdingDao.UpsertAddQuantity(ctx, tx, userID, assetID, amount)
		if err != nil {
			return err
		}
		ledgerNo, err = util.NewLedgerNo()
		if err != nil {
			return err
		}
		return s.ledgerDao.Create(ctx, tx, &model.AccountLedger{
			LedgerNo:     ledgerNo,
			UserID:       userID,
			AssetCode:    assetCode,
			ChangeAmount: amount,
			BusinessType: model.LedgerBusinessTypeReward,
			BusinessID:   bizID,
			BalanceAfter: balanceAfter,
		})
	})
	return ledgerNo, err
}

func (s *RewardServiceImpl) rewardOnDuplicate(ctx context.Context, bizID, assetCode string, err error) (*RewardResult, error) {
	// Concurrent reward: unique key conflict rolls back holding+ledger; return existing.
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil, err
	}
	existing, lookupErr := s.ledgerDao.GetByBusinessKey(ctx, nil, model.LedgerBusinessTypeReward, bizID, assetCode)
	if lookupErr != nil {
		return nil, lookupErr
	}
	if existing != nil {
		return &RewardResult{TransactionID: existing.LedgerNo}, nil
	}
	return nil, err
}
