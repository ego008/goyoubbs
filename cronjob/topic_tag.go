package cronjob

import (
	"bytes"
	"fmt"
	"goyoubbs/model"
	"goyoubbs/util"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"
)

// getTagFromTitle remote
func getTagFromTitle(db *mdb.DB, apiUrl string) {
	aobj := model.Topic{}

	var tid, tval []byte

	_ = db.View(func(tx *bbolt.Tx) error {
		_ = db.HScanFunc(tx, "task_to_get_tag", nil, 1, func(key, val []byte) bool {
			tid = bytes.Clone(key)
			tval = bytes.Clone(val)
			return true
		})
		return nil
	})

	if len(tid) == 0 {
		return
	}

	var tVal []byte
	_ = db.View(func(tx *bbolt.Tx) error {
		_ = db.HGetFunc(tx, "topic", tid, func(val []byte) error {
			tVal = bytes.Clone(val)
			return nil
		})
		return nil
	})
	if len(tVal) == 0 {
		_ = db.Update(func(tx *bbolt.Tx) error {
			return db.HDel(tx, "task_to_get_tag", tid)
		})
		return
	}

	err := json.Unmarshal(tVal, &aobj)
	if err != nil {
		_ = db.Update(func(tx *bbolt.Tx) error {
			return db.HDel(tx, "task_to_get_tag", tid)
		})
		return
	}
	if aobj.ID == 0 {
		_ = db.Update(func(tx *bbolt.Tx) error {
			return db.HDel(tx, "task_to_get_tag", tid)
		})
		return
	}

	oldTags := aobj.Tags

	// 1. 构建 url.Values 参数
	formData := url.Values{
		"state": {"ok"},
		"ms":    {string(tval)},
	}

	// 2. 发送 POST 请求（自动设置 Content-Type 为 application/x-www-form-urlencoded）
	httpClient := &http.Client{Timeout: 10 * time.Second}
	res, err := httpClient.PostForm(apiUrl, formData)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	// 3. 检查状态码
	if res.StatusCode != http.StatusOK {
		fmt.Println("res.StatusCode", res.StatusCode)
		return
	}

	// 4. 读取响应 Body（如需要）
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	t := struct {
		Code int    `json:"code"`
		Tag  string `json:"tag"`
	}{}

	err = json.Unmarshal(body, &t)
	if err != nil {
		fmt.Println(err)
		return
	}
	// log.Println(t.Code, t.Tag)
	if t.Code == 200 {
		_ = db.Update(func(tx *bbolt.Tx) error {
			if len(t.Tag) > 0 {
				tags := util.StringSplit(t.Tag, ",")
				if len(tags) > 5 {
					tags = tags[:5]
				}

				// get once more
				var tv []byte
				_ = db.HGetFunc(tx, "topic", mdb.I2b(aobj.ID), func(val []byte) error {
					tv = bytes.Clone(val)
					return nil
				})
				if len(tv) == 0 {
					return nil
				}

				aobj := model.Topic{}
				_ = json.Unmarshal(tv, &aobj)
				aobj.Tags = strings.Join(tags, ",")
				jb, _ := json.Marshal(aobj)
				_ = db.HSet(tx, "topic", mdb.I2b(aobj.ID), jb)

				// tag send task work，自动处理tag与文章id
				at := model.TopicTag{
					ID:      aobj.ID,
					OldTags: oldTags,
					NewTags: aobj.Tags,
				}
				jb, _ = json.Marshal(at)
				_ = db.HSet(tx, "task_to_set_tag", mdb.I2b(at.ID), jb)

			}
			_ = db.HDel(tx, "task_to_get_tag", mdb.I2b(aobj.ID))
			return nil
		})
	}
	// log.Println("done")
}

func setArticleTag(mc *fastcache.Cache, db *mdb.DB) {
	_ = db.Update(func(tx *bbolt.Tx) error {
		var tKey, tVal []byte
		_ = db.HScanFunc(tx, "task_to_set_tag", nil, 1, func(key, val []byte) bool {
			tKey = bytes.Clone(key)
			tVal = bytes.Clone(val)
			return true
		})
		if len(tVal) == 0 {
			return nil
		}

		var tagChanged bool
		info := model.TopicTag{}
		err := json.Unmarshal(tVal, &info)
		if err != nil {
			return db.HDel(tx, "task_to_set_tag", tKey)
		}

		// set tag
		oldTag := util.StringSplit(info.OldTags, ",")
		newTag := util.StringSplit(info.NewTags, ",")

		// remove
		for _, tag1 := range oldTag {
			contains := false
			for _, tag2 := range newTag {
				if tag1 == tag2 {
					contains = true
					break
				}
			}
			if !contains {
				// 删除
				tagLower := strings.ToLower(tag1)
				tagLowerB := []byte(tagLower)
				_ = db.HDel(tx, "tag:"+tagLower, tKey)

				var ok bool
				_ = db.HScanFunc(tx, "tag:"+tagLower, nil, 1, func(key, val []byte) bool {
					_, _ = db.ZIncr(tx, "tag_article_num", tagLowerB, -1) // 热门标签排序
					ok = true
					return true
				})

				if !ok {
					// 删除
					_ = db.ZDel(tx, "tag_article_num", tagLowerB)
					_ = db.HDel(tx, model.TagTbName, tagLowerB)
					_, _ = db.HIncr(tx, model.CountTb, []byte(model.TagTbName), -1)
				}
				tagChanged = true
			}
		}

		// add
		for _, tag1 := range newTag {
			contains := false
			for _, tag2 := range oldTag {
				if tag1 == tag2 {
					contains = true
					break
				}
			}
			if !contains {
				tagLower := strings.ToLower(tag1)
				tagLowerB := mdb.S2b(tagLower)

				_ = db.HGetFunc(tx, model.TagTbName, tagLowerB, func(val []byte) error {
					_ = db.HSet(tx, model.TagTbName, tagLowerB, nil)
					_, _ = db.HIncr(tx, model.CountTb, mdb.S2b(model.TagTbName), 1)
					return nil
				})

				// check if not exist !important
				var ok bool
				_ = db.HGetFunc(tx, "tag:"+tagLower, tKey, func(val []byte) error {
					ok = true
					return nil
				})
				if !ok {
					_ = db.HSet(tx, "tag:"+tagLower, tKey, nil)
					_, _ = db.ZIncr(tx, "tag_article_num", tagLowerB, 1)
				}
				tagChanged = true
			}
		}

		_ = db.HDel(tx, "task_to_set_tag", tKey)

		if tagChanged {
			mc.Del([]byte("GetTagsForSide"))
		}

		return nil
	})
}
