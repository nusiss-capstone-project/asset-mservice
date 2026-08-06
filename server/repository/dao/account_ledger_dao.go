package dao

import (
	"context"
	"sync"

	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"gorm.io/gorm"
)

type AccountLedgerDao interface {
	Create(ctx context.Context, tx *gorm.DB, ledger *model.AccountLedger) error
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
