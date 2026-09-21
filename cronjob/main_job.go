package cronjob

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"sync/atomic"
	"time"

	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"
)

type BaseHandler struct {
	App *model.Application
}

func (h *BaseHandler) MainCronJob() {

	db := h.App.Db

	// do prepare
	loadDb2Mc(db)

	if h.App.Cf.Site.AutoDecodeMp4 && util.CmdExists("ffmpeg") {
		go func(db *mdb.DB) {
			tickA := time.Tick(3 * time.Minute)
			for {
				select {
				case <-tickA:
					decodeMp4(db)
				default:
					time.Sleep(time.Second)
				}
			}
		}(db)
	}

	spl := h.App.Spl
	tick1 := time.Tick(3 * time.Minute)   // 清理过期一些 keys
	tick3 := time.Tick(30 * time.Minute)  // 数据库备份
	tick6 := time.Tick(9 * time.Second)   // 从 title 提取 tag
	tick7 := time.Tick(29 * time.Second)  // 设置 topic tag
	tick8 := time.Tick(2 * time.Second)   // check bot ip
	tick10 := time.Tick(21 * time.Second) // avatar
	tick11 := time.Tick(39 * time.Second) // send mail

	daySecond := int64(3600 * 24)

	for {
		if atomic.LoadUint32(&model.AppStop) > 0 {
			time.Sleep(time.Second * 3)
			continue
		}

		select {
		case <-tick1:
			lk := spl.Init("tk1")
			if !lk.IsLocked() {
				lk.Lock()
				// 做一些清理工作
				limit := 10
				timeBefore := uint64(time.Now().UTC().Unix() - daySecond)
				zbnList := []string{
					"article_detail_token",
					// "user_login_token", // 登录出错限制，未实现
				}
				_ = db.Update(func(tx *bbolt.Tx) error {
					for _, bn := range zbnList {
						var keys [][]byte
						_ = db.ZRScanFunc(tx, bn, nil, timeBefore, 0, limit, func(k []byte, s uint64) bool {
							keys = append(keys, k)
							return true
						})
						if len(keys) > 0 {
							_ = db.ZMDel(tx, bn, keys)
						}
					}
					return nil
				})
				lk.UnLock()
			}
		case <-tick3:
			if h.App.Cf.Site.AutoDataBackup {
				dataBackup(db, h.App.Cf.Site.DataBackupDir)
			}
		case <-tick6:
			if len(h.App.Cf.Site.GetTagApi) > 0 {
				lk := spl.Init("tk6")
				if !lk.IsLocked() {
					lk.Lock()
					getTagFromTitle(db, h.App.Cf.Site.GetTagApi)
					lk.UnLock()
				}
			}
		case <-tick7:
			lk := spl.Init("tk7")
			if !lk.IsLocked() {
				lk.Lock()
				setArticleTag(h.App.Mc, db)
				lk.UnLock()
			}
		case <-tick8:
			lk := spl.Init("tk8")
			if !lk.IsLocked() {
				lk.Lock()
				spiderIpCheck(db)
				lk.UnLock()
			}
		case <-tick10:
			lk := spl.Init("tk10")
			if !lk.IsLocked() {
				lk.Lock()
				SetAvatar(db, h.App.Cf.Site.Socks5Proxy)
				lk.UnLock()
			}
		case <-tick11:
			if h.App.Cf.Site.SendEmail {
				lk := spl.Init("tk11")
				if !lk.IsLocked() {
					lk.Lock()
					sendMail(db, h.App.Cf.Site)
					lk.UnLock()
				}
			}
		}
	}
}
