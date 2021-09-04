package cronjob

import (
	"fmt"
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"log"
	"strings"
)

// getTagFromTitle remote
func getTagFromTitle(db *sdb.DB, apiUrl string) {
	rs := db.Hscan("task_to_get_tag", nil, 1)
	if !rs.OK() {
		return
	}
	tid := rs.Data[0].Bytes()

	rs2 := db.Hget("topic", tid)
	if !rs2.OK() {
		_ = db.Hdel("task_to_get_tag", tid)
		return
	}
	aobj := model.Topic{}
	err := json.Unmarshal(rs2.Data[0].Bytes(), &aobj)
	if err != nil {
		_ = db.Hdel("task_to_get_tag", tid)
		return
	}
	if aobj.ID == 0 {
		_ = db.Hdel("task_to_get_tag", tid)
		return
	}
	oldTags := aobj.Tags
	//if len(oldTags) > 0 {
	//	_ = db.Hdel("task_to_get_tag", tid)
	//	return
	//}

	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()

	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()

	// 默认是application/x-www-form-urlencoded
	// req.Header.SetContentType("application/json")
	req.Header.SetContentType("application/x-www-form-urlencoded") // !important
	req.Header.SetMethod("POST")

	req.SetRequestURI(apiUrl)
	req.SetBodyString(`state=ok&ms=` + rs.Data[1].String())

	err = fastHttpClient.Do(req, res)
	if err != nil {
		fmt.Println(err)
		return
	}

	if res.StatusCode() != fasthttp.StatusOK {
		fmt.Println("res.StatusCode", res.StatusCode())
		return
	}

	t := struct {
		Code int    `json:"code"`
		Tag  string `json:"tag"`
	}{}

	err = json.Unmarshal(res.Body(), &t)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Println(t.Code, t.Tag)
	if t.Code == 200 {
		if len(t.Tag) > 0 {
			tags := strings.Split(t.Tag, ",")
			if len(tags) > 5 {
				tags = tags[:5]
			}

			// get once more
			rs2 := db.Hget("topic", sdb.I2b(aobj.ID))
			if rs2.OK() {
				aobj := model.Topic{}
				_ = json.Unmarshal(rs2.Data[0].Bytes(), &aobj)
				aobj.Tags = strings.Join(tags, ",")
				jb, _ := json.Marshal(aobj)
				_ = db.Hset("topic", sdb.I2b(aobj.ID), jb)

				// tag send task work，自动处理tag与文章id
				at := model.TopicTag{
					ID:      aobj.ID,
					OldTags: oldTags,
					NewTags: aobj.Tags,
				}
				jb, _ = json.Marshal(at)
				_ = db.Hset("task_to_set_tag", sdb.I2b(at.ID), jb)
			}
		}
		_ = db.Hdel("task_to_get_tag", sdb.I2b(aobj.ID))
	}
	log.Println("done")
}

func setArticleTag(mc *fastcache.Cache, db *sdb.DB) {
	rs := db.Hscan("task_to_set_tag", nil, 1)
	if rs.OK() {
		var tagChanged bool
		info := model.TopicTag{}
		err := json.Unmarshal(rs.Data[1].Bytes(), &info)
		if err != nil {
			return
		}
		//log.Println("aid", info.Id)

		// set tag
		oldTag := strings.Split(info.OldTags, ",")
		newTag := strings.Split(info.NewTags, ",")

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
				tagLowerB := sdb.S2b(tagLower)
				_ = db.Hdel("tag:"+tagLower, sdb.I2b(info.ID))

				if db.Hscan("tag:"+tagLower, nil, 1).OK() {
					_, _ = db.Zincr("tag_article_num", tagLowerB, -1) // 热门标签排序
				} else {
					// 删除
					_ = db.Zdel("tag_article_num", tagLowerB)
					_ = db.Hdel(model.TagTbName, tagLowerB)
					_, _ = db.Hincr(model.CountTb, sdb.S2b(model.TagTbName), -1)
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
				tagLowerB := sdb.S2b(tagLower)
				if !db.Hget(model.TagTbName, tagLowerB).OK() {
					_ = db.Hset(model.TagTbName, tagLowerB, nil)
					_, _ = db.Hincr(model.CountTb, sdb.S2b(model.TagTbName), 1)
				}
				// check if not exist !important
				if !db.Hget("tag:"+tagLower, sdb.I2b(info.ID)).OK() {
					_ = db.Hset("tag:"+tagLower, sdb.I2b(info.ID), nil)
					_, _ = db.Zincr("tag_article_num", tagLowerB, 1)
				}
				tagChanged = true
			}
		}

		_ = db.Hdel("task_to_set_tag", sdb.I2b(info.ID))

		if tagChanged {
			mc.Del([]byte("GetTagsForSide"))
		}
	}
}
