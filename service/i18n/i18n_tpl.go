package i18n

import (
	"errors"
	"fmt"
	"logtech.com/exchange/ltrader/model"
)

// 渲染多语言模板，输出渲染结果
func RenderMultilingualTpl(configKey string, langKey string, properties ...any) (string, error) {

	item := model.GetConfigLanguage(configKey, langKey)
	if item == nil {
		return "", errors.New("tpl not found!")
	}
	tpl := item.Content
	return fmt.Sprintf(tpl, properties...), nil
}
