package repository

import (
	"database/sql"
	"os"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/repository/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	DB  *gorm.DB
	err error
)

type TxBeginner interface {
	Transaction(fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error
}

var _ TxBeginner = (*gorm.DB)(nil) // Compile-time interface check

func Init() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		panic("MYSQL_DSN environment variable is not set")
	}
	DB, err = gorm.Open(mysql.Open(dsn),
		&gorm.Config{
			PrepareStmt:            true,
			SkipDefaultTransaction: true,
		},
	)
	if err != nil {
		panic(err)
	}
	err = DB.AutoMigrate(
		&model.Item{},
	)
	if err != nil {
		panic(err)
	}
}
