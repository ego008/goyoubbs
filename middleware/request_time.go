package mdw

import (
	"github.com/valyala/fasthttp"
	"runtime"
	"strconv"
	"time"
)

var goVersion = runtime.Version()

// RspTime RequestTimeWrap
func RspTime(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		startTime := time.Now()
		next(ctx)
		_, _ = ctx.WriteString(`<div class="rsptime">Based on Golang + fastHTTP + sdb` + ` | ` + goVersion + ` Processed in ` + strconv.FormatInt(time.Now().Sub(startTime).Milliseconds(), 10) + "ms</div>")
		//log.Println("wrapHandler ed <-", time.Now().Sub(startTime).String())
	}
}
