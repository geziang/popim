package model

import (
	"time"
)

const (
	UserTableName = "users"
)

type User struct {
	Id        uint `gorm:"primary_key;AUTO_INCREMENT"`
	CreatedAt time.Time
	PhoneNumber string `gorm:"type:varchar(20);unique_index"`
	NickName string `gorm:"type:varchar(40);index:nick_name"`
	MusicId uint
}
