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
	"github.com/ego008/youdb"
	"sort"
)

type Link struct {
	Id    uint64 `json:"id"`
	Name  string `json:"name"`
	Url   string `json:"url"`
	Score int    `json:"score"`
}

func LinkGetById(db *youdb.DB, lid string) Link {
	var item Link
	rs := db.Hget("link", youdb.DS2b(lid))
	if rs.State == "ok" {
		json.Unmarshal(rs.Data[0], &item)
	}
	return item
}

func LinkSet(db *youdb.DB, obj Link) {
	if obj.Id == 0 {
		// add
		newId, _ := db.HnextSequence("link")
		obj.Id = newId
	}
	jb, _ := json.Marshal(obj)
	db.Hset("link", youdb.I2b(obj.Id), jb)
}

func LinkList(db *youdb.DB, getAll bool) []Link {
	var items []Link
	itemMap := map[uint64]Link{}

	startKey := []byte("")

	for {
		rs := db.Hscan("link", startKey, 20)
		if rs.State == "ok" {
			for i := 0; i < len(rs.Data)-1; i += 2 {
				startKey = rs.Data[i]
				item := Link{}
				json.Unmarshal(rs.Data[i+1], &item)
				if getAll {
					// included score == 0
					itemMap[youdb.B2i(rs.Data[i])] = item
				} else {
					if item.Score > 0 {
						itemMap[youdb.B2i(rs.Data[i])] = item
					}
				}
			}
		} else {
			break
		}
	}

	if len(itemMap) > 0 {
		type Kv struct {
			Key   uint64
			Value int
		}

		var ss []Kv
		for k, v := range itemMap {
			ss = append(ss, Kv{k, v.Score})
		}

		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Value > ss[j].Value
		})

		for _, kv := range ss {
			items = append(items, itemMap[kv.Key])
		}
	}

	return items
}
