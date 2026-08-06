package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	kproducer "github.com/nusiss-capstone-project/asset-mservice/server/kafka/producer"
	producermocks "github.com/nusiss-capstone-project/asset-mservice/server/kafka/producer/mocks"
	"github.com/nusiss-capstone-project/asset-mservice/server/proxy"
	proxymocks "github.com/nusiss-capstone-project/asset-mservice/server/proxy/mocks"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/dao"
	daomocks "github.com/nusiss-capstone-project/asset-mservice/server/repository/dao/mocks"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	"github.com/nusiss-capstone-project/identity-mservice/common/identitypb"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDepositServiceForTest(
	fiatTxnDao *daomocks.FiatTransactionDao,
	fiatAccountDao *daomocks.UserFiatAccountDao,
	ledgerDao *daomocks.AccountLedgerDao,
	userProxy *proxymocks.UserProxy,
	paymentProxy *proxymocks.PaymentProxy,
	resultProducer *producermocks.DepositPaymentResultProducer,
) *DepositServiceImpl {
	return &DepositServiceImpl{
		fiatTxnDao:                   fiatTxnDao,
		fiatAccountDao:               fiatAccountDao,
		ledgerDao:                    ledgerDao,
		userProxy:                    userProxy,
		paymentProxy:                 paymentProxy,
		depositPaymentResultProducer: resultProducer,
	}
}

func sampleProfile(market string) *identitypb.GetUserProfileResponse {
	return &identitypb.GetUserProfileResponse{Market: market}
}

func TestCreateDeposit_ValidationErrors(t *testing.T) {
	initServiceTestEnv(t)
	svc := newDepositServiceForTest(
		new(daomocks.FiatTransactionDao), new(daomocks.UserFiatAccountDao), new(daomocks.AccountLedgerDao),
		new(proxymocks.UserProxy), new(proxymocks.PaymentProxy), new(producermocks.DepositPaymentResultProducer),
	)

	_, err := svc.CreateDeposit(context.Background(), 1, &data.CreateDepositRequest{})
	require.ErrorContains(t, err, "idempotent_key")

	_, err = svc.CreateDeposit(context.Background(), 1, &data.CreateDepositRequest{IdempotentKey: "k"})
	require.ErrorContains(t, err, "payment_method_id")

	_, err = svc.CreateDeposit(context.Background(), 1, &data.CreateDepositRequest{
		IdempotentKey: "k", PaymentMethodID: 1, Currency: "USD", Amount: "-1",
	})
	require.ErrorContains(t, err, "invalid amount")
}

