package controller

import (
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/admin"
	"sort"
	"strings"
)

func (h *BaseHandler) AdminRateLimitSetting(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.RateLimitSetting{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "Rate Limit Setting"
	evn.PageName = "admin_RateLimitSetting"

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)
	evn.LinkLst = model.LinkList(h.App.Mc, h.App.Db, true)

	evn.MyIp = ReadUserIP(ctx)
	var stLst []model.SettingKv
	stLst = model.SettingGetByKeys(h.App.Db, model.SettingKeys)
	sort.Slice(stLst, func(i, j int) bool {
		return stLst[i].Key < stLst[j].Key
	})
	evn.SettingLst = stLst

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ctx.SetContentType("text/html; charset=utf-8")
	admin.WritePageTemplate(ctx, evn)
}

func (h *BaseHandler) AdminRateLimitSettingPost(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	stMp := map[string]string{}
	var kvs [][]byte
	for _, v := range model.SettingKeys {
		stValue := strings.TrimSpace(string(ctx.FormValue(v)))
		stValue = util.SliceUniqStr(stValue, ",")

		// reset stValue
		if v == model.SettingKeyAllowIp || v == model.SettingKeyBadIp {
			var lis []string
			for _, ip := range strings.Split(stValue, ",") {
				lis = append(lis, util.IpTrimRightDot(ip))
			}
			stValue = strings.Join(lis, ",")
		}

		kvs = append(kvs, []byte(v), []byte(stValue))
		stMp[v] = stValue
	}

	if len(kvs) == 0 {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/ratelimit/setting", 302)
		return
	}

	// get old value
	var stLst []model.SettingKv
	stLst = model.SettingGetByKeys(h.App.Db, model.SettingKeys)
	stMpOld := map[string]string{} // old value
	for i := 0; i < len(stLst); i++ {
		stMpOld[stLst[i].Key] = stLst[i].Value
	}

	if err := h.App.Db.Hmset(model.TbnSetting, kvs...); err != nil {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/ratelimit/setting", 302)
		return
	}

	// update

	// BadBotNameMap
	if stMp[model.SettingKeyBadBot] != stMpOld[model.SettingKeyBadBot] {
		model.UpdateBadBotName(h.App.Db)
	}
	// BadIpPrefixLst
	if stMp[model.SettingKeyBadIp] != stMpOld[model.SettingKeyBadIp] {
		model.UpdateBadIpPrefix(h.App.Db)
	}
	// AllowIpPrefixLst
	if stMp[model.SettingKeyAllowIp] != stMpOld[model.SettingKeyAllowIp] {
		model.UpdateAllowIpPrefix(h.App.Db)
	}

	ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/ratelimit/setting", 302)
}
