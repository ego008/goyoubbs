package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/views/ybs"
	"strconv"
)

func (h *BaseHandler) AdminTopicEditPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin", 302)
		return
	}

	scf := h.App.Cf.Site

	evn := &ybs.AdminTopicAdd{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "编辑帖子"
	evn.PageName = "admin_topic_edit"

	tid := sdb.B2s(ctx.FormValue("id")) // 编辑帖子id
	tidI, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"id 不是数字"}`)
		return
	}

	//var rec model.Topic
	rec := model.TopicGetById(h.App.Db, tidI)
	if rec.ID == 0 {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"该 id 帖子不存在"}`)
		return
	}

	// 是不是删除
	isDel := sdb.B2s(ctx.FormValue("del")) // 删除帖子
	if isDel == "1" {
		model.TopicDel(h.App.Mc, h.App.Db, rec)
		if h.App.Cf.Site.SaveTopicIcon {
			// 删九宫格图片
			_ = h.App.Db.Hdel("topic_icon", sdb.I2b(tidI))
		}
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/topic/add", 302)
		return
	}

	var author model.User
	if rec.UserId > 0 {
		author, _ = model.UserGetById(h.App.Db, rec.UserId)
	}
	if author.ID == 0 {
		author = curUser
	}

	evn.DefaultTopic = rec
	if evn.DefaultTopic.NodeId == 0 {
		evn.DefaultTopic.NodeId = 1
	}

	evn.DefaultUser = author
	// evn.DefaultNode, _ = model.NodeGetById(h.App.Db, evn.DefaultTopic.NodeId)
	// evn.DefaultNode = model.Node{}
	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)

	// evn.UserLst = model.UserGetAllAdmin(h.App.Db)
	evn.UserLst = []model.User{author}

	if len(ctx.QueryArgs().Peek("back")) > 0 {
		evn.GoBack = true
	}

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ybs.WritePageTemplate(ctx, evn)
	ctx.SetContentType("text/html; charset=utf-8")

	//_ = h.Render(ctx, evn, "admin/layout.html", "admin/topic_add.html")
}
