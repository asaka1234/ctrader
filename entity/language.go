package entity

//语言枚举

var (
	Chinese             = NewLanguage2("zh", 1, "zh_CN", "cnName", 1, 1, "简体中文", "4,1", "{familyName}{givenName}", "中文", "￥", "CNY", 2)
	English             = NewLanguage2("en", 2, "en_US", "enName", 1, 1, "English", "5,2", "{givenName}{familyName}", "英文", "$", "USD", 2)
	Korean              = NewLanguage2("ko", 3, "ko_KR", "koName", 1, 1, "한국어", "6,3", "{familyName}{givenName}", "韩文", "₩", "KRW", 0)
	Traditional_Chinese = NewLanguage2("el", 4, "el_GR", "cnName", 1, 1, "繁体中文", "8,7", "{familyName}{givenName}", "繁体中文", "￥", "CNY", 2)
	Mongolian           = NewLanguage2("mn", 5, "mn_MN", "mnName", 1, 1, "Монгол хэл", "10,9", "{familyName}{givenName}", "蒙文", "₮", "MNT", 2)
	Russian             = NewLanguage2("ru", 6, "ru_RU", "ruName", 1, 1, "русский язык", "12,11", "{familyName}{givenName}", "俄语", "₽", "RUB", 2)
	Japanese            = NewLanguage2("ja", 7, "ja_JP", "jaName", 1, 1, "日本語", "14,13", "{familyName}{givenName}", "日语", "¥", "JPY", 2)
	Vietnamese          = NewLanguage2("vi", 8, "vi_VN", "viName", 1, 1, "Tiếng Việt", "16,15", "{familyName}{givenName}", "越南语", "₫", "VND", 2)
)

//-----------------------------------------------------

type Language struct {
	Lang             string
	LangTypeId       int
	LangType         string
	PhoneCountryName string
	Status           int
	OperateOpen      int
	ShowName         string
	CmsTypeId        string
	NameOrder        string
	Description      string
	MoneySymbol      string
	CountryCoin      string
	CoinPrecision    int
}

func NewLanguage(lang string, langTypeId int, langType string, phoneCountryName string, status int, operateOpen int, showName string, cmsTypeId string, nameOrder string, description string, moneySymbol string, countryCoin string) Language {
	return Language{
		Lang:             lang,
		LangTypeId:       langTypeId,
		LangType:         langType,
		PhoneCountryName: phoneCountryName,
		Status:           status,
		OperateOpen:      operateOpen,
		ShowName:         showName,
		CmsTypeId:        cmsTypeId,
		NameOrder:        nameOrder,
		Description:      description,
		MoneySymbol:      moneySymbol,
		CountryCoin:      countryCoin,
	}
}

func NewLanguage2(lang string, langTypeId int, langType string, phoneCountryName string, status int, operateOpen int, showName string, cmsTypeId string, nameOrder string, description string, moneySymbol string, countryCoin string, coinPrecision int) Language {
	return Language{
		Lang:             lang,
		LangTypeId:       langTypeId,
		LangType:         langType,
		PhoneCountryName: phoneCountryName,
		Status:           status,
		OperateOpen:      operateOpen,
		ShowName:         showName,
		CmsTypeId:        cmsTypeId,
		NameOrder:        nameOrder,
		Description:      description,
		MoneySymbol:      moneySymbol,
		CountryCoin:      countryCoin,
		CoinPrecision:    coinPrecision, //这个有区别
	}
}
