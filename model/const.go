package model

import "time"

const (
	CountTb         = "count"        // 计数专用
	KeyValueTb      = "keyValue"     // 存放一些配置
	TbnSitemapIndex = "sm_i"         // key: indexStr, value: time
	TbnPostUpdate   = "topic_update" // key: topicId, value: addTime
	TbnDbImg        = "dbi"          // 上传图片 key: sdb.I2b(imgHashValue), value: img data
	TbnIpInfo       = "ip"           // user ip info
	TbnSetting      = "setting"      // key: settingKey, value: setting value
	TbnV2DecMp4     = "v2dec_mp4"    // key: saveFullPath, value: nil

	SettingKeyBadBot  = "BadBotName"
	SettingKeyBadIp   = "BadIpPrefix"
	SettingKeyAllowIp = "AllowIpPrefix"
)

var (
	TimeOffSet  = time.Duration(8) * time.Hour // 时差， 图个方便
	SettingKeys = []string{SettingKeyBadBot, SettingKeyBadIp, SettingKeyAllowIp}
)
