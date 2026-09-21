package controller

import (
	"bytes"
	"goyoubbs/model"
	"net/http"
	"strconv"
	"strings"

	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) DbImageHandle(c *gin.Context) {
	if h.App.Cf.Site.Authorized {
		ssValue := h.GetCookie(c, "SessionID")
		if len(ssValue) == 0 {
			c.String(200, "401")
			return
		}
	}

	keys := c.Param("key")
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
		c.Status(http.StatusNotFound)
		return
	}

	var imgData []byte
	_ = h.App.Db.View(func(tx *bbolt.Tx) error {
		val := h.App.Db.HGet(tx, model.TbnDbImg, mdb.I2b(uidInt))
		if len(val) > 0 {
			imgData = bytes.Clone(val)
		}
		return nil
	})
	if len(imgData) == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", "image/"+exn)

	etag := key
	c.Header("Etag", `"`+etag+`"`)
	// private public
	c.Header("Cache-Control", "public, max-age=25920000") // 300 days

	if match := c.GetHeader("If-None-Match"); len(match) > 5 {
		if strings.Trim(match, `"`) == etag {
			c.Status(http.StatusNotModified)
			return
		}
	}

	c.Data(http.StatusOK, "image/jpeg", imgData)
}
