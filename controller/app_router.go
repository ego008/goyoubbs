package controller

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"goyoubbs/model"

	"github.com/ego008/goutils/json"
	"github.com/gin-gonic/gin"
	"github.com/mileusna/useragent"
	"go.etcd.io/bbolt"
)

func RouterReload(ap *model.Application) {
	// 假设在 main.go 中已初始化 router 并传入，或在此处创建并赋值给 ap.GinEngine
	var router *gin.Engine
	if ap.Mux != nil {
		router = ap.Mux
	} else {
		router = gin.New()
		router.Use(gin.Logger(), gin.Recovery())
		ap.Mux = router
	}
	MainRouter(ap, router)
}

func MainRouter(ap *model.Application, router *gin.Engine) {
	h := BaseHandler{App: ap}

	// 自定义路径及内容 参见 https://youbbs.org/t/3324
	// 浏览需登录设置对自定义路由无效
	var keyStart []byte
	_ = ap.Db.View(func(tx *bbolt.Tx) error {
		for {
			var ok bool
			_ = ap.Db.HScanFunc(tx, "custom_router", keyStart, 20, func(key, val []byte) bool {
				ok = true

				keyStart = key
				obj := model.CustomRouter{}
				err := json.Unmarshal(val, &obj)
				if err != nil {
					return true
				}
				// 默认 GET
				router.GET(obj.Router, func(c *gin.Context) {
					reqKey := c.Request.URL.Path
					var ok2 bool
					_ = ap.Db.HGetFunc(tx, "custom_router", []byte(reqKey), func(v []byte) error {
						ok2 = true
						obj2 := model.CustomRouter{}
						_ = json.Unmarshal(v, &obj2)
						if strings.HasPrefix(obj2.Content, "goto:") {
							// match goto url
							// goto: https://youbbs.org/
							c.Redirect(http.StatusFound, strings.TrimSpace(obj2.Content[5:]))
							return nil
						}
						c.Data(http.StatusOK, obj2.MimeType, []byte(obj2.Content))
						return nil
					})

					if !ok2 {
						c.String(http.StatusNotFound, "404 page not found")
						return
					}

				})

				return true
			})
			if !ok {
				break
			}
		}
		return nil
	})

	// https://youbbs.org/avatar/1.jpg
	// 自定义 avatar handle
	// old /avatar/:uid.jpg
	router.GET("/avatar/:uid.jpg", h.UserAvatarHandle)
	router.GET("/icon/t/:tid.jpg", h.TopicIconHandle)

	// db img
	router.GET("/dbi/:key", h.DbImageHandle)

	// 用户上传图片
	log.Printf("UploadDir from %q", ap.Cf.Site.UploadDir)
	if _, err := os.Stat(ap.Cf.Site.UploadDir); err != nil {
		// Dir not exist
		err = os.MkdirAll(ap.Cf.Site.UploadDir, os.ModePerm)
		if err != nil {
			log.Println("#os.MkdirAll UploadDir", err)
		}
	}
	// old /static/upload/*filepath
	router.GET("/upload/*filepath", func(c *gin.Context) {
		if ap.Cf.Site.Authorized {
			ssValue := h.GetCookie(c, "SessionID")
			if len(ssValue) == 0 {
				c.String(http.StatusUnauthorized, "401")
				return
			}
		}

		filepath := c.Param("filepath")
		fp := ap.Cf.Site.UploadDir + filepath
		c.File(fp)
	})

	// use embed.FS for static file
	sub, err := fs.Sub(ap.Assets, "static")
	if err != nil {
		log.Panicln("#fs.Sub error:", err)
	}

	router.GET("/static/*filepath", func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// 如果请求路径以 /static/upload/ 开头，转向头像处理函数
		if strings.HasPrefix(reqPath, "/static/upload/") {
			if len(reqPath) < 19 {
				c.Status(404)
				return
			}
			fp := ap.Cf.Site.UploadDir + "/" + reqPath[15:]
			c.File(fp)
			return
		}

		// 其他普通静态资源（如 /static/css/style.css）交给 embed.FS
		fileServer := http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	router.GET("/captcha/*filepath", h.CaptchaHandle)

	router.GET("/robots.txt", h.Robots)
	router.GET("/ads.txt", h.Ads)
	router.GET("/feed", h.FeedHandler)
	router.GET("/sitemap.xml", h.SiteMapHandler)
	router.GET("/sitemap/:xmlFile", h.SitemapIndexHandler)
	router.GET("/favicon.ico", h.ShowIcon)

	router.POST("/get/link/count", h.GetLinkCount)
	router.POST("/api/post/content", h.ApiAdminRemotePost) // 管理员发帖、评论接口
	router.POST("/api/post/print", h.ApiAdminPrintPost)    // 管理员打印post

	router.GET("/login", h.UserLoginPage)
	router.POST("/login", h.UserLoginPost)
	router.GET("/register", h.UserLoginPage)
	router.POST("/register", h.UserLoginPost)
	router.GET("/logout", h.UserLogout)

	router.GET("/qqlogin", h.QQOauthHandler)
	router.GET("/oauth/qq/callback", h.QQOauthCallback)
	router.GET("/wblogin", h.WeiboOauthHandler)
	router.GET("/oauth/wb/callback", h.WeiboOauthCallback)
	router.GET("/githublogin", h.GithubOauthHandler)
	router.GET("/oauth/github/callback", h.GithubOauthCallback)

	// admin
	// only post method
	router.POST("/content/preview", h.ContentPreview)
	router.POST("/user/avatar/upload", h.AvatarUpload)
	router.POST("/file/upload", h.FileUpload)

	admin := router.Group("/admin")
	{
		admin.GET("/", h.AdminHomePage)
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
	}

	router.GET("/name/:uname", mdwRateLimit(), h.MemberNamePage)
	router.GET("/member/:uid", mdwRateLimit(), h.MemberPage)
	router.GET("/t/:tid", mdwRateLimit(), h.TopicDetailPage)
	router.POST("/t/:tid", h.TopicDetailPost)

	router.GET("/n/:nid", mdwRateLimit(), h.NodePage)
	router.GET("/tag/:tag", mdwRateLimit(), h.TagPage)
	router.GET("/q", mdwRateLimit(), h.SearchPage)

	// login
	router.GET("/my/msg", h.MyMsgPage)
	router.GET("/topic/add", h.TopicAddPage)
	router.POST("/topic/add", h.TopicAddPost)
	router.GET("/setting", h.UserSettingPage)
	router.POST("/setting", h.UserSettingPost)

	// old
	router.GET("/topic/:tid/:title", func(c *gin.Context) {
		tid := c.Param("tid")
		c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s/t/%s", ap.Cf.Site.MainDomain, tid))
	})
	router.GET("/category/:nid/:title", func(c *gin.Context) {
		nid := c.Param("nid")
		c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s/n/%s", ap.Cf.Site.MainDomain, nid))
	})

	router.GET("/:filepath", h.StaticFile)
	router.GET("/", h.HomePage)
}

// mdwRateLimit 改写为符合 Gin 规范的中间件构造函数
func mdwRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if model.RateLimitDay == 0 || model.RateLimitHour == 0 {
			c.Next()
			return
		}

		// ua
		ua := useragent.Parse(c.GetHeader("User-Agent"))
		if len(ua.String) == 0 {
			c.String(http.StatusForbidden, "403")
			c.Abort()
			return
		}

		m := model.BadBotNameMap.Load().(model.Map)
		if _, ok := m[ua.Name]; ok {
			c.String(http.StatusForbidden, "403")
			c.Abort()
			return
		}

		// user ip
		uip := ReadUserIP(c)
		if len(uip) == 0 {
			c.String(http.StatusForbidden, "403")
			c.Abort()
			return
		}

		// BadIpPrefix
		if model.BadIpPrefixLst.ItemInPrefix(uip) {
			c.String(http.StatusForbidden, "403")
			c.Abort()
			return
		}

		t1 := time.Now()

		// RateLimit if not a bot
		// check ip white ip prefix
		if !model.AllowIpPrefixLst.ItemInPrefix(uip) {
			_, cntH, underRateLimit := model.Limiter.Incr(uint64(t1.UTC().Unix()), uip)
			if !underRateLimit {
				c.String(http.StatusForbidden, "429")
				c.Abort()
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

		c.Next()
	}
}
