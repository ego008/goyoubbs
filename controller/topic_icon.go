package controller

import (
	"bytes"
	"github.com/ego008/goutils/lst"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"io"
	"log"
	"strconv"
)

func serveFileCon(ctx *fasthttp.RequestCtx, db *sdb.DB, uid uint64) {
	etag := strconv.FormatUint(uid, 10)
	ctx.Response.Header.Set("Etag", `"`+etag+`"`)
	ctx.Response.Header.Set("Cache-Control", "public, max-age=25920000") // 300 days

	if match := ctx.Request.Header.Peek("If-None-Match"); len(match) > 2 {
		if bytes.Equal(match[1:len(match)-1], []byte(etag)) {
			ctx.SetStatusCode(fasthttp.StatusNotModified)
			return
		}
	}

	rs := db.Hget("user_avatar", sdb.I2b(uid))
	if !rs.OK() {
		ctx.NotFound()
		return
	}
	data := rs.Bytes()

	ctx.Response.Header.Set("Content-Length", strconv.Itoa(len(data)))
	ctx.SetBody(data)
}

func serveFileCon2(ctx *fasthttp.RequestCtx, data []byte, tid, rn uint64) {
	etag := strconv.FormatUint(tid, 10) + "a" + strconv.FormatUint(rn, 10)
	ctx.Response.Header.Set("Etag", `"`+etag+`"`)
	ctx.Response.Header.Set("Cache-Control", "public, max-age=25920000") // 300 days

	ctx.Response.Header.Set("Content-Length", strconv.Itoa(len(data)))
	ctx.SetBody(data)
}

func (h *BaseHandler) TopicIconHandle(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("image/jpeg")

	tid := ctx.UserValue("tid").(string)
	tidInt, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		ctx.NotFound()
		return
	}
	db := h.App.Db
	topic := model.TopicGetById(db, tidInt)
	if topic.ID == 0 {
		// 不存在
		ctx.NotFound()
		return
	}

	tidByte := sdb.I2b(tidInt)

	var commentLst []model.CommentFmt

	topic.Comments = db.HgetInt(model.CommentNumTbName, tidByte)
	if topic.Comments == 0 {
		// 实际没走这里，在前端已指定 src="/static/avatar/*"
		serveFileCon(ctx, db, topic.UserId)
		return
	}

	if match := ctx.Request.Header.Peek("If-None-Match"); len(match) > 2 {
		etag := tid + "a" + strconv.FormatUint(topic.Comments, 10)
		if bytes.Equal(match[1:len(match)-1], []byte(etag)) {
			ctx.Response.Header.Set("Etag", `"`+etag+`"`)
			ctx.SetStatusCode(fasthttp.StatusNotModified)
			return
		}
	}

	// get from db
	commentNumB := sdb.I2b(topic.Comments)
	if h.App.Cf.Site.SaveTopicIcon {
		rs := db.Hget("topic_icon", tidByte)
		if rs.OK() {
			if bytes.Equal(rs.Bytes()[:8], commentNumB) {
				serveFileCon2(ctx, rs.Bytes()[8:], tidInt, topic.Comments)
				return
			}
		}
	}

	commentLst = model.GetAllTopicComment(h.App.Mc, db, topic, false, false)

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
		serveFileCon(ctx, db, topic.UserId)
		return
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
		ksbLst = append(ksbLst, sdb.I2b(uint64(v)))
	}

	var srcLst []io.Reader
	db.Hmget("user_avatar", ksbLst).KvEach(func(_, value sdb.BS) {
		srcLst = append(srcLst, bytes.NewReader(value))
	})

	//for _, v := range uIds {
	//	dat, err := os.ReadFile("static/avatar/" + strconv.Itoa(v) + ".jpg")
	//	if err != nil {
	//		log.Println("Read avatar err", err)
	//		continue
	//	}
	//	srcLst = append(srcLst, bytes.NewReader(dat))
	//}

	dst := new(bytes.Buffer)
	err = util.Merge(srcLst, dst)
	if err != nil {
		log.Println("Merge err", err)
		serveFileCon(ctx, db, topic.UserId)
		return
	}

	// save icon to db
	if h.App.Cf.Site.SaveTopicIcon {
		_ = db.Hset("topic_icon", tidByte, sdb.Bconcat(commentNumB, dst.Bytes()))
	}

	serveFileCon2(ctx, dst.Bytes(), tidInt, topic.Comments)
}
