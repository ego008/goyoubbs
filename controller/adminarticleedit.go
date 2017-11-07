package controller

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/ego008/goyoubbs/model"
	"github.com/ego008/goyoubbs/util"
	"github.com/ego008/youdb"
	"github.com/rs/xid"
	"goji.io/pat"
	"net/http"
	"strconv"
	"strings"
)

func (h *BaseHandler) ArticleEdit(w http.ResponseWriter, r *http.Request) {
	aid := pat.Param(r, "aid")
	_, err := strconv.Atoi(aid)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"cid type err"}`))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		w.Write([]byte(`{"retcode":401,"retmsg":"authored err"}`))
		return
	}
	if currentUser.Flag < 99 {
		w.Write([]byte(`{"retcode":403,"retmsg":"flag forbidden}`))
		return
	}

	db := h.App.Db

	aobj, err := model.ArticleGetById(db, aid)
	if err != nil {
		w.Write([]byte(`{"retcode":403,"retmsg":"aid not found"}`))
		return
	}
	aidB := youdb.I2b(aobj.Id)

	cobj, err := model.CategoryGetById(db, strconv.FormatUint(aobj.Cid, 10))
	if err != nil {
		w.Write([]byte(`{"retcode":404,"retmsg":"` + err.Error() + `"}`))
		return
	}

	act := r.FormValue("act")

	if act == "del" {
		// remove
		// 总文章列表
		db.Zdel("article_timeline", aidB)
		// 分类文章列表
		db.Zdel("category_article_timeline:"+strconv.FormatUint(aobj.Cid, 10), aidB)
		// 用户文章列表
		db.Hdel("user_article_timeline:"+strconv.FormatUint(aobj.Uid, 10), youdb.I2b(aobj.Id))
		// 分类下文章数
		db.Zincr("category_article_num", youdb.I2b(aobj.Cid), -1)

		// set
		db.Hset("article_hidden", aidB, []byte(""))
		aobj.Hidden = true
		jb, _ := json.Marshal(aobj)
		db.Hset("article", aidB, jb)
		uobj, _ := model.UserGetById(db, aobj.Uid)
		uobj.Articles--
		jb, _ = json.Marshal(uobj)
		db.Hset("user", youdb.I2b(uobj.Id), jb)

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	type pageData struct {
		PageData
		Cobj      model.Category
		MainNodes []model.CategoryMini
		Aobj      model.Article
	}

	tpl := h.CurrentTpl(r)
	evn := &pageData{}
	evn.SiteCf = h.App.Cf.Site
	evn.Title = "修改文章"
	evn.IsMobile = tpl == "mobile"
	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "article_edit"

	evn.Cobj = cobj
	evn.MainNodes = model.CategoryGetMain(db, cobj)
	evn.Aobj = aobj

	h.SetCookie(w, "token", xid.New().String(), 1)
	h.Render(w, tpl, evn, "layout.html", "adminarticleedit.html")
}

func (h *BaseHandler) ArticleEditPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	aid := pat.Param(r, "aid")
	aidI, err := strconv.Atoi(aid)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"cid type err"}`))
		return
	}

	token := h.GetCookie(r, "token")
	if len(token) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"token cookie missed"}`))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		w.Write([]byte(`{"retcode":401,"retmsg":"authored require"}`))
		return
	}
	if currentUser.Flag < 99 {
		w.Write([]byte(`{"retcode":403,"retmsg":"flag forbidden}`))
		return
	}

	type recForm struct {
		Aid          uint64 `json:"aid"`
		Act          string `json:"act"`
		Cid          uint64 `json:"cid"`
		Title        string `json:"title"`
		Content      string `json:"content"`
		Tags         string `json:"tags"`
		CloseComment string `json:"closecomment"`
	}

	decoder := json.NewDecoder(r.Body)
	var rec recForm
	err = decoder.Decode(&rec)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"json Decode err:` + err.Error() + `"}`))
		return
	}
	defer r.Body.Close()

	rec.Aid = uint64(aidI)

	aidS := strconv.FormatUint(rec.Aid, 10)
	aidB := youdb.I2b(rec.Aid)

	rec.Title = strings.TrimSpace(rec.Title)
	rec.Content = strings.TrimSpace(rec.Content)
	rec.Tags = util.CheckTags(rec.Tags)

	db := h.App.Db
	if rec.Act == "preview" {
		tmp := struct {
			normalRsp
			Html string `json:"html"`
		}{
			normalRsp{200, ""},
			util.ContentFmt(db, rec.Content),
		}
		json.NewEncoder(w).Encode(tmp)
		return
	}

	// check title
	hash := md5.Sum([]byte(rec.Title))
	titleMd5 := hex.EncodeToString(hash[:])
	rs0 := db.Hget("title_md5", []byte(titleMd5))
	if rs0.State == "ok" && !bytes.Equal(rs0.Data[0], aidB) {
		w.Write([]byte(`{"retcode":403,"retmsg":"title has existed"}`))
		return
	}

	scf := h.App.Cf.Site

	if rec.Cid == 0 || len(rec.Title) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"missed args"}`))
		return
	}
	if len(rec.Title) > scf.TitleMaxLen {
		w.Write([]byte(`{"retcode":403,"retmsg":"TitleMaxLen limited"}`))
		return
	}
	if len(rec.Content) > scf.ContentMaxLen {
		w.Write([]byte(`{"retcode":403,"retmsg":"ContentMaxLen limited"}`))
		return
	}

	_, err = model.CategoryGetById(db, strconv.FormatUint(rec.Cid, 10))
	if err != nil {
		w.Write([]byte(`{"retcode":404,"retmsg":"` + err.Error() + `"}`))
		return
	}

	aobj, err := model.ArticleGetById(db, aidS)
	if err != nil {
		w.Write([]byte(`{"retcode":403,"retmsg":"aid not found"}`))
		return
	}

	var closeComment bool
	if rec.CloseComment == "1" {
		closeComment = true
	}

	if aobj.Cid == rec.Cid && aobj.Title == rec.Title && aobj.Content == rec.Content && aobj.Tags == rec.Tags && aobj.CloseComment == closeComment {
		w.Write([]byte(`{"retcode":201,"retmsg":"nothing changed"}`))
		return
	}

	oldCid := aobj.Cid
	oldTitle := aobj.Title
	oldTags := aobj.Tags

	aobj.Cid = rec.Cid
	aobj.Title = rec.Title
	aobj.Content = rec.Content
	aobj.Tags = rec.Tags
	aobj.CloseComment = closeComment

	jb, _ := json.Marshal(aobj)
	db.Hset("article", aidB, jb)

	if oldCid != rec.Cid {
		db.Zincr("category_article_num", youdb.I2b(rec.Cid), 1)
		db.Zincr("category_article_num", youdb.I2b(oldCid), -1)

		db.Zset("category_article_timeline:"+strconv.FormatUint(rec.Cid, 10), aidB, aobj.EditTime)
		db.Zdel("category_article_timeline:"+strconv.FormatUint(oldCid, 10), aidB)
	}

	if oldTitle != rec.Title {
		hash0 := md5.Sum([]byte(oldTitle))
		titleMd50 := hex.EncodeToString(hash0[:])
		db.Hdel("title_md5", []byte(titleMd50))
		db.Hset("title_md5", []byte(titleMd5), aidB)
	}

	if oldTags != rec.Tags {
		oldTag := strings.Split(oldTags, ",")
		newTag := strings.Split(rec.Tags, ",")

		// remove
		for _, tag1 := range oldTag {
			contains := false
			for _, tag2 := range newTag {
				if tag1 == tag2 {
					contains = true
					break
				}
			}
			if contains == false {
				tagLower := strings.ToLower(tag1)
				db.Hdel("tag", []byte(tagLower))
				db.Hdel("tag:"+tagLower, aidB)
				db.Zincr("tag_article_num", []byte(tagLower), -1)
			}
		}
		// add
		for _, tag1 := range newTag {
			contains := false
			for _, tag2 := range oldTag {
				if tag1 == tag2 {
					contains = true
					break
				}
			}
			if contains == false {
				tagLower := strings.ToLower(tag1)
				if db.Hget("tag", []byte(tagLower)).State != "ok" {
					db.Hset("tag", []byte(tagLower), []byte(""))
					db.HnextSequence("tag")
				}
				// check if not exist !important
				if db.Hget("tag:"+tagLower, aidB).State != "ok" {
					db.Hset("tag:"+tagLower, aidB, []byte(""))
					db.Zincr("tag_article_num", []byte(tagLower), 1)
				}
			}
		}
	}

	h.DelCookie(w, "token")

	tmp := struct {
		normalRsp
		Aid uint64 `json:"aid"`
	}{
		normalRsp{200, "ok"},
		aobj.Id,
	}
	json.NewEncoder(w).Encode(tmp)
}
