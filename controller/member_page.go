package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"net/url"
	"strconv"
	"strings"
)

func (h *BaseHandler) MemberNamePage(ctx *fasthttp.RequestCtx) {
	// 内容 @用户名 的链接
	uNameRaw := strings.TrimSpace(ctx.UserValue("uname").(string))
	if uName, err := url.QueryUnescape(uNameRaw); err == nil {
		uNameRaw = uName
	}
	user, err := model.UserGetByName(h.App.Db, uNameRaw)
	if err != nil {
		unUint64, err := strconv.ParseUint(uNameRaw, 10, 64)
		if err == nil {
			var code int
			user, code = model.UserGetById(h.App.Db, unUint64)
			if code != 1 {
				ctx.NotFound()
				return
			}
		} else {
			ctx.NotFound()
			return
		}
	}
	ctx.Redirect(h.App.Cf.Site.MainDomain+"/member/"+strconv.FormatUint(user.ID, 10), 301)
}

func (h *BaseHandler) MemberPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)

	if h.App.Cf.Site.Authorized && curUser.Flag < model.FlagAuthor {
		if curUser.ID == 0 {
			ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
			return
		}
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/setting", 302)
		return
	}

	uid := strings.TrimSpace(ctx.UserValue("uid").(string))
	uidInt, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		// 不是数字，取用户名
		ctx.NotFound()
		_, _ = ctx.WriteString(uid + " uid not found")
		return
	}
	user, code := model.UserGetById(h.App.Db, uidInt)
	if code != 1 {
		ctx.NotFound()
		_, _ = ctx.WriteString(uid + " user not found")
		return
	}

	btn, key, score := sdb.B2s(ctx.FormValue("btn")), sdb.B2s(ctx.FormValue("key")), sdb.B2s(ctx.FormValue("score"))
	if len(key) > 0 {
		_, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			ctx.Redirect(h.App.Cf.Site.MainDomain+"/", 302)
			return
		}
	}
	if len(score) > 0 {
		_, err := strconv.ParseUint(score, 10, 64)
		if err != nil {
			ctx.Redirect(h.App.Cf.Site.MainDomain+"/", 302)
			return
		}
	}

	cmd := "zrscan"
	if btn == "prev" {
		cmd = "zscan"
	}

	var titleText string
	lstType := sdb.B2s(ctx.FormValue("type"))
	if lstType == "comment" {
		titleText = "评论的主题"
	} else {
		titleText = "发表的主题"
		lstType = "topic"
	}

	db := h.App.Db
	scf := h.App.Cf.Site

	//topicPageInfo := model.GetTopicList(db, cmd, "topic_update", key, score, scf.PageShowNum)
	tbName := "user_" + lstType + ":" + strconv.FormatUint(user.ID, 10)
	topicPageInfo := model.GetTopicList(db, cmd, tbName, key, score, scf.PageShowNum)

	evn := &ybs.MemberPage{}
	evn.SiteCf = scf
	evn.Title = "会员: " + user.Name + " 最近" + titleText + " - " + scf.Name
	evn.CurrentUser = curUser

	evn.NodeLst = model.NodeGetAll(h.App.Mc, db)
	evn.TopicPageInfo = topicPageInfo
	evn.UserFmt = model.UserFmt{
		User:       user,
		RegTimeFmt: util.TimeFmt(int64(user.RegTime), "2006-01-02 15:04"),
	}
	evn.LstType = lstType
	evn.TitleText = titleText

	if curUser.ID == user.ID {
		if lstType == "comment" {
			evn.CommentReviewLst = model.CommentGetReview(db, curUser.ID)
		} else {
			evn.TopicLst = model.TopicGetV2Review(db, curUser.ID)
		}
	}

	evn.TagCloud = model.GetTagsForSide(h.App.Mc, db, 100)
	evn.RangeTopicLst = rangeTopicLst[:]
	evn.RecentComment = model.CommentGetRecent(h.App.Mc, db, scf.RecentCommentNum)

	if curUser.ID > 0 {
		evn.HasMsg = model.MsgCheckHasOne(db, curUser.ID)
		if curUser.Flag >= model.FlagAdmin {
			evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
			evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)
		}
	}

	ybs.WritePageTemplate(ctx, evn)
	ctx.SetContentType("text/html; charset=utf-8")

}
