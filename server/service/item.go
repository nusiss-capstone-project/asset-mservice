package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/http/data"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/log"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/repository/dao"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/repository/model"
)

type ItemService interface {
	CreateItem(ctx context.Context, itemVo *data.ItemVO) (int, error)
	GetItemById(ctx context.Context, id int) (*data.ItemVO, error)
}

type ItemServiceImpl struct {
	itemDao dao.ItemDao
}

var (
	itemServiceSyncOnce sync.Once
	itemServiceInst     *ItemServiceImpl
)

func GetItemService() *ItemServiceImpl {
	itemServiceSyncOnce.Do(func() {
		itemServiceInst = &ItemServiceImpl{itemDao: dao.GetItemDao()}
	})
	return itemServiceInst
}

func (ls *ItemServiceImpl) CreateItem(ctx context.Context, itemVo *data.ItemVO) (int, error) {
	if itemVo == nil {
		return 0, errors.New("itemVO is nil")
	}
	item := &model.Item{
		ID:   itemVo.ID,
		Name: itemVo.Name,
	}
	id, err := ls.itemDao.Create(ctx, item)
	if err != nil {
		log.Logger.Errorf("Failed to create item: %v", err)
		return 0, fmt.Errorf("failed to create item: %w", err)
	}
	return id, nil
}

func (ls *ItemServiceImpl) GetItemById(ctx context.Context, id int) (*data.ItemVO, error) {
	item, err := ls.itemDao.GetById(ctx, id)
	if err != nil {
		log.Logger.Errorf("Failed to get item by ID: %v", err)
		return nil, fmt.Errorf("failed to get item by ID: %w", err)
	}
	if item == nil {
		return nil, nil
	}
	itemVo := &data.ItemVO{
		ID:   item.ID,
		Name: item.Name,
	}
	return itemVo, nil
}
