package service

import (
	"fmt"
	"github.com/geziang/popim"
	"github.com/geziang/popim/server/cache"
	"github.com/geziang/popim/server/model"
	"github.com/geziang/popim/server/util"
	"github.com/gomodule/redigo/redis"
	"log"
	"time"
)

const (
	friendRequestExpireTime = 24*60*60 //好友请求过期时间:24h
)

var (
	FriendService = &friendService{}
)

type friendService struct {}

/*
 判断是否为好友
 */
func (svc *friendService) IsFriends(popId1, popId2 uint) bool {
	userId1, userId2 := util.GetOrderedIds(popId1,popId2)

	var count int
	model.DB.Table(model.FriendTableName).Where("user_id1 = ? and user_id2 = ?",userId1,userId2).Count(&count)
	if count > 0 {
		return true
	}
	return false
}

/*
 加好友
 */
func (svc *friendService) AddOrRequestFriend(srcPopId, targetPopId uint) bool {
	if svc.IsFriends(srcPopId, targetPopId) {
		return true
	}

	userId1, userId2 := util.GetOrderedIds(srcPopId,targetPopId)
	key := fmt.Sprint("friendreq_",userId1,"_",userId2)

	conn := cache.RCPool.Get()
	defer conn.Close()
	if err := conn.Err(); err != nil {
		log.Println("redis conn failed! ",err)
		return false
	}

	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		log.Println("redis exists command failed! ",err)
		return false
	}

	if exists {
		//关系打通
		friend := model.Friend{
			CreatedAt: time.Now(),
			UserId1:   userId1,
			UserId2:   userId2,
		}
		model.DB.Create(&friend)

		//给双方发送新好友消息
		user := UserService.GetUserById(srcPopId)
		MsgService.PushMsg(targetPopId, &popim.PushMsg{
			Timestamp:     uint64(time.Now().Unix()),
			Type:          popim.PushMsg_NEWFRIENDMSG,
			FriendMsgData: &popim.UserInfo{
				PopId: uint64(user.Id),
				NickName: user.NickName,
				MusicId: uint64(user.MusicId),
			},
		})

		user = UserService.GetUserById(targetPopId)
		MsgService.PushMsg(srcPopId, &popim.PushMsg{
			Timestamp:     uint64(time.Now().Unix()),
			Type:          popim.PushMsg_NEWFRIENDMSG,
			FriendMsgData: &popim.UserInfo{
				PopId: uint64(user.Id),
				NickName: user.NickName,
				MusicId: uint64(user.MusicId),
			},
		})
	} else {
		//请求
		_, err := conn.Do("SET", key, 1,"EX",friendRequestExpireTime)
		if err != nil {
			log.Println("redis set failed! ",err)
			return false
		}

		user := UserService.GetUserById(srcPopId)

		MsgService.PushMsg(targetPopId, &popim.PushMsg{
			Timestamp:     uint64(time.Now().Unix()),
			Type:          popim.PushMsg_REQFRIENDMSG,
			FriendMsgData: &popim.UserInfo{
				PopId: uint64(user.Id),
				NickName: user.NickName,
				MusicId: uint64(user.MusicId),
			},
		})
	}

	return true
}

/*
 获取所有好友
 */
func (svc *friendService) GetFriendList(popId uint) []model.User {
	users := make([]model.User,0)
	model.DB.Raw("SELECT user_id1 AS id FROM `"+model.FriendTableName+"` WHERE user_id2 = ? UNION ALL SELECT user_id2 AS id FROM `"+model.FriendTableName+"` WHERE user_id1 = ?",popId,popId).Scan(&users)

	return users
}

/*
 以昵称搜索好友
*/
func (svc *friendService) SearchFriend(popId uint, key string) []model.User {
	users := make([]model.User,0)
	model.DB.Raw("SELECT * FROM `"+model.UserTableName+"` WHERE nick_name LIKE ? AND id IN (SELECT user_id1 AS id FROM `"+model.FriendTableName+"` WHERE user_id2 = ? UNION ALL SELECT user_id2 AS id FROM `"+model.FriendTableName+"` WHERE user_id1 = ?)","%"+key+"%",popId,popId).Scan(&users)
	return users
}