package controller

import (
	"bytes"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"net/http"
)

var defaultRobots = `User-agent: *
Disallow: /admin
`

func (h *BaseHandler) Robots(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/plain; charset=utf-8")
	buf, err := ioutil.ReadFile("static/robots.txt")
	if err != nil {
		_, _ = ctx.WriteString(defaultRobots + "\nSitemap: " + h.App.Cf.Site.MainDomain + "/sitemap.xml")
		return
	}
	if !bytes.Contains(buf, []byte("Sitemap:")) {
		buf = append(buf, []byte("\nSitemap: "+h.App.Cf.Site.MainDomain+"/sitemap.xml")...)
	}
	_, _ = ctx.Write(buf)
}

func (h *BaseHandler) Ads(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/plain; charset=utf-8")
	buf, err := ioutil.ReadFile("static/ads.txt")
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		_, _ = ctx.WriteString("404: not found")
		return
	}
	_, _ = ctx.Write(buf)
}

func (h *BaseHandler) StaticFile(ctx *fasthttp.RequestCtx) {
	filePath := ctx.UserValue("filepath").(string)
	buf, err := ioutil.ReadFile("static/" + filePath)
	if err != nil {
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		_, _ = ctx.WriteString(err.Error())
		return
	}
	fileType := http.DetectContentType(buf)
	ctx.SetContentType(fileType)
	_, _ = ctx.Write(buf)
}

func (h *BaseHandler) ShowIcon(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("image/png")
	buf, err := ioutil.ReadFile("static/logo.png")
	if err != nil {
		ctx.NotFound()
		return
	}
	_, _ = ctx.Write(buf)
}
