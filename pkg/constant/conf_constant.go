package constant

//---kv-store里的key-------

const (
	DeviceIpChangeSwitch string = "device_ip_change_switch" //是否限制同一端多个IP同时在线；0关闭 1开启
	SystemDateFormat     string = "system_date_format"
	DefaultLanguage      string = "exchange_default_language" //默认语言 zh_CN
	LoginAutoTimeoutOpen string = "login_auto_timeout_open"   //web端循环自动请求是否影响用户登录超时 0：正常超时，1：不超时
)
