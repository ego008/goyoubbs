package model

import (
	"bytes"
	"embed"
	"goyoubbs/util"
	"log"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/goutils/lfqueue"
	"github.com/ego008/goutils/ratelimit"
	"github.com/ego008/goutils/splock"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"go.etcd.io/bbolt"
)

var (
	Limiter *ratelimit.Cache
	IpQue   = lfqueue.NewLKQueue() // ip wait to lookup
)

type MyAppConf struct {
	Main *MainConf
	Site *SiteConf
}

type Application struct {
	Cf     *MyAppConf
	Db     *mdb.DB
	Sc     *securecookie.SecureCookie
	Mc     *fastcache.Cache // 数量不固定的缓存，或者是不需要序列化的内容
	Mux    *gin.Engine
	Assets *embed.FS
	Spl    *splock.SimpleLock
}

func (app *Application) Init(addr, sdbDir string, assetFs *embed.FS) {

	mcf := &MainConf{
		Addr:   addr,
		SdbDir: sdbDir,
	}

	db, err := mdb.Open(mcf.SdbDir)
	if err != nil {
		log.Fatalf("Connect Error: %v", err)
	}
	app.Db = db

	scf := SiteConf{}
	var hashKey []byte
	var blockKey []byte
	_ = db.Update(func(tx *bbolt.Tx) error {
		// 从数据库读取网站配置
		SiteConfLoad(&scf, db, tx)

		// 简单识别本地开发 go run main.go
		// /var/folders/bw/8bnjyv6j4k73h6j2qwh9s7xr0000gn/T/go-build1539771127/b001/exe/main
		scf.IsDevMod = strings.HasSuffix(os.Args[0], "exe/main")

		scf.SelfHash = util.HashFile(os.Args[0])
		log.Println("SelfHash:", scf.SelfHash)

		app.Cf = &MyAppConf{
			Main: mcf,
			Site: &scf,
		}

		hkb := []byte("hashKey")
		bkb := []byte("blockKey")
		if scf.ResetCookieKey {
			hashKey = securecookie.GenerateRandomKey(64)
			blockKey = securecookie.GenerateRandomKey(32)
			_ = db.HMSet(tx, KeyValueTb, hkb, hashKey, bkb, blockKey)
		} else {
			_ = db.HMGetFunc(tx, KeyValueTb, [][]byte{hkb, bkb}, func(key, val []byte) error {
				if val != nil {
					if string(key) == "hashKey" {
						hashKey = bytes.Clone(val)
					}
					if string(key) == "blockKey" {
						blockKey = bytes.Clone(val)
					}
				}
				return nil
			})
			if len(hashKey) == 0 {
				hashKey = securecookie.GenerateRandomKey(64)
				blockKey = securecookie.GenerateRandomKey(32)
				_ = db.HMSet(tx, KeyValueTb, hkb, hashKey, bkb, blockKey)
			}
		}

		// 新站初始化
		// 通过取第一个分类来判断
		nodeNum := db.HGetInt(tx, CountTb, []byte(NodeTbName))
		if nodeNum == 0 {
			nt := uint64(time.Now().UTC().Unix())
			//
			_ = db.HSet(tx, CountTb, []byte("site_create_time"), mdb.I2b(nt))
			// Node
			newCid, err2 := db.HIncr(tx, CountTb, []byte(NodeTbName), 1)
			if err2 == nil {
				obj := Node{
					ID:    newCid,
					Name:  "默认分类",
					About: "默认第一个分类",
				}
				jb, _ := json.Marshal(obj)
				_ = db.HSet(tx, NodeTbName, mdb.I2b(obj.ID), jb)
			}
			// link
			LinkSet(db, tx, Link{
				Name:  "youBBS",
				Url:   "https://youbbs.org",
				Score: 100,
			})
			// User
			userId, _ := db.HIncr(tx, CountTb, mdb.S2b(UserTbName), 1)
			UserSet(db, tx, User{
				ID:            userId,
				Name:          "admin",
				Flag:          FlagAdmin,
				Password:      util.Md5("admin"),
				RegTime:       nt,
				LastLoginTime: nt,
			})
			_ = db.HSet(tx, "user_name2uid", []byte("admin"), mdb.I2b(userId))
			_ = db.HSet(tx, "user_flag:"+strconv.Itoa(FlagAdmin), mdb.I2b(userId), nil)
			//生成头像
			_ = util.GenAvatar(db, tx, userId, "admin")
		}
		return nil
	})

	app.Sc = securecookie.New(hashKey, blockKey)
	app.Mc = fastcache.New(scf.CachedSize * 1024 * 1024)

	// rateLimit
	Limiter = ratelimit.NewCache(1000, scf.RateLimitDay, scf.RateLimitHour)
	// if Limiter in mc, load them
	if mcValue := app.Mc.GetBig(nil, []byte("mc_Limiter")); len(mcValue) > 0 {
		Limiter.Load(mcValue)
	}

	app.Assets = assetFs
	app.Spl = &splock.SimpleLock{}
}

func (app *Application) Close() {
	// 改变中断状态
	atomic.StoreUint32(&AppStop, 1)
	// 检测是否还存在后台运行的函数
	for {
		if hasLock, _ := app.Spl.HasLocked(); hasLock {
			time.Sleep(time.Millisecond * 10)
			continue
		}
		log.Println("break!")
		break
	}

	_ = app.Db.Close()

	// set limiter data to mc
	if jb := Limiter.Dump(); len(jb) > 0 {
		app.Mc.SetBig([]byte("mc_Limiter"), jb)
	}

	log.Println("db closed")
	app.Mc.Reset()
	log.Println("mc Reset")
}
