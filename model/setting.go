package model

import (
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"

	"goyoubbs/util"
	"strings"
)

type SettingKv struct {
	Key   string
	Value string
}

func SettingGetByKeys(db *mdb.DB, tx *bbolt.Tx, keySs []string) (items []SettingKv) {
	kvm := map[string]struct{}{}
	var keyBs [][]byte
	for _, v := range keySs {
		keyBs = append(keyBs, mdb.S2b(v))
		kvm[v] = struct{}{}
	}

	_ = db.HMGetFunc(tx, TbnSetting, keyBs, func(key, val []byte) error {
		if len(val) == 0 {
			return nil
		}
		keyStr := string(key)
		items = append(items, SettingKv{
			Key:   keyStr,
			Value: string(val),
		})
		delete(kvm, keyStr)
		return nil
	})

	if len(kvm) > 0 {
		for k := range kvm {
			items = append(items, SettingKv{Key: k})
		}
	}

	return
}

func UpdateBadBotName(db *mdb.DB, tx *bbolt.Tx) {
	// BadBotNameMap
	_ = db.HGetFunc(tx, TbnSetting, []byte(SettingKeyBadBot), func(val []byte) error {
		if len(val) == 0 {
			return nil
		}
		curMap := Map{}
		for _, line := range util.StringSplit(string(val), ",") {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			curMap[line] = struct{}{}
		}

		cm := BadBotNameMap.Load().(Map)
		cm.Update(curMap)
		BadBotNameMap.Store(cm)
		return nil
	})
}

func UpdateBadIpPrefix(db *mdb.DB, tx *bbolt.Tx) {
	// BadIpPrefixLst
	_ = db.HGetFunc(tx, TbnSetting, []byte(SettingKeyBadIp), func(val []byte) error {
		if len(val) == 0 {
			return nil
		}
		var tmpLst []string
		kMap := map[string]struct{}{}
		for _, line := range util.StringSplit(string(val), ",") {
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
		return nil
	})
}

func UpdateAllowIpPrefix(db *mdb.DB, tx *bbolt.Tx) {
	// AllowIpPrefixLst
	_ = db.HGetFunc(tx, TbnSetting, []byte(SettingKeyAllowIp), func(val []byte) error {
		var tmpLst []string
		kMap := map[string]struct{}{}
		for _, line := range util.StringSplit(string(val), ",") {
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
		return nil
	})
}
