package model

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAIHubFeaturesDatabaseCompatibility(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver, logDriver gorm.Dialector
			dbType := common.DatabaseTypeSQLite
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "aihub.db"))
				logDriver = sqlite.Open(filepath.Join(t.TempDir(), "logs.db"))
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN is not configured")
				}
				driver = mysql.Open(dsn)
				if logDSN := os.Getenv("TEST_MYSQL_LOG_DSN"); logDSN != "" {
					logDriver = mysql.Open(logDSN)
				}
				dbType = common.DatabaseTypeMySQL
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN is not configured")
				}
				driver = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
				dbType = common.DatabaseTypePostgreSQL
				if logDSN := os.Getenv("TEST_POSTGRES_LOG_DSN"); logDSN != "" {
					logDriver = postgres.New(postgres.Config{DSN: logDSN, PreferSimpleProtocol: true})
				}
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			logDB := db
			if logDriver != nil {
				logDB, err = gorm.Open(logDriver, &gorm.Config{})
				require.NoError(t, err)
			}
			var version string
			versionSQL := "SELECT version()"
			if dialect == "sqlite" {
				versionSQL = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(versionSQL).Scan(&version).Error)
			t.Logf("database version: %s; separate log database: %t", version, logDB != db)
			oldDB, oldLogDB := DB, LOG_DB
			oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
			DB, LOG_DB = db, logDB
			common.SetDatabaseTypes(dbType, dbType)
			initCol()
			t.Cleanup(func() {
				DB, LOG_DB = oldDB, oldLogDB
				common.SetDatabaseTypes(oldMainType, oldLogType)
				initCol()
				sqlDB, err := db.DB()
				require.NoError(t, err)
				require.NoError(t, sqlDB.Close())
				if logDB != db {
					logSQLDB, err := logDB.DB()
					require.NoError(t, err)
					require.NoError(t, logSQLDB.Close())
				}
			})
			require.NoError(t, db.AutoMigrate(&User{}, &Token{}, &Model{}, &Channel{}, &IpAudit{}, &IpAuditStatus{}, &IpAuditAlertLog{}))

			require.NoError(t, logDB.AutoMigrate(&Log{}))

			for i, field := range []string{"username", "oidc_id"} {
				employee := []string{"90000101", "90000102"}[i]
				user, err := CreateAIHubSSOUser(employee, "SSO user", 0, "default", field)
				require.NoError(t, err)
				found, err := GetUserByAIHubEmployNo(employee, field)
				require.NoError(t, err)
				assert.Equal(t, user.Id, found.Id)
				t.Cleanup(func() {
					require.NoError(t, db.Where("user_id = ?", user.Id).Delete(&Token{}).Error)
					require.NoError(t, logDB.Where("user_id = ?", user.Id).Delete(&Log{}).Error)
					require.NoError(t, db.Unscoped().Delete(&User{}, user.Id).Error)
				})
				var token Token
				require.NoError(t, db.Where("user_id = ?", user.Id).First(&token).Error)
				assert.Equal(t, "default", token.Group)
			}
			user, err := GetUserByAIHubEmployNo("90000101", "username")
			require.NoError(t, err)
			ip := "10.0.0.1"
			month := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
			logs := []Log{
				{UserId: user.Id, Type: LogTypeConsume, Ip: ip, TokenName: "A", CreatedAt: month.AddDate(0, -1, 0).Unix(), Quota: 10},
				{UserId: user.Id, Type: LogTypeConsume, Ip: ip, TokenName: "B", CreatedAt: month.AddDate(0, -1, 1).Unix(), Quota: 20},
				{UserId: user.Id, Type: LogTypeConsume, Ip: ip, TokenName: "B", CreatedAt: month.AddDate(0, -1, 2).Unix(), Quota: 30},
				{UserId: user.Id, Type: LogTypeConsume, Ip: ip, TokenName: "A", CreatedAt: month.Unix(), Quota: 40},
				{UserId: user.Id, Type: LogTypeConsume, Ip: ip, TokenName: "B", CreatedAt: month.Unix(), Quota: 50},
				{UserId: user.Id, Type: LogTypeConsume, Ip: ip, TokenName: "C", CreatedAt: month.Unix(), Quota: 60},
			}
			require.NoError(t, logDB.Create(&logs).Error)
			rows, err := GetIpAuditRows("2026-09", []IpAuditAccount{{UserId: user.Id}})
			require.NoError(t, err)
			require.Len(t, rows, 3)
			expected := map[string][2]int64{"A": {1, 10}, "B": {2, 50}, "C": {0, 0}}
			for _, row := range rows {
				assert.Equal(t, expected[row.TokenName][0], row.PrevCalls)
				assert.Equal(t, expected[row.TokenName][1], row.PrevQuota)
				assert.False(t, row.IsNew, "a new token on a known IP is not a new IP")
				assert.Equal(t, logs[0].CreatedAt, row.FirstSeen)
			}
			metadata := Model{ModelName: "private-model", Tags: "私有化部署"}
			require.NoError(t, db.Create(&metadata).Error)
			t.Cleanup(func() { require.NoError(t, db.Unscoped().Delete(&metadata).Error) })
			url := "http://10.0.0.2:8000"
			channel := Channel{Name: "private-channel", Key: "secret-not-for-disclosure", BaseURL: &url, Models: "private-model", Status: common.ChannelStatusEnabled}
			require.NoError(t, db.Create(&channel).Error)
			t.Cleanup(func() { require.NoError(t, db.Delete(&channel).Error) })
			nodes, err := GetDeploymentNodes("私有化部署")
			require.NoError(t, err)
			require.Len(t, nodes, 1)
			assert.Equal(t, "10.0.0.2", nodes[0].IP)
			encoded, err := common.Marshal(nodes)
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), "secret-not-for-disclosure")
			assert.NotContains(t, string(encoded), `"key"`)
		})
	}
}
