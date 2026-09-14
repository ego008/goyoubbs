package controller

import (
	"goyoubbs/util"
	"net/http"
	"strconv"
	"strings"

	"github.com/ego008/sdb"
	"github.com/gin-gonic/gin"
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
	rs := h.App.Db.Hget("user_avatar", sdb.I2b(uidInt))
	if !rs.OK() {
		c.Status(http.StatusNotFound)
		return
	}

	imgData := rs.Bytes()
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
