package proxy

import (
	"context"
	"fmt"
	"sync"

	"github.com/nusiss-capstone-project/payment-mservice/common/paymentpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
)

type CreatePaymentRequest struct {
	BizID           string
	UserID          int64
	Amount          int64
	Currency        string
	PaymentMethodID int64
}

type CreatePaymentResult struct {
	PaymentID string
	Status    string
}

type PaymentProxy interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResult, error)
}

type paymentProxyImpl struct {
	client paymentpb.PaymentServiceClient
}

var (
	paymentProxyOnce sync.Once
	paymentProxyInst PaymentProxy
)

func GetPaymentProxy() PaymentProxy {
	paymentProxyOnce.Do(func() {
		cfg := config.Config.PaymentGrpcConfig
		if cfg == nil {
			panic("payment_grpc config is nil")
		}
		conn, err := grpc.NewClient(
			fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(1024*1024),
				grpc.MaxCallSendMsgSize(1024*1024),
			),
		)
		if err != nil {
			panic(err)
		}
		paymentProxyInst = &paymentProxyImpl{
			client: paymentpb.NewPaymentServiceClient(conn),
		}
		log.Logger.Infow("payment grpc client initialized", "host", cfg.Host, "port", cfg.Port)
	})
	return paymentProxyInst
}

func (p *paymentProxyImpl) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResult, error) {
	log.WithContext(ctx).Infow("create payment request",
		"biz_id", req.BizID,
		"user_id", req.UserID,
		"amount", req.Amount,
		"currency", req.Currency,
		"payment_method_id", req.PaymentMethodID,
	)
	resp, err := p.client.CreatePayment(ctx, &paymentpb.CreatePaymentRequest{
		BizId:           req.BizID,
		UserId:          req.UserID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		PaymentMethodId: req.PaymentMethodID,
	})
	if err != nil {
		log.WithContext(ctx).Errorw("create payment grpc failed",
			"biz_id", req.BizID,
			"user_id", req.UserID,
			"error", err,
		)
		return nil, err
	}
	if resp.GetBaseInfo() != nil &&
		resp.GetBaseInfo().GetCode() != paymentpb.ErrorCode_ERROR_CODE_OK &&
		resp.GetBaseInfo().GetCode() != paymentpb.ErrorCode_ERROR_CODE_UNSPECIFIED {
		err := fmt.Errorf("create payment failed: code=%v message=%s",
			resp.GetBaseInfo().GetCode(), resp.GetBaseInfo().GetMessage())
		log.WithContext(ctx).Errorw("create payment business error",
			"biz_id", req.BizID,
			"error", err,
		)
		return nil, err
	}
	log.WithContext(ctx).Infow("create payment response",
		"biz_id", req.BizID,
		"payment_id", resp.GetPaymentId(),
		"status", resp.GetStatus(),
	)
	return &CreatePaymentResult{
		PaymentID: resp.GetPaymentId(),
		Status:    resp.GetStatus(),
	}, nil
}
