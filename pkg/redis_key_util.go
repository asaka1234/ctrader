package pkg

import (
	"fmt"
	"logtech.com/exchange/ltrader/pkg/constant"
)

func UserTokenKey(token string) string {
	return fmt.Sprintf("%s%s", constant.UserTokenKeyPrefix, token)
}
