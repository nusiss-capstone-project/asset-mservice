package model

const (
	UserStatusInactive = -1
	UserStatusActive   = 1
)

type Item struct {
	ID   int    `gorm:"primaryKey"`
	Name string `gorm:"type:varchar(128);unique;not null"`
}

// TableName sets the insert table name for this struct type
func (Item) TableName() string {
	return "items"
}
