package model

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// GetUserByAIHubEmployNo 将 AI Hub employNo 映射到已存在的本地用户。
// 第一版刻意不自动创建用户，也不引入表结构变更。
func GetUserByAIHubEmployNo(employNo string, matchField string) (*User, error) {
	employNo = strings.TrimSpace(employNo)
	if employNo == "" {
		return nil, errors.New("employNo is empty")
	}

	column := "username"
	if strings.EqualFold(strings.TrimSpace(matchField), "oidc_id") {
		column = "oidc_id"
	}

	user := &User{}
	err := DB.Omit("password").Where(column+" = ?", employNo).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

// IsAIHubUserNotFound 隔离 GORM 的 not found 判断，避免 controller 直接依赖 GORM 细节。
func IsAIHubUserNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
