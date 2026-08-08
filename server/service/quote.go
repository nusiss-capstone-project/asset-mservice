package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	cacheredis "github.com/nusiss-capstone-project/asset-mservice/server/repository/redis"
	"github.com/nusiss-capstone-project/asset-mservice/server/util"
)

// GenQuoter generates a short-lived quote for an asset.
type GenQuoter interface {
	GenQuote(ctx context.Context, userID int64, req *data.GenQuoteRequest) (*data.QuoteVO, error)
}

type QuoteServiceImpl struct {
	assetDao dao.AssetDao
}

var (
	quoteServiceOnce sync.Once
	quoteServiceInst GenQuoter
)

func GetQuoteService() GenQuoter {
	quoteServiceOnce.Do(func() {
		quoteServiceInst = &QuoteServiceImpl{
			assetDao: dao.GetAssetDao(),
		}
	})
	return quoteServiceInst
}

func (s *QuoteServiceImpl) GenQuote(ctx context.Context, userID int64, req *data.GenQuoteRequest) (*data.QuoteVO, error) {
	if !cacheredis.Available() {
		return nil, fmt.Errorf("quote service unavailable: redis disabled")
	}
	assetID, err := strconv.ParseInt(strings.TrimSpace(req.AssetID), 10, 64)
	if err != nil || assetID <= 0 {
		return nil, fmt.Errorf("invalid asset_id")
	}
	currency := normalizeCurrency(req.Currency)

	asset, err := s.assetDao.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, fmt.Errorf("asset not found")
	}
	if asset.Status != model.AssetStatusActive {
		return nil, fmt.Errorf("asset is not active")
	}
	if !strings.EqualFold(asset.Currency, currency) {
		return nil, fmt.Errorf("asset currency mismatch")
	}

	quoteID, err := util.NewQuoteID()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(cacheredis.QuoteTTL)
	quote := &cacheredis.QuoteCache{
		QuoteID:   quoteID,
		AssetID:   asset.ID,
		Currency:  currency,
		UnitPrice: asset.CurrentPrice,
		UserID:    userID,
		ExpiresAt: expiresAt.Unix(),
	}
	if err := cacheredis.SaveQuote(ctx, quote); err != nil {
		log.WithContext(ctx).Errorw("gen quote save failed", "quote_id", quoteID, "error", err)
		return nil, err
	}
	log.WithContext(ctx).Infow("quote generated",
		"quote_id", quoteID,
		"asset_id", asset.ID,
		"user_id", userID,
		"unit_price", asset.CurrentPrice,
	)
	return &data.QuoteVO{
		QuoteID:   quoteID,
		AssetID:   strconv.FormatInt(asset.ID, 10),
		Currency:  currency,
		UnitPrice: asset.CurrentPrice,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}
