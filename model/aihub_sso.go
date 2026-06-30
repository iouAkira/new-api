package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

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
func CreateAIHubSSOUser(employNo string, initialBalanceRMB int, group string) (*User, error) {
	employNo = strings.TrimSpace(employNo)
	user := &User{
		Username:    employNo,
		Password:    common.GetRandomString(16),
		DisplayName: employNo,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       group,
	}
	if err := user.Insert(0); err != nil {
		return nil, err
	}
	initialQuota := aiHubSSOInitialBalanceQuota(initialBalanceRMB)
	if initialQuota > 0 {
		if err := IncreaseUserQuota(user.Id, initialQuota, true); err != nil {
			return nil, err
		}
		RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("AI Hub SSO 自动开户赠送 %d 元人民币钱包余额", initialBalanceRMB))
	}
	if err := createAIHubSSODefaultToken(user, group); err != nil {
		return nil, err
	}
	return GetUserByAIHubEmployNo(employNo, "username")
}

func createAIHubSSODefaultToken(user *User, group string) error {
	key, err := common.GenerateKey()
	if err != nil {
		return err
	}
	now := common.GetTimestamp()
	token := &Token{
		UserId:             user.Id,
		Key:                key,
		Name:               user.Username + "默认令牌",
		CreatedTime:        now,
		AccessedTime:       now,
		ExpiredTime:        -1,
		UnlimitedQuota:     true,
		ModelLimitsEnabled: false,
		Group:              group,
	}
	if err := token.Insert(); err != nil {
		return err
	}
	RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("AI Hub SSO 自动开户创建 %s 分组无限制默认令牌", group))
	return nil
}

func aiHubSSOInitialBalanceQuota(initialBalanceRMB int) int {
	if initialBalanceRMB <= 0 || operation_setting.USDExchangeRate <= 0 {
		return 0
	}
	return int(float64(initialBalanceRMB) / operation_setting.USDExchangeRate * common.QuotaPerUnit)
}

// IsAIHubUserNotFound 隔离 GORM 的 not found 判断，避免 controller 直接依赖 GORM 细节。
func IsAIHubUserNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
