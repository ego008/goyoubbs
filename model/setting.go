package model

import (
	"github.com/ego008/sdb"
	"strings"
)

type SettingKv struct {
	Key   string
	Value string
}

func SettingGetByKey(db *sdb.DB, key string) (item SettingKv) {
	item.Key = key
	rs := db.Hget(TbnSetting, sdb.S2b(key))
	if !rs.OK() {
		return
	}
	item.Value = rs.String()
	return
}

func SettingGetByKeys(db *sdb.DB, keySs []string) (items []SettingKv) {
	kvm := map[string]struct{}{}
	var keyBs [][]byte
	for _, v := range keySs {
		keyBs = append(keyBs, sdb.S2b(v))
		kvm[v] = struct{}{}
	}

	db.Hmget(TbnSetting, keyBs).KvEach(func(key, value sdb.BS) {
		keyStr := sdb.B2s(key)
		items = append(items, SettingKv{
			Key:   keyStr,
			Value: string(value),
		})
		delete(kvm, keyStr)
	})

	if len(kvm) > 0 {
		for k := range kvm {
			items = append(items, SettingKv{Key: k})
		}
	}

	return
}

func UpdateBadBotName(db *sdb.DB) {
	// BadBotNameMap
	if rs := db.Hget(TbnSetting, sdb.S2b(SettingKeyBadBot)); rs.OK() {
		curMap := Map{}
		for _, line := range strings.Split(string(rs.Data[0]), ",") {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			curMap[line] = struct{}{}
		}

		cm := BadBotNameMap.Load().(Map)
		cm.Update(curMap)
		BadBotNameMap.Store(cm)
	}
}

func UpdateBadIpPrefix(db *sdb.DB) {
	// BadIpPrefixLst
	if rs := db.Hget(TbnSetting, sdb.S2b(SettingKeyBadIp)); rs.OK() {
		var tmpLst []string
		kMap := map[string]struct{}{}
		for _, line := range strings.Split(string(rs.Data[0]), ",") {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			if _, ok := kMap[line]; ok {
				continue
			}
			tmpLst = append(tmpLst, line)
		}
		BadIpPrefixLst.Copy(tmpLst)
	}
}

func UpdateAllowIpPrefix(db *sdb.DB) {
	// AllowIpPrefixLst
	if rs := db.Hget(TbnSetting, sdb.S2b(SettingKeyAllowIp)); rs.OK() {
		var tmpLst []string
		kMap := map[string]struct{}{}
		for _, line := range strings.Split(string(rs.Data[0]), ",") {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			if _, ok := kMap[line]; ok {
				continue
			}
			tmpLst = append(tmpLst, line)
		}
		AllowIpPrefixLst.Copy(tmpLst)
	}
}
