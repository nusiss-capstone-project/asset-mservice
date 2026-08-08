package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	kproducer "github.com/nusiss-capstone-project/asset-mservice/server/kafka/producer"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/market"
	"github.com/nusiss-capstone-project/asset-mservice/server/proxy"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/nusiss-capstone-project/asset-mservice/server/util"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ListLedgersQuery struct {
	UserID       int64
	AssetCode    string
	BusinessType string
	CreatedFrom  int64
	CreatedTo    int64
	Cursor       int64
	Limit        int
}

type DepositService interface {
	CreateDeposit(ctx context.Context, userID int64, req *data.CreateDepositRequest) (*data.FiatTransactionVO, error)
	ListFiatAccounts(ctx context.Context, userID int64, currenciesFilter string) (*data.FiatAccountListVO, error)
	ListLedgers(ctx context.Context, query ListLedgersQuery) (*data.LedgerListVO, error)
}

type DepositServiceImpl struct {
	fiatTxnDao                   dao.FiatTransactionDao
	fiatAccountDao               dao.UserFiatAccountDao
	ledgerDao                    dao.AccountLedgerDao
	userProxy                    proxy.UserProxy
	paymentProxy                 proxy.PaymentProxy
	depositPaymentResultProducer kproducer.DepositPaymentResultProducer
}

var (
	depositServiceOnce sync.Once
	depositServiceInst DepositService
)

func GetDepositService() DepositService {
	depositServiceOnce.Do(func() {
		depositServiceInst = &DepositServiceImpl{
			fiatTxnDao:                   dao.GetFiatTransactionDao(),
			fiatAccountDao:               dao.GetUserFiatAccountDao(),
			ledgerDao:                    dao.GetAccountLedgerDao(),
			userProxy:                    proxy.GetUserProxy(),
			paymentProxy:                 proxy.GetPaymentProxy(),
			depositPaymentResultProducer: kproducer.GetDepositPaymentResultProducer(),
		}
	})
	return depositServiceInst
}

func (s *DepositServiceImpl) CreateDeposit(ctx context.Context, userID int64, req *data.CreateDepositRequest) (*data.FiatTransactionVO, error) {
	idempotentKey, currency, amount, err := parseCreateDepositRequest(req)
	if err != nil {
		return nil, err
	}

	existing, err := s.fiatTxnDao.GetByIdempotentKey(ctx, userID, idempotentKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return toFiatTransactionVO(existing), nil
	}

	if err := s.ensureDepositCurrencyAllowed(ctx, userID, currency); err != nil {
		return nil, err
	}

	minorAmount, err := util.ToMinorUnits(strings.TrimSpace(req.Amount), currency)
	if err != nil {
		return nil, err
	}

	txn, err := s.createPendingDeposit(ctx, userID, idempotentKey, currency, amount, req.PaymentMethodID)
	if err != nil {
		return s.createDepositOnDuplicate(ctx, userID, idempotentKey, err)
	}

	payResult, err := s.paymentProxy.CreatePayment(ctx, proxy.CreatePaymentRequest{
		BizID:           txn.TransactionNo,
		UserID:          userID,
		Amount:          minorAmount,
		Currency:        currency,
		PaymentMethodID: req.PaymentMethodID,
	})
	if err != nil {
		_ = s.applyDepositPaymentResult(ctx, txn, "", model.FiatTxnStatusFailed, err.Error())
		log.WithContext(ctx).Errorw("create payment failed",
			"user_id", userID,
			"transaction_no", txn.TransactionNo,
			"error", err,
		)
		return nil, err
	}

	if err := s.applyDepositPaymentResult(ctx, txn, payResult.PaymentID, mapDepositPaymentStatus(payResult.Status), ""); err != nil {
		return nil, err
	}

	latest, err := s.fiatTxnDao.GetByID(ctx, txn.ID)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return toFiatTransactionVO(txn), nil
	}
	return toFiatTransactionVO(latest), nil
}

func parseCreateDepositRequest(req *data.CreateDepositRequest) (idempotentKey, currency string, amount decimal.Decimal, err error) {
	idempotentKey = strings.TrimSpace(req.IdempotentKey)
	currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if idempotentKey == "" {
		return "", "", decimal.Zero, fmt.Errorf("idempotent_key is required")
	}
	if req.PaymentMethodID <= 0 {
		return "", "", decimal.Zero, fmt.Errorf("payment_method_id is required")
	}
	if currency == "" {
		return "", "", decimal.Zero, fmt.Errorf("currency is required")
	}
	amount, err = decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || !amount.IsPositive() {
		return "", "", decimal.Zero, fmt.Errorf("invalid amount")
	}
	return idempotentKey, currency, amount, nil
}

func (s *DepositServiceImpl) ensureDepositCurrencyAllowed(ctx context.Context, userID int64, currency string) error {
	profile, err := s.userProxy.GetUserProfile(ctx, userID)
	if err != nil {
		return err
	}
	if !market.SupportsCurrency(profile.GetMarket(), currency) {
		return fmt.Errorf("currency %s is not supported for market %s", currency, profile.GetMarket())
	}
	return nil
}

