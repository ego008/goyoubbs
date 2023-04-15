package model

import (
	"errors"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/tidwall/gjson"
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

func UserGetByName(db *sdb.DB, name string) (User, error) {
	obj := User{}
	rs := db.Hget("user_name2uid", []byte(name))
	if rs.OK() {
		rs2 := db.Hget(UserTbName, rs.Data[0])
		if rs2.OK() {
			_ = json.Unmarshal(rs2.Data[0], &obj)
			return obj, nil
		}
		return obj, errors.New(rs2.State)
	}
	return obj, errors.New(rs.State)
}

func UserGetById(db *sdb.DB, uid uint64) (obj User, code int) {
	if rs := db.Hget(UserTbName, sdb.I2b(uid)); rs.OK() {
		err := json.Unmarshal(rs.Bytes(), &obj)
		if err != nil {
			return
		}
		code = 1 //  存在 1
	}
	return
}

func UserGetByIds(db *sdb.DB, ids []uint64) (userLst []User) {
	if len(ids) > 0 {
		var idsb [][]byte
		for _, k := range ids {
			idsb = append(idsb, sdb.I2b(k))
		}
		db.Hmget(UserTbName, idsb).KvEach(func(_, value sdb.BS) {
			user := User{}
			err := json.Unmarshal(value.Bytes(), &user)
			if err != nil {
				return
			}
			userLst = append(userLst, user)
		})
	}
	return
}

// UserGetNamesByIds 根据 ids 取 name ，返回id:name 的map
// 只解析 Name 字段，性能提高一丁点
func UserGetNamesByIds(db *sdb.DB, ids []uint64) map[uint64]string {
	id2name := map[uint64]string{}
	if len(ids) == 0 {
		return id2name
	}
	var idsb [][]byte
	for _, k := range ids {
		if v, ok := userNameMap.Load(k); ok {
			id2name[k] = v.(string)
		} else {
			idsb = append(idsb, sdb.I2b(k))
		}
	}

	if len(idsb) > 0 {
		db.Hmget(UserTbName, idsb).KvEach(func(key, value sdb.BS) {
			name := gjson.Get(value.String(), "Name").String()
			uId := sdb.B2i(key.Bytes())
			id2name[uId] = name
			userNameMap.Store(uId, name)
		})
	}
	return id2name
}

func UserGetRecent(db *sdb.DB, tbn string, limit int) (userLst []User) {
	db.Hrscan(tbn, nil, limit).KvEach(func(_, value sdb.BS) {
		obj := User{}
		err := json.Unmarshal(value, &obj)
		if err != nil {
			return
		}
		userLst = append(userLst, obj)
	})
	return
}

func UserGetRecentByKw(db *sdb.DB, kw string, limit int) (userLst []User) {
	var keyStart []byte
	for {
		rs := db.Hrscan(UserTbName, keyStart, 50)
		if !rs.OK() {
			break
		}
		rs.KvEach(func(key, value sdb.BS) {
			if len(userLst) == limit {
				return
			}
			keyStart = key
			obj := User{}
			err := json.Unmarshal(value, &obj)
			if err != nil {
				return
			}
			if strconv.FormatUint(obj.ID, 10) == kw || strings.Contains(obj.Name, kw) {
				userLst = append(userLst, obj)
				if len(userLst) == limit {
					return
				}
			}
		})

		if len(userLst) >= limit {
			break
		}
	}
	return
}

func UserGetRecentByFlag(db *sdb.DB, tbn string, limit int) (userLst []User) {
	if tbn == UserTbName {
		db.Hrscan(tbn, nil, limit).KvEach(func(_, value sdb.BS) {
			obj := User{}
			err := json.Unmarshal(value, &obj)
			if err != nil {
				return
			}
			userLst = append(userLst, obj)
		})
		return
	}
	var idBLst [][]byte
	db.Hrscan(tbn, nil, limit).KvEach(func(key, _ sdb.BS) {
		idBLst = append(idBLst, key)
	})
	db.Hmget(UserTbName, idBLst).KvEach(func(_, value sdb.BS) {
		obj := User{}
		err := json.Unmarshal(value, &obj)
		if err != nil {
			return
		}
		userLst = append(userLst, obj)
	})
	return
}

func UserSet(db *sdb.DB, obj User) User {
	jb, err := json.Marshal(obj)
	if err != nil {
		return User{}
	}
	_ = db.Hset(UserTbName, sdb.I2b(obj.ID), jb)
	return obj
}

func UserGetAllAdmin(db *sdb.DB) (userLst []User) {
	var userIds [][]byte
	var keyStart []byte
	for {
		if rs := db.Hscan("user_flag:99", keyStart, 10); rs.OK() {
			rs.KvEach(func(key, _ sdb.BS) {
				keyStart = key
				userIds = append(userIds, key)
			})
		} else {
			break
		}
	}
	if len(userIds) == 0 {
		return
	}
	db.Hmget(UserTbName, userIds).KvEach(func(_, value sdb.BS) {
		obj := User{}
		err := json.Unmarshal(value, &obj)
		if err != nil {
			return
		}
		userLst = append(userLst, obj)
	})
	return
}
