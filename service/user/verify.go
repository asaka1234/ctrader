package user

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"logtech.com/exchange/ltrader/entity"
	"logtech.com/exchange/ltrader/model"
	"logtech.com/exchange/ltrader/pkg"
	"logtech.com/exchange/ltrader/pkg/client"
	"logtech.com/exchange/ltrader/pkg/constant"
	log "logtech.com/exchange/ltrader/pkg/logger"
	"logtech.com/exchange/ltrader/pkg/redis"
	"logtech.com/exchange/ltrader/service/i18n"
)

// 验证Token
func verifyToken(c *gin.Context) {

	//根据token判断是否存在，是否可以获取到user
	token:=c.GetHeader(constant.ExchangeTokenName)
	if token!=""{
		var redisData string
		redisKey:=pkg.UserTokenKey(token)
		if err := redis.Get(redisKey, &redisData); err != nil {
			log.Infof("get redis data fail, redisKey=%v", redisKey)
		}

		if redisData!=""{
			//拿到redis存里的user
			var user model.User
			json.Unmarshal([]byte(redisData), &user)

			//终端设备类型
			uidTerminalType := getUidLoginTerminalType(c, user.ID) //uid_pc
			lean:= getLoginUidAndIpAndDevice(c,user,token,uidTerminalType)
			if (lean){//ip如果发生变化；
				return
			}
			//model.addAttribute("user", JSON.parseObject(userJsonStr, User.class))
			request.setAttribute("uid", user.ID)
			redis.Set(WebApiConstants.LAST_OPERATE_TIME_PRE + user.ID, new Date().getTime() + "");
			redis.expire(WebApiConstants.LAST_OPERATE_TIME_PRE + user.ID, WebApiConstants.LOGIN_TIMEOUT);

			//kv配置此开关，0表示正常超时
			//来源请求头 需求正常超时 并且 无auto头或者其值为0，为用户请求
			tOpen:= false
			loginAutoTimeoutOpen := model.GetConfigKvStore(constant.LoginAutoTimeoutOpen)
			if loginAutoTimeoutOpen!=nil && loginAutoTimeoutOpen.Value=="1"{
				tOpen = true
			}

			if tOpen && c.GetHeader(constant.ExchangeAuto) == "1" {
				log.Debug("自动刷新接口，无需更新token");
			} else {
				redis.Expire(WebApiConstants.TOKEN_USER + token, WebApiConstants.LOGIN_TIMEOUT);
			}
		}
	}

}

/**
 * 获取用户登录终端类型
 * 登录uid+(pc、h5)
 */
func getUidLoginTerminalType(c *gin.Context, uid int) string {
	 exchangeClient:= getClientType(c) //设备
	 if uid>0 && exchangeClient!=""{
		return fmt.Sprintf("%d_%s",uid,exchangeClient)
	 }
	return ""
}



/**
 * 获取客户的类型 pc/h5
 *
 * @param request
 * @return
 */
func getClientType(c *gin.Context) string {
	client := c.GetHeader(constant.ExchangeClientName)
	if client=="" {
		client = "pc"
	}
	return client
}

/**
 * 4.0由于ip获取方法改变，重写了ip获取方法
 *
 * @param request
 * @return
 */
 func getRemoteAddr(c *gin.Context) string{
	 return c.RemoteIP()
 }

func getDomainName(c *gin.Context)string{
	return c.Request.Host
}

//当前访问token的ip发生变化；需要将token失效；通知下面短信
func getLoginUidAndIpAndDevice(c *gin.Context, user model.User, token, uidTerminalType string)bool {

	//获取kv配置是否限制同一端多个IP同时在线；0关闭 1开启
	device_ip_change_switch := model.GetConfigKvStore(constant.DeviceIpChangeSwitch)
	log.Infof("device_ip_change_switch==%v,uidTerminalType==%s", device_ip_change_switch,uidTerminalType)
	ip := getRemoteAddr(c) //ip
	if device_ip_change_switch!=nil && device_ip_change_switch.Value=="1"{
		loginTokens,_ := redis.HGet(constant.LoginUserTokenList, uidTerminalType)//记录全部登录的token
		if len(loginTokens)>0 {
			var loginInfoList []entity.LoginInfo //登录在线的token
			json.Unmarshal([]byte(loginTokens),&loginInfoList)

			log.Infof("List<LoginInfo> loginInfoList %d",len(loginInfoList))//在线当前用户在线token数量
			if len(loginInfoList)>0{
				if loginInfoList[0].LoginIp!=ip{
					redis.Del(WebApiConstants.TOKEN_USER+token);                       //清空redis key
					redis.hdel(WebApiConstants.LOGIN_USER_TOKEN_LIST, uidTerminalType) //干掉这个IP  因为APP内有加载H5导致两端IP不一致,下次登录还会这样。所以干掉,H5和APP都重新登录
					loginOutRemove(uidTerminalType,token);                             //remove-->login_user_token_list 中token
					if len(user.MobileNumber)>0{
						sendUnusualSms(user, c)
					} else {
						sendUnusualEmail(user, c)
					}
					return true
				}
			}
		}
	}
	return false
}


