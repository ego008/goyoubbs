package util

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
)

// ObjCachedSet 存缓存
// v 为 string、[]byte 直存，为struct、slice 则 json.Marshal 序列化后再存
func ObjCachedSet(mc *fastcache.Cache, k []byte, v interface{}) {
	switch v2 := v.(type) {
	case string:
		mc.Set(k, sdb.S2b(v2))
	case []byte:
		mc.Set(k, v2)
	default:
		if jb, err := json.Marshal(v); err == nil {
			mc.Set(k, jb)
		}
	}
}

func ObjCachedSetBig(mc *fastcache.Cache, k []byte, v interface{}) {
	switch v2 := v.(type) {
	case string:
		mc.SetBig(k, sdb.S2b(v2))
	case []byte:
		mc.SetBig(k, v2)
	default:
		if jb, err := json.Marshal(v); err == nil {
			mc.SetBig(k, jb)
		}
	}
}

// ObjCachedGet 取缓存
// 输入参数 getByte 指定直取 byte 内容
func ObjCachedGet(mc *fastcache.Cache, k []byte, v interface{}, getByte bool) (mcValue []byte, exist bool) {
	if mcValue = mc.Get(nil, k); len(mcValue) > 0 {
		if getByte {
			exist = true
			return
		} else {
			err := json.Unmarshal(mcValue, v)
			if err == nil {
				exist = true
				return
			}
		}
	}
	return
}

func ObjCachedGetBig(mc *fastcache.Cache, k []byte, v interface{}, getByte bool) (mcValue []byte, exist bool) {
	if mcValue = mc.GetBig(nil, k); len(mcValue) > 0 {
		if getByte {
			exist = true
			return
		} else {
			err := json.Unmarshal(mcValue, v)
			if err == nil {
				exist = true
				return
			}
		}
	}
	return
}
