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
	"github.com/ego008/youdb"
	"github.com/goxforum/xforum/pkg/model"
	"goji.io/pat"
	"net/http"
	"strconv"
)

func (h *BaseHandler) CategoryDetail(w http.ResponseWriter, r *http.Request) {
	btn, key, score := r.FormValue("btn"), r.FormValue("key"), r.FormValue("score")
	if len(key) > 0 {
		_, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"key type err"}`))
			return
		}
	}
	if len(score) > 0 {
		_, err := strconv.ParseUint(score, 10, 64)
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"score type err"}`))
			return
		}
	}

	cid := pat.Param(r, "cid")
	_, err := strconv.Atoi(cid)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"cid type err"}`))
		return
	}

	cmd := "zrscan"
	if btn == "prev" {
		cmd = "zscan"
	}

	db := h.App.Db
	scf := h.App.Cf.Site
	cobj, err := model.CategoryGetById(db, cid)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)

	if cobj.Hidden && currentUser.Flag < 99 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"retcode":404,"retmsg":"not found"}`))
		return
	}
	cobj.Articles = db.Zget("category_article_num", youdb.I2b(cobj.Id)).Uint64()
	pageInfo := model.ArticleList(db, cmd, "category_article_timeline:"+cid, key, score, scf.HomeShowNum, scf.TimeZone)

	type pageData struct {
		PageData
		Cobj     model.Category
		PageInfo model.ArticlePageInfo
	}

	tpl := h.CurrentTpl(r)

	evn := &pageData{}
	evn.SiteCf = scf
	evn.Title = cobj.Name + " - " + scf.Name
	evn.Keywords = cobj.Name
	evn.Description = cobj.About
	evn.IsMobile = tpl == "mobile"

	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "category_detail"
	evn.HotNodes = model.CategoryHot(db, scf.CategoryShowNum)
	evn.NewestNodes = model.CategoryNewest(db, scf.CategoryShowNum)

	evn.Cobj = cobj
	evn.PageInfo = pageInfo

	h.Render(w, tpl, evn, "layout.html", "category.html")
}
