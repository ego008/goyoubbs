package cronjob

import (
	"log"
	"os"
	"time"

	"github.com/ego008/mdb"
)

func dataBackup(db *mdb.DB, bakDir string) {
	if _, err := os.Stat(bakDir); err != nil {
		//Dir not exist
		err = os.MkdirAll(bakDir, os.ModePerm)
		if err != nil {
			log.Println("#os.MkdirAll dataBackup", err)
			return
		}
	}

	mdbFile := bakDir + "/" + time.Now().UTC().Format("20060102") + ".zip"

	if _, err := os.Stat(mdbFile); err == nil {
		//log.Println("mdbFile exist", mdbFile)
		return
	}

	t1 := time.Now()

	_ = db.CompactZip("", mdbFile)

	// 删掉n天前的备份一个
	oldBakFile := bakDir + "/" + time.Now().UTC().AddDate(0, 0, -14).Format("20060102") + ".zip"
	_ = os.RemoveAll(oldBakFile)

	log.Println("data backup done", time.Now().Sub(t1))
}
