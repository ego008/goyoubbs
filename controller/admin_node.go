package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/views/admin"
	"strconv"
)

func (h *BaseHandler) AdminNodePage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.Node{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "分区管理"
	evn.PageName = "admin_node"

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)

	//
	evn.Act = "添加"
	evn.Node = model.Node{}
	_id := sdb.B2s(ctx.FormValue("id"))
	if len(_id) > 0 {
		idi, _ := strconv.ParseUint(_id, 10, 64)
		evn.Node = model.Node{}
		node, code := model.NodeGetById(h.App.Db, idi)
		if code == 1 {
			evn.Node = node
			evn.Act = "编辑"
		}
	}

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ctx.SetContentType("text/html; charset=utf-8")
	admin.WritePageTemplate(ctx, evn)
}

func (h *BaseHandler) AdminNodePost(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	var id uint64
	_id := string(ctx.FormValue("id"))
	if len(_id) > 0 {
		idI, err := strconv.ParseUint(_id, 10, 64)
		if err == nil {
			id = idI
		}
	}

	obj := model.Node{}
	obj.ID = id
	obj.Name = string(ctx.FormValue("Name"))
	obj.Score, _ = strconv.Atoi(string(ctx.FormValue("Score")))
	obj.About = string(ctx.FormValue("About"))

	_, _ = model.NodeSet(h.App.Db, obj)

	// 删除缓存
	h.App.Mc.Del([]byte("NodeGetAll"))

	ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/node", 302)
}
