package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zip"
)

func (h *BaseHandler) AdminCurDbPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	t1 := time.Now()

	ts := util.TimeFmt(time.Now().Unix(), "20060102150405")
	zipName := "db_" + ts + ".zip"
	defer func() {
		_ = os.Remove(zipName)
	}()

	err := h.App.Db.CompactZip("", zipName)

	if err != nil {
		log.Println(err)
		return
	}
	log.Println("cur data copy done", time.Now().Sub(t1))

	c.Header("Content-Type", "application/zip")

	c.Header("Content-Disposition", "attachment; filename="+zipName)
	c.File(zipName)
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
