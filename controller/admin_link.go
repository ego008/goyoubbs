package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/views/ybs"
	"strconv"
)

func (h *BaseHandler) AdminLinkPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	scf := h.App.Cf.Site

	evn := &ybs.AdminLink{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "链接管理"
	evn.PageName = "admin_link"

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)
	evn.LinkLst = model.LinkList(h.App.Mc, h.App.Db, true)

	//
	evn.Act = "添加"
	evn.Link = model.Link{}
	ids := sdb.B2s(ctx.FormValue("id"))
	if len(ids) > 0 {
		evn.Link = model.Link{}
		link := model.LinkGetById(h.App.Db, ids)
		if link.ID > 0 {
			evn.Link = link
			evn.Act = "编辑"
		}
	}

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ctx.SetContentType("text/html; charset=utf-8")
	ybs.WritePageTemplate(ctx, evn)
}

func (h *BaseHandler) AdminLinkPost(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	var id uint64
	ids := sdb.B2s(ctx.FormValue("id"))
	if len(ids) > 0 {
		idI, err := strconv.ParseUint(ids, 10, 64)
		if err == nil {
			id = idI
		}
	}

	obj := model.Link{}
	obj.ID = id
	obj.Name = sdb.B2s(ctx.FormValue("Name"))
	obj.Score, _ = strconv.Atoi(sdb.B2s(ctx.FormValue("Score")))
	obj.Url = sdb.B2s(ctx.FormValue("Url"))

	model.LinkSet(h.App.Db, obj)

	// 删除缓存
	h.App.Mc.Del([]byte("LinkList"))

	ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/link", 302)
}
