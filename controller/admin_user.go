package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"strconv"
	"strings"
)

func (h *BaseHandler) AdminUserPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	scf := h.App.Cf.Site

	evn := &ybs.AdminUser{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "用户管理"

	evn.FlagLst = []model.Flag{
		{model.FlagForbidden, "0 禁用"},
		{model.FlagReview, "1 待审核"},
		{model.FlagAuthor, "5 普通"},
		{model.FlagTrust, "10 可信任"},
		{model.FlagAdmin, "99 管理员"},
	}

	//
	evn.Act = "添加"
	evn.User = model.User{}
	_id := sdb.B2s(ctx.FormValue("id"))
	if len(_id) > 0 {
		idi, _ := strconv.ParseUint(_id, 10, 64)
		evn.User = model.User{}
		user, code := model.UserGetById(h.App.Db, idi)
		if code == 1 {
			evn.User = user
			evn.Act = "编辑"
		}
	}

	if evn.User.ID == 0 {
		var tbn string
		flag := sdb.B2s(ctx.FormValue("flag"))
		if len(flag) > 0 {
			tbn = "user_flag:" + flag
		} else {
			tbn = model.UserTbName
		}

		var userLst []model.User
		q := strings.TrimSpace(sdb.B2s(ctx.FormValue("q")))
		if len(q) > 0 {
			// 搜索用户
			userLst = model.UserGetRecentByKw(h.App.Db, q, 100)
		} else {
			userLst = model.UserGetRecentByFlag(h.App.Db, tbn, 100)
		}
		evn.UserLst = userLst
	}

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ctx.SetContentType("text/html; charset=utf-8")
	ybs.WritePageTemplate(ctx, evn)
}

func (h *BaseHandler) AdminUserPost(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	var id uint64
	_id := sdb.B2s(ctx.FormValue("id"))
	if len(_id) > 0 {
		idI, err := strconv.ParseUint(_id, 10, 64)
		if err == nil {
			id = idI
		}
	}

	var obj model.User
	var isAdd bool
	var oldFlag int

	db := h.App.Db

	if id > 0 {
		// 编辑
		obj, _ = model.UserGetById(db, id)
		if obj.ID == 0 {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"not has this id"}`)
			return
		}
		oldFlag = obj.Flag
	} else {
		// 添加
		// 检测重名
		nameLow := strings.ToLower(strings.TrimSpace(sdb.B2s(ctx.FormValue("Name"))))
		if !util.IsNickname(nameLow) {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"name fmt err"}`)
			return
		}
		tmpObj, _ := model.UserGetByName(db, nameLow)
		if tmpObj.ID > 0 {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"name is exist"}`)
			return
		}

		isAdd = true
		userId, _ := db.Hincr(model.CountTb, sdb.S2b(model.UserTbName), 1)
		obj = model.User{
			ID:      userId,
			Name:    strings.TrimSpace(sdb.B2s(ctx.FormValue("Name"))),
			RegTime: uint64(util.GetCNTM(model.TimeOffSet)),
		}
	}

	pw := strings.TrimSpace(sdb.B2s(ctx.FormValue("Password")))
	if len(pw) > 0 {
		obj.Password = util.Md5(pw)
	}

	obj.Flag, _ = strconv.Atoi(sdb.B2s(ctx.FormValue("Flag")))
	obj.Url = sdb.B2s(ctx.FormValue("Url"))
	obj.About = sdb.B2s(ctx.FormValue("About"))

	obj = model.UserSet(db, obj)

	if isAdd {
		nameLow := strings.ToLower(obj.Name)
		_ = db.Hset("user_name2uid", []byte(nameLow), sdb.I2b(obj.ID))
		_ = db.Hset("user_flag:"+strconv.Itoa(obj.Flag), sdb.I2b(obj.ID), nil)
		//生成头像
		_ = util.GenAvatar(db, obj.ID, obj.Name)
	} else {
		if oldFlag != obj.Flag {
			_ = db.Hset("user_flag:"+strconv.Itoa(obj.Flag), sdb.I2b(obj.ID), nil)
			_ = db.Hdel("user_flag:"+strconv.Itoa(oldFlag), sdb.I2b(obj.ID))
		}
	}

	ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/user", 302)
}
