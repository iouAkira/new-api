package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

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

// CreateAIHubSSOUser 为通过规则校验的 AI Hub 工号创建本地普通用户。
// 密码使用随机值，避免 SSO 自动创建的账户可被猜测密码直接登录。
func CreateAIHubSSOUser(employNo string) (*User, error) {
	employNo = strings.TrimSpace(employNo)
	user := &User{
		Username:    employNo,
		Password:    common.GetRandomString(16),
		DisplayName: employNo,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	if err := user.Insert(0); err != nil {
		return nil, err
	}
	return GetUserByAIHubEmployNo(employNo, "username")
}

// IsAIHubUserNotFound 隔离 GORM 的 not found 判断，避免 controller 直接依赖 GORM 细节。
func IsAIHubUserNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
