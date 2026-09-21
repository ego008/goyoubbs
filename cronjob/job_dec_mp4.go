package cronjob

import (
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"

	"goyoubbs/model"
	"goyoubbs/util"
	"log"
	"os"
	"os/exec"
	"strings"
)

func decodeMp4(db *mdb.DB) {
	isRunning, _ := util.FindInPs("ffmpeg", "ffmpeg -y")
	if isRunning {
		// still running
		log.Println("ffmpeg isRunning")
		return
	}

	var canBreak bool
	var iFileMp4Str, iFileWebmStr string

	_ = db.Update(func(tx *bbolt.Tx) error {
		var value string
		_ = db.HScanFunc(tx, model.TbnV2DecMp4, nil, 1, func(key, val []byte) bool {
			iFileMp4Str = string(key)
			value = string(val)
			iFileWebmStr = strings.Replace(iFileMp4Str, ".mp4", ".webm", 1)
			return true
		})

		if len(iFileMp4Str) == 0 {
			canBreak = true
			return nil
		}

		if value == "1" {
			_, err := os.Stat(iFileWebmStr)
			if os.IsNotExist(err) {
				// NotExist
			} else {
				// Exist
				_ = db.HDel(tx, model.TbnV2DecMp4, []byte(iFileMp4Str))
				canBreak = true
				return nil
			}
		}
		return nil
	})

	if canBreak {
		return
	}

	log.Println("ffmpeg decoding ", iFileMp4Str)
	runFfmpeg := exec.Command("ffmpeg", "-y", "-i", iFileMp4Str, "-b:v", "0", "-crf", "30", iFileWebmStr)
	_ = runFfmpeg.Run()

	_ = db.Update(func(tx *bbolt.Tx) error {
		_ = db.HSet(tx, model.TbnV2DecMp4, []byte(iFileMp4Str), []byte("1"))
		return nil
	})

	log.Println("ffmpeg decode done ", iFileWebmStr)
}