func (s *DepositServiceImpl) createPendingDeposit(
	ctx context.Context, userID int64, idempotentKey, currency string, amount decimal.Decimal, paymentMethodID int64,
) (*model.FiatTransaction, error) {
	txnNo, err := util.NewFiatTransactionNo()
	if err != nil {
		return nil, err
	}
	txn := &model.FiatTransaction{
		TransactionNo:   txnNo,
		UserID:          userID,
		IdempotentKey:   idempotentKey,
		Amount:          amount,
		Currency:        currency,
		TransactionType: model.FiatTxnTypeDeposit,
		Status:          model.FiatTxnStatusPending,
		PaymentMethodID: paymentMethodID,
	}
	if err := s.fiatTxnDao.Create(ctx, nil, txn); err != nil {
		return nil, err
	}
	return txn, nil
}

func (s *DepositServiceImpl) createDepositOnDuplicate(
	ctx context.Context, userID int64, idempotentKey string, err error,
) (*data.FiatTransactionVO, error) {
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil, err
	}
	existing, lookupErr := s.fiatTxnDao.GetByIdempotentKey(ctx, userID, idempotentKey)
	if lookupErr != nil {
		return nil, lookupErr
	}
	if existing != nil {
		return toFiatTransactionVO(existing), nil
	}
	return nil, err
}

func (s *DepositServiceImpl) applyDepositPaymentResult(
	ctx context.Context,
	txn *model.FiatTransaction,
	paymentID, status, failureReason string,
) error {
	if status != model.FiatTxnStatusSucceeded {
		return s.markDepositNotSucceeded(ctx, txn, paymentID, status, failureReason)
	}
	return s.settleSucceededDeposit(ctx, txn, paymentID)
}

func (s *DepositServiceImpl) markDepositNotSucceeded(
	ctx context.Context, txn *model.FiatTransaction, paymentID, status, failureReason string,
) error {
	rows, err := s.fiatTxnDao.UpdatePaymentResult(
		ctx, nil, txn.ID, paymentID, model.FiatTxnStatusPending, status, failureReason,
	)
	if err != nil {
		return err
	}
	if rows == 0 {
		return nil
	}
	log.WithContext(ctx).Infow("update payment result success",
		"user_id", txn.UserID,
		"transaction_no", txn.TransactionNo,
		"payment_id", paymentID,
		"status", status,
		"failure_reason", failureReason,
	)
	return s.publishDepositPaymentResult(ctx, txn, status)
}

func (s *DepositServiceImpl) settleSucceededDeposit(ctx context.Context, txn *model.FiatTransaction, paymentID string) error {
	var accountID int64
	err := repository.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.creditDepositInTx(ctx, tx, txn, paymentID, &accountID)
	})
	if err != nil {
		if errors.Is(err, errDepositAlreadySettled) {
			return nil
		}
		return err
	}
	txn.AccountID = accountID
	txn.ExternalPaymentID = paymentID
	txn.Status = model.FiatTxnStatusSucceeded
	return s.publishDepositPaymentResult(ctx, txn, model.FiatTxnStatusSucceeded)
}

func (s *DepositServiceImpl) creditDepositInTx(
	ctx context.Context, tx *gorm.DB, txn *model.FiatTransaction, paymentID string, accountID *int64,
) error {
	rows, err := s.fiatTxnDao.UpdatePaymentResult(
		ctx, tx, txn.ID, paymentID, model.FiatTxnStatusPending, model.FiatTxnStatusSucceeded, "",
	)
	if err != nil {
		return err
	}
	if rows == 0 {
		return errDepositAlreadySettled
	}
	balanceAfter, acctID, err := s.fiatAccountDao.UpsertAddBalance(ctx, tx, txn.UserID, txn.Currency, txn.Amount)
	if err != nil {
		return err
	}
	*accountID = acctID
	if err := s.fiatTxnDao.UpdateAccountID(ctx, tx, txn.ID, acctID); err != nil {
		return err
	}
	ledgerNo, err := util.NewLedgerNo()
	if err != nil {
		return err
	}
	return s.ledgerDao.Create(ctx, tx, &model.AccountLedger{
		LedgerNo:     ledgerNo,
		UserID:       txn.UserID,
		AssetCode:    txn.Currency,
		ChangeAmount: txn.Amount,
		BusinessType: model.LedgerBusinessTypeDeposit,
		BusinessID:   txn.TransactionNo,
		BalanceAfter: balanceAfter,
	})
}

var errDepositAlreadySettled = errors.New("deposit already settled")

func (s *DepositServiceImpl) publishDepositPaymentResult(ctx context.Context, txn *model.FiatTransaction, status string) error {
	event := kproducer.DepositPaymentResultEvent{
		UserID:        txn.UserID,
		TransactionID: txn.ID,
		Status:        status,
		Currency:      txn.Currency,
		Amount:        txn.Amount.String(),
		EventTime:     time.Now().Unix(),
	}
	if err := s.depositPaymentResultProducer.PublishDepositPaymentResult(ctx, event); err != nil {
		log.WithContext(ctx).Errorw("publish deposit payment result failed",
			"transaction_id", txn.ID,
			"transaction_no", txn.TransactionNo,
			"status", status,
			"error", err,
		)
		return err
	}
	log.WithContext(ctx).Infow("publish deposit payment result success",
		"transaction_id", txn.ID,
		"transaction_no", txn.TransactionNo,
		"status", status,
	)
	return nil
}

