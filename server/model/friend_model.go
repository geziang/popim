package model

import "time"

const (
	FriendTableName = "friends"
)

type Friend struct {
	Id        uint `gorm:"primary_key;AUTO_INCREMENT"`
	CreatedAt time.Time
	UserId1 uint `gorm:"index:user_id1_user_id2"`
	UserId2 uint `gorm:"index:user_id1_user_id2"`
}