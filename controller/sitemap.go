package controller

import (
	"bytes"
	"github.com/ego008/youdb"
	"goyoubbs/util"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

const maxUrlInSitemap = 50000

func (h *BaseHandler) SiteMapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")

	db := h.App.Db

	var lastID uint64
	var lastTime int64
	db.Zrscan("article_timeline", nil, nil, 1).KvEach(func(key, value youdb.BS) {
		lastID = youdb.B2i(key.Bytes())
		lastTime = int64(youdb.B2i(value.Bytes()))
	})
	if lastID == 0 {
		return
	}

	fPath := "static/sitemap.xml"
	var f os.FileInfo
	var err error
	f, err = os.Stat(fPath)
	if err == nil {
		if lastTime <= f.ModTime().UTC().Unix() {
			http.ServeFile(w, r, fPath)
			return
		}
	}

	//

	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	buf.WriteString("\n")
	buf.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	buf.WriteString("\n")

	var keyStart []byte
	var scoreStart []byte

	tt := 0
	for {
		rs := db.Zrscan("article_timeline", keyStart, scoreStart, 100)
		if !rs.OK() {
			break
		}

		rs.KvEach(func(key, value youdb.BS) {
			keyStart = key.Bytes()
			scoreStart = value.Bytes()
			tt++
			if tt <= maxUrlInSitemap {
				lastmod := util.TimeFmt(youdb.B2i(value.Bytes()), "2006-01-02", h.App.Cf.Site.TimeZone)
				buf.WriteString("<url>\n<loc>" + h.App.Cf.Site.MainDomain + "/t/" + strconv.FormatUint(youdb.B2i(key.Bytes()), 10) + "</loc>\n<lastmod>" + lastmod + "</lastmod>\n</url>\n")
			}
		})
		if tt >= maxUrlInSitemap {
			break
		}
	}

	buf.WriteString("</urlset>\n")

	_ = ioutil.WriteFile(fPath, buf.Bytes(), 0644)

	_, _ = w.Write(buf.Bytes())
}
