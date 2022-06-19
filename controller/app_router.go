package controller

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"goyoubbs/app"
	"goyoubbs/middleware"
	"goyoubbs/model"
	"log"
	"os"
	"strings"
)

func RouterReload(ap *app.Application) {
	mux := router.New()
	MainRouter(ap, mux)
	ap.Mux = mux
}

func MainRouter(ap *app.Application, sm *router.Router) {
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
		fp := ap.Cf.Site.UploadDir + sdb.B2s(ctx.Path()[14:])
		fasthttp.ServeFile(ctx, fp)
	})

	// static file server
	//mux.ServeFiles("/static/{filepath:*}", staticPath)
	sm.ServeFilesCustom("/static/{filepath:*}", &fasthttp.FS{
		Root: "static",
		//IndexNames:         []string{"index.html"},
		GenerateIndexPages: false, // 设为 false 时不能浏览目录
		AcceptByteRange:    true,
	})

	sm.GET("/captcha/{filepath:*}", h.CaptchaHandle)

	sm.GET("/robots.txt", h.Robots)
	sm.GET("/ads.txt", h.Ads)
	sm.GET("/feed", h.FeedHandler)
	sm.GET("/sitemap.xml", h.SiteMapHandler)

	sm.POST("/get/link/count", h.GetLinkCount)
	sm.POST("/api/post/content", h.ApiAdminRemotePost) // 管理员发帖、评论接口

	sm.GET("/login", mdw.RspTime(h.UserLoginPage))
	sm.POST("/login", h.UserLoginPost)
	sm.GET("/register", mdw.RspTime(h.UserLoginPage))
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
	admin.GET("", h.AdminHomePage) // 后面没有斜杠
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

	sm.GET("/name/{uname}", h.MemberNamePage)
	sm.GET("/member/{uid}", mdw.RspTime(h.MemberPage))
	sm.GET("/t/{tid}", mdw.RspTime(h.TopicDetailPage))
	sm.POST("/t/{tid}", h.TopicDetailPost)

	sm.GET("/n/{nid}", mdw.RspTime(h.NodePage))
	sm.GET("/tag/{tag}", mdw.RspTime(h.TagPage))
	sm.GET("/q", mdw.RspTime(h.SearchPage))

	// login
	sm.GET("/my/msg", mdw.RspTime(h.MyMsgPage))
	sm.GET("/topic/add", mdw.RspTime(h.TopicAddPage))
	sm.POST("/topic/add", h.TopicAddPost)
	sm.GET("/setting", mdw.RspTime(h.UserSettingPage))
	sm.POST("/setting", h.UserSettingPost)

	// old
	sm.GET("/topic/{tid}/{title}", func(ctx *fasthttp.RequestCtx) {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/t/"+ctx.UserValue("tid").(string), 301)
	})
	sm.GET("/category/{nid}/{title}", func(ctx *fasthttp.RequestCtx) {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/n/"+ctx.UserValue("nid").(string), 301)
	})

	sm.GET("/{filepath}", h.StaticFile)
	sm.GET("/", mdw.RspTime(h.HomePage))
}
