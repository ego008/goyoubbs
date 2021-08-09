package controller

import (
	"bytes"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/util"
	"strconv"
)

func (h *BaseHandler) UserAvatarHandle(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("image/jpeg")
	uid := ctx.UserValue("uid").(string)
	uidInt, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		// 不是数字，取用户名
		ctx.NotFound()
		return
	}
	rs := h.App.Db.Hget("user_avatar", sdb.I2b(uidInt))
	if !rs.OK() {
		ctx.NotFound()
		return
	}

	imgData := rs.Bytes()
	etag := strconv.FormatUint(util.Xxhash(imgData), 10)
	ctx.Response.Header.Set("Etag", `"`+etag+`"`)
	// private public
	ctx.Response.Header.Set("Cache-Control", "public, max-age=25920000") // 300 days

	if match := ctx.Request.Header.Peek("If-None-Match"); len(match) > 5 {
		if bytes.Equal(match[1:len(match)-1], []byte(etag)) {
			ctx.SetStatusCode(fasthttp.StatusNotModified)
			return
		}
	}

	ctx.Response.Header.Set("Content-Length", strconv.Itoa(len(imgData)))
	ctx.SetBody(imgData)
}
