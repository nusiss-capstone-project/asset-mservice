package dao

import (
	"context"
	"errors"
	"sync"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/log"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/repository"
	"gorm.io/gorm"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/repository/model"
)

type ItemDao interface {
	Create(ctx context.Context, item *model.Item) (int, error)
	GetById(ctx context.Context, id int) (*model.Item, error)
}

type ItemDaoImpl struct {
	db *gorm.DB
}

var (
	itemOnce sync.Once
	itemDao  *ItemDaoImpl
)

func GetItemDao() *ItemDaoImpl {
	itemOnce.Do(func() {
		if itemDao == nil {
			itemDao = &ItemDaoImpl{db: repository.DB}
		}
	})
	return itemDao
}

func (dao *ItemDaoImpl) Create(ctx context.Context, item *model.Item) (int, error) {
	ret := dao.db.WithContext(ctx).Save(&item)
	if ret.Error != nil {
		if errors.Is(ret.Error, gorm.ErrDuplicatedKey) {
			return 0, errors.New("item already exists")
		}
		log.Logger.Errorf("Failed to create item: %v", ret.Error)
		return 0, ret.Error
	}
	log.Logger.Infof("item created with ID: %d", item.ID)
	return item.ID, nil
}

func (dao *ItemDaoImpl) GetById(ctx context.Context, id int) (*model.Item, error) {
	var item model.Item
	ret := dao.db.WithContext(ctx).Where("id = ?", id).First(&item)
	if ret.Error != nil {
		if errors.Is(ret.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		log.Logger.Errorf("Failed to get item by email: %v", ret.Error)
		return nil, ret.Error
	}
	return &item, nil
}
