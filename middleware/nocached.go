package mdw

import (
	"github.com/gin-gonic/gin"
)

var (
	// 按照你原代码中的定义的变量名称保持一致
	etagHeaders = []string{
		"ETag",
		"If-Modified-Since",
		"If-Match",
		"If-None-Match",
		"If-Range",
		"If-Unmodified-Since",
	}

	noCacheHeaders = map[string]string{
		"Expires":         "Thu, 01 Jan 1970 00:00:00 UTC",
		"Cache-Control":   "no-cache, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}
)

func RspNoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 删除请求头中的 ETag 相关 Header
		for _, v := range etagHeaders {
			if c.GetHeader(v) != "" {
				c.Request.Header.Del(v)
			}
		}

		// 2. 设置禁止缓存的响应头 (Response Headers)
		for k, v := range noCacheHeaders {
			c.Header(k, v)
		}

		// 3. 执行后续 Handler
		c.Next()
	}
}
