package service

import (
	"fmt"
	"github.com/geziang/popim/server/cache"
	"github.com/geziang/popim/server/model"
	"github.com/gomodule/redigo/redis"
	"log"
	"math/rand"
)

var (
	UserService = &userService{}
)

type userService struct {}

/*
 判断用户是否存在（以手机号）
 */
func (*userService) IsUserExistByPhoneNumber(phoneNumber string) bool {
	var count int
	model.DB.Table(model.UserTableName).Where("phone_number = ?",phoneNumber).Count(&count)
	if count > 0 {
		return true
	}

	return false
}

/*
 判断用户是否存在（以id）
 */
func (*userService) IsUserExistByPopId(popId uint) bool {
	var count int
	model.DB.Table(model.UserTableName).Where("id = ?",popId).Count(&count)
	if count > 0 {
		return true
	}

	return false
}

/*
 以手机号获取用户id
*/
func (*userService) GetPopIdByPhoneNumber(phoneNumber string) uint {
	var user model.User
	model.DB.Select("id").Where("phone_number = ?",phoneNumber).Take(&user)

	return user.Id
}

/*
 获取一个新token
 */
func (*userService) GetNewTokenByPopId(popId uint) (string, error) {
	token := fmt.Sprint(popId,rand.Int())
	conn := cache.RCPool.Get()
	defer conn.Close()
	if err := conn.Err(); err != nil {
		log.Println("redis conn failed! ",err)
		return "", err
	}
	_, err := conn.Do("SET", popId, token)
	if err != nil {
		log.Println("redis set failed! ",err)
		return "", err
	}

	return token, nil
}

/*
 校验token
 */
func (*userService) CheckToken(popId uint, token string) (bool, error) {
	conn := cache.RCPool.Get()
	defer conn.Close()
	if err := conn.Err(); err != nil {
		log.Println("redis conn failed! ",err)
		return false, err
	}

	recordToken, err := redis.String(conn.Do("GET", popId))
	if err != nil {
		log.Println("redis get failed! ",err)
		return false, err
	}

	if token != recordToken {
		return false, nil
	}

	return true, nil
}

/*
 获取用户信息
 */
func (*userService) GetUserById(popId uint) model.User {
	var user model.User
	model.DB.Where("id = ?", popId).Take(&user)
	return user
}

func (*userService) RegisterUser(user *model.User) *model.User {
	model.DB.Create(user)
	return user
}