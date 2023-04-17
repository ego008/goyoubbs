package model

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/sdb"
	"github.com/fasthttp/router"
	"github.com/gorilla/securecookie"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"log"
	"os"
	"strings"
)

type MyAppConf struct {
	Main *MainConf
	Site *SiteConf
}

type Application struct {
	Cf  *MyAppConf
	Db  *sdb.DB
	Sc  *securecookie.SecureCookie
	Mc  *fastcache.Cache // 数量不固定的缓存，或者是不需要序列化的内容
	Mux *router.Router
}

func (app *Application) Init(addr, sdbDir string) {

	mcf := &MainConf{
		Addr:   addr,
		SdbDir: sdbDir,
	}

	db, err := sdb.Open(mcf.SdbDir, &opt.Options{
		Filter: filter.NewBloomFilter(10), // 一般取10
	})
	if err != nil {
		log.Fatalf("Connect Error: %v", err)
	}
	app.Db = db

	scf := SiteConf{}
	// 从数据库读取网站配置
	SiteConfLoad(&scf, db)

	// 简单识别本地开发 go run main.go
	// /var/folders/bw/8bnjyv6j4k73h6j2qwh9s7xr0000gn/T/go-build1539771127/b001/exe/main
	scf.IsDevMod = strings.HasSuffix(os.Args[0], "exe/main")

	app.Cf = &MyAppConf{mcf, &scf}

	var hashKey []byte
	var blockKey []byte
	if scf.ResetCookieKey {
		hashKey = securecookie.GenerateRandomKey(64)
		blockKey = securecookie.GenerateRandomKey(32)
		_ = db.Hmset(KeyValueTb, []byte("hashKey"), hashKey, []byte("blockKey"), blockKey)
	} else {
		hashKey = append(hashKey, db.Hget(KeyValueTb, []byte("hashKey")).Bytes()...)
		blockKey = append(blockKey, db.Hget(KeyValueTb, []byte("blockKey")).Bytes()...)
		if len(hashKey) == 0 {
			hashKey = securecookie.GenerateRandomKey(64)
			blockKey = securecookie.GenerateRandomKey(32)
			_ = db.Hmset(KeyValueTb, []byte("hashKey"), hashKey, []byte("blockKey"), blockKey)
		}
	}

	app.Sc = securecookie.New(hashKey, blockKey)
	app.Mc = fastcache.New(scf.CachedSize * 1024 * 1024)
}

func (app *Application) Close() {
	_ = app.Db.Close()
	log.Println("db closed")
	app.Mc.Reset()
	log.Println("mc Reset")
}
