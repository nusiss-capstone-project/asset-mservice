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

type FiatTransactionDao interface {
	Create(ctx context.Context, tx *gorm.DB, txn *model.FiatTransaction) error
	GetByID(ctx context.Context, id int64) (*model.FiatTransaction, error)
	GetByIdempotentKey(ctx context.Context, userID int64, key string) (*model.FiatTransaction, error)
	UpdatePaymentResult(ctx context.Context, tx *gorm.DB, id int64, paymentID, fromStatus, toStatus, failureReason string) (int64, error)
	UpdateAccountID(ctx context.Context, tx *gorm.DB, id, accountID int64) error
}

type FiatTransactionDaoImpl struct {
	db *gorm.DB
}

var (
	fiatTxnOnce sync.Once
	fiatTxnDao  *FiatTransactionDaoImpl
)

func GetFiatTransactionDao() *FiatTransactionDaoImpl {
	fiatTxnOnce.Do(func() {
		fiatTxnDao = &FiatTransactionDaoImpl{db: repository.DB}
	})
	return fiatTxnDao
}

func (dao *FiatTransactionDaoImpl) dbOr(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return dao.db
}

func (dao *FiatTransactionDaoImpl) Create(ctx context.Context, tx *gorm.DB, txn *model.FiatTransaction) error {
	if err := dao.dbOr(tx).WithContext(ctx).Create(txn).Error; err != nil {
		log.WithContext(ctx).Errorw("create fiat transaction failed",
			"user_id", txn.UserID,
			"transaction_no", txn.TransactionNo,
			"error", err,
		)
		return err
	}
	log.WithContext(ctx).Infow("create fiat transaction success",
		"user_id", txn.UserID,
		"transaction_no", txn.TransactionNo,
	)
	return nil
}

func (dao *FiatTransactionDaoImpl) GetByID(ctx context.Context, id int64) (*model.FiatTransaction, error) {
	var txn model.FiatTransaction
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&txn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get fiat transaction by id failed", "id", id, "error", err)
		return nil, err
	}
	return &txn, nil
}

func (dao *FiatTransactionDaoImpl) GetByIdempotentKey(ctx context.Context, userID int64, key string) (*model.FiatTransaction, error) {
	var txn model.FiatTransaction
	err := dao.db.WithContext(ctx).
		Where("user_id = ? AND idempotent_key = ?", userID, key).
		First(&txn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.WithContext(ctx).Errorw("get fiat transaction by idempotent_key failed",
			"user_id", userID,
			"idempotent_key", key,
			"error", err,
		)
		return nil, err
	}
	return &txn, nil
}

func (dao *FiatTransactionDaoImpl) UpdatePaymentResult(
	ctx context.Context,
	tx *gorm.DB,
	id int64,
	paymentID, fromStatus, toStatus, failureReason string,
) (int64, error) {
	updates := map[string]interface{}{
		"status": toStatus,
	}
	if paymentID != "" {
		updates["external_payment_id"] = paymentID
	}
	if failureReason != "" {
		updates["failure_reason"] = failureReason
	}
	ret := dao.dbOr(tx).WithContext(ctx).Model(&model.FiatTransaction{}).
		Where("id = ? AND status = ?", id, fromStatus).
		Updates(updates)
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("update fiat transaction payment result failed",
			"id", id,
			"from_status", fromStatus,
			"to_status", toStatus,
			"error", ret.Error,
		)
		return 0, ret.Error
	}
	return ret.RowsAffected, nil
}

func (dao *FiatTransactionDaoImpl) UpdateAccountID(ctx context.Context, tx *gorm.DB, id, accountID int64) error {
	ret := dao.dbOr(tx).WithContext(ctx).Model(&model.FiatTransaction{}).
		Where("id = ?", id).
		Update("account_id", accountID)
	if ret.Error != nil {
		log.WithContext(ctx).Errorw("update fiat transaction account_id failed",
			"id", id,
			"account_id", accountID,
			"error", ret.Error,
		)
		return ret.Error
	}
	return nil
}
