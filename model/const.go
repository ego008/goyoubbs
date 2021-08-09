package model

import "time"

const (
	CountTb    = "count"    // 计数专用
	StatusKvTb = "status"   // 状态或游标
	KeyValueTb = "keyValue" // 存放一些配置
)

var TimeOffSet = time.Duration(8) * time.Hour // 时差， 图个方便
