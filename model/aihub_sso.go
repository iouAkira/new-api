package model

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

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

func IsAIHubUserNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