/**
 * 退出需要将 LOGIN_USER_TOKEN_LIST 中 token 删除
 */
func loginOutRemove( uidTerminalType, token string){
	loginTokens := redis.HGet(WebApiConstants.LOGIN_USER_TOKEN_LIST, uidTerminalType)//记录全部登录的token
	if loginTokens!=""{
		List<LoginInfo> loginInfoList = JSON.parseArray(loginTokens, LoginInfo.class)//登录在线的token
		for (Iterator<LoginInfo> iterator = loginInfoList.iterator(); iterator.hasNext() ) {
			LoginInfo loinfo = iterator.next()
			if (loinfo.LoginToken.equals(token)){
				iterator.remove()
			}
		}
		//删除后存放回去
		loginTokens=JSON.toJSONString(loginInfoList)
		redis.HGet(WebApiConstants.LOGIN_USER_TOKEN_LIST, uidTerminalType, loginTokens)//设置当前登录token、ip
	}
}


/**
 * 发送登陆短信
 * @param loginUser
 * @param request
 */
func sendUnusualSms(loginUser model.User,c *gin.Context) {
	mobile := "**"

	mobileLen:= len(loginUser.MobileNumber)
	if mobileLen>4 {
		//只保留最后4位, 前边的都**掩码[为了匿名展示]
		mobile = mobile+ loginUser.MobileNumber[len(loginUser.MobileNumber)-4:len(loginUser.MobileNumber)]
	}else {
		mobile = mobile+loginUser.MobileNumber
	}

	dateFormat := model.GetConfigKvStore(constant.SystemDateFormat)
	//SimpleDateFormat sdf = new SimpleDateFormat(date_format);
	//date := sdf.format(loginUser.getLastLoginTime());

	lang:= getLanguage(c) //zh_CN
	companyName := model.GetExLanguageConfig(lang).SmsHeader

	//读取:一些文案等的多语言模板(db)
	content,_ := i18n.RenderMultilingualTpl(constant.DeviceIpChangeSwitchMobileTip, lang, companyName, mobile, dateFormat.Value)
	//实际发送,调用sdk
	client.SendSmsValidataCode(loginUser.CountryCode, loginUser.MobileNumber, content)
}

/**
 * 发送登陆email
 * @param loginUser
 * @param request
 */
func sendUnusualEmail(loginUser model.User, c *gin.Context) {

	dateFormat := model.GetConfigKvStore(constant.SystemDateFormat)

	//SimpleDateFormat sdf = new SimpleDateFormat(date_format);
	//date := sdf.format(loginUser.getLastLoginTime());

	lang:= getLanguage(c) //zh_CN
	//公司名(不同国家/语言可以显示不一样的)
	companyName := model.GetExLanguageConfig(lang).SmsHeader

	//读取:一些文案等的多语言模板(db)
	content,_ := i18n.RenderMultilingualTpl(constant.DeviceIpChangeSwitchEmailTip, lang, companyName, dateFormat.Value)
	subject := companyName+ string(constant.UnusualLoginReminding)
	//实际发送,调用sdk
	client.SendEmailValidataCode(loginUser.Email, subject, content)
}

/**
 * 获取语言 zh_CH  语言编码+国家编码
 *
 * @param request
 * @return
 */
func getLanguage(c *gin.Context) string {
	language := c.GetHeader(constant.ExchangeLanguage)
	if language==""{
		kvInfo := model.GetConfigKvStore(constant.DefaultLanguage)
		if kvInfo!=nil && kvInfo.Value!=""{
			language = kvInfo.Value
		}else{
			language = constant.SystemDefaultLanguage
		}
	}
	return language
}