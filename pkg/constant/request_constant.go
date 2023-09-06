package constant

// -----------http请求-------------
type HeaderName string

const (
	ExchangeTokenName  HeaderName = "exchange-token"  //请求Token存放的header名字
	ExchangeClientName HeaderName = "exchange-client" //请求client类型存放的header名字
	ExchangeLanguage   HeaderName = "exchange-i18n"   //客户端发过来的请求的语言 zn_CN等
	ExchangeAuto       HeaderName = "exchange-auto"   //0,1 代表是否自动登录??TODO
)

const (
	SystemDefaultLanguage string = "zh_CN" //默认的语言
)
