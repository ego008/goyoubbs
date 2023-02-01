package controller

import (
	"bytes"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"strconv"
	"strings"
)

func (h *BaseHandler) DbImageHandle(ctx *fasthttp.RequestCtx) {
	if h.App.Cf.Site.Authorized {
		ssValue := h.GetCookie(ctx, "SessionID")
		if len(ssValue) == 0 {
			_, _ = ctx.WriteString("401")
			return
		}
	}

	keys := ctx.UserValue("key").(string)
	index := strings.Index(keys, ".")
	var key, exn string
	if index > 0 {
		key, exn = keys[:index], keys[index+1:]
	} else {
		key, exn = keys, "jpeg"
	}
	uidInt, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		// 不是数字
		ctx.NotFound()
		return
	}
	rs := h.App.Db.Hget(model.TbnDbImg, sdb.I2b(uidInt))
	if !rs.OK() {
		ctx.NotFound()
		return
	}

	ctx.SetContentType("image/" + exn)

	imgData := rs.Bytes()
	etag := key
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
