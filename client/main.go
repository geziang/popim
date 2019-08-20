package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/geziang/popim"
	"google.golang.org/grpc"
	"io"
	"log"
	"os"
	"time"
)

const (
	fileBlockSize = 1024
)

var (
	userC popim.UserClient
	friendC popim.FriendClient
	msgC popim.MsgClient
	authContext *popim.AuthContext
)

func main(){
	address := flag.String("addr", "", "server address")
	flag.Parse()

	conn, err := grpc.Dial(*address,grpc.WithInsecure())
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer conn.Close()

	userC = popim.NewUserClient(conn)
	friendC = popim.NewFriendClient(conn)
	msgC = popim.NewMsgClient(conn)

	fmt.Println("欢迎使用我的IM系统原始模型！")
	fmt.Println()
	fmt.Println("请选择操作:")
	fmt.Println("1) 模拟登录")
	fmt.Println("2) 模拟注册")

	for {
		op := <- readOp()
		if op == 1 {
			login()
			break
		} else if op == 2 {
			register()
			break
		}
	}
}

func readOp() <- chan int {
	fmt.Print("请输入数字选择:")
	ch := make(chan int)
	go func() {
		var ret int
		fmt.Scanln(&ret)
		ch <- ret
	}()
	return ch
}

func register() {
	var phoneNumber,nickName string
	for phoneNumber == "" {
		fmt.Print("请输入手机号:")
		fmt.Scanln(&phoneNumber)
	}

	for nickName == "" {
		fmt.Print("请输入昵称:")
		fmt.Scanln(&nickName)
	}

	res, err := userC.Register(context.Background(), &popim.RegisterReq{
		PhoneNumber: phoneNumber,
		NickName:    nickName,
	})
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(res.Result.Msg)
		if res.Result.Code == 200 {
			fmt.Println("注册用户ID: ",res.PopId)
		}
	}
}

func login() {
	req := &popim.LoginReq{}

	fmt.Println("请选择登录凭据:")
	fmt.Println("1) 手机号")
	fmt.Println("2) ID")

	for {
		op := <- readOp()
		if op == 1 {
			var phoneNumber string
			for phoneNumber == "" {
				fmt.Print("请输入手机号:")
				fmt.Scanln(&phoneNumber)
			}
			req.PhoneNumber = phoneNumber
			break
		} else if op == 2 {
			var id uint64
			for id == 0 {
				fmt.Print("请输入ID:")
				fmt.Scanln(&id)
			}
			req.PopId = id
			break
		}
	}

	res, err := userC.Login(context.Background(), req)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(res.Result.Msg)
		if res.Result.Code == 200 {
			authContext = res.Context
			loop()
		}
	}
}

func getUserInfo(popId uint64) {
	res, err := userC.GetUserInfo(context.Background(), &popim.QueryByIdReq{
		Context:    authContext,
		QueryPopId: popId,
	})
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(res.Result.Msg)
		if res.Result.Code == 200 {
			fmt.Println("ID: ", res.Data.PopId)
			fmt.Println("昵称: ", res.Data.NickName)
			fmt.Println("背景音乐id: ", res.Data.MusicId)
		}
	}
}

func getAvatar(popId uint64) {
	stream, err := userC.GetAvatar(context.Background(), &popim.QueryByIdReq{
		Context:    authContext,
		QueryPopId: popId,
	})
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println("开始接收文件avatar.jpg...")

		//准备资源
		file, err := os.Create("./avatar.jpg")
		if err != nil {
			log.Println("Create file fail err:",err)
			return
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		defer writer.Flush()

		var totalBlocks uint64
		nowBlock := 0
		for {
			block, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					log.Println("block recv error ", err)
				}
				break
			}

			totalBlocks = block.TotalCount
			nowBlock ++
			_, err = writer.Write(block.Data)
			if err != nil {
				log.Println("file write error ", err)
				break
			}
			fmt.Print("\r进度:",nowBlock,"/",totalBlocks)
		}
		fmt.Println()
	}
}

func changeAvatar() {
	var filePath string
	fmt.Print("请输入头像文件路径:")
	fmt.Scanln(&filePath)

	info, err := os.Stat(filePath)
	if err != nil {
		log.Println(err)
		return
	}
	fileSize := info.Size()
	totalBlocks := fileSize / fileBlockSize
	if fileSize %fileBlockSize != 0 {
		totalBlocks ++
	}
	log.Println("Sending file ", filePath, " size=", fileSize, " total_blocks=", totalBlocks)

	stream, err := userC.UpdateAvatar(context.Background())
	if err != nil {
		log.Println(err)
		return
	}
	defer stream.CloseSend()

	//准备资源
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Open file ", filePath, " fail err:",err)
		return
	}
	reader := bufio.NewReader(file)
	defer file.Close()
	buf := make([]byte, 1024)

	//开始传输
	nowBlock := 0
	aContext := authContext
	for {
		n, err := reader.Read(buf)
		if err != nil{
			if err != io.EOF {
				log.Println("File read error ", err, "file=", filePath)
			}
			break
		}

		if nowBlock > 0 {
			aContext = &popim.AuthContext{}
		}
		nowBlock ++
		err = stream.Send(&popim.StreamUploadFileBlock{
			Context: aContext,
			TotalCount: uint64(totalBlocks),
			Data:       buf[:n],
		})
		if err != nil{
			log.Println("File block send error ", err, "file=", filePath)
			break
		}
		fmt.Print("\r进度:",nowBlock,"/",totalBlocks)
	}

	fmt.Println()
}

func changeUserInfo() {
	req := &popim.UpdateUserInfoReq{
		Context: authContext,
		Data: &popim.UserInfo{
			PopId:authContext.PopId,
		},
	}

	fmt.Println("选择要修改的信息:")
	fmt.Println("1) 昵称")
	fmt.Println("2) 背景音乐id")
	op := <- readOp()
	if op == 1 {
		var nickName string
		fmt.Print("请输入新昵称:")
		fmt.Scanln(&nickName)
		if nickName != "" {
			req.Data.NickName = nickName
		}
	} else if op == 2 {
		var musicId uint64
		fmt.Print("请输入新背景音乐id:")
		fmt.Scanln(&musicId)
		if musicId != 0 {
			req.Data.MusicId = musicId
		}
	}

	res, err := userC.UpdateUserInfo(context.Background(), req)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(res.Msg)
	}
}

func getFriendList() {
	stream, err := friendC.GetFriendList(context.Background(), &popim.QueryByIdReq{
		Context:    authContext,
		QueryPopId: authContext.PopId,
	})
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println("好友列表:")
		first := true
		for {
			list, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					fmt.Println("结束.")
				} else {
					log.Println(err)
				}
				break
			}

			if first {
				first = false
				fmt.Println("共",list.TotalCount,"人")
			}

			for _, v := range list.UserInfo {
				fmt.Println("好友: ID=",(*v).PopId," 昵称=",(*v).NickName," 背景音乐id=",(*v).MusicId)
			}
		}
	}
}

func searchFriend() {
	var key string
	fmt.Print("请输入搜索昵称关键词:")
	fmt.Scanln(&key)
	if key == "" {
		return
	}

	stream, err := friendC.SearchFriend(context.Background(), &popim.SearchFriendReq{
		Context:    authContext,
		Keyword: key,
	})
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println("搜索结果:")
		first := true
		for {
			list, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					fmt.Println("结束.")
				} else {
					log.Println(err)
				}
				break
			}

			if first {
				first = false
				fmt.Println("共",list.TotalCount,"人")
			}

			for _, v := range list.UserInfo {
				fmt.Println("好友: ID=",(*v).PopId," 昵称=",(*v).NickName," 背景音乐id=",(*v).MusicId)
			}
		}
	}
}

