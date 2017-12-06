package system

import (
	"log"
	"runtime"

	"github.com/ego008/goyoubbs/util"
	"github.com/ego008/youdb"
	"github.com/gorilla/securecookie"
	"github.com/qiniu/api.v7/storage"
	"github.com/weint/config"
	"net/url"
	"strings"
)

type MainConf struct {
	ListenAddr     string
	Domain         string // 若启用https 则该domain 为注册的域名，eg: domain.com、www.domain.com
	HttpsOn        bool
	ListenPort     int
	PubDir         string
	ViewDir        string
	Youdb          string
	CookieSecure   bool
	CookieHttpOnly bool
	OldSiteDomain  string
}

type SiteConf struct {
	GoVersion         string
	MD5Sums           string
	Name              string
	Desc              string
	AdminEmail        string
	MainDomain        string // 上传图片后添加网址前缀, eg: http://domian.com 、http://234.21.35.89:8082
	MainNodeIds       string
	HomeShowNum       int
	PageShowNum       int
	TagShowNum        int
	TitleMaxLen       int
	ContentMaxLen     int
	PostInterval      int
	CommentListNum    int
	CommentInterval   int
	Authorized        bool
	RegReview         bool
	CloseReg          bool
	AutoDataBackup    bool
	AutoGetTag        bool
	GetTagApi         string
	QQClientID        int
	QQClientSecret    string
	WeiboClientID     int
	WeiboClientSecret string
	QiniuAccessKey    string
	QiniuSecretKey    string
	QiniuDomain       string
	QiniuBucket       string
	UpyunDomain       string
	UpyunBucket       string
	UpyunUser         string
	UpyunPw           string
}

type AppConf struct {
	Main *MainConf
	Site *SiteConf
}

type Application struct {
	Cf     *AppConf
	Db     *youdb.DB
	Sc     *securecookie.SecureCookie
	QnZone *storage.Zone
}

func LoadConfig(filename string) *config.Engine {
	c := &config.Engine{}
	c.Load(filename)
	return c
}

func (app *Application) Init(c *config.Engine, currentFilePath string) {

	mcf := &MainConf{}
	c.GetStruct("Main", mcf)

	// check domain
	if strings.HasPrefix(mcf.Domain, "http") {
		dm, err := url.Parse(mcf.Domain)
		if err != nil {
			log.Fatal("domain fmt err", err)
		}
		mcf.Domain = dm.Host
	} else {
		mcf.Domain = strings.Trim(mcf.Domain, "/")
	}

	scf := &SiteConf{}
	c.GetStruct("Site", scf)
	scf.GoVersion = runtime.Version()
	fMd5, _ := util.HashFileMD5(currentFilePath)
	scf.MD5Sums = fMd5
	scf.MainDomain = strings.Trim(scf.MainDomain, "/")
	log.Println("MainDomain:", scf.MainDomain)

	app.Cf = &AppConf{mcf, scf}
	db, err := youdb.Open(mcf.Youdb)
	if err != nil {
		log.Fatalf("Connect Error: %v", err)
	}
	app.Db = db

	// set main node
	db.Hset("keyValue", []byte("main_category"), []byte(scf.MainNodeIds))

	app.Sc = securecookie.New(securecookie.GenerateRandomKey(64),
		securecookie.GenerateRandomKey(32))
	//app.Sc.SetSerializer(securecookie.JSONEncoder{})

	log.Println("youdb Connect to", mcf.Youdb)
}

func (app *Application) Close() {
	app.Db.Close()
	log.Println("db cloded")
}
