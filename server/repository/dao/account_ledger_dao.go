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

type AccountLedgerDao interface {
	Create(ctx context.Context, tx *gorm.DB, ledger *model.AccountLedger) error
	GetByBusinessKey(ctx context.Context, tx *gorm.DB, businessType, businessID, assetCode string) (*model.AccountLedger, error)
}

type AccountLedgerDaoImpl struct {
	db *gorm.DB
}

var (
	accountLedgerOnce sync.Once
	accountLedgerDao  *AccountLedgerDaoImpl
)

func GetAccountLedgerDao() *AccountLedgerDaoImpl {
	accountLedgerOnce.Do(func() {
		accountLedgerDao = &AccountLedgerDaoImpl{db: repository.DB}
	})
	return accountLedgerDao
}

func (dao *AccountLedgerDaoImpl) dbOr(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return dao.db
}

func (dao *AccountLedgerDaoImpl) Create(ctx context.Context, tx *gorm.DB, ledger *model.AccountLedger) error {
	if err := dao.dbOr(tx).WithContext(ctx).Create(ledger).Error; err != nil {
		log.WithContext(ctx).Errorw("create account ledger failed",
			"user_id", ledger.UserID,
			"ledger_no", ledger.LedgerNo,
			"business_type", ledger.BusinessType,
			"business_id", ledger.BusinessID,
			"asset_code", ledger.AssetCode,
			"error", err,
		)
		return err
	}
	log.WithContext(ctx).Infow("account ledger created",
		"ledger_id", ledger.ID,
		"ledger_no", ledger.LedgerNo,
		"user_id", ledger.UserID,
		"business_type", ledger.BusinessType,
		"business_id", ledger.BusinessID,
		"asset_code", ledger.AssetCode,
		"change_amount", ledger.ChangeAmount.String(),
		"balance_after", ledger.BalanceAfter.String(),
	)
	return nil
}

func (dao *AccountLedgerDaoImpl) GetByBusinessKey(
	ctx context.Context,
	tx *gorm.DB,
	businessType, businessID, assetCode string,
) (*model.AccountLedger, error) {
	var ledger model.AccountLedger
	err := dao.dbOr(tx).WithContext(ctx).
		Where("business_type = ? AND business_id = ? AND asset_code = ?", businessType, businessID, assetCode).
		First(&ledger).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get account ledger by business key failed",
			"business_type", businessType,
			"business_id", businessID,
			"asset_code", assetCode,
			"error", err,
		)
		return nil, err
	}
	return &ledger, nil
}
