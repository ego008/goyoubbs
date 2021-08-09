package cronjob

import (
	"crypto/tls"
	"github.com/valyala/fasthttp"
	"time"
)

var (
	defaultUA      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_0_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36"
	fastHttpClient = &fasthttp.Client{
		TLSConfig:                     &tls.Config{InsecureSkipVerify: true},
		NoDefaultUserAgentHeader:      true, // Don't send: User-Agent: fasthttp
		MaxConnsPerHost:               12000,
		ReadBufferSize:                4096, // Make sure to set this big enough that your whole request can be read at once.
		WriteBufferSize:               4096, // Same but for your response.
		ReadTimeout:                   time.Second,
		WriteTimeout:                  time.Second,
		MaxIdleConnDuration:           time.Minute,
		DisableHeaderNamesNormalizing: true, // If you set the case on your headers correctly you can enable this.
	}
)
