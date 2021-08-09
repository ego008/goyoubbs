package controller

import (
	"bytes"
	"github.com/ego008/sdb"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"io/ioutil"
	"os"
	"strconv"
)

const maxUrlInSitemap = 50000

func (h *BaseHandler) SiteMapHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/xml; charset=utf-8")

	db := h.App.Db

	var lastID uint64
	var lastTime int64

	var keyStart []byte
	for {
		rs := db.Hrscan(model.TopicTbName, keyStart, 1)
		if !rs.OK() {
			return
		}
		rs.KvEach(func(key, value sdb.BS) {
			keyStart = key.Bytes()
			obj := model.Topic{}
			err := json.Unmarshal(value.Bytes(), &obj)
			if err != nil {
				return
			}
			lastID = sdb.B2i(key.Bytes())
			lastTime = obj.AddTime // EditTime
		})
		if lastID > 0 {
			break
		}
	}
	if lastID == 0 {
		return
	}

	fPath := "static/sitemap.xml"
	var f os.FileInfo
	var err error
	f, err = os.Stat(fPath)
	if err == nil {
		if lastTime <= f.ModTime().UTC().Unix() {
			fasthttp.ServeFile(ctx, fPath)
			return
		}
	}

	//

	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	buf.WriteString("\n")
	buf.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	buf.WriteString("\n")

	keyStart = keyStart[:0]

	tt := 0
	for {
		rs := db.Hrscan(model.TopicTbName, keyStart, 100)
		if !rs.OK() {
			break
		}

		rs.KvEach(func(key, value sdb.BS) {
			keyStart = key.Bytes()
			tt++
			if tt <= maxUrlInSitemap {
				lastmod := util.TimeFmt(gjson.Get(value.String(), "EditTime").Int(), "2006-01-02")
				buf.WriteString("<url>\n<loc>" + h.App.Cf.Site.MainDomain + "/t/" + strconv.FormatUint(sdb.B2i(key.Bytes()), 10) + "</loc>\n<lastmod>" + lastmod + "</lastmod>\n</url>\n")
			}
		})
		if tt >= maxUrlInSitemap {
			break
		}
	}

	buf.WriteString("</urlset>\n")

	_ = ioutil.WriteFile(fPath, buf.Bytes(), 0644)

	_, _ = ctx.Write(buf.Bytes())
}
