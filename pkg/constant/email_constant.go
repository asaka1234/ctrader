package constant

type EmailOperateTypeEnum string

//发送的邮件的标题

const (
	RegByEmail            EmailOperateTypeEnum = "邮箱注册"
	BindEmail             EmailOperateTypeEnum = "绑定邮箱"
	FindPassword          EmailOperateTypeEnum = "找回密码"
	EmailLogin            EmailOperateTypeEnum = "邮箱登录"
	LoginAgain            EmailOperateTypeEnum = "二次登录"
	AddCryptoAddr         EmailOperateTypeEnum = "添加提现地址"
	ModifyEmail           EmailOperateTypeEnum = "修改邮箱"
	LoginReminding        EmailOperateTypeEnum = "登录提醒"
	CryptoWithdraw        EmailOperateTypeEnum = "数字货币提现"
	NewCoinPurchase       EmailOperateTypeEnum = "IEO新币申购"
	UnusualLoginReminding EmailOperateTypeEnum = "异常登录"
	InnerWithdraw         EmailOperateTypeEnum = "内部转账/站内直转"
)
