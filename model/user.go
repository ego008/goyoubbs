package model

import (
	"errors"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/tidwall/gjson"
	"go.etcd.io/bbolt"

	"strconv"
	"strings"
	"sync"
)

const (
	UserTbName    = "user"
	FlagAdmin     = 99 // 管理员
	FlagTrust     = 10 // 可信任用户，发帖回帖不审核
	FlagAuthor    = 5  // 普通用户，可正常浏览、发草稿、评论
	FlagReview    = 1  // 待审核用户
	FlagForbidden = 0  // 禁止发草稿、评论
)

var userNameMap = sync.Map{} // 缓存

type User struct {
	ID            uint64
	Name          string
	Flag          int
	Password      string
	Email         string
	Url           string
	Topics        uint64 // 发表文章数
	Replies       uint64 // 回复数
	RegTime       uint64
	LastPostTime  uint64
	LastReplyTime uint64
	LastLoginTime uint64
	About         string
	Hidden        bool // 隐藏该用户界面或该用户界面不显示广告
	Session       string
}

type UserFmt struct {
	User
	RegTimeFmt string
}

type Flag struct {
	Flag int
	Name string
}

func UserGetByName(db *mdb.DB, tx *bbolt.Tx, name string) (User, error) {
	obj := User{}
	var uVal []byte
	_ = db.HGetFunc(tx, "user_name2uid", []byte(name), func(val []byte) error {
		uVal = val
		return nil
	})

	if len(uVal) > 0 {
		_ = db.HGetFunc(tx, UserTbName, uVal, func(val []byte) error {
			_ = json.Unmarshal(val, &obj)
			return nil
		})
		if obj.ID > 0 {
			return obj, nil
		}
		return obj, errors.New("user not found")
	}
	return obj, errors.New("user not found")
}

func UserGetById(db *mdb.DB, tx *bbolt.Tx, uid uint64) (obj User, code int) {
	_ = db.HGetFunc(tx, UserTbName, mdb.I2b(uid), func(val []byte) error {
		err := json.Unmarshal(val, &obj)
		if err != nil {
			return nil
		}
		code = 1 //  存在 1
		return nil
	})
	return
}

func UserGetByIds(db *mdb.DB, tx *bbolt.Tx, ids []uint64) (userLst []User) {
	if len(ids) > 0 {
		var idsb [][]byte
		for _, k := range ids {
			idsb = append(idsb, mdb.I2b(k))
		}
		_ = db.HMGetFunc(tx, UserTbName, idsb, func(key, val []byte) error {
			if len(val) == 0 {
				return nil
			}
			user := User{}
			err := json.Unmarshal(val, &user)
			if err != nil {
				return nil
			}
			userLst = append(userLst, user)
			return nil
		})
	}
	return
}

// UserGetNamesByIds 根据 ids 取 name ，返回id:name 的map
// 只解析 Name 字段，性能提高一丁点
func UserGetNamesByIds(db *mdb.DB, tx *bbolt.Tx, ids []uint64) map[uint64]string {
	id2name := map[uint64]string{}
	if len(ids) == 0 {
		return id2name
	}
	var idsb [][]byte
	for _, k := range ids {
		if v, ok := userNameMap.Load(k); ok {
			id2name[k] = v.(string)
		} else {
			idsb = append(idsb, mdb.I2b(k))
		}
	}

	if len(idsb) > 0 {
		_ = db.HMGetFunc(tx, UserTbName, idsb, func(key, val []byte) error {
			if len(val) == 0 {
				return nil
			}
			name := gjson.Get(string(val), "Name").String()
			uId := mdb.B2i(key)
			id2name[uId] = name
			userNameMap.Store(uId, name)
			return nil
		})
	}
	return id2name
}

func UserGetRecentByKw(db *mdb.DB, tx *bbolt.Tx, kw string, limit int) (userLst []User) {
	var keyStart []byte
	for {
		var ok bool
		_ = db.HRScanFunc(tx, UserTbName, keyStart, 50, func(key, val []byte) bool {
			if len(userLst) == limit {
				return false
			}
			keyStart = key
			obj := User{}
			err := json.Unmarshal(val, &obj)
			if err != nil {
				return true
			}
			if strconv.FormatUint(obj.ID, 10) == kw || strings.Contains(obj.Name, kw) {
				userLst = append(userLst, obj)
				if len(userLst) == limit {
					return false
				}
			}
			ok = true
			return true
		})

		if !ok || len(userLst) >= limit {
			break
		}
	}
	return
}

func UserGetRecentByFlag(db *mdb.DB, tx *bbolt.Tx, tbn string, limit int) (userLst []User) {
	if tbn == UserTbName {
		_ = db.HRScanFunc(tx, tbn, nil, limit, func(_, val []byte) bool {
			obj := User{}
			err := json.Unmarshal(val, &obj)
			if err != nil {
				return true
			}
			userLst = append(userLst, obj)
			return true
		})
		return
	}
	var idBLst [][]byte
	_ = db.HRScanFunc(tx, tbn, nil, limit, func(key, _ []byte) bool {
		idBLst = append(idBLst, key)
		return true
	})

	_ = db.HMGetFunc(tx, UserTbName, idBLst, func(_, val []byte) error {
		if len(val) == 0 {
			return nil
		}
		obj := User{}
		err := json.Unmarshal(val, &obj)
		if err != nil {
			return nil
		}
		userLst = append(userLst, obj)
		return nil
	})
	return
}

func UserSet(db *mdb.DB, tx *bbolt.Tx, obj User) User {
	jb, err := json.Marshal(obj)
	if err != nil {
		return User{}
	}
	_ = db.HSet(tx, UserTbName, mdb.I2b(obj.ID), jb)
	// update UserMap
	UserMapMux.Lock()
	if _, ok := UserMap[obj.ID]; ok {
		UserMap[obj.ID] = &obj
	}
	UserMapMux.Unlock()
	return obj
}

func UserGetAllAdmin(db *mdb.DB, tx *bbolt.Tx) (userLst []User) {
	var userIds [][]byte
	var keyStart []byte
	for {
		var ok bool
		_ = db.HScanFunc(tx, "user_flag:99", keyStart, 10, func(key, val []byte) bool {
			ok = true
			keyStart = key
			userIds = append(userIds, key)
			return true
		})
		if !ok {
			break
		}
	}
	if len(userIds) == 0 {
		return
	}
	_ = db.HMGetFunc(tx, UserTbName, userIds, func(key, val []byte) error {
		if len(val) == 0 {
			return nil
		}
		obj := User{}
		err := json.Unmarshal(val, &obj)
		if err != nil {
			return nil
		}
		userLst = append(userLst, obj)
		return nil
	})
	return
}
