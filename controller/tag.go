package controller

import (
	"github.com/ego008/goyoubbs/model"
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
	rs := db.Hget("tag", []byte(tagLow))
	if rs.State != "ok" {
		w.Write([]byte(`{"retcode":404,"retmsg":"not found"}`))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)

	pageInfo := model.UserArticleList(db, cmd, "tag:"+tagLow, key, h.App.Cf.Site.PageShowNum)

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
	evn.SiteCf = h.App.Cf.Site
	evn.Title = tag + " - " + h.App.Cf.Site.Name
	evn.Keywords = tag
	evn.Description = tag
	evn.IsMobile = tpl == "mobile"

	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "category_detail"
	evn.HotNodes = model.CategoryHot(db, 20)
	evn.NewestNodes = model.CategoryNewest(db, 20)

	evn.Tag = tagDetail{
		Name:   tag,
		Number: db.Zget("tag_article_num", []byte(tagLow)).Uint64(),
	}
	evn.PageInfo = pageInfo

	h.Render(w, tpl, evn, "layout.html", "tag.html")
}
