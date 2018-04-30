/*
MIT License

Copyright (c) 2017

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"github.com/goxforum/xforum/pkg/model"
	"goji.io/pat"
	"net/http"
	"strconv"
	"strings"
)

func (h *BaseHandler) TagDetail(w http.ResponseWriter, r *http.Request) {
	btn, key := r.FormValue("btn"), r.FormValue("key")
	if len(key) > 0 {
		_, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"key type err"}`))
			return
		}
	}

	tag := pat.Param(r, "tag")
	tagLow := strings.ToLower(tag)

	cmd := "hrscan"
	if btn == "prev" {
		cmd = "hscan"
	}

	db := h.App.Db
	scf := h.App.Cf.Site
	rs := db.Hget("tag", []byte(tagLow))
	if rs.State != "ok" {
		w.Write([]byte(`{"retcode":404,"retmsg":"not found"}`))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)

	pageInfo := model.UserArticleList(db, cmd, "tag:"+tagLow, key, scf.PageShowNum, scf.TimeZone)

	type tagDetail struct {
		Name   string
		Number uint64
	}
	type pageData struct {
		PageData
		Tag      tagDetail
		PageInfo model.ArticlePageInfo
	}

	tpl := h.CurrentTpl(r)

	evn := &pageData{}
	evn.SiteCf = scf
	evn.Title = tag + " - " + scf.Name
	evn.Keywords = tag
	evn.Description = tag
	evn.IsMobile = tpl == "mobile"

	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "category_detail"
	evn.HotNodes = model.CategoryHot(db, scf.CategoryShowNum)
	evn.NewestNodes = model.CategoryNewest(db, scf.CategoryShowNum)

	evn.Tag = tagDetail{
		Name:   tag,
		Number: db.Zget("tag_article_num", []byte(tagLow)).Uint64(),
	}
	evn.PageInfo = pageInfo

	h.Render(w, tpl, evn, "layout.html", "tag.html")
}
