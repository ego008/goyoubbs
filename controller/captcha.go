package controller

import (
	"bytes"
	"github.com/ego008/captcha"
	"github.com/valyala/fasthttp"
	"path"
	"strings"
)

var (
	captchaImgWidth  = captcha.StdWidth
	captchaImgHeight = captcha.StdHeight
)

func (h *BaseHandler) CaptchaHandle(ctx *fasthttp.RequestCtx) {
	dir, file := path.Split(string(ctx.URI().Path()))
	ext := path.Ext(file)
	id := file[:len(file)-len(ext)]
	if ext == "" || id == "" {
		ctx.NotFound()
		return
	}
	if len(ctx.FormValue("reload")) > 0 {
		captcha.Reload(id)
	}
	lang := strings.ToLower(string(ctx.FormValue("lang")))
	download := path.Base(dir) == "download"
	if captchaServeFastHTTP(ctx, id, ext, lang, download) == captcha.ErrNotFound {
		ctx.NotFound()
	}
}

func captchaServeFastHTTP(ctx *fasthttp.RequestCtx, id, ext, lang string, download bool) error {
	ctx.Response.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	ctx.Response.Header.Set("Pragma", "no-cache")
	ctx.Response.Header.Set("Expires", "0")

	var content bytes.Buffer
	switch ext {
	case ".png":
		ctx.Response.Header.Set("Content-Type", "image/png")
		_ = captcha.WriteImage(&content, id, captchaImgWidth, captchaImgHeight)
	case ".wav":
		ctx.Response.Header.Set("Content-Type", "audio/x-wav")
		_ = captcha.WriteAudio(&content, id, lang)
	default:
		return captcha.ErrNotFound
	}

	if download {
		ctx.Response.Header.Set("Content-Type", "application/octet-stream")
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(content.Bytes())

	return nil
}
