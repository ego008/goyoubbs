package controller

import (
	"goyoubbs/model"
	"goyoubbs/views/admin"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) AdminRateLimitIpLookup(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.IpLookup{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "Ip Lookup"
	evn.PageName = "admin_IpLookup"

	var items []model.KvStr

	_ = h.App.Db.View(func(tx *bbolt.Tx) error {
		evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)
		evn.LinkLst = model.LinkList(h.App.Mc, h.App.Db, tx, true)

		stLst := model.SettingGetByKeys(h.App.Db, tx, model.SettingKeys)
		sort.Slice(stLst, func(i, j int) bool {
			return stLst[i].Key < stLst[j].Key
		})

		var whiteItems []string
		var blackItems []string

		// get from cached
		whiteItems = model.AllowIpPrefixLst.Items()
		blackItems = model.BadIpPrefixLst.Items()

		limit := 100
		// items
		startKeyStr := c.Query("key")
		for _, item := range model.IpInfoGetByKeyStart(h.App.Db, tx, startKeyStr, limit) {
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

		evn.HasMsg = model.MsgCheckHasOne(h.App.Db, tx, curUser.ID)
		evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
		evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)

		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}
