package controller

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const maxUrlInSitemap = 50000

type locItem struct {
	loc string
	tm  int64
	pid uint64
}

func (h *BaseHandler) SiteMapHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/xml; charset=utf-8")

	db := h.App.Db

	// post index
	// newest
	var obj model.TopicLoc
	db.Hrscan(model.TopicTbName, nil, 1).KvEach(func(_, value sdb.BS) {
		_ = json.Unmarshal(value, &obj)
	})

	if obj.Id == 0 {
		_, _ = ctx.WriteString("nil")
		return
	}

	// index
	indexTmMap := map[string]locItem{}
	var indexByteLst [][]byte
	maxId := int(obj.Id)
	maxTm := obj.AddTime
	for i := 1; i < maxId/maxUrlInSitemap+2; i++ {
		li := "posts_" + strconv.Itoa(i) + ".xml"
		indexTmMap[li] = locItem{
			loc: li,
			tm:  0,
			pid: uint64(i * maxUrlInSitemap),
		}
		indexByteLst = append(indexByteLst, sdb.S2b(li))
	}
	for k := range indexByteLst {

		kByte := indexByteLst[k]
		kStr := sdb.B2s(kByte)
		locLi := indexTmMap[kStr]
		rs := db.Hget(model.TbnSitemapIndex, kByte)
		if rs.OK() {
			locLi.tm = rs.Int64()
		} else {
			tmI64 := db.Zget(model.TbnPostUpdate, sdb.I2b(locLi.pid))
			if tmI64 > 0 {
				locLi.tm = int64(tmI64)
				_ = db.Hset(model.TbnSitemapIndex, kByte, sdb.I2b(tmI64))
			} else {
				locLi.tm = maxTm
			}
		}
		indexTmMap[kStr] = locLi
	}

	// output
	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	buf.WriteString("\n")
	buf.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	buf.WriteString("\n")

	for k := range indexTmMap {
		item := indexTmMap[k]
		buf.WriteString("<sitemap>\n")
		buf.WriteString("<loc>" + h.App.Cf.Site.MainDomain + "/sitemap/" + item.loc + "</loc>\n")
		buf.WriteString("<lastmod>" + util.TimeFmt(item.tm, "2006-01-02T15:04:05+00:00") + "</lastmod>\n")
		buf.WriteString("</sitemap>\n")
	}

	buf.WriteString("</sitemapindex>")

	_, _ = ctx.Write(buf.Bytes())
}

func (h *BaseHandler) SitemapIndexHandler(ctx *fasthttp.RequestCtx) {
	//ctx.SetContentType("application/xml; charset=utf-8")

	xmlFile := strings.TrimSpace(ctx.UserValue("xmlFile").(string))

	// get type & index form xmlFile = "posts_1.xml"
	index := strings.Index(xmlFile, "_")
	typeStr, indexStr := xmlFile[:index], xmlFile[index+1:len(xmlFile)-4] // posts, 1
	var indexInt uint64
	var err error
	indexInt, err = strconv.ParseUint(indexStr, 10, 64)
	if err != nil {
		ctx.NotFound()
		return
	}

	filename := "static/sitemap/" + xmlFile
	var buf []byte
	buf, err = ioutil.ReadFile(filename)
	if err == nil {
		// get sitemap_x.xml mtime
		var fileInfo os.FileInfo
		if fileInfo, err = os.Stat(filename); err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			_, _ = ctx.WriteString("500: InternalServerError")
			return
		}
		modifiedTime := fileInfo.ModTime()
		//
		getPidByte := sdb.I2b(indexInt * maxUrlInSitemap)
		var modifiedTm int64
		modifiedTmTmp := h.App.Db.Zget(model.TbnPostUpdate, getPidByte)
		if modifiedTmTmp > 0 {
			modifiedTm = int64(modifiedTmTmp)
		} else {
			h.App.Db.Zrscan(model.TbnPostUpdate, nil, nil, 1).KvEach(func(_, value sdb.BS) {
				modifiedTm = value.Int64()
			})
		}

		if modifiedTime.UTC().Unix() >= modifiedTm {
			ctx.SetContentType("application/xml; charset=utf-8")
			_, _ = ctx.Write(buf)
			return
		}

	}

	// write to file
	// scan
	var locLst []locItem

	bn := model.TopicTbName
	if typeStr == "posts" {
		bn = model.TopicTbName
	} // other todo
	fromKeyB, toKeyB := sdb.I2b((indexInt-1)*maxUrlInSitemap), sdb.I2b(indexInt*maxUrlInSitemap)

	keyStart := fromKeyB
	for {
		rs := h.App.Db.Hscan(bn, keyStart, 100)
		if !rs.OK() {
			break
		}
		keyStart = rs.Data[len(rs.Data)-2]
		rs.KvEach(func(key, value sdb.BS) {
			if bytes.Compare(key, toKeyB) > 0 {
				return
			}
			obj := model.TopicLoc{}
			err = json.Unmarshal(value, &obj)
			if err != nil {
				return
			}
			locLst = append(locLst, locItem{
				loc: "/t/" + strconv.FormatUint(obj.Id, 10),
				tm:  obj.AddTime,
			})
		})
	}

	if len(locLst) == 0 {
		ctx.NotFound()
		return
	}

	err = writeXmlToFile(xmlFile, h.App.Cf.Site.MainDomain, locLst)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	buf, err = ioutil.ReadFile("static/sitemap/" + xmlFile)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/xml; charset=utf-8")
	_, _ = ctx.Write(buf)
}

func writeXmlToFile(fn, domain string, locLst []locItem) error {
	if err := util.AutoCreateDir("static/sitemap"); err != nil {
		return err
	}
	f, err := os.Create("static/sitemap/" + fn)
	if err != nil {
		return err
	}

	buf := bufio.NewWriter(f)

	_, err = buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	if err != nil {
		return err
	}
	_, err = buf.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	if err != nil {
		return err
	}

	for _, item := range locLst {
		line := fmt.Sprintf("<url>\n<loc>%s%s</loc>\n<lastmod>%s</lastmod>\n</url>\n", domain, item.loc, util.TimeFmt(item.tm, "2006-01-02T15:04:05+00:00"))
		_, err = buf.WriteString(line)
		if err != nil {
			return err
		}
	}
	_, err = buf.WriteString("</urlset>\n")
	if err != nil {
		return err
	}

	err = buf.Flush()
	if err != nil {
		return err
	}
	return f.Close()
}
