package controller

import (
	"bytes"
	"goyoubbs/util"
	"net/http"
	"strconv"
	"strings"

	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) UserAvatarHandle(c *gin.Context) {
	c.Header("Content-Type", "image/jpeg")
	uid := c.Param("uid.jpg")
	if len(uid) < 5 {
		c.Status(http.StatusNotFound)
		return
	}
	uid = uid[:len(uid)-4]
	uidInt, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		// 不是数字，取用户名
		c.Status(http.StatusNotFound)
		return
	}
	var imgData []byte

	_ = h.App.Db.View(func(tx *bbolt.Tx) error {
		_ = h.App.Db.HGetFunc(tx, "user_avatar", mdb.I2b(uidInt), func(val []byte) error {
			imgData = bytes.Clone(val)
			return nil
		})
		return nil
	})

	if len(imgData) == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	etag := strconv.FormatUint(util.Xxhash(imgData), 10)
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
