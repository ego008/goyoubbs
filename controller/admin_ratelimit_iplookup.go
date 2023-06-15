package controller

import (
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/views/admin"
	"sort"
	"strings"
)

func (h *BaseHandler) AdminRateLimitIpLookup(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.IpLookup{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "Ip Lookup"
	evn.PageName = "admin_IpLookup"

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)
	evn.LinkLst = model.LinkList(h.App.Mc, h.App.Db, true)

	stLst := model.SettingGetByKeys(h.App.Db, model.SettingKeys)
	sort.Slice(stLst, func(i, j int) bool {
		return stLst[i].Key < stLst[j].Key
	})

	var items []model.KvStr
	var whiteItems []string
	var blackItems []string

	// get from cached
	whiteItems = model.AllowIpPrefixLst.Items()
	blackItems = model.BadIpPrefixLst.Items()

	limit := 100
	// items
	startKeyStr := string(ctx.FormValue("key"))
	for _, item := range model.IpInfoGetByKeyStart(h.App.Db, startKeyStr, limit) {
		items = append(items, model.KvStr{
			Key:   item.Ip,
			Value: item.Names,
		})
	}

	var keyStart string // for next page
	if len(items) > 0 {
		keyStart = items[len(items)-1].Key
	}

	for i := 0; i < len(items); i++ {
		for _, v := range blackItems {
			if strings.HasPrefix(items[i].Key, v) {
				items[i].Key = `<del>` + v + `</del>` + items[i].Key[len(v):]
			}
		}
		for _, v := range whiteItems {
			if strings.HasPrefix(items[i].Key, v) {
				items[i].Key = `<span class="red">` + v + `</span>` + items[i].Key[len(v):]
			}
		}
	}

	evn.Limit = limit
	evn.KeyStart = keyStart
	evn.ShowNext = len(items) == limit
	evn.Items = items

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ctx.SetContentType("text/html; charset=utf-8")
	admin.WritePageTemplate(ctx, evn)
}
