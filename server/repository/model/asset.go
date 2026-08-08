package model

import "time"

const (
	AssetStatusActive   = "active"
	AssetStatusInactive = "inactive"
)

type Asset struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	Name         string    `gorm:"type:varchar(128);not null"`
	Symbol       string    `gorm:"type:varchar(32);not null;index"`
	IconURL      string    `gorm:"column:icon_url;type:varchar(512)"`
	Currency     string    `gorm:"type:varchar(16);not null;index"`
	CurrentPrice string    `gorm:"column:current_price;type:varchar(64);not null"`
	Status       string    `gorm:"type:varchar(32);not null;index"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (Asset) TableName() string {
	return "assets"
}
