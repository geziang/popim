package service

import (
	"encoding/json"
	"fmt"
	"github.com/geziang/popim"
	"github.com/geziang/popim/server/cache"
	"github.com/geziang/popim/server/util"
	"github.com/gomodule/redigo/redis"
	"log"
	"sync"
	"time"
)

const (
	imMsgExpireTime = 24*60*60 //消息过期时间:24h
)

var (
	MsgService = &msgService{}
)

type msgService struct {
	mPusher sync.Map
}

/*
 推送服务端循环
 */
func (svc *msgService) PusherLoop(popId uint, stream popim.Msg_PusherServer) {
	ch := make(chan popim.PushMsg)
	svc.mPusher.Store(popId, ch)
	defer close(ch)
	defer svc.mPusher.Delete(popId)

	for {
		msg := <- ch
		stream.Send(&msg)
	}
}

/*
 发送推送
 */
func (svc *msgService) PushMsg(popId uint, msg *popim.PushMsg) bool {
	ch, ok := svc.mPusher.Load(popId)
	if !ok {
		return false
	}
	ch.(chan popim.PushMsg) <- *msg
	return true
}

func (svc *msgService) SendIMMsg(msg *popim.IMMsg) bool {
	ts := time.Now().Unix()
	userId1, userId2 := util.GetOrderedIds(uint(msg.FromId),uint(msg.ToId))
	key := fmt.Sprint("immsg_",userId1,"_",userId2,"_",ts)

	data, err := json.Marshal(&cache.IMMsgInStore{
		Timestamp: ts,
		FromId:    uint(msg.FromId),
		ToId:      uint(msg.ToId),
		Content:   msg.Content,
	})

	if err != nil {
		log.Println("json marshal failed! ",err)
		return false
	}

	conn := cache.RCPool.Get()
	defer conn.Close()
	if err := conn.Err(); err != nil {
		log.Println("redis conn failed! ",err)
		return false
	}

	_, err = conn.Do("SET", key, data,"EX",imMsgExpireTime)
	if err != nil {
		log.Println("redis set failed! ",err)
		return false
	}

	return true
}

func (svc *msgService) GetIMMsgHistory(popId1, popId2 uint) <-chan cache.IMMsgInStore {
	userId1, userId2 := util.GetOrderedIds(popId1, popId2)
	pattern := fmt.Sprint("immsg_",userId1,"_",userId2,"_*")

	ch := make(chan cache.IMMsgInStore)
	go func() {
		defer close(ch)

		conn := cache.RCPool.Get()
		defer conn.Close()
		if err := conn.Err(); err != nil {
			log.Println("redis conn failed! ",err)
			return
		}

		keys, err := redis.Strings(conn.Do("KEYS", pattern))
		if err != nil {
			log.Println("redis keys command failed! ",err)
			return
		}

		for _, key := range keys {
			data, err := redis.Bytes(conn.Do("GET", key))
			if err != nil {
				log.Println("redis get failed! ",err)
				continue
			}

			msg := cache.IMMsgInStore{}
			err = json.Unmarshal(data, &msg)
			if err != nil {
				log.Println("json unmarshal failed! ",err)
				continue
			}

			ch <- msg
		}
	}()
	return ch
}