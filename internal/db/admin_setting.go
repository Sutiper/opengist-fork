package db

import (
	"fmt"
	"gorm.io/gorm/clause"
)

type AdminSetting struct {
	Key   string `gorm:"index:,unique"`
	Value string
}

const (
	SettingDisableSignup          = "disable-signup"
	SettingRequireLogin           = "require-login"
	SettingAllowGistsWithoutLogin = "allow-gists-without-login"
	SettingDisableLoginForm       = "disable-login-form"
	SettingDisableGravatar        = "disable-gravatar"
	SettingApiEnabled             = "api-enabled"

	// Anonymous gist creation
	SettingAllowAnonymousCreate      = "allow-anonymous-create"
	SettingAnonymousGistVisibility   = "anonymous-gist-visibility" // "public" or "unlisted"
	SettingAnonymousGistTTL          = "anonymous-gist-ttl"        // minutes, 0 = disabled
	SettingAnonymousGistInFeed       = "anonymous-gist-in-feed"
	SettingAllowAnonymousUpload      = "allow-anonymous-upload"    // allow file uploads for anonymous gists
)

func GetSetting(key string) (string, error) {
	var setting AdminSetting
	var err error
	switch db.Name() {
	case "mysql", "sqlite":
		err = db.Where("`key` = ?", key).First(&setting).Error
	case "postgres":
		err = db.Where("key = ?", key).First(&setting).Error
	}
	return setting.Value, err
}

func GetSettings() (map[string]string, error) {
	var settings []AdminSetting
	err := db.Find(&settings).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}

	return result, nil
}

func UpdateSetting(key string, value string) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}}, // key column
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&AdminSetting{
		Key:   key,
		Value: value,
	}).Error
}

func setSetting(key string, value string) error {
	return db.FirstOrCreate(&AdminSetting{Key: key, Value: value}, &AdminSetting{Key: key}).Error
}

func initAdminSettings(settings map[string]string) error {
	for key, value := range settings {
		if err := setSetting(key, value); err != nil {
			if !IsUniqueConstraintViolation(err) {
				return err
			}
		}
	}

	return nil
}

type AuthInfo struct{}

func (auth AuthInfo) RequireLogin() (bool, error) {
	s, err := GetSetting(SettingRequireLogin)
	if err != nil {
		return true, err
	}
	return s == "1", nil
}

func (auth AuthInfo) AllowGistsWithoutLogin() (bool, error) {
	s, err := GetSetting(SettingAllowGistsWithoutLogin)
	if err != nil {
		return false, err
	}
	return s == "1", nil
}

func (auth AuthInfo) AllowAnonymousUpload() (bool, error) {
	s, err := GetSetting(SettingAllowAnonymousUpload)
	if err != nil {
		return false, err
	}
	return s == "1", nil
}

func (auth AuthInfo) AllowAnonymousCreate() (bool, error) {
	s, err := GetSetting(SettingAllowAnonymousCreate)
	if err != nil {
		return false, err
	}
	return s == "1", nil
}

func (auth AuthInfo) AnonymousGistVisibility() (string, error) {
	s, err := GetSetting(SettingAnonymousGistVisibility)
	if err != nil || s == "" {
		return "unlisted", err
	}
	return s, nil
}

func (auth AuthInfo) AnonymousGistTTL() (int, error) {
	s, err := GetSetting(SettingAnonymousGistTTL)
	if err != nil || s == "" {
		return 0, err
	}
	n := 0
	fmt.Sscanf(s, "%d", &n)
	return n, nil
}

func (auth AuthInfo) AnonymousGistInFeed() (bool, error) {
	s, err := GetSetting(SettingAnonymousGistInFeed)
	if err != nil {
		return false, err
	}
	return s == "1", nil
}
