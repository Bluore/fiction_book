package response

import (
	"fiction-book/internal/service"
)

func getErrorList() *map[error]errorResponse {
	return &map[error]errorResponse{
		service.ErrBookNotFound: errorResponse{
			Code:    4001,
			Message: "书籍未找到",
		},
		service.ErrUserNotFound: errorResponse{
			Code:    4002,
			Message: "用户未找到",
		},
		service.ErrVerification: errorResponse{
			Code:    4003,
			Message: "验证码验证验证失败",
		},
		service.ErrEmailVerifySend: errorResponse{
			Code:    4004,
			Message: "请先发送邮箱验证码",
		},
		service.ErrSizeExceed: errorResponse{
			Code:    4005,
			Message: "文件大小超出预期",
		},
		service.ErrInvalidPassword: errorResponse{
			Code:    4006,
			Message: "用户名或密码错误",
		},
		service.ErrUserExists: errorResponse{
			Code:    4007,
			Message: "用户已存在",
		},
		service.ErrBookExists: errorResponse{
			Code:    4008,
			Message: "书籍已存在",
		},
		service.ErrBookAlreadyBuy: errorResponse{
			Code:    4009,
			Message: "书籍已购买",
		},
		service.ErrBookChapterNotFound: errorResponse{
			Code:    4010,
			Message: "章节未找到",
		},
		service.ErrFileType: errorResponse{
			Code:    4011,
			Message: "文件类型出错",
		},
		service.ErrImage: errorResponse{
			Code:    4012,
			Message: "图片出错",
		},
		service.ErrEmailExists: errorResponse{
			Code:    4013,
			Message: "邮箱已存在",
		},
		service.ErrUserNotLogin: errorResponse{
			Code:    4014,
			Message: "请先完成登录",
		},
		service.ErrVIPLevel: errorResponse{
			Code:    4015,
			Message: "会员等级不足",
		},
		service.ErrUserPurchasedBookNotFound: errorResponse{
			Code:    4016,
			Message: "没有购买此书籍的记录",
		},
		service.ErrParam: errorResponse{
			Code:    4017,
			Message: "参数错误",
		},
		service.ErrSufficientFunds: errorResponse{
			Code:    4018,
			Message: "你的余额不足了，充钱使你变得强大",
		},
		service.ErrPermission: errorResponse{
			Code:    4020,
			Message: "缺失对应权限",
		},
		service.ErrOrderNotFound: errorResponse{
			Code:    4021,
			Message: "订单没找到",
		},
	}
}