func TestCreateDeposit_IdempotentHit(t *testing.T) {
	initServiceTestEnv(t)
	fiatTxnDao := new(daomocks.FiatTransactionDao)
	svc := newDepositServiceForTest(
		fiatTxnDao, new(daomocks.UserFiatAccountDao), new(daomocks.AccountLedgerDao),
		new(proxymocks.UserProxy), new(proxymocks.PaymentProxy), new(producermocks.DepositPaymentResultProducer),
	)
	existing := &model.FiatTransaction{
		ID: 9, UserID: 42, TransactionNo: "FT1", Currency: "USD",
		Amount: decimal.RequireFromString("10.00"), Status: model.FiatTxnStatusSucceeded,
		IdempotentKey: "idem-1", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	fiatTxnDao.On("GetByIdempotentKey", mock.Anything, int64(42), "idem-1").Return(existing, nil)

	vo, err := svc.CreateDeposit(context.Background(), 42, &data.CreateDepositRequest{
		IdempotentKey: "idem-1", PaymentMethodID: 1, Currency: "USD", Amount: "10.00",
	})
	require.NoError(t, err)
	require.Equal(t, "9", vo.TransactionID)
	require.Equal(t, model.FiatTxnStatusSucceeded, vo.Status)
}

func TestCreateDeposit_UnsupportedCurrency(t *testing.T) {
	initServiceTestEnv(t)
	fiatTxnDao := new(daomocks.FiatTransactionDao)
	userProxy := new(proxymocks.UserProxy)
	svc := newDepositServiceForTest(
		fiatTxnDao, new(daomocks.UserFiatAccountDao), new(daomocks.AccountLedgerDao),
		userProxy, new(proxymocks.PaymentProxy), new(producermocks.DepositPaymentResultProducer),
	)
	fiatTxnDao.On("GetByIdempotentKey", mock.Anything, int64(1), "k").Return(nil, nil)
	userProxy.On("GetUserProfile", mock.Anything, int64(1)).Return(sampleProfile("US"), nil)

	_, err := svc.CreateDeposit(context.Background(), 1, &data.CreateDepositRequest{
		IdempotentKey: "k", PaymentMethodID: 1, Currency: "SGD", Amount: "10.00",
	})
	require.ErrorContains(t, err, "not supported")
}

func TestCreateDeposit_PaymentSucceededSettles(t *testing.T) {
	initServiceTestEnv(t)
	fiatTxnDao := new(daomocks.FiatTransactionDao)
	fiatAccountDao := new(daomocks.UserFiatAccountDao)
	ledgerDao := new(daomocks.AccountLedgerDao)
	userProxy := new(proxymocks.UserProxy)
	paymentProxy := new(proxymocks.PaymentProxy)
	resultProducer := new(producermocks.DepositPaymentResultProducer)
	svc := newDepositServiceForTest(fiatTxnDao, fiatAccountDao, ledgerDao, userProxy, paymentProxy, resultProducer)

	fiatTxnDao.On("GetByIdempotentKey", mock.Anything, int64(42), "idem-ok").Return(nil, nil)
	userProxy.On("GetUserProfile", mock.Anything, int64(42)).Return(sampleProfile("SG"), nil)
	fiatTxnDao.On("Create", mock.Anything, (*gorm.DB)(nil), mock.AnythingOfType("*model.FiatTransaction")).
		Run(func(args mock.Arguments) {
			txn := args.Get(2).(*model.FiatTransaction)
			txn.ID = 100
		}).Return(nil)
	paymentProxy.On("CreatePayment", mock.Anything, mock.MatchedBy(func(req proxy.CreatePaymentRequest) bool {
		return req.UserID == 42 && req.Amount == 5000 && req.Currency == "USD" && req.PaymentMethodID == 7
	})).Return(&proxy.CreatePaymentResult{PaymentID: "pay_1", Status: "SUCCEEDED"}, nil)

	fiatTxnDao.On("UpdatePaymentResult", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(100), "pay_1",
		model.FiatTxnStatusPending, model.FiatTxnStatusSucceeded, "").Return(int64(1), nil)
	fiatAccountDao.On("UpsertAddBalance", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(42), "USD",
		decimal.RequireFromString("50.00")).Return(decimal.RequireFromString("50.00"), int64(55), nil)
	fiatTxnDao.On("UpdateAccountID", mock.Anything, mock.AnythingOfType("*gorm.DB"), int64(100), int64(55)).Return(nil)
	ledgerDao.On("Create", mock.Anything, mock.AnythingOfType("*gorm.DB"), mock.MatchedBy(func(l *model.AccountLedger) bool {
		return l.UserID == 42 &&
			l.AssetCode == "USD" &&
			l.BusinessType == model.LedgerBusinessTypeDeposit &&
			l.ChangeAmount.Equal(decimal.RequireFromString("50.00")) &&
			l.BalanceAfter.Equal(decimal.RequireFromString("50.00")) &&
			l.LedgerNo != ""
	})).Return(nil)
	resultProducer.On("PublishDepositPaymentResult", mock.Anything, mock.MatchedBy(func(e kproducer.DepositPaymentResultEvent) bool {
		return e.UserID == 42 && e.TransactionID == 100 && e.Status == model.FiatTxnStatusSucceeded &&
			e.Currency == "USD" && e.Amount == "50" && e.EventTime > 0
	})).Return(nil)

	succeeded := &model.FiatTransaction{
		ID: 100, UserID: 42, TransactionNo: "FT100", AccountID: 55, Currency: "USD",
		Amount: decimal.RequireFromString("50.00"), Status: model.FiatTxnStatusSucceeded,
		ExternalPaymentID: "pay_1", PaymentMethodID: 7, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	fiatTxnDao.On("GetByID", mock.Anything, int64(100)).Return(succeeded, nil)

	vo, err := svc.CreateDeposit(context.Background(), 42, &data.CreateDepositRequest{
		IdempotentKey: "idem-ok", PaymentMethodID: 7, Currency: "USD", Amount: "50.00",
	})
	require.NoError(t, err)
	require.Equal(t, "100", vo.TransactionID)
	require.Equal(t, model.FiatTxnStatusSucceeded, vo.Status)
	require.Equal(t, "55", vo.AccountID)
	require.Equal(t, "pay_1", vo.ExternalPaymentID)
	fiatTxnDao.AssertExpectations(t)
	fiatAccountDao.AssertExpectations(t)
	ledgerDao.AssertExpectations(t)
	resultProducer.AssertExpectations(t)
}

func TestCreateDeposit_PaymentRPCFailedDoesNotCredit(t *testing.T) {
	initServiceTestEnv(t)
	fiatTxnDao := new(daomocks.FiatTransactionDao)
	fiatAccountDao := new(daomocks.UserFiatAccountDao)
	userProxy := new(proxymocks.UserProxy)
	paymentProxy := new(proxymocks.PaymentProxy)
	resultProducer := new(producermocks.DepositPaymentResultProducer)
	svc := newDepositServiceForTest(
		fiatTxnDao, fiatAccountDao, new(daomocks.AccountLedgerDao), userProxy, paymentProxy, resultProducer,
	)

	fiatTxnDao.On("GetByIdempotentKey", mock.Anything, int64(1), "idem-fail").Return(nil, nil)
	userProxy.On("GetUserProfile", mock.Anything, int64(1)).Return(sampleProfile("SG"), nil)
	fiatTxnDao.On("Create", mock.Anything, (*gorm.DB)(nil), mock.AnythingOfType("*model.FiatTransaction")).
		Run(func(args mock.Arguments) {
			args.Get(2).(*model.FiatTransaction).ID = 11
		}).Return(nil)
	paymentProxy.On("CreatePayment", mock.Anything, mock.Anything).Return(nil, errors.New("grpc down"))
	fiatTxnDao.On("UpdatePaymentResult", mock.Anything, (*gorm.DB)(nil), int64(11), "",
		model.FiatTxnStatusPending, model.FiatTxnStatusFailed, "grpc down").Return(int64(1), nil)
	resultProducer.On("PublishDepositPaymentResult", mock.Anything, mock.MatchedBy(func(e kproducer.DepositPaymentResultEvent) bool {
		return e.Status == model.FiatTxnStatusFailed && e.TransactionID == 11
	})).Return(nil)

	_, err := svc.CreateDeposit(context.Background(), 1, &data.CreateDepositRequest{
		IdempotentKey: "idem-fail", PaymentMethodID: 1, Currency: "USD", Amount: "10.00",
	})
	require.ErrorContains(t, err, "grpc down")
	fiatAccountDao.AssertNotCalled(t, "UpsertAddBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	resultProducer.AssertExpectations(t)
}

func TestListFiatAccounts_Placeholders(t *testing.T) {
	initServiceTestEnv(t)
	fiatAccountDao := new(daomocks.UserFiatAccountDao)
	userProxy := new(proxymocks.UserProxy)
	svc := newDepositServiceForTest(
		new(daomocks.FiatTransactionDao), fiatAccountDao, new(daomocks.AccountLedgerDao),
		userProxy, new(proxymocks.PaymentProxy), new(producermocks.DepositPaymentResultProducer),
	)
	userProxy.On("GetUserProfile", mock.Anything, int64(7)).Return(sampleProfile("SG"), nil)
	fiatAccountDao.On("ListByUser", mock.Anything, int64(7)).Return([]*model.UserFiatAccount{
		{ID: 3, UserID: 7, Currency: "USD", Balance: decimal.RequireFromString("12.50"), UpdatedAt: time.Unix(100, 0)},
	}, nil)

	list, err := svc.ListFiatAccounts(context.Background(), 7, "")
	require.NoError(t, err)
	require.Len(t, list.Items, 2)
	require.Equal(t, "3", list.Items[0].AccountID)
	require.Equal(t, "USD", list.Items[0].Currency)
	require.Equal(t, "12.50", list.Items[0].Balance)
	require.Equal(t, "0", list.Items[1].AccountID)
	require.Equal(t, "SGD", list.Items[1].Currency)
	require.Equal(t, "0.00", list.Items[1].Balance)
}

func TestListFiatAccounts_EmptyMarket(t *testing.T) {
	initServiceTestEnv(t)
	userProxy := new(proxymocks.UserProxy)
	svc := newDepositServiceForTest(
		new(daomocks.FiatTransactionDao), new(daomocks.UserFiatAccountDao), new(daomocks.AccountLedgerDao),
		userProxy, new(proxymocks.PaymentProxy), new(producermocks.DepositPaymentResultProducer),
	)
	userProxy.On("GetUserProfile", mock.Anything, int64(1)).Return(sampleProfile(""), nil)

	_, err := svc.ListFiatAccounts(context.Background(), 1, "")
	require.ErrorContains(t, err, "market is required")
}

func TestListLedgers_CursorAndFilter(t *testing.T) {
	initServiceTestEnv(t)
	ledgerDao := new(daomocks.AccountLedgerDao)
	svc := newDepositServiceForTest(
		new(daomocks.FiatTransactionDao), new(daomocks.UserFiatAccountDao), ledgerDao,
		new(proxymocks.UserProxy), new(proxymocks.PaymentProxy), new(producermocks.DepositPaymentResultProducer),
	)
	now := time.Now()
	rows := []*model.AccountLedger{
		{ID: 3, LedgerNo: "L3", AssetCode: "USD", ChangeAmount: decimal.RequireFromString("10"), BusinessType: "DEPOSIT", BusinessID: "FT1", BalanceAfter: decimal.RequireFromString("10"), CreatedAt: now},
		{ID: 2, LedgerNo: "L2", AssetCode: "BTC", ChangeAmount: decimal.RequireFromString("0.1"), BusinessType: "PURCHASE", BusinessID: "AO1", BalanceAfter: decimal.RequireFromString("0.1"), CreatedAt: now},
		{ID: 1, LedgerNo: "L1", AssetCode: "USD", ChangeAmount: decimal.RequireFromString("5"), BusinessType: "DEPOSIT", BusinessID: "FT0", BalanceAfter: decimal.RequireFromString("5"), CreatedAt: now},
	}
	ledgerDao.On("ListByCursor", mock.Anything, dao.AccountLedgerListFilter{
		UserID: 9, AssetCode: "USD", BusinessType: "DEPOSIT", CreatedFrom: 1, CreatedTo: 2, Cursor: 0, Limit: 3,
	}).Return(rows, nil)

	list, err := svc.ListLedgers(context.Background(), ListLedgersQuery{
		UserID: 9, AssetCode: "usd", BusinessType: "deposit", CreatedFrom: 1, CreatedTo: 2, Limit: 2,
	})
	require.NoError(t, err)
	require.Equal(t, "2", list.NextCursor)
	require.Len(t, list.Items, 2)
	require.Equal(t, "+10", list.Items[0].ChangeAmount)
	require.Equal(t, "USD", list.Items[0].AssetCode)
}
