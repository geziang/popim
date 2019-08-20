package rpc

import "github.com/geziang/popim"

var (
	InternalErrorResult = &popim.OperationResult{
		Code:                 500,
		Msg:                  "内部错误",
	}
	SuccessResult = &popim.OperationResult{
		Code:                 200,
		Msg:                  "成功",
	}
	AuthErrorResult = &popim.OperationResult{
		Code:                 403,
		Msg:                  "认证错误",
	}
	BadRequestResult = &popim.OperationResult{
		Code:                 400,
		Msg:                  "无效请求",
	}

)
