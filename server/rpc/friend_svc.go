package rpc

import (
	"context"
	"github.com/geziang/popim"
	"github.com/geziang/popim/server/service"
	"log"
)

type FriendServer struct {}

/*
 流式传输好友列表
 */
func (s *FriendServer) GetFriendList(in *popim.QueryByIdReq, stream popim.Friend_GetFriendListServer) error {
	popId := uint(in.Context.PopId)
	token := in.Context.Token

	success, err := service.UserService.CheckToken(popId,token)
	if err != nil {
		return nil
	}
	if !success {
		return nil
	}

	users := service.FriendService.GetFriendList(popId)
	totalCount := len(users)
	for _, user := range users{
		userFromDb := service.UserService.GetUserById(user.Id)

		err := stream.Send(&popim.FriendList{
			TotalCount: uint64(totalCount),
			UserInfo:   []*popim.UserInfo {
				&popim.UserInfo{
					PopId:    uint64(userFromDb.Id),
					NickName: userFromDb.NickName,
					MusicId:  uint64(userFromDb.MusicId),
				},
			},
		})
		if err != nil {
			log.Println("Send friend list failed err:",err)
			break
		}
	}

	return nil
}
/*
 以昵称搜索好友
 流式传输搜索结果
*/
func (s *FriendServer) SearchFriend(in *popim.SearchFriendReq, stream popim.Friend_SearchFriendServer) error {
	popId := uint(in.Context.PopId)
	token := in.Context.Token

	success, err := service.UserService.CheckToken(popId,token)
	if err != nil {
		return nil
	}
	if !success {
		return nil
	}

	users := service.FriendService.SearchFriend(popId, in.Keyword)
	totalCount := len(users)
	for _, user := range users{
		err := stream.Send(&popim.FriendList{
			TotalCount: uint64(totalCount),
			UserInfo:   []*popim.UserInfo {
				&popim.UserInfo{
					PopId:    uint64(user.Id),
					NickName: user.NickName,
					MusicId:  uint64(user.MusicId),
				},
			},
		})
		if err != nil {
			log.Println("Send friend list failed err:",err)
			break
		}
	}

	return nil
}

/*
 用id加好友
 发起者调用则发送好友请求
 接收者调用则完成加好友过程
 */
func (s *FriendServer) AddFriend(ctx context.Context, in *popim.QueryByIdReq) (*popim.OperationResult,error) {
	popId := uint(in.Context.PopId)
	token := in.Context.Token

	success, err := service.UserService.CheckToken(popId,token)
	if err != nil {
		return InternalErrorResult, nil
	}
	if !success {
		return AuthErrorResult, nil
	}

	srcId := uint(in.Context.PopId)
	targetId := uint(in.QueryPopId)
	if srcId == targetId || !service.UserService.IsUserExistByPopId(targetId) {
		return BadRequestResult, nil
	}

	success = service.FriendService.AddOrRequestFriend(srcId, targetId)
	if !success {
		return InternalErrorResult, nil
	}
	return SuccessResult, nil
}