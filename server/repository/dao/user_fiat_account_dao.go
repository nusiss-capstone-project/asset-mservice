package dao

import (
	"context"
	"errors"
	"sync"

	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserFiatAccountDao interface {
	ListByUser(ctx context.Context, userID int64) ([]*model.UserFiatAccount, error)
	GetByUserAndCurrency(ctx context.Context, userID int64, currency string) (*model.UserFiatAccount, error)
	UpsertAddBalance(ctx context.Context, tx *gorm.DB, userID int64, currency string, amount decimal.Decimal) (decimal.Decimal, int64, error)
}

type UserFiatAccountDaoImpl struct {
	db *gorm.DB
}

var (
	fiatAccountOnce sync.Once
	fiatAccountDao  *UserFiatAccountDaoImpl
)

func GetUserFiatAccountDao() *UserFiatAccountDaoImpl {
	fiatAccountOnce.Do(func() {
		fiatAccountDao = &UserFiatAccountDaoImpl{db: repository.DB}
	})
	return fiatAccountDao
}

func (dao *UserFiatAccountDaoImpl) dbOr(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return dao.db
}

func (dao *UserFiatAccountDaoImpl) ListByUser(ctx context.Context, userID int64) ([]*model.UserFiatAccount, error) {
	var accounts []*model.UserFiatAccount
	if err := dao.db.WithContext(ctx).Where("user_id = ?", userID).Order("id asc").Find(&accounts).Error; err != nil {
		log.WithContext(ctx).Errorw("list user fiat accounts failed", "user_id", userID, "error", err)
		return nil, err
	}
	return accounts, nil
}

func (dao *UserFiatAccountDaoImpl) GetByUserAndCurrency(ctx context.Context, userID int64, currency string) (*model.UserFiatAccount, error) {
	var account model.UserFiatAccount
	err := dao.db.WithContext(ctx).
		Where("user_id = ? AND currency = ?", userID, currency).
		First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get user fiat account failed",
			"user_id", userID,
			"currency", currency,
			"error", err,
		)
		return nil, err
	}
	return &account, nil
}

// UpsertAddBalance adds amount to the fiat balance and returns (balance_after, account_id).
func (dao *UserFiatAccountDaoImpl) UpsertAddBalance(
	ctx context.Context,
	tx *gorm.DB,
	userID int64,
	currency string,
	amount decimal.Decimal,
) (decimal.Decimal, int64, error) {
	account := &model.UserFiatAccount{
		UserID:   userID,
		Currency: currency,
		Balance:  amount,
	}
	db := dao.dbOr(tx).WithContext(ctx)
	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "currency"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"balance":    gorm.Expr("balance + ?", amount),
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(account).Error
	if err != nil {
		log.WithContext(ctx).Errorw("upsert user fiat account failed",
			"user_id", userID,
			"currency", currency,
			"amount", amount.String(),
			"error", err,
		)
		return decimal.Zero, 0, err
	}

	var updated model.UserFiatAccount
	if err := db.Where("user_id = ? AND currency = ?", userID, currency).First(&updated).Error; err != nil {
		log.WithContext(ctx).Errorw("read user fiat account after upsert failed",
			"user_id", userID,
			"currency", currency,
			"error", err,
		)
		return decimal.Zero, 0, err
	}
	return updated.Balance, updated.ID, nil
}
