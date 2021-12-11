package cronjob

import (
	"github.com/ego008/sdb"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	ldbUtil "github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"os"
	"time"
)

func dataBackup(db *sdb.DB, bakDir string) {
	if _, err := os.Stat(bakDir); err != nil {
		//Dir not exist
		err = os.MkdirAll(bakDir, os.ModePerm)
		if err != nil {
			log.Println("#os.MkdirAll dataBackup", err)
			return
		}
	}

	sdbFold := bakDir + "/" + time.Now().UTC().Format("20060102")

	if _, err := os.Stat(sdbFold); err == nil {
		//log.Println("sdbFold exist", sdbFold)
		return
	}

	t1 := time.Now()
	db2, err := sdb.Open(sdbFold, &opt.Options{
		Filter: filter.NewBloomFilter(10), // 一般取10
	})
	if err != nil {
		return
	}
	defer func() {
		_ = db2.Close()
	}()

	batchNum := 500 // 批量写条数
	iter := db.NewIterator(nil, nil)
	ic := 0
	ic2 := 0
	batch := new(leveldb.Batch)
	for iter.Next() {
		if batch.Len() > 0 && (ic%batchNum) == 0 {
			ic2 += batch.Len()
			err = db2.Write(batch, nil)
			if err != nil {
				return
			}
			batch = new(leveldb.Batch)
		}
		batch.Put(iter.Key(), iter.Value())
		ic++
	}
	if batch.Len() > 0 {
		ic2 += batch.Len()
		err = db2.Write(batch, nil)
		if err != nil {
			return
		}
	}

	iter.Release()
	err = iter.Error()
	if err != nil {
		return
	}

	err = db2.CompactRange(ldbUtil.Range{})
	if err != nil {
		log.Println(err)
		return
	}

	err = db2.Close() // !important
	if err != nil {
		log.Println(err)
		return
	}

	// 删掉n天前的备份一个
	sdbFold = bakDir + "/" + time.Now().UTC().AddDate(0, 0, -14).Format("20060102")
	_ = os.RemoveAll(sdbFold)

	log.Println("databackup done", ic, ic2, time.Now().Sub(t1))
}
