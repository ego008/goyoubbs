package model

import "time"

const (
	CountTb         = "count"        // 计数专用
	StatusKvTb      = "status"       // 状态或游标
	KeyValueTb      = "keyValue"     // 存放一些配置
	TbnSitemapIndex = "sm_i"         // key: indexStr, value: time
	TbnPostUpdate   = "topic_update" // key: topicId, value: addTime
)

var TimeOffSet = time.Duration(8) * time.Hour // 时差， 图个方便