func reqFriend() {
	var id uint64
	fmt.Print("请输入对方ID:")
	fmt.Scanln(&id)

	if id == 0 {
		return
	}

	addFriend(id)
}

func addFriend(popId uint64) {
	res, err := friendC.AddFriend(context.Background(), &popim.QueryByIdReq{
		Context:    authContext,
		QueryPopId: popId,
	})
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(res.Msg)
	}
}

func sendMsg() {
	var id uint64
	fmt.Print("请输入对方ID:")
	fmt.Scanln(&id)
	if id == 0 {
		return
	}

	var content string
	fmt.Print("请输入消息内容:")
	fmt.Scanln(&content)
	if content == "" {
		return
	}

	res, err := msgC.SendIMMsg(context.Background(), &popim.SendIMMsgReq{
		Context:    authContext,
		ImMsg: &popim.IMMsg{
			Timestamp: uint64(time.Now().Unix()),
			FromId:    authContext.PopId,
			ToId:      id,
			Content:   content,
		},
	})
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(res.Msg)
	}
}

func fetchMsgHistory() {
	var id uint64
	fmt.Print("请输入好友ID:")
	fmt.Scanln(&id)
	if id == 0 {
		return
	}

	stream, err := msgC.GetIMMsgHistoryBetweenSingleUser(context.Background(), &popim.QueryByIdReq{
		Context:    authContext,
		QueryPopId: id,
	})
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println("消息记录（保留24小时）:")

		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					fmt.Println("结束.")
				} else {
					log.Println(err)
				}
				break
			}

			fmt.Println((*msg).Timestamp,"时间 ", (*msg).FromId,"发给",(*msg).ToId,": ",(*msg).Content)
		}
	}
}

func loop() {
	fmt.Println("登录成功！")
	fmt.Println()
	fmt.Println("获取当前用户信息...")
	getUserInfo(authContext.PopId)

	fmt.Println("获取当前用户头像...")
	getAvatar(authContext.PopId)

	stream, err := msgC.Pusher(context.Background(), authContext)
	if err != nil {
		log.Println(err)
		return
	} else {
		fmt.Println("消息推送通道已打开")
	}

	ch := make(chan *popim.PushMsg)
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Println(err)
				break
			}
			ch <- msg
		}
	}()

	for {
		fmt.Println("菜单:")
		fmt.Println("1) 修改个人信息")
		fmt.Println("2) 修改头像")
		fmt.Println("3) 获取好友列表")
		fmt.Println("4) 搜索好友")
		fmt.Println("5) 加好友")
		fmt.Println("6) 发送消息")
		fmt.Println("7) 拉取消息记录")
		select {
		case op := <- readOp():
			if op == 1 {
				changeUserInfo()
			} else if op == 2 {
				changeAvatar()
			} else if op == 3 {
				getFriendList()
			} else if op == 4 {
				searchFriend()
			} else if op == 5 {
				reqFriend()
			} else if op == 6 {
				sendMsg()
			} else if op == 7 {
				fetchMsgHistory()
			}
			break
		case msg := <-ch:
			if msg.Type == popim.PushMsg_REQFRIENDMSG {
				fmt.Println("收到 ",msg.FriendMsgData.NickName,"(",msg.FriendMsgData.PopId,") 的好友请求，自动同意")
				addFriend(msg.FriendMsgData.PopId)
			} else if msg.Type == popim.PushMsg_NEWFRIENDMSG {
				fmt.Println("新增了好友 ",msg.FriendMsgData.NickName,"(",msg.FriendMsgData.PopId,")")
			} else if msg.Type == popim.PushMsg_IMMSG {
				fmt.Println("收到新消息",msg.ImMsgData.TotalCount,"条:")
				for _, v := range msg.ImMsgData.ImMsg {
					fmt.Println((*v).Timestamp,"时间 ", (*v).FromId,"发给",(*v).ToId,": ",(*v).Content)
				}
			}
			break
		}


	}
}