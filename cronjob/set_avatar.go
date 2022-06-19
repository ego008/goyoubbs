package cronjob

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"goyoubbs/model"
	"goyoubbs/util"
	"log"
)

func SetAvatar(db *sdb.DB, s5 string) {
	rs := db.Hscan("task_to_get_avatar", nil, 1)
	if !rs.OK() {
		return
	}

	uidB := rs.Data[0]
	//log.Println("SetAvatar task_to_get_avatar", sdb.B2i(rs.Data[0]))

	taskObj := model.AvatarTask{}
	err := json.Unmarshal(rs.Data[1], &taskObj)
	if err != nil {
		log.Println("Unmarshal err ", err)
		_ = db.Hdel("task_to_get_avatar", uidB)
		return
	}

	err = FetchAvatar(db, taskObj.Uid, taskObj.Avatar, taskObj.SavePath, taskObj.Agent, s5)
	if err != nil {
		log.Println("FetchAvatar err :", err)
		err = util.GenAvatar(db, taskObj.Uid, taskObj.Name)
		if err != nil {
			log.Println("GenAvatar err", err)
			return
		}
	}

	_ = db.Hdel("task_to_get_avatar", uidB)
}
