package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"time"
)

var (
	siteConfKeyName = []byte("site_config")
)

type MainConf struct {
	Addr   string // :8080
	SdbDir string // sdb 文件夹
}

type SiteConf struct {
	Name               string // 网站名
	SubTitle           string // 网站副标题
	Desc               string // 网站描述
	AdminEmail         string // 提供对外联系方式
	MainDomain         string // 主域名，sitemap 等调用 https://example.com !! 最后字符不包含"/"
	HeaderPartCon      string // 直接显示在页面 Header 里的内容
	GoogleAutoAdJs     string // 放在Header 里的google adsense js 代码
	FooterPartHtml     string // 直接显示在页脚的 html 内容，如备案信息及统计js代码
	TimeZone           int    // 时区
	PageShowNum        int    // 每页显示文章数
	TopRateNum         int    // 侧栏top rate 显示文章数
	RecentCommentNum   int    // 侧栏显示最近评论条数
	TitleMaxLen        int    // 标题最多字数
	TopicConMaxLen     int    // 主贴内容最大字数，管理员没限制
	CommentConMaxLen   int    // 评论内容最大字数，管理员没限制
	AutoDataBackup     bool   // 自动备份数据库
	DataBackupDir      string // 存放备份数据库目录
	Authorized         bool   // 需要登录才能浏览
	AllowNameReg       bool   // 是否允许用户名注册（false 时只允许第三方用户登录）
	RegReview          bool   // 用户注册审核
	CloseReg           bool   // 关闭用户注册
	CloseReply         bool   // 关闭评论
	PostReview         bool   // 发帖、回复审核
	ResetCookieKey     bool   // 重设cookie key （强迫重新登录）
	AutoDecodeMp4      bool   // 自动转码 mp4 -> webm，需要调用 ffmpeg
	GetTagApi          string // 远程取 tag 网址
	UploadLimit        bool   // 上传图片限制，只允许管理员上传
	UploadDir          string // 存放用户上传图片目录
	UploadMaxSize      int    // 上传文件最大 m
	UploadMaxSizeByte  int64  // 上传文件最大 b
	CachedSize         int    // 缓存大小 x MB
	RateLimitDay       int    // 120 日访问限制
	RateLimitHour      int    // 60 小时访问限制
	SaveTopicIcon      bool   // 帖子九宫格图片保存到数据库（以空间换CPU）
	SaveImg2db         bool   // 上传的图片保存到数据库
	RemotePostPw       string // 供管理员远程发布帖子、评论密码
	IsDevMod           bool   // 开发模式
	SelfHash           string // 自身hash
	Socks5Proxy        string // 122.33.44.55:123
	QQClientID         string
	QQClientSecret     string
	WeiboClientID      string
	WeiboClientSecret  string
	GithubClientID     string
	GithubClientSecret string
	SendEmail          bool // 有待验证帖子、回复是否发邮件
	SmtpHost           string
	SmtpPort           int
	SmtpEmail          string // 登录email
	SmtpPassword       string
	SendToEmail        string // 发到通知邮箱
}

func SiteConfLoad(scf *SiteConf, db *sdb.DB) {
	rs := db.Hget(KeyValueTb, siteConfKeyName)
	if rs.OK() {
		_ = json.Unmarshal(rs.Bytes(), &scf)
	} else {
		// 设置一些必要的默认值
		scf.Name = "GoYouBBS"
		scf.MainDomain = "http://127.0.0.1:8080"
		scf.TimeZone = 8
		scf.PageShowNum = 30
		scf.TopRateNum = 10
		scf.RecentCommentNum = 10
		scf.TitleMaxLen = 110
		scf.TopicConMaxLen = 12000
		scf.CommentConMaxLen = 5000
		scf.Authorized = false
		scf.AllowNameReg = true
		scf.RegReview = false
		scf.CloseReg = false
		scf.PostReview = false
		scf.AutoDataBackup = false
		scf.DataBackupDir = "data_backup"
		scf.ResetCookieKey = false
		scf.GetTagApi = ""
		scf.UploadLimit = true
		scf.UploadDir = "upload"
		scf.UploadMaxSize = 20
		scf.CachedSize = 5

		scf.UploadMaxSizeByte = int64(scf.UploadMaxSize) << 20

		// 保存
		jb, _ := json.Marshal(scf)
		_ = db.Hset(KeyValueTb, siteConfKeyName, jb)

		TimeOffSet = time.Duration(scf.TimeZone) * time.Hour
	}
}

func ConfLoad2MC(db *sdb.DB) {
	obj := SiteConf{}
	rs := db.Hget(KeyValueTb, []byte("site_config"))
	if !rs.OK() {
		return
	}
	_ = json.Unmarshal(rs.Bytes(), &obj)
	RateLimitDay = obj.RateLimitDay
	RateLimitHour = obj.RateLimitHour
}
