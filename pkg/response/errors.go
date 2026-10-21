package response

import "fmt"

type Err struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *Err) Error() string {
	return fmt.Sprintf("request bad :( Code:%d Message:%s", e.Code, e.Msg)
}

// 业务请求错误定义，service 层直接返回这些错误，handler 通过 PushError 统一处理
var (
	ErrBookNotFound                 = &Err{Code: 4001, Msg: "书籍未找到"}
	ErrUserNotFound                 = &Err{Code: 4002, Msg: "用户未找到"}
	ErrVerification                 = &Err{Code: 4003, Msg: "验证码验证验证失败"}
	ErrEmailVerifySend              = &Err{Code: 4004, Msg: "请先发送邮箱验证码"}
	ErrSizeExceed                   = &Err{Code: 4005, Msg: "文件大小超出预期"}
	ErrInvalidPassword              = &Err{Code: 4006, Msg: "用户名或密码错误"}
	ErrUserExists                   = &Err{Code: 4007, Msg: "用户已存在"}
	ErrBookExists                   = &Err{Code: 4008, Msg: "书籍已存在"}
	ErrBookAlreadyBuy               = &Err{Code: 4009, Msg: "书籍已购买"}
	ErrBookChapterNotFound          = &Err{Code: 4010, Msg: "章节未找到"}
	ErrFileType                     = &Err{Code: 4011, Msg: "文件类型出错"}
	ErrImage                        = &Err{Code: 4012, Msg: "图片出错"}
	ErrEmailExists                  = &Err{Code: 4013, Msg: "邮箱已存在"}
	ErrUserNotLogin                 = &Err{Code: 4014, Msg: "请先完成登录"}
	ErrVIPLevel                     = &Err{Code: 4015, Msg: "会员等级不足"}
	ErrUserPurchasedBookNotFound    = &Err{Code: 4016, Msg: "没有购买此书籍的记录"}
	ErrParam                        = &Err{Code: 4017, Msg: "参数错误"}
	ErrSufficientFunds              = &Err{Code: 4018, Msg: "你的余额不足了，充钱使你变得强大"}
	ErrPermission                   = &Err{Code: 4020, Msg: "缺失对应权限"}
	ErrOrderNotFound                = &Err{Code: 4021, Msg: "订单没找到"}
	ErrRepetitionRequest            = &Err{Code: 4022, Msg: "重复的请求"}
	ErrUserPurchasedChapterNotFound = &Err{Code: 4023, Msg: "没有购买此章节的记录"}
	ErrCreateAlipayOrder            = &Err{Code: 4024, Msg: "创建支付宝订单失败"}
	ErrAlipayNotifyVerify           = &Err{Code: 4025, Msg: "支付宝回调验签失败"}
	ErrOrderAmountMismatch          = &Err{Code: 4026, Msg: "订单金额不匹配"}
)
