package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/admin"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) AdminSiteConfigPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.SiteConfig{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "网站设置"
	evn.PageName = "admin_site_setting"

	siteConf := model.SiteConf{}

	_ = h.App.Db.View(func(tx *bbolt.Tx) error {
		evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)

		//
		model.SiteConfLoad(&siteConf, h.App.Db, tx)
		evn.SiteConf = siteConf

		evn.HasMsg = model.MsgCheckHasOne(h.App.Db, tx, curUser.ID)
		evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
		evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)
		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}

func s2int(s string, df int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return df
	}
	return i
}

func s2bool(s string, df bool) bool {
	i, err := strconv.ParseBool(s)
	if err != nil {
		return df
	}
	return i
}

func (h *BaseHandler) AdminSiteConfigPost(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/login")
		return
	}

	_ = h.App.Db.Update(func(tx *bbolt.Tx) error {
		obj := model.SiteConf{}
		model.SiteConfLoad(&obj, h.App.Db, tx)

		obj.Name = c.PostForm("Name")
		obj.Desc = c.PostForm("Desc")
		obj.MainDomain = strings.TrimSuffix(c.PostForm("MainDomain"), "/")
		obj.HeaderPartCon = c.PostForm("HeaderPartCon")
		obj.GoogleAutoAdJs = c.PostForm("GoogleAutoAdJs")
		obj.FooterPartHtml = c.PostForm("FooterPartHtml")

		obj.TimeZone = s2int(c.PostForm("TimeZone"), 8)
		if obj.TimeZone < -12 || obj.TimeZone > 12 {
			obj.TimeZone = 8
		}
		model.TimeOffSet = time.Duration(obj.TimeZone) * time.Hour

		obj.PageShowNum = s2int(c.PostForm("PageShowNum"), 32)
		obj.TopRateNum = s2int(c.PostForm("TopRateNum"), 10)
		obj.RecentCommentNum = s2int(c.PostForm("RecentCommentNum"), 10)
		obj.TitleMaxLen = s2int(c.PostForm("TitleMaxLen"), 110)
		obj.TopicConMaxLen = s2int(c.PostForm("TopicConMaxLen"), 12000)
		obj.CommentConMaxLen = s2int(c.PostForm("CommentConMaxLen"), 5000)

		obj.AutoDataBackup = s2bool(c.PostForm("AutoDataBackup"), false)
		obj.DataBackupDir = strings.TrimSuffix(c.PostForm("DataBackupDir"), "/")
		if obj.UploadDir == "" {
			obj.UploadDir = "data_backup"
		}

		obj.Authorized = s2bool(c.PostForm("Authorized"), false)
		obj.AllowNameReg = s2bool(c.PostForm("AllowNameReg"), true)
		obj.RegReview = s2bool(c.PostForm("RegReview"), false)
		obj.CloseReg = s2bool(c.PostForm("CloseReg"), false)
		obj.CloseReply = s2bool(c.PostForm("CloseReply"), false)
		obj.PostReview = s2bool(c.PostForm("PostReview"), false)

		obj.ResetCookieKey = s2bool(c.PostForm("ResetCookieKey"), false)
		if obj.ResetCookieKey {
			hashKey := securecookie.GenerateRandomKey(64)
			blockKey := securecookie.GenerateRandomKey(32)
			_ = h.App.Db.HMSet(tx, model.KeyValueTb, []byte("hashKey"), hashKey, []byte("blockKey"), blockKey)
			h.App.Sc = securecookie.New(hashKey, blockKey)
		}

		obj.AutoDecodeMp4 = s2bool(c.PostForm("AutoDecodeMp4"), false)
		// check ffmpeg exist
		if obj.AutoDecodeMp4 {
			obj.AutoDecodeMp4 = util.CmdExists("ffmpeg")
		}

		obj.GetTagApi = c.PostForm("GetTagApi")

		obj.UploadLimit = s2bool(c.PostForm("UploadLimit"), false)

		var reloadRouter bool
		oldUploadDir := obj.UploadDir
		obj.UploadDir = strings.TrimSuffix(c.PostForm("UploadDir"), "/")
		if obj.UploadDir == "" {
			obj.UploadDir = "upload"
		}
		if obj.UploadDir != oldUploadDir {
			reloadRouter = true
		}

		obj.UploadMaxSize = s2int(c.PostForm("UploadMaxSize"), 20)
		if obj.UploadMaxSize < 1 {
			obj.UploadMaxSize = 1
		}
		obj.UploadMaxSizeByte = int64(obj.UploadMaxSize) << 20

		oldCachedSize := obj.CachedSize
		obj.CachedSize = s2int(c.PostForm("CachedSize"), 1)
		if obj.CachedSize < 1 {
			obj.CachedSize = 1
		}

		obj.RateLimitDay = s2int(c.PostForm("RateLimitDay"), 0)
		model.RateLimitDay = obj.RateLimitDay
		obj.RateLimitHour = s2int(c.PostForm("RateLimitHour"), 0)
		model.RateLimitHour = obj.RateLimitHour

		oldSaveTopicIcon := obj.SaveTopicIcon
		obj.SaveTopicIcon = s2bool(c.PostForm("SaveTopicIcon"), false)

		obj.SaveImg2db = s2bool(c.PostForm("SaveImg2db"), false)
		obj.RemotePostPw = c.PostForm("RemotePostPw")
		obj.QQClientID = c.PostForm("QQClientID")
		obj.QQClientSecret = c.PostForm("QQClientSecret")
		obj.WeiboClientID = c.PostForm("WeiboClientID")
		obj.WeiboClientSecret = c.PostForm("WeiboClientSecret")
		obj.GithubClientID = c.PostForm("GithubClientID")
		obj.GithubClientSecret = c.PostForm("GithubClientSecret")
		obj.SendEmail = s2bool(c.PostForm("SendEmail"), false)
		obj.SmtpHost = c.PostForm("SmtpHost")
		obj.SmtpPort = s2int(c.PostForm("SmtpPort"), 465)
		obj.SmtpEmail = c.PostForm("SmtpEmail")
		obj.SmtpPassword = c.PostForm("SmtpPassword")
		obj.SendToEmail = c.PostForm("SendToEmail")

		jb, _ := json.Marshal(obj)
		_ = h.App.Db.HSet(tx, model.KeyValueTb, []byte("site_config"), jb)

		// in old conf
		obj.IsDevMod = h.App.Cf.Site.IsDevMod
		obj.SelfHash = h.App.Cf.Site.SelfHash

		h.App.Cf.Site = &obj

		if reloadRouter {
			// 在保存后 reload
			RouterReload(h.App)
		}

		// 清空缓存
		h.App.Mc.Reset()
		if obj.CachedSize != oldCachedSize {
			h.App.Mc = fastcache.New(obj.CachedSize * 1024 * 1024)
		}

		// 清除帖子九宫格图片
		if (oldSaveTopicIcon != obj.SaveTopicIcon) && !obj.SaveTopicIcon {
			_ = h.App.Db.HScanFunc(tx, "topic_icon", nil, 1, func(key, val []byte) bool {
				_ = h.App.Db.HDelBucket(tx, "topic_icon")
				return true
			})
		}

		return nil
	})

	c.Redirect(302, "/admin/site/conf")
}
