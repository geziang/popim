package rpc

import (
	"context"
	"github.com/geziang/popim"
	"github.com/geziang/popim/server/service"
	"log"
	"time"
)

type MsgServer struct {}

func (s *MsgServer) Pusher(in *popim.AuthContext, stream popim.Msg_PusherServer) error {
	popId := in.PopId
	token := in.Token

	success, err := service.UserService.CheckToken(uint(popId),token)
	if err != nil {
		return nil
	}
	if !success {
		return nil
	}

	service.MsgService.PusherLoop(uint(popId), stream)

	return nil
}

func (s *MsgServer) SendIMMsg(ctx context.Context, in *popim.SendIMMsgReq) (*popim.OperationResult,error) {
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
	targetId := uint(in.ImMsg.ToId)
	if srcId == targetId || in.Context.PopId != in.ImMsg.FromId || !service.FriendService.IsFriends(srcId,targetId) {
		return BadRequestResult, nil
	}

	success = service.MsgService.SendIMMsg(in.ImMsg)
	if !success {
		return InternalErrorResult, nil
	}

	service.MsgService.PushMsg(targetId, &popim.PushMsg{
		Timestamp:     uint64(time.Now().Unix()),
		Type:          popim.PushMsg_IMMSG,
		ImMsgData:     &popim.IMMsgs{
			TotalCount: 1,
			ImMsg:      []*popim.IMMsg{
				in.ImMsg,
			},
		},
	})

	return SuccessResult, nil
}

func (s *MsgServer) GetIMMsgHistoryBetweenSingleUser(in *popim.QueryByIdReq, stream popim.Msg_GetIMMsgHistoryBetweenSingleUserServer) error {
	popId := uint(in.Context.PopId)
	token := in.Context.Token

	success, err := service.UserService.CheckToken(popId,token)
	if err != nil {
		return nil
	}
	if !success {
		return nil
	}

	srcId := uint(in.Context.PopId)
	targetId := uint(in.QueryPopId)
	chMsgs := service.MsgService.GetIMMsgHistory(srcId,targetId)
	for {
		msg, ok := <- chMsgs
		if !ok {
			break
		}

		err := stream.Send(&popim.IMMsg{
			Timestamp: uint64(msg.Timestamp),
			FromId:    uint64(msg.FromId),
			ToId:      uint64(msg.ToId),
			Content:   msg.Content,
		})
		if err != nil {
			log.Println("Send IMMsg History failed ",err)
			break
		}
	}

	return nil
}
