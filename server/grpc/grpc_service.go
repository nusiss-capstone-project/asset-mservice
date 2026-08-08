package grpc

import (
	"context"
	"errors"

	"github.com/nusiss-capstone-project/asset-mservice/common/assetpb"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/service"
)

type AssetService struct {
	assetpb.UnimplementedAssetServiceServer
}

func (s *AssetService) Reward(ctx context.Context, in *assetpb.RewardRequest) (*assetpb.RewardResponse, error) {
	result, err := service.GetRewardService().Reward(ctx, service.RewardInput{
		BizID:     in.GetBizId(),
		UserID:    in.GetUserId(),
		AssetCode: in.GetAssetCode(),
		Amount:    in.GetAmount(),
	})
	if err != nil {
		log.WithContext(ctx).Warnw("reward failed",
			"biz_id", in.GetBizId(),
			"user_id", in.GetUserId(),
			"asset_code", in.GetAssetCode(),
			"error", err,
		)
		return &assetpb.RewardResponse{
			BaseInfo: rewardBaseInfo(err),
		}, nil
	}
	return &assetpb.RewardResponse{
		TransactionId: result.TransactionID,
		BaseInfo: &assetpb.BaseResponseInfo{
			Code:    assetpb.ErrorCode_ERROR_CODE_OK,
			Message: "ok",
		},
	}, nil
}

func rewardBaseInfo(err error) *assetpb.BaseResponseInfo {
	switch {
	case errors.Is(err, service.ErrRewardInvalidArgument):
		return &assetpb.BaseResponseInfo{
			Code:    assetpb.ErrorCode_ERROR_CODE_INVALID_ARGUMENT,
			Message: err.Error(),
		}
	case errors.Is(err, service.ErrRewardAssetNotFound):
		return &assetpb.BaseResponseInfo{
			Code:    assetpb.ErrorCode_ERROR_CODE_NOT_FOUND,
			Message: err.Error(),
		}
	default:
		return &assetpb.BaseResponseInfo{
			Code:    assetpb.ErrorCode_ERROR_CODE_INTERNAL,
			Message: err.Error(),
		}
	}
}
