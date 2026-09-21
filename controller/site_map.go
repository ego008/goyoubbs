package controller

import (
	"bufio"
	"bytes"
	"fmt"
	"goyoubbs/model"
	"goyoubbs/util"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

const maxUrlInSitemap = 50000

type locItem struct {
	loc string
	tm  int64
	pid uint64
}

func (h *BaseHandler) SiteMapHandler(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if h.App.Cf.Site.Authorized && curUser.Flag < model.FlagAuthor {
		if curUser.ID == 0 {
			c.Redirect(302, "/login")
			return
		}
		c.Redirect(302, "/setting")
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")

	db := h.App.Db

	// post index
	// newest
	var obj model.TopicLoc
	indexTmMap := map[string]locItem{}

	_ = db.Update(func(tx *bbolt.Tx) error {

		_ = db.HRScanFunc(tx, model.TopicTbName, nil, 1, func(_, val []byte) bool {
			_ = json.Unmarshal(val, &obj)
			return true
		})

		if obj.Id == 0 {
			c.String(200, "nil")
			return nil
		}

		// index

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
			indexByteLst = append(indexByteLst, mdb.S2b(li))
		}
		for k := range indexByteLst {

			kByte := indexByteLst[k]
			kStr := mdb.B2s(kByte)
			locLi := indexTmMap[kStr]

			var ok bool

			_ = db.HGetFunc(tx, model.TbnSitemapIndex, kByte, func(val []byte) error {
				locLi.tm = int64(mdb.B2i(val))
				ok = true
				return nil
			})

			if !ok {
				tmI64 := db.ZGetInt(tx, model.TbnPostUpdate, mdb.I2b(locLi.pid))
				if tmI64 > 0 {
					locLi.tm = int64(tmI64)
					_ = db.HSet(tx, model.TbnSitemapIndex, kByte, mdb.I2b(tmI64))
				} else {
					locLi.tm = maxTm
				}
			}
			indexTmMap[kStr] = locLi
		}

		return nil
	})

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

	c.String(200, buf.String())
}

func (h *BaseHandler) SitemapIndexHandler(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if h.App.Cf.Site.Authorized && curUser.Flag < model.FlagAuthor {
		if curUser.ID == 0 {
			c.Redirect(302, "/login")
			return
		}
		c.Redirect(302, "/setting")
		return
	}

	//c.Header("Content-Type", "application/xml; charset=utf-8")

	xmlFile := strings.TrimSpace(c.Param("xmlFile"))

	// get type & index form xmlFile = "posts_1.xml"
	index := strings.Index(xmlFile, "_")
	typeStr, indexStr := xmlFile[:index], xmlFile[index+1:len(xmlFile)-4] // posts, 1
	var indexInt uint64
	var err error
	indexInt, err = strconv.ParseUint(indexStr, 10, 64)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	filename := "static/sitemap/" + xmlFile
	var buf []byte
	buf, err = os.ReadFile(filename)

	_ = h.App.Db.View(func(tx *bbolt.Tx) error {

		if err == nil {
			// get sitemap_x.xml mtime
			var fileInfo os.FileInfo
			if fileInfo, err = os.Stat(filename); err != nil {
				c.Status(http.StatusInternalServerError)
				c.String(200, "500: InternalServerError")
				return nil
			}
			modifiedTime := fileInfo.ModTime()
			//
			getPidByte := mdb.I2b(indexInt * maxUrlInSitemap)
			var modifiedTm int64
			modifiedTmTmp := h.App.Db.ZGetInt(tx, model.TbnPostUpdate, getPidByte)
			if modifiedTmTmp > 0 {
				modifiedTm = int64(modifiedTmTmp)
			} else {
				_ = h.App.Db.ZRScanFunc(tx, model.TbnPostUpdate, nil, 0, 0, 1, func(key []byte, score uint64) bool {
					modifiedTm = int64(score)
					return true
				})
			}

			if modifiedTime.UTC().Unix() >= modifiedTm {
				c.Header("Content-Type", "application/xml; charset=utf-8")
				c.String(200, string(buf))
				return nil
			}

		}

		// write to file
		// scan
		var locLst []locItem

		bn := model.TopicTbName
		if typeStr == "posts" {
			bn = model.TopicTbName
		} // other todo
		fromKeyB, toKeyB := mdb.I2b((indexInt-1)*maxUrlInSitemap), mdb.I2b(indexInt*maxUrlInSitemap)

		keyStart := fromKeyB
		for {
			var ok bool
			_ = h.App.Db.HScanFunc(tx, bn, keyStart, 100, func(key, val []byte) bool {
				ok = true
				keyStart = key

				if bytes.Compare(key, toKeyB) > 0 {
					return true
				}
				obj := model.TopicLoc{}
				err = json.Unmarshal(val, &obj)
				if err != nil {
					return true
				}
				locLst = append(locLst, locItem{
					loc: "/t/" + strconv.FormatUint(obj.Id, 10),
					tm:  obj.AddTime,
				})
				return true
			})

			if !ok {
				break
			}
		}

		if len(locLst) == 0 {
			c.Status(http.StatusNotFound)
			return nil
		}

		err = writeXmlToFile(xmlFile, h.App.Cf.Site.MainDomain, locLst)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return nil
		}

		buf, err = os.ReadFile("static/sitemap/" + xmlFile)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return nil
		}

		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(200, string(buf))

		return nil
	})
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
