package controller

import (
	"bytes"
	"goyoubbs/model"
	"goyoubbs/util"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ego008/goutils/lst"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func serveFileCon(c *gin.Context, db *mdb.DB, tx *bbolt.Tx, uid uint64) {
	etag := strconv.FormatUint(uid, 10)
	c.Header("Etag", `"`+etag+`"`)
	c.Header("Cache-Control", "public, max-age=25920000") // 300 days

	if match := c.GetHeader("If-None-Match"); len(match) > 2 {
		if strings.Trim(match, `"`) == etag {
			c.Status(http.StatusNotModified)
			return
		}
	}

	var data []byte

	_ = db.HGetFunc(tx, "user_avatar", mdb.I2b(uid), func(val []byte) error {
		data = bytes.Clone(val)
		return nil
	})

	if len(data) == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	c.Data(200, "image/jpeg", data)
}

func serveFileCon2(c *gin.Context, data []byte, tid, rn uint64) {
	etag := strconv.FormatUint(tid, 10) + "a" + strconv.FormatUint(rn, 10)
	c.Header("Etag", `"`+etag+`"`)
	c.Header("Cache-Control", "public, max-age=25920000") // 300 days

	c.Data(200, "image/jpeg", data)
}

func (h *BaseHandler) TopicIconHandle(c *gin.Context) {
	c.Header("Content-Type", "image/jpeg")
	tid := c.Param("tid.jpg")
	if len(tid) < 5 {
		c.Status(http.StatusNotFound)
		return
	}
	tid = tid[:len(tid)-4]
	tidInt, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	db := h.App.Db

	dst := new(bytes.Buffer)
	var tidByte, commentNumB []byte

	_ = db.View(func(tx *bbolt.Tx) error {

		topic := model.TopicGetById(db, tx, tidInt)
		if topic.ID == 0 {
			// 不存在
			c.Status(http.StatusNotFound)
			return nil
		}

		tidByte = mdb.I2b(tidInt)

		var commentLst []model.CommentFmt

		topic.Comments = db.HGetInt(tx, model.CommentNumTbName, tidByte)
		if topic.Comments == 0 {
			// 实际没走这里，在前端已指定 src="/avatar/*"
			serveFileCon(c, db, tx, topic.UserId)
			return nil
		}

		if match := c.GetHeader("If-None-Match"); len(match) > 2 {
			etag := tid + "a" + strconv.FormatUint(topic.Comments, 10)
			if strings.Trim(match, `"`) == etag {
				c.Header("Etag", `"`+etag+`"`)
				c.Status(http.StatusNotModified)
				return nil
			}
		}

		// get from db
		commentNumB = mdb.I2b(topic.Comments)
		if h.App.Cf.Site.SaveTopicIcon {
			var ok bool
			_ = db.HGetFunc(tx, "topic_icon", tidByte, func(val []byte) error {
				if bytes.Equal(val[:8], commentNumB) {
					serveFileCon2(c, val[8:], tidInt, topic.Comments)
					ok = true
				}
				return nil
			})
			if ok {
				return nil
			}
		}

		commentLst = model.GetAllTopicComment(h.App.Mc, db, tx, topic, false, false)

		// 图片9宫格
		// user uIds
		var uIds lst.IntLst
		uIds = append(uIds, int(topic.UserId))
		for _, cObj := range commentLst {
			if !uIds.Has(int(cObj.UserId)) {
				uIds = append(uIds, int(cObj.UserId))
			}
		}
		if len(uIds) == 1 {
			serveFileCon(c, db, tx, topic.UserId)
			return nil
		}
		if len(uIds) > 9 {
			uIds = append(uIds[:1], uIds[len(uIds)-8:]...)
		}
		// reverse
		for i, j := 0, len(uIds)-1; i < j; i, j = i+1, j-1 {
			uIds[i], uIds[j] = uIds[j], uIds[i]
		}

		//
		var ksbLst [][]byte
		for _, v := range uIds {
			ksbLst = append(ksbLst, mdb.I2b(uint64(v)))
		}

		var srcLst []io.Reader
		_ = db.HMGetFunc(tx, "user_avatar", ksbLst, func(_, val []byte) error {
			if len(val) == 0 {
				return nil
			}
			srcLst = append(srcLst, bytes.NewReader(val))
			return nil
		})

		err = util.Merge(srcLst, dst)
		if err != nil {
			log.Println("Merge err", err)
			serveFileCon(c, db, tx, topic.UserId)
			return nil
		}

		serveFileCon2(c, dst.Bytes(), tidInt, topic.Comments)
		return nil
	})

	// save icon to db
	if h.App.Cf.Site.SaveTopicIcon {
		_ = db.Update(func(tx *bbolt.Tx) error {
			_ = db.HSet(tx, "topic_icon", tidByte, mdb.BConcat(commentNumB, dst.Bytes()))
			return nil
		})
	}
}
