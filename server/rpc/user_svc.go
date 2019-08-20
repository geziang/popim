package rpc

import (
	"context"
	"github.com/geziang/popim"
	"github.com/geziang/popim/server/model"
	"github.com/geziang/popim/server/service"
	"time"
)

type UserServer struct {}

/*
 注册账户
 */
func (s *UserServer) Register(ctx context.Context, in *popim.RegisterReq) (*popim.RegisterRes,error) {
	phoneNumber := in.PhoneNumber
	//TODO: 判断手机号是否合法

	//先判断是否已注册
	if service.UserService.IsUserExistByPhoneNumber(phoneNumber) {
		return &popim.RegisterRes{
			Result: &popim.OperationResult{
				Code:                 403,
				Msg:                  "这个手机号已经注册过了",
			},
		}, nil
	}

	user := model.User{
		CreatedAt:   time.Now(),
		PhoneNumber: phoneNumber,
		NickName:    in.NickName,
		MusicId:     0,
	}

	service.UserService.RegisterUser(&user)

	return &popim.RegisterRes{
		Result: SuccessResult,
		PopId:  uint64(user.Id),
	}, nil
}

/*
 模拟登录
 */
func (s *UserServer) Login(ctx context.Context, in *popim.LoginReq) (*popim.LoginRes,error) {
	popId := uint(in.PopId)
	phoneNumber := in.PhoneNumber

	//是否存在用户
	if popId != 0 {
		if !service.UserService.IsUserExistByPopId(popId) {
			return &popim.LoginRes{
				Result: &popim.OperationResult{
					Code:                 403,
					Msg:                  "POP ID不存在",
				},
			}, nil
		}
	} else {
		//TODO: 判断手机号是否合法
		popId = service.UserService.GetPopIdByPhoneNumber(phoneNumber)
		if popId == 0 {
			return &popim.LoginRes{
				Result: &popim.OperationResult{
					Code:                 403,
					Msg:                  "手机号未注册",
				},
			}, nil
		}
	}

	token, err := service.UserService.GetNewTokenByPopId(popId)
	if err != nil {
		return &popim.LoginRes{
			Result: InternalErrorResult,
		}, nil
	}
	return &popim.LoginRes{
		Result: SuccessResult,
		Context: &popim.AuthContext{
			Token: token,
			PopId: uint64(popId),
		},
	}, nil
}

/*
 以用户id获取用户信息
 (已登录用户可调用)
 */
func (s *UserServer) GetUserInfo(ctx context.Context, in *popim.QueryByIdReq) (*popim.UserInfoRes,error) {
	popId := uint(in.Context.PopId)
	token := in.Context.Token

	success, err := service.UserService.CheckToken(popId,token)
	if err != nil {
		return &popim.UserInfoRes{
			Result: InternalErrorResult,
		}, nil
	}
	if !success {
		return &popim.UserInfoRes{
			Result: AuthErrorResult,
		}, nil
	}

	user := service.UserService.GetUserById(uint(in.QueryPopId))
	return &popim.UserInfoRes{
		Result: SuccessResult,
		Data: &popim.UserInfo{
			PopId:    uint64(user.Id),
			NickName: user.NickName,
			MusicId:  uint64(user.MusicId),
		},
	}, nil
}

/*
 以用户id获取用户头像
 (已登录用户可调用)
*/
func (s *UserServer) GetAvatar(in *popim.QueryByIdReq, stream popim.User_GetAvatarServer) error {
	popId := uint(in.Context.PopId)
	token := in.Context.Token

	success, err := service.UserService.CheckToken(popId,token)
	if err != nil {
		return nil
	}
	if !success {
		return nil
	}

	service.FileService.SendAvatarFile(popId, stream)

	return nil
}

/*
 修改用户信息
 (登录用户自身)
*/
func (s *UserServer) UpdateUserInfo(ctx context.Context, in *popim.UpdateUserInfoReq) (*popim.OperationResult,error) {
	popId := uint(in.Context.PopId)
	token := in.Context.Token

	success, err := service.UserService.CheckToken(popId,token)
	if err != nil {
		return InternalErrorResult, nil
	}
	if !success {
		return AuthErrorResult, nil
	}

	user := model.User{
		Id:          popId,
		NickName:    in.Data.NickName,
		MusicId:     uint(in.Data.MusicId),
	}
	model.DB.Model(&user).Update(user)

	return SuccessResult, nil
}

/*
 修改用户头像
 (登录用户自身)
*/
func (s *UserServer) UpdateAvatar(stream popim.User_UpdateAvatarServer) error {
	service.FileService.RecvAvatarFile(stream)
	return nil
}