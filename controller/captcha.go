package controller

import (
	"bytes"
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/ego008/captcha"
	"github.com/gin-gonic/gin"
)

var (
	captchaImgWidth  = captcha.StdWidth
	captchaImgHeight = captcha.StdHeight
)

func (h *BaseHandler) CaptchaHandle(c *gin.Context) {
	dir, file := path.Split(c.Request.URL.Path)
	ext := path.Ext(file)
	id := file[:len(file)-len(ext)]
	if ext == "" || id == "" {
		c.Status(http.StatusNotFound)
		return
	}
	if len(c.Query("reload")) > 0 {
		captcha.Reload(id)
	}
	lang := strings.ToLower(c.Query("lang"))
	download := path.Base(dir) == "download"
	if errors.Is(captchaServeFastHTTP(c, id, ext, lang, download), captcha.ErrNotFound) {
		c.Status(http.StatusNotFound)
	}
}

func captchaServeFastHTTP(c *gin.Context, id, ext, lang string, download bool) error {
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	var content bytes.Buffer
	switch ext {
	case ".png":
		c.Header("Content-Type", "image/png")
		_ = captcha.WriteImage(&content, id, captchaImgWidth, captchaImgHeight)
	case ".wav":
		c.Header("Content-Type", "audio/x-wav")
		_ = captcha.WriteAudio(&content, id, lang)
	default:
		return captcha.ErrNotFound
	}

	if download {
		c.Header("Content-Type", "application/octet-stream")
	}

	c.String(http.StatusOK, content.String())
	return nil
}
