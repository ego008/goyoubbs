package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/admin"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *BaseHandler) AdminRateLimitSetting(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.RateLimitSetting{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "Rate Limit Setting"
	evn.PageName = "admin_RateLimitSetting"

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)
	evn.LinkLst = model.LinkList(h.App.Mc, h.App.Db, true)

	evn.MyIp = ReadUserIP(c)
	var stLst []model.SettingKv
	stLst = model.SettingGetByKeys(h.App.Db, model.SettingKeys)
	sort.Slice(stLst, func(i, j int) bool {
		return stLst[i].Key < stLst[j].Key
	})
	evn.SettingLst = stLst

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}

func (h *BaseHandler) AdminRateLimitSettingPost(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/login")
		return
	}

	stMp := map[string]string{}
	var kvs [][]byte
	for _, v := range model.SettingKeys {
		stValue := strings.TrimSpace(c.Query(v))
		stValue = util.SliceUniqStr(stValue, ",")

		// reset stValue
		if v == model.SettingKeyAllowIp || v == model.SettingKeyBadIp {
			var lis []string
			for _, ip := range util.StringSplit(stValue, ",") {
				lis = append(lis, util.IpTrimRightDot(ip))
			}
			stValue = strings.Join(lis, ",")
		}

		kvs = append(kvs, []byte(v), []byte(stValue))
		stMp[v] = stValue
	}

	if len(kvs) == 0 {
		c.Redirect(302, "/admin/ratelimit/setting")
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
		c.Redirect(302, "/admin/ratelimit/setting")
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

	c.Redirect(302, "/admin/ratelimit/setting")
}
