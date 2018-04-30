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
	"strconv"
	"strings"
)

type Category struct {
	Id       uint64 `json:"id"`
	Name     string `json:"name"`
	Articles uint64 `json:"articles"`
	About    string `json:"about"`
	Hidden   bool   `json:"hidden"`
}

type CategoryMini struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`
}

type CategoryPageInfo struct {
	Items    []Category `json:"items"`
	HasPrev  bool       `json:"hasprev"`
	HasNext  bool       `json:"hasnext"`
	FirstKey uint64     `json:"firstkey"`
	LastKey  uint64     `json:"lastkey"`
}

func CategoryGetById(db *youdb.DB, cid string) (Category, error) {
	obj := Category{}
	rs := db.Hget("category", youdb.DS2b(cid))
	if rs.State == "ok" {
		json.Unmarshal(rs.Data[0], &obj)
		return obj, nil
	}
	return obj, errors.New(rs.State)
}

func CategoryHot(db *youdb.DB, limit int) []CategoryMini {
	var items []CategoryMini
	rs := db.Zrscan("category_article_num", []byte(""), []byte(""), limit)
	if rs.State == "ok" {
		var keys [][]byte
		for i := 0; i < len(rs.Data)-1; i += 2 {
			keys = append(keys, rs.Data[i])
		}
		if len(keys) > 0 {
			rs2 := db.Hmget("category", keys)
			if rs2.State == "ok" {
				for i := 0; i < len(rs2.Data)-1; i += 2 {
					item := CategoryMini{}
					json.Unmarshal(rs2.Data[i+1], &item)
					items = append(items, item)
				}
			}
		}
	}
	return items
}

func CategoryNewest(db *youdb.DB, limit int) []CategoryMini {
	var items []CategoryMini
	rs := db.Hrscan("category", []byte(""), limit)
	if rs.State == "ok" {
		for i := 0; i < len(rs.Data)-1; i += 2 {
			item := CategoryMini{}
			json.Unmarshal(rs.Data[i+1], &item)
			items = append(items, item)
		}
	}
	return items
}

func CategoryGetMain(db *youdb.DB, currentCobj Category) []CategoryMini {
	var items []CategoryMini
	item := CategoryMini{
		Id:   currentCobj.Id,
		Name: currentCobj.Name,
	}
	items = append(items, item)

	rs := db.Hget("keyValue", []byte("main_category"))
	if rs.State == "ok" {
		var keys [][]byte
		currentCidStr := strconv.FormatUint(currentCobj.Id, 10)
		cids := strings.Split(rs.String(), ",")
		for _, v := range cids {
			if v != currentCidStr {
				keys = append(keys, youdb.DS2b(v))
			}
		}
		if len(keys) > 0 {
			rs2 := db.Hmget("category", keys)
			if rs2.State == "ok" {
				for i := 0; i < len(rs2.Data)-1; i += 2 {
					item := CategoryMini{}
					json.Unmarshal(rs2.Data[i+1], &item)
					items = append(items, item)
				}
			}
		}
	}

	return items
}

func CategoryList(db *youdb.DB, cmd, key string, limit int) CategoryPageInfo {
	tb := "category"
	var items []Category
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
		rs := db.Hmget("category", keys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := Category{}
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

	return CategoryPageInfo{
		Items:    items,
		HasPrev:  hasPrev,
		HasNext:  hasNext,
		FirstKey: firstKey,
		LastKey:  lastKey,
	}
}
