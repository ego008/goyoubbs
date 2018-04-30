/*
MIT License

Copyright (c) 2017

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package model

import (
	"encoding/json"
	"errors"
	"github.com/ego008/youdb"
)

type User struct {
	Id            uint64 `json:"id"`
	Name          string `json:"name"`
	Gender        string `json:"gender"`
	Flag          int    `json:"flag"`
	Avatar        string `json:"avatar"`
	Password      string `json:"password"`
	Email         string `json:"email"`
	Url           string `json:"url"`
	Articles      uint64 `json:"articles"`
	Replies       uint64 `json:"replies"`
	RegTime       uint64 `json:"regtime"`
	LastPostTime  uint64 `json:"lastposttime"`
	LastReplyTime uint64 `json:"lastreplytime"`
	LastLoginTime uint64 `json:"lastlogintime"`
	About         string `json:"about"`
	Notice        string `json:"notice"`
	NoticeNum     int    `json:"noticenum"`
	Hidden        bool   `json:"hidden"`
	Session       string `json:"session"`
}

type UserMini struct {
	Id     uint64 `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type UserPageInfo struct {
	Items    []User `json:"items"`
	HasPrev  bool   `json:"hasprev"`
	HasNext  bool   `json:"hasnext"`
	FirstKey uint64 `json:"firstkey"`
	LastKey  uint64 `json:"lastkey"`
}

func UserGetById(db *youdb.DB, uid uint64) (User, error) {
	obj := User{}
	rs := db.Hget("user", youdb.I2b(uid))
	if rs.State == "ok" {
		json.Unmarshal(rs.Data[0], &obj)
		return obj, nil
	}
	return obj, errors.New(rs.State)
}

func UserUpdate(db *youdb.DB, obj User) error {
	jb, _ := json.Marshal(obj)
	return db.Hset("user", youdb.I2b(obj.Id), jb)
}

func UserGetByName(db *youdb.DB, name string) (User, error) {
	obj := User{}
	rs := db.Hget("user_name2uid", []byte(name))
	if rs.State == "ok" {
		rs2 := db.Hget("user", rs.Data[0])
		if rs2.State == "ok" {
			json.Unmarshal(rs2.Data[0], &obj)
			return obj, nil
		}
		return obj, errors.New(rs2.State)
	}
	return obj, errors.New(rs.State)
}

func UserGetIdByName(db *youdb.DB, name string) string {
	rs := db.Hget("user_name2uid", []byte(name))
	if rs.State == "ok" {
		return youdb.B2ds(rs.Data[0])
	}
	return ""
}

func UserListByFlag(db *youdb.DB, cmd, tb, key string, limit int) UserPageInfo {
	var items []User
	var keys [][]byte
	var hasPrev, hasNext bool
	var firstKey, lastKey uint64

	keyStart := youdb.DS2b(key)
	if cmd == "hrscan" {
		rs := db.Hrscan(tb, keyStart, limit)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				keys = append(keys, rs.Data[i])
			}
		}
	} else if cmd == "hscan" {
		rs := db.Hscan(tb, keyStart, limit)
		if rs.State == "ok" {
			for i := len(rs.Data) - 2; i >= 0; i -= 2 {
				keys = append(keys, rs.Data[i])
			}
		}
	}

	if len(keys) > 0 {
		rs := db.Hmget("user", keys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := User{}
				json.Unmarshal(rs.Data[i+1], &item)
				items = append(items, item)
				if firstKey == 0 {
					firstKey = item.Id
				}
				lastKey = item.Id
			}

			rs = db.Hscan(tb, youdb.I2b(firstKey), 1)
			if rs.State == "ok" {
				hasPrev = true
			}
			rs = db.Hrscan(tb, youdb.I2b(lastKey), 1)
			if rs.State == "ok" {
				hasNext = true
			}
		}
	}

	return UserPageInfo{
		Items:    items,
		HasPrev:  hasPrev,
		HasNext:  hasNext,
		FirstKey: firstKey,
		LastKey:  lastKey,
	}
}
