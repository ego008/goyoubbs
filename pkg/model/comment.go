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
	"github.com/goxforum/xforum/pkg/util"
	"html/template"
)

type Comment struct {
	Id       uint64 `json:"id"`
	Aid      uint64 `json:"aid"`
	Uid      uint64 `json:"uid"`
	Content  string `json:"content"`
	ClientIp string `json:"clientip"`
	AddTime  uint64 `json:"addtime"`
}

type CommentListItem struct {
	Id         uint64 `json:"id"`
	Aid        uint64 `json:"aid"`
	Uid        uint64 `json:"uid"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
	Content    string `json:"content"`
	ContentFmt template.HTML
	AddTime    uint64 `json:"addtime"`
	AddTimeFmt string `json:"addtimefmt"`
}

type CommentPageInfo struct {
	Items    []CommentListItem `json:"items"`
	HasPrev  bool              `json:"hasprev"`
	HasNext  bool              `json:"hasnext"`
	FirstKey uint64            `json:"firstkey"`
	LastKey  uint64            `json:"lastkey"`
}

func CommentGetByKey(db *youdb.DB, aid string, cid uint64) (Comment, error) {
	obj := Comment{}
	rs := db.Hget("article_comment:"+aid, youdb.I2b(cid))
	if rs.State == "ok" {
		json.Unmarshal(rs.Data[0], &obj)
		return obj, nil
	}
	return obj, errors.New(rs.State)
}

func CommentSetByKey(db *youdb.DB, aid string, cid uint64, obj Comment) error {
	jb, _ := json.Marshal(obj)
	return db.Hset("article_comment:"+aid, youdb.I2b(cid), jb)
}

func CommentDelByKey(db *youdb.DB, aid string, cid uint64) error {
	return db.Hdel("article_comment:"+aid, youdb.I2b(cid))
}

func CommentList(db *youdb.DB, cmd, tb, key string, limit, tz int) CommentPageInfo {
	var items []CommentListItem
	var citems []Comment
	userMap := map[uint64]UserMini{}
	var userKeys [][]byte
	var hasPrev, hasNext bool
	var firstKey, lastKey uint64

	keyStart := youdb.DS2b(key)
	if cmd == "hrscan" {
		rs := db.Hrscan(tb, keyStart, limit)
		if rs.State == "ok" {
			for i := len(rs.Data) - 2; i >= 0; i -= 2 {
				item := Comment{}
				json.Unmarshal(rs.Data[i+1], &item)
				citems = append(citems, item)
				userMap[item.Uid] = UserMini{}
				userKeys = append(userKeys, youdb.I2b(item.Uid))
			}
		}
	} else if cmd == "hscan" {
		rs := db.Hscan(tb, keyStart, limit)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := Comment{}
				json.Unmarshal(rs.Data[i+1], &item)
				citems = append(citems, item)
				userMap[item.Uid] = UserMini{}
				userKeys = append(userKeys, youdb.I2b(item.Uid))
			}
		}
	}

	if len(citems) > 0 {
		rs := db.Hmget("user", userKeys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := UserMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				userMap[item.Id] = item
			}
		}

		for _, citem := range citems {
			user := userMap[citem.Uid]
			item := CommentListItem{
				Id:         citem.Id,
				Aid:        citem.Aid,
				Uid:        citem.Uid,
				Name:       user.Name,
				Avatar:     user.Avatar,
				AddTime:    citem.AddTime,
				AddTimeFmt: util.TimeFmt(citem.AddTime, "2006-01-02 15:04", tz),
				ContentFmt: template.HTML(util.ContentFmt(db, citem.Content)),
			}
			items = append(items, item)
			if firstKey == 0 {
				firstKey = item.Id
			}
			lastKey = item.Id
		}

		rs = db.Hrscan(tb, youdb.I2b(firstKey), 1)
		if rs.State == "ok" {
			hasPrev = true
		}
		rs = db.Hscan(tb, youdb.I2b(lastKey), 1)
		if rs.State == "ok" {
			hasNext = true
		}
	}

	return CommentPageInfo{
		Items:    items,
		HasPrev:  hasPrev,
		HasNext:  hasNext,
		FirstKey: firstKey,
		LastKey:  lastKey,
	}
}