func (s *DepositServiceImpl) ListFiatAccounts(ctx context.Context, userID int64, currenciesFilter string) (*data.FiatAccountListVO, error) {
	profile, err := s.userProxy.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	supported, err := market.CurrenciesForMarket(profile.GetMarket())
	if err != nil {
		return nil, err
	}

	wanted := filterFiatCurrencies(profile.GetMarket(), supported, currenciesFilter)
	accounts, err := s.fiatAccountDao.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &data.FiatAccountListVO{Items: buildFiatAccountItems(wanted, accounts)}, nil
}

func filterFiatCurrencies(marketCode string, supported []string, currenciesFilter string) []string {
	filter := strings.TrimSpace(currenciesFilter)
	if filter == "" {
		return supported
	}
	wanted := make([]string, 0)
	for _, part := range strings.Split(filter, ",") {
		c := strings.ToUpper(strings.TrimSpace(part))
		if c == "" || !market.SupportsCurrency(marketCode, c) {
			continue
		}
		wanted = append(wanted, c)
	}
	return wanted
}

func buildFiatAccountItems(wanted []string, accounts []*model.UserFiatAccount) []*data.FiatAccountVO {
	byCurrency := make(map[string]*model.UserFiatAccount, len(accounts))
	for _, a := range accounts {
		byCurrency[a.Currency] = a
	}
	items := make([]*data.FiatAccountVO, 0, len(wanted))
	for _, currency := range wanted {
		if a, ok := byCurrency[currency]; ok {
			items = append(items, &data.FiatAccountVO{
				AccountID: strconv.FormatInt(a.ID, 10),
				Currency:  a.Currency,
				Balance:   a.Balance.StringFixed(2),
				UpdatedAt: a.UpdatedAt.Unix(),
			})
			continue
		}
		items = append(items, &data.FiatAccountVO{
			AccountID: "0",
			Currency:  currency,
			Balance:   "0.00",
			UpdatedAt: 0,
		})
	}
	return items
}

func (s *DepositServiceImpl) ListLedgers(ctx context.Context, query ListLedgersQuery) (*data.LedgerListVO, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	items, err := s.ledgerDao.ListByCursor(ctx, dao.AccountLedgerListFilter{
		UserID:       query.UserID,
		AssetCode:    strings.ToUpper(strings.TrimSpace(query.AssetCode)),
		BusinessType: strings.ToUpper(strings.TrimSpace(query.BusinessType)),
		CreatedFrom:  query.CreatedFrom,
		CreatedTo:    query.CreatedTo,
		Cursor:       query.Cursor,
		Limit:        limit + 1,
	})
	if err != nil {
		return nil, err
	}
	nextCursor := ""
	if len(items) > limit {
		nextCursor = strconv.FormatInt(items[limit-1].ID, 10)
		items = items[:limit]
	}
	out := make([]*data.LedgerVO, 0, len(items))
	for _, item := range items {
		out = append(out, toLedgerVO(item))
	}
	return &data.LedgerListVO{NextCursor: nextCursor, Items: out}, nil
}

func mapDepositPaymentStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCEEDED":
		return model.FiatTxnStatusSucceeded
	case "FAILED":
		return model.FiatTxnStatusFailed
	default:
		return model.FiatTxnStatusPending
	}
}

func toFiatTransactionVO(txn *model.FiatTransaction) *data.FiatTransactionVO {
	return &data.FiatTransactionVO{
		TransactionID:     strconv.FormatInt(txn.ID, 10),
		TransactionNo:     txn.TransactionNo,
		AccountID:         strconv.FormatInt(txn.AccountID, 10),
		UserID:            strconv.FormatInt(txn.UserID, 10),
		Currency:          txn.Currency,
		Amount:            txn.Amount.StringFixed(2),
		TransactionType:   txn.TransactionType,
		Status:            txn.Status,
		PaymentMethodID:   strconv.FormatInt(txn.PaymentMethodID, 10),
		ExternalPaymentID: txn.ExternalPaymentID,
		FailureReason:     txn.FailureReason,
		CreatedAt:         txn.CreatedAt.Unix(),
		UpdatedAt:         txn.UpdatedAt.Unix(),
	}
}

func toLedgerVO(l *model.AccountLedger) *data.LedgerVO {
	change := l.ChangeAmount.String()
	if l.ChangeAmount.IsPositive() {
		change = "+" + l.ChangeAmount.String()
	}
	return &data.LedgerVO{
		LedgerID:     strconv.FormatInt(l.ID, 10),
		LedgerNo:     l.LedgerNo,
		AssetCode:    l.AssetCode,
		ChangeAmount: change,
		BusinessType: l.BusinessType,
		BusinessID:   l.BusinessID,
		BalanceAfter: l.BalanceAfter.String(),
		CreatedAt:    l.CreatedAt.Unix(),
	}
}
