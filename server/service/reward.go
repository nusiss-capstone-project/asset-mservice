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

type RewardService interface {
	Reward(ctx context.Context, in RewardInput) (*RewardResult, error)
}

type RewardServiceImpl struct {
	assetDao   dao.AssetDao
	holdingDao dao.UserAssetHoldingDao
	ledgerDao  dao.AccountLedgerDao
}

var (
	rewardServiceOnce sync.Once
	rewardServiceInst RewardService
)

func GetRewardService() RewardService {
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
	bizID := strings.TrimSpace(in.BizID)
	assetCode := strings.ToUpper(strings.TrimSpace(in.AssetCode))
	if bizID == "" {
		return nil, fmt.Errorf("%w: biz_id is required", ErrRewardInvalidArgument)
	}
	if in.UserID <= 0 {
		return nil, fmt.Errorf("%w: user_id is required", ErrRewardInvalidArgument)
	}
	if assetCode == "" {
		return nil, fmt.Errorf("%w: asset_code is required", ErrRewardInvalidArgument)
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(in.Amount))
	if err != nil || !amount.IsPositive() {
		return nil, fmt.Errorf("%w: invalid amount", ErrRewardInvalidArgument)
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

	var ledgerNo string
	err = repository.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		balanceAfter, err := s.holdingDao.UpsertAddQuantity(ctx, tx, in.UserID, asset.ID, amount)
		if err != nil {
			return err
		}
		ledgerNo, err = util.NewLedgerNo()
		if err != nil {
			return err
		}
		ledger := &model.AccountLedger{
			LedgerNo:     ledgerNo,
			UserID:       in.UserID,
			AssetCode:    assetCode,
			ChangeAmount: amount,
			BusinessType: model.LedgerBusinessTypeReward,
			BusinessID:   bizID,
			BalanceAfter: balanceAfter,
		}
		return s.ledgerDao.Create(ctx, tx, ledger)
	})
	if err != nil {
		// Concurrent reward: unique key conflict rolls back holding+ledger; return existing.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, lookupErr := s.ledgerDao.GetByBusinessKey(ctx, nil, model.LedgerBusinessTypeReward, bizID, assetCode)
			if lookupErr != nil {
				return nil, lookupErr
			}
			if existing != nil {
				return &RewardResult{TransactionID: existing.LedgerNo}, nil
			}
		}
		return nil, err
	}
	return &RewardResult{TransactionID: ledgerNo}, nil
}
