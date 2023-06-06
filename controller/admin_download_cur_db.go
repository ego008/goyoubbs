package controller

import (
	"github.com/ego008/sdb"
	"github.com/klauspost/compress/zip"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	ldbUtil "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/valyala/fasthttp"
	"goyoubbs/util"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (h *BaseHandler) AdminCurDbPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.ID == 0 {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	dir, err := os.MkdirTemp("", "sdb")
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	t1 := time.Now()
	db2, err := sdb.Open(dir, &opt.Options{
		Filter: filter.NewBloomFilter(10),
	})
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		_ = db2.Close()
	}()

	batchNum := 500 // 批量写条数
	iter := h.App.Db.NewIterator(nil, nil)
	ic := 0
	ic2 := 0
	batch := new(leveldb.Batch)
	for iter.Next() {
		if batch.Len() > 0 && (ic%batchNum) == 0 {
			ic2 += batch.Len()
			err = db2.Write(batch, nil)
			if err != nil {
				log.Println(err)
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
			log.Println(err)
			return
		}
	}

	iter.Release()
	err = iter.Error()
	if err != nil {
		log.Println(err)
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

	log.Println("cur data copy done", ic, ic2, time.Now().Sub(t1))

	ts := util.TimeFmt(time.Now().Unix(), "20060102150405")
	zipName := "db_" + ts + ".zip"
	defer func() {
		_ = os.Remove(zipName)
	}()

	err = zipIt(dir, zipName)
	if err != nil {
		log.Println(err)
		return
	}

	ctx.SetContentType("application/zip")

	ctx.Response.Header.Set("Content-Disposition", "attachment; filename="+zipName)
	ctx.SendFile(zipName)
	return
}

func zipIt(source, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		_ = zipFile.Close()
	}()

	archive := zip.NewWriter(zipFile)
	defer func() {
		_ = archive.Close()
	}()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	_ = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			_ = file.Close()
		}()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}
