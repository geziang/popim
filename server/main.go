package main

import (
	"flag"
	"github.com/geziang/popim"
	"github.com/geziang/popim/server/cache"
	"github.com/geziang/popim/server/model"
	"github.com/geziang/popim/server/rpc"
	"github.com/gomodule/redigo/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"google.golang.org/grpc"
	"net"
	"time"
)

func main(){
	listen := flag.String("listen", ":50513", "server listen address")
	mysqlStr := flag.String("mysql", "", "mysql conn str")
	redisAddr := flag.String("redis", "127.0.0.1:6379", "redis connect address")
	flag.Parse()

	var err error
	model.DB, err = gorm.Open("mysql", *mysqlStr)
	if err != nil {
		panic(err)
	}
	defer model.DB.Close()

	model.MigrateDB()

	// 建立连接池
	cache.RCPool = &redis.Pool{
		MaxIdle:     100,
		MaxActive:   1000,
		IdleTimeout: 60 * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) { return redis.Dial("tcp", *redisAddr) },
	}

	//监听端口
	lis,err := net.Listen("tcp",*listen)
	if err != nil{
		panic(err)
	}
	//创建一个grpc 服务器
	s := grpc.NewServer()
	//注册事件
	popim.RegisterUserServer(s,&rpc.UserServer{})
	popim.RegisterFriendServer(s,&rpc.FriendServer{})
	popim.RegisterMsgServer(s,&rpc.MsgServer{})

	//处理链接
	s.Serve(lis)
}