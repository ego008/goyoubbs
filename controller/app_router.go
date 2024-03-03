package controller

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/fasthttp/router"
	"github.com/mileusna/useragent"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"
)

func RouterReload(ap *model.Application) {
	mux := router.New()
	MainRouter(ap, mux)
	ap.Mux = mux
}

func MainRouter(ap *model.Application, sm *router.Router) {
	h := BaseHandler{App: ap}

	// 自定义路径及内容 参见 https://youbbs.org/t/3324
	// 浏览需登录设置对自定义路由无效
	var keyStart []byte
	for {
		rs := ap.Db.Hscan("custom_router", keyStart, 20)
		if !rs.OK() {
			break
		}
		rs.KvEach(func(key, value sdb.BS) {
			keyStart = key
			obj := model.CustomRouter{}
			err := json.Unmarshal(value, &obj)
			if err != nil {
				return
			}
			// 默认 GET
			sm.GET(obj.Router, func(ctx *fasthttp.RequestCtx) {
				key := ctx.Path()
				rs2 := ap.Db.Hget("custom_router", key)
				if !rs2.OK() {
					ctx.NotFound()
					return
				}
				obj2 := model.CustomRouter{}
				_ = json.Unmarshal(rs2.Bytes(), &obj2)
				if strings.HasPrefix(obj2.Content, "goto:") {
					// match goto url
					// goto: https://youbbs.org/
					ctx.Redirect(strings.TrimSpace(obj2.Content[5:]), 302)
					return
				}
				ctx.SetContentType(obj2.MimeType)
				_, _ = ctx.WriteString(obj2.Content)
			})
		})
	}

	// https://youbbs.org/static/avatar/1.jpg
	// 自定义 avatar handle
	sm.GET("/static/avatar/{uid}.jpg", h.UserAvatarHandle)
	sm.GET("/icon/t/{tid}.jpg", h.TopicIconHandle)

	// db img
	sm.GET("/dbi/{key}", h.DbImageHandle)

	// 用户上传图片
	log.Printf("UploadDir from %q", ap.Cf.Site.UploadDir)
	if _, err := os.Stat(ap.Cf.Site.UploadDir); err != nil {
		//Dir not exist
		err = os.MkdirAll(ap.Cf.Site.UploadDir, os.ModePerm)
		if err != nil {
			log.Println("#os.MkdirAll UploadDir", err)
		}
	}
	sm.GET("/static/upload/{filepath:*}", func(ctx *fasthttp.RequestCtx) {
		if ap.Cf.Site.Authorized {
			ssValue := h.GetCookie(ctx, "SessionID")
			if len(ssValue) == 0 {
				_, _ = ctx.WriteString("401")
				return
			}
		}

		fp := ap.Cf.Site.UploadDir + sdb.B2s(ctx.Path()[14:])
		fasthttp.ServeFile(ctx, fp)
	})

	// static file server
	//mux.ServeFiles("/static/{filepath:*}", staticPath)
	//sm.ServeFilesCustom("/static/{filepath:*}", &fasthttp.FS{
	//	Root:               "static",
	//	GenerateIndexPages: false,
	//	AcceptByteRange:    true,
	//})

	// use embed.FS for static file
	sub, _ := fs.Sub(ap.Assets, "static")
	sm.ServeFilesCustom("/static/{filepath:*}", &fasthttp.FS{
		FS:              sub,
		Root:            "",
		AcceptByteRange: true,
	})

	sm.GET("/captcha/{filepath:*}", h.CaptchaHandle)

	sm.GET("/robots.txt", h.Robots)
	sm.GET("/ads.txt", h.Ads)
	sm.GET("/feed", h.FeedHandler)
	sm.GET("/sitemap.xml", h.SiteMapHandler)
	sm.GET("/sitemap/{xmlFile}", h.SitemapIndexHandler)
	sm.GET("/favicon.ico", h.ShowIcon)

	sm.POST("/get/link/count", h.GetLinkCount)
	sm.POST("/api/post/content", h.ApiAdminRemotePost) // 管理员发帖、评论接口

	sm.GET("/login", h.UserLoginPage)
	sm.POST("/login", h.UserLoginPost)
	sm.GET("/register", h.UserLoginPage)
	sm.POST("/register", h.UserLoginPost)
	sm.GET("/logout", h.UserLogout)

	sm.GET("/qqlogin", h.QQOauthHandler)
	sm.GET("/oauth/qq/callback", h.QQOauthCallback)
	sm.GET("/wblogin", h.WeiboOauthHandler)
	sm.GET("/oauth/wb/callback", h.WeiboOauthCallback)
	sm.GET("/githublogin", h.GithubOauthHandler)
	sm.GET("/oauth/github/callback", h.GithubOauthCallback)

	// admin
	// only post method
	sm.POST("/content/preview", h.ContentPreview)
	sm.POST("/user/avatar/upload", h.AvatarUpload)
	sm.POST("/file/upload", h.FileUpload)

	admin := sm.Group("/admin")
	admin.GET("/", h.AdminHomePage)
	//
	admin.GET("/node", h.AdminNodePage)
	admin.POST("/node", h.AdminNodePost)
	admin.GET("/link", h.AdminLinkPage)
	admin.POST("/link", h.AdminLinkPost)
	admin.GET("/site/conf", h.AdminSiteConfigPage)
	admin.POST("/site/conf", h.AdminSiteConfigPost)
	admin.GET("/site/router", h.AdminSiteRouterPage)
	admin.POST("/site/router", h.AdminSiteRouterPost)
	admin.GET("/site/download/cur/db", h.AdminCurDbPage)
	admin.GET("/site/download/cur/img", h.AdminImgPage)
	admin.GET("/user", h.AdminUserPage)
	admin.POST("/user", h.AdminUserPost)
	admin.GET("/topic/add", h.AdminTopicAddPage)
	admin.POST("/topic/add", h.AdminTopicAddPost)
	admin.GET("/topic/edit", h.AdminTopicEditPage)
	admin.POST("/topic/edit", h.AdminTopicAddPost)
	admin.GET("/topic/review", h.AdminTopicReviewPage)
	admin.POST("/topic/review", h.AdminTopicAddPost)
	admin.GET("/comment/review", h.AdminCommentReviewPage)
	admin.POST("/comment/review", h.AdminCommentReviewPost)
	admin.GET("/comment/edit", h.AdminCommentEditPage)
	admin.POST("/comment/edit", h.AdminCommentReviewPost)
	admin.GET("/ratelimit/iplookup", h.AdminRateLimitIpLookup)
	admin.GET("/ratelimit/setting", h.AdminRateLimitSetting)
	admin.POST("/ratelimit/setting", h.AdminRateLimitSettingPost)

	sm.GET("/name/{uname}", mdwRateLimit(h.MemberNamePage))
	sm.GET("/member/{uid}", mdwRateLimit(h.MemberPage))
	sm.GET("/t/{tid}", mdwRateLimit(h.TopicDetailPage))
	sm.POST("/t/{tid}", h.TopicDetailPost)

	sm.GET("/n/{nid}", mdwRateLimit(h.NodePage))
	sm.GET("/tag/{tag}", mdwRateLimit(h.TagPage))
	sm.GET("/q", mdwRateLimit(h.SearchPage))

	// login
	sm.GET("/my/msg", h.MyMsgPage)
	sm.GET("/topic/add", h.TopicAddPage)
	sm.POST("/topic/add", h.TopicAddPost)
	sm.GET("/setting", h.UserSettingPage)
	sm.POST("/setting", h.UserSettingPost)

	// old
	sm.GET("/topic/{tid}/{title}", func(ctx *fasthttp.RequestCtx) {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/t/"+ctx.UserValue("tid").(string), 301)
	})
	sm.GET("/category/{nid}/{title}", func(ctx *fasthttp.RequestCtx) {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/n/"+ctx.UserValue("nid").(string), 301)
	})

	sm.GET("/{filepath}", h.StaticFile)
	sm.GET("/", h.HomePage)
}

// mdwRateLimit simple mdw for RateLimit
func mdwRateLimit(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if model.RateLimitDay == 0 || model.RateLimitHour == 0 {
			// ok, go next
			next(ctx)
			return
		}

		// ua
		ua := useragent.Parse(string(ctx.Request.Header.Peek("User-Agent")))
		if len(ua.String) == 0 {
			ctx.Error("403", fasthttp.StatusForbidden)
			_, _ = ctx.Write(nil)
			return
		}

		m := model.BadBotNameMap.Load().(model.Map)
		if _, ok := m[ua.Name]; ok {
			ctx.Error("403", fasthttp.StatusForbidden)
			_, _ = ctx.Write(nil)
			return
		}

		// user ip
		uip := ReadUserIP(ctx)
		if len(uip) == 0 {
			ctx.Error("403", fasthttp.StatusForbidden)
			_, _ = ctx.Write(nil)
			return
		}

		// BadIpPrefix
		if model.BadIpPrefixLst.ItemInPrefix(uip) {
			ctx.Error("403", fasthttp.StatusForbidden)
			_, _ = ctx.Write(nil)
			return
		}

		t1 := time.Now()

		// RateLimit if not a bot
		// the maximum number of items I want from this user in one hour
		// check ip white ip prefix
		if !model.AllowIpPrefixLst.ItemInPrefix(uip) {
			// log.Println(uip, "not in white list")
			// if not in white list
			// hour
			_, cntH, underRateLimit := model.Limiter.Incr(uint64(t1.UTC().Unix()), uip)
			// log.Println(cntD, cntH, underRateLimit)
			if !underRateLimit {
				ctx.Error("429", fasthttp.StatusForbidden)
				_, _ = ctx.Write(nil)
				return
			}

			if ua.Bot {
				model.IpQue.Enqueue(uip)
			} else {
				// check ip dns, some time ua.Bot is false but are welcome
				if cntH%10 == 0 {
					model.IpQue.Enqueue(uip)
				}
			}
		}

		// ok, go next
		next(ctx)
	}
}
