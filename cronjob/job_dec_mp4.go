package cronjob

import (
	"github.com/ego008/sdb"
	"goyoubbs/model"
	"goyoubbs/util"
	"log"
	"os"
	"os/exec"
	"strings"
)

func decodeMp4(db *sdb.DB) {
	rs := db.Hscan(model.TbnV2DecMp4, nil, 1)
	if !rs.OK() {
		return
	}

	iFileMp4Str := string(rs.Data[0])
	value := string(rs.Data[1])
	iFileWebmStr := strings.Replace(iFileMp4Str, ".mp4", ".webm", 1)

	isRunning, _ := util.FindInPs("ffmpeg", "ffmpeg -y")
	if isRunning {
		// still running
		log.Println("ffmpeg isRunning")
		return
	}

	if value == "1" {
		_, err := os.Stat(iFileWebmStr)
		if os.IsNotExist(err) {
			// NotExist
		} else {
			// Exist
			_ = db.Hdel(model.TbnV2DecMp4, []byte(iFileMp4Str))
			return
		}
	}

	log.Println("ffmpeg decoding ", iFileMp4Str)
	runFfmpeg := exec.Command("ffmpeg", "-y", "-i", iFileMp4Str, "-b:v", "0", "-crf", "30", iFileWebmStr)
	_ = runFfmpeg.Run()

	_ = db.Hset(model.TbnV2DecMp4, []byte(iFileMp4Str), []byte("1"))
	log.Println("ffmpeg decode done ", iFileWebmStr)
}
