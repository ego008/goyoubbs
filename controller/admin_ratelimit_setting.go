package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/admin"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
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

	var stLst []model.SettingKv

	_ = h.App.Db.View(func(tx *bbolt.Tx) error {
		evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)
		evn.LinkLst = model.LinkList(h.App.Mc, h.App.Db, tx, true)

		evn.MyIp = ReadUserIP(c)

		stLst = model.SettingGetByKeys(h.App.Db, tx, model.SettingKeys)
		sort.Slice(stLst, func(i, j int) bool {
			return stLst[i].Key < stLst[j].Key
		})
		evn.SettingLst = stLst

		evn.HasMsg = model.MsgCheckHasOne(h.App.Db, tx, curUser.ID)
		evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
		evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)
		return nil
	})

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

	_ = h.App.Db.Update(func(tx *bbolt.Tx) error {
		// get old value
		var stLst []model.SettingKv
		stLst = model.SettingGetByKeys(h.App.Db, tx, model.SettingKeys)
		stMpOld := map[string]string{} // old value
		for i := 0; i < len(stLst); i++ {
			stMpOld[stLst[i].Key] = stLst[i].Value
		}

		if err := h.App.Db.HMSet(tx, model.TbnSetting, kvs...); err != nil {
			return err
		}

		// update

		// BadBotNameMap
		if stMp[model.SettingKeyBadBot] != stMpOld[model.SettingKeyBadBot] {
			model.UpdateBadBotName(h.App.Db, tx)
		}
		// BadIpPrefixLst
		if stMp[model.SettingKeyBadIp] != stMpOld[model.SettingKeyBadIp] {
			model.UpdateBadIpPrefix(h.App.Db, tx)
		}
		// AllowIpPrefixLst
		if stMp[model.SettingKeyAllowIp] != stMpOld[model.SettingKeyAllowIp] {
			model.UpdateAllowIpPrefix(h.App.Db, tx)
		}
		return nil
	})

	c.Redirect(302, "/admin/ratelimit/setting")
}
