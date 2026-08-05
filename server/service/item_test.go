package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/config"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/http/data"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/log"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/repository/dao/mocks"
	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/repository/model"
	"github.com/stretchr/testify/mock"
)

func initEnv() {
	config.Config = &config.Conf{
		LogConfig: &config.LogConfig{
			Level:    "debug",
			FilePath: "",
		},
	}
	log.InitLogger()
}
func TestGetItemService(t *testing.T) {
	initEnv()
	service1 := GetItemService()
	service2 := GetItemService()

	if service1 != service2 {
		t.Errorf("Expected the same instance of ItemServiceImpl, got different instances")
	}
}
func TestCreateItem(t *testing.T) {
	initEnv()
	itemDao := new(mocks.ItemDao)
	service := &ItemServiceImpl{
		itemDao: itemDao, // In a real test, you might want to mock this
	}
	tests := []struct {
		name     string
		itemVo   *data.ItemVO
		expected int
		wantErr  bool
	}{
		{
			name:     "Valid item",
			itemVo:   &data.ItemVO{ID: 1, Name: "Test Item"},
			expected: 1, // Assuming the ID returned is 1
			wantErr:  false,
		},
		{
			name:     "Nil itemVO",
			itemVo:   nil,
			expected: 0,
			wantErr:  true,
		},
	}
	itemDao.On("Create", mock.Anything, mock.Anything).Return(1, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.CreateItem(context.Background(), tt.itemVo)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("CreateItem() = %v, want %v", got, tt.expected)
			}
		})
	}
}
func TestGetItemById(t *testing.T) {
	initEnv()
	itemDao := new(mocks.ItemDao)
	service := &ItemServiceImpl{
		itemDao: itemDao,
	}
	tests := []struct {
		name     string
		id       int
		expected *data.ItemVO
		wantErr  bool
	}{
		{
			name:     "Valid ID",
			id:       1,
			expected: &data.ItemVO{ID: 1, Name: "Test Item"},
			wantErr:  false,
		},
		{
			name:     "Item not found",
			id:       2,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "Invalid ID",
			id:       -1,
			expected: nil,
			wantErr:  true,
		},
	}

	itemDao.On("GetById", mock.Anything, 1).Return(&model.Item{ID: 1, Name: "Test Item"}, nil)
	itemDao.On("GetById", mock.Anything, 2).Return(nil, nil)
	itemDao.On("GetById", mock.Anything, -1).Return(nil, errors.New("invalid ID"))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.GetItemById(context.Background(), tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetItemById() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GetItemById() = %v, want %v", got, tt.expected)
			}
		})
	}
}
