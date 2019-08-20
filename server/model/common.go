package model

import "github.com/jinzhu/gorm"

var (
	DB *gorm.DB
)

func MigrateDB() {
	DB.AutoMigrate(&User{},&Friend{})
}