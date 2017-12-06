package controller

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/ego008/goyoubbs/model"
	"github.com/ego008/goyoubbs/util"
	"github.com/ego008/youdb"
	"github.com/rs/xid"
	"goji.io/pat"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *BaseHandler) ArticleAdd(w http.ResponseWriter, r *http.Request) {
	cid := pat.Param(r, "cid")
	_, err := strconv.Atoi(cid)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"cid type err"}`))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		w.Write([]byte(`{"retcode":401,"retmsg":"authored err"}`))
		return
	}
	if currentUser.Flag < 5 {
		var msg string
		if currentUser.Flag == 1 {
			msg = "注册验证中，等待管理员通过"
		} else {
			msg = "您已被禁用"
		}
		w.Write([]byte(`{"retcode":401,"retmsg":"` + msg + `"}`))
		return
	}

	db := h.App.Db

	cobj, err := model.CategoryGetById(db, cid)
	if err != nil {
		w.Write([]byte(`{"retcode":404,"retmsg":"` + err.Error() + `"}`))
		return
	}

	if cobj.Hidden {
		w.Write([]byte(`{"retcode":403,"retmsg":"category is Hidden"}`))
		return
	}

	type pageData struct {
		PageData
		Cobj      model.Category
		MainNodes []model.CategoryMini
	}

	tpl := h.CurrentTpl(r)
	evn := &pageData{}
	evn.SiteCf = h.App.Cf.Site
	evn.Title = "发表文章"
	evn.IsMobile = tpl == "mobile"
	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "article_add"

	evn.Cobj = cobj
	evn.MainNodes = model.CategoryGetMain(db, cobj)

	h.SetCookie(w, "token", xid.New().String(), 1)
	h.Render(w, tpl, evn, "layout.html", "articlecreate.html")
}

func (h *BaseHandler) ArticleAddPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

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
	if currentUser.Flag < 5 {
		w.Write([]byte(`{"retcode":403,"retmsg":"user flag err"}`))
		return
	}

	type recForm struct {
		Act     string `json:"act"`
		Cid     uint64 `json:"cid"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	decoder := json.NewDecoder(r.Body)
	var rec recForm
	err := decoder.Decode(&rec)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"json Decode err:` + err.Error() + `"}`))
		return
	}
	defer r.Body.Close()

	rec.Title = strings.TrimSpace(rec.Title)
	rec.Content = strings.TrimSpace(rec.Content)

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
	if db.Hget("title_md5", []byte(titleMd5)).State == "ok" {
		w.Write([]byte(`{"retcode":403,"retmsg":"title has existed"}`))
		return
	}

	now := uint64(time.Now().UTC().Unix())
	scf := h.App.Cf.Site

	if currentUser.Flag < 99 && currentUser.LastPostTime > 0 {
		if (now - currentUser.LastPostTime) < uint64(scf.PostInterval) {
			w.Write([]byte(`{"retcode":403,"retmsg":"PostInterval limited"}`))
			return
		}
	}

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

	cobj, err := model.CategoryGetById(db, strconv.FormatUint(rec.Cid, 10))
	if err != nil {
		w.Write([]byte(`{"retcode":404,"retmsg":"` + err.Error() + `"}`))
		return
	}

	if cobj.Hidden {
		w.Write([]byte(`{"retcode":403,"retmsg":"category is Hidden"}`))
		return
	}

	newAid, _ := db.HnextSequence("article")
	aobj := model.Article{
		Id:       newAid,
		Uid:      currentUser.Id,
		Cid:      rec.Cid,
		Title:    rec.Title,
		Content:  rec.Content,
		AddTime:  now,
		EditTime: now,
		ClientIp: r.Header.Get("X-FORWARDED-FOR"),
	}

	jb, _ := json.Marshal(aobj)
	aidB := youdb.I2b(newAid)
	db.Hset("article", aidB, jb)
	// 总文章列表
	db.Zset("article_timeline", aidB, aobj.EditTime)
	// 分类文章列表
	db.Zset("category_article_timeline:"+strconv.FormatUint(aobj.Cid, 10), aidB, aobj.EditTime)
	// 用户文章列表
	db.Hset("user_article_timeline:"+strconv.FormatUint(aobj.Uid, 10), youdb.I2b(aobj.Id), []byte(""))
	// 分类下文章数
	db.Zincr("category_article_num", youdb.I2b(aobj.Cid), 1)

	currentUser.LastPostTime = now
	currentUser.Articles++

	jb, _ = json.Marshal(currentUser)
	db.Hset("user", youdb.I2b(aobj.Uid), jb)

	// title md5
	db.Hset("title_md5", []byte(titleMd5), aidB)

	// send task work
	// get tag from title
	if scf.AutoGetTag && len(scf.GetTagApi) > 0 {
		db.Hset("task_to_get_tag", aidB, []byte(rec.Title))
	}

	// @ somebody in content
	sbs := util.GetMention(rec.Content,
		[]string{currentUser.Name, strconv.FormatUint(currentUser.Id, 10)})

	aid := strconv.FormatUint(newAid, 10)
	for _, sb := range sbs {
		var sbObj model.User
		sbu, err := strconv.ParseUint(sb, 10, 64)
		if err != nil {
			// @ user name
			sbObj, err = model.UserGetByName(db, strings.ToLower(sb))
		} else {
			// @ user id
			sbObj, err = model.UserGetById(db, sbu)
		}

		if err == nil {
			if len(sbObj.Notice) > 0 {
				aidList := util.SliceUniqStr(strings.Split(aid+","+sbObj.Notice, ","))
				if len(aidList) > 100 {
					aidList = aidList[:100]
				}
				sbObj.Notice = strings.Join(aidList, ",")
				sbObj.NoticeNum = len(aidList)
			} else {
				sbObj.Notice = aid
				sbObj.NoticeNum = 1
			}
			jb, _ := json.Marshal(sbObj)
			db.Hset("user", youdb.I2b(sbObj.Id), jb)
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

func (h *BaseHandler) ArticleHomeList(w http.ResponseWriter, r *http.Request) {
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

	cmd := "zrscan"
	if btn == "prev" {
		cmd = "zscan"
	}

	db := h.App.Db
	pageInfo := model.ArticleList(db, cmd, "article_timeline", key, score, h.App.Cf.Site.HomeShowNum)

	type siteInfo struct {
		Days     int
		UserNum  uint64
		NodeNum  uint64
		TagNum   uint64
		PostNum  uint64
		ReplyNum uint64
	}

	type pageData struct {
		PageData
		SiteInfo siteInfo
		PageInfo model.ArticlePageInfo
		Links    []model.Link
	}

	si := siteInfo{}
	rs := db.Hget("count", []byte("site_create_time"))
	var siteCreateTime uint64
	if rs.State == "ok" {
		siteCreateTime = rs.Data[0].Uint64()
	} else {
		rs2 := db.Hscan("user", []byte(""), 1)
		if rs2.State == "ok" {
			user := model.User{}
			json.Unmarshal(rs2.Data[1], &user)
			siteCreateTime = user.RegTime
		} else {
			siteCreateTime = uint64(time.Now().UTC().Unix())
		}
		db.Hset("count", []byte("site_create_time"), youdb.I2b(siteCreateTime))
	}
	then := time.Unix(int64(siteCreateTime), 0)
	diff := time.Now().UTC().Sub(then)
	si.Days = int(diff.Hours() / 24)
	si.UserNum = db.Hsequence("user")
	si.NodeNum = db.Hsequence("category")
	si.TagNum = db.Hsequence("tag")
	si.PostNum = db.Hsequence("article")
	si.ReplyNum = db.Hget("count", []byte("comment_num")).Uint64()

	// fix
	if si.NodeNum == 0 {
		newCid, err2 := db.HnextSequence("category")
		if err2 == nil {
			cobj := model.Category{
				Id:    newCid,
				Name:  "默认分类",
				About: "默认第一个分类",
			}
			jb, _ := json.Marshal(cobj)
			db.Hset("category", youdb.I2b(cobj.Id), jb)
			si.NodeNum = 1
		}
		// link
		model.LinkSet(db, model.Link{
			Name:  "youBBS",
			Url:   "https://www.youbbs.org",
			Score: 100,
		})
	}

	tpl := h.CurrentTpl(r)
	evn := &pageData{}
	evn.SiteCf = h.App.Cf.Site
	evn.Title = h.App.Cf.Site.Name
	evn.Keywords = evn.Title
	evn.Description = h.App.Cf.Site.Desc
	evn.IsMobile = tpl == "mobile"
	currentUser, _ := h.CurrentUser(w, r)
	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "home"
	evn.HotNodes = model.CategoryHot(db, 20)
	evn.NewestNodes = model.CategoryNewest(db, 20)

	evn.SiteInfo = si
	evn.PageInfo = pageInfo
	evn.Links = model.LinkList(db, false)

	h.Render(w, tpl, evn, "layout.html", "index.html")
}

func (h *BaseHandler) ArticleDetail(w http.ResponseWriter, r *http.Request) {
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

	aid := pat.Param(r, "aid")
	_, err := strconv.Atoi(aid)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"aid type err"}`))
		return
	}

	cmd := "hscan"
	if btn == "prev" {
		cmd = "hrscan"
	}

	db := h.App.Db
	aobj, err := model.ArticleGetById(db, aid)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	currentUser, _ := h.CurrentUser(w, r)

	if len(currentUser.Notice) > 0 && len(currentUser.Notice) >= len(aid) {
		if len(aid) == len(currentUser.Notice) && aid == currentUser.Notice {
			currentUser.Notice = ""
			currentUser.NoticeNum = 0
			jb, _ := json.Marshal(currentUser)
			db.Hset("user", youdb.I2b(currentUser.Id), jb)
		} else {
			subStr := "," + aid + ","
			newNotice := "," + currentUser.Notice + ","
			if strings.Index(newNotice, subStr) >= 0 {
				currentUser.Notice = strings.Trim(strings.Replace(newNotice, subStr, "", 1), ",")
				currentUser.NoticeNum--
				jb, _ := json.Marshal(currentUser)
				db.Hset("user", youdb.I2b(currentUser.Id), jb)
			}
		}
	}

	if aobj.Hidden && currentUser.Flag < 99 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"retcode":404,"retmsg":"not found"}`))
		return
	}
	cobj, err := model.CategoryGetById(db, strconv.FormatUint(aobj.Cid, 10))
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	if cobj.Hidden && currentUser.Flag < 99 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"retcode":404,"retmsg":"not found"}`))
		return
	}

	// Authorized
	if h.App.Cf.Site.Authorized && currentUser.Flag < 5 {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"retcode":401,"retmsg":"Unauthorized"}`))
		return
	}

	cobj.Articles = db.Zget("category_article_num", youdb.I2b(cobj.Id)).Uint64()
	pageInfo := model.CommentList(db, cmd, "article_comment:"+aid, key, h.App.Cf.Site.CommentListNum)

	type articleForDetail struct {
		model.Article
		ContentFmt  template.HTML
		TagStr      template.HTML
		Name        string
		Avatar      string
		Views       uint64
		AddTimeFmt  string
		EditTimeFmt string
	}

	type pageData struct {
		PageData
		Aobj     articleForDetail
		Author   model.User
		Cobj     model.Category
		Relative model.ArticleRelative
		PageInfo model.CommentPageInfo
		Views    uint64
	}

	tpl := h.CurrentTpl(r)
	evn := &pageData{}
	evn.SiteCf = h.App.Cf.Site
	evn.Title = aobj.Title + " - " + cobj.Name + " - " + h.App.Cf.Site.Name
	evn.Keywords = aobj.Tags
	evn.Description = cobj.Name + " - " + aobj.Title + " - " + aobj.Tags
	evn.IsMobile = tpl == "mobile"

	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "article_detail"
	evn.HotNodes = model.CategoryHot(db, 20)
	evn.NewestNodes = model.CategoryNewest(db, 20)

	author, _ := model.UserGetById(db, aobj.Uid)
	viewsNum, _ := db.Hincr("article_views", youdb.I2b(aobj.Id), 1)
	evn.Aobj = articleForDetail{
		Article:     aobj,
		ContentFmt:  template.HTML(util.ContentFmt(db, aobj.Content)),
		Name:        author.Name,
		Avatar:      author.Avatar,
		Views:       viewsNum,
		AddTimeFmt:  util.TimeFmt(aobj.AddTime, "2006-01-02 15:04"),
		EditTimeFmt: util.TimeFmt(aobj.EditTime, "2006-01-02 15:04"),
	}

	if len(aobj.Tags) > 0 {
		var tags []string
		for _, v := range strings.Split(aobj.Tags, ",") {
			tags = append(tags, `<a href="/tag/`+v+`">`+v+`</a>`)
		}
		evn.Aobj.TagStr = template.HTML(strings.Join(tags, ", "))
	}

	evn.Cobj = cobj
	evn.Relative = model.ArticleGetRelative(db, aobj.Id, aobj.Tags)
	evn.PageInfo = pageInfo

	token := h.GetCookie(r, "token")
	if len(token) == 0 {
		token := xid.New().String()
		h.SetCookie(w, "token", token, 1)
	}

	h.Render(w, tpl, evn, "layout.html", "article.html")
}

func (h *BaseHandler) ArticleDetailPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	token := h.GetCookie(r, "token")
	if len(token) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"token cookie missed"}`))
		return
	}

	aid := pat.Param(r, "aid")
	_, err := strconv.Atoi(aid)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"aid type err:` + err.Error() + `"}`))
		return
	}

	type recForm struct {
		Act     string `json:"act"`
		Link    string `json:"link"`
		Content string `json:"content"`
	}

	type response struct {
		normalRsp
		Content string        `json:"content"`
		Html    template.HTML `json:"html"`
	}

	decoder := json.NewDecoder(r.Body)
	var rec recForm
	err = decoder.Decode(&rec)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"json Decode err:` + err.Error() + `"}`))
		return
	}
	defer r.Body.Close()

	db := h.App.Db
	rsp := response{}

	if rec.Act == "link_click" {
		rsp.Retcode = 200
		if len(rec.Link) > 0 {
			hash := md5.Sum([]byte(rec.Link))
			urlMd5 := hex.EncodeToString(hash[:])
			bn := "article_detail_token"
			clickKey := []byte(token + ":click:" + urlMd5)
			if db.Zget(bn, clickKey).State == "ok" {
				w.Write([]byte(`{"retcode":403,"retmsg":"err"}`))
				return
			}
			db.Zset(bn, clickKey, uint64(time.Now().UTC().Unix()))
			db.Hincr("url_md5_click", []byte(urlMd5), 1)

			w.Write([]byte(`{"retcode":200,"retmsg":"ok"}`))
			return
		}
	} else if rec.Act == "comment_preview" {
		rsp.Retcode = 200
		rsp.Html = template.HTML(util.ContentFmt(db, rec.Content))
	} else if rec.Act == "comment_submit" {
		timeStamp := uint64(time.Now().UTC().Unix())
		currentUser, _ := h.CurrentUser(w, r)
		if currentUser.Flag < 5 {
			w.Write([]byte(`{"retcode":403,"retmsg":"forbidden"}`))
			return
		}
		if (timeStamp - currentUser.LastReplyTime) < uint64(h.App.Cf.Site.CommentInterval) {
			w.Write([]byte(`{"retcode":403,"retmsg":"out off comment interval"}`))
			return
		}
		aobj, err := model.ArticleGetById(db, aid)
		if err != nil {
			w.Write([]byte(`{"retcode":404,"retmsg":"not found"}`))
			return
		}
		if aobj.CloseComment {
			w.Write([]byte(`{"retcode":403,"retmsg":"comment forbidden"}`))
			return
		}
		commentId, _ := db.HnextSequence("article_comment:" + aid)
		obj := model.Comment{
			Id:       commentId,
			Aid:      aobj.Id,
			Uid:      currentUser.Id,
			Content:  rec.Content,
			AddTime:  timeStamp,
			ClientIp: r.Header.Get("X-FORWARDED-FOR"),
		}
		jb, _ := json.Marshal(obj)

		db.Hset("article_comment:"+aid, youdb.I2b(obj.Id), jb) // 文章评论bucket
		db.Hincr("count", []byte("comment_num"), 1)            // 评论总数
		// 用户回复文章列表
		db.Zset("user_article_reply:"+strconv.FormatUint(obj.Uid, 10), youdb.I2b(obj.Aid), obj.AddTime)

		// 更新文章列表时间

		aobj.Comments = commentId
		aobj.RUid = currentUser.Id
		aobj.EditTime = timeStamp
		jb2, _ := json.Marshal(aobj)
		db.Hset("article", youdb.I2b(aobj.Id), jb2)

		currentUser.LastReplyTime = timeStamp
		currentUser.Replies += 1
		jb3, _ := json.Marshal(currentUser)
		db.Hset("user", youdb.I2b(currentUser.Id), jb3)

		// 总文章列表
		db.Zset("article_timeline", youdb.I2b(aobj.Id), timeStamp)
		// 分类文章列表
		db.Zset("category_article_timeline:"+strconv.FormatUint(aobj.Cid, 10), youdb.I2b(aobj.Id), timeStamp)

		// @ somebody in comment & topic author
		sbs := util.GetMention("@"+strconv.FormatUint(aobj.Uid, 10)+" "+rec.Content,
			[]string{currentUser.Name, strconv.FormatUint(currentUser.Id, 10)})
		for _, sb := range sbs {
			var sbObj model.User
			sbu, err := strconv.ParseUint(sb, 10, 64)
			if err != nil {
				// @ user name
				sbObj, err = model.UserGetByName(db, strings.ToLower(sb))
			} else {
				// @ user id
				sbObj, err = model.UserGetById(db, sbu)
			}

			if err == nil {
				if len(sbObj.Notice) > 0 {
					aidList := util.SliceUniqStr(strings.Split(aid+","+sbObj.Notice, ","))
					if len(aidList) > 100 {
						aidList = aidList[:100]
					}
					sbObj.Notice = strings.Join(aidList, ",")
					sbObj.NoticeNum = len(aidList)
				} else {
					sbObj.Notice = aid
					sbObj.NoticeNum = 1
				}
				jb, _ := json.Marshal(sbObj)
				db.Hset("user", youdb.I2b(sbObj.Id), jb)
			}
		}

		rsp.Retcode = 200
	}

	json.NewEncoder(w).Encode(rsp)
}

func (h *BaseHandler) ContentPreviewPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	token := h.GetCookie(r, "token")
	if len(token) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"token cookie missed"}`))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Flag < 5 {
		w.Write([]byte(`{"retcode":403,"retmsg":"forbidden"}`))
		return
	}

	type recForm struct {
		Act     string `json:"act"`
		Link    string `json:"link"`
		Content string `json:"content"`
	}

	type response struct {
		normalRsp
		Content string        `json:"content"`
		Html    template.HTML `json:"html"`
	}

	decoder := json.NewDecoder(r.Body)
	var rec recForm
	err := decoder.Decode(&rec)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"json Decode err:` + err.Error() + `"}`))
		return
	}
	defer r.Body.Close()

	db := h.App.Db
	rsp := response{}

	if rec.Act == "preview" && len(rec.Content) > 0 {
		rsp.Retcode = 200
		rsp.Html = template.HTML(util.ContentFmt(db, rec.Content))
	}
	json.NewEncoder(w).Encode(rsp)
}
