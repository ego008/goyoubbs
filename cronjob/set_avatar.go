package cronjob

import (
	"bytes"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"

	"goyoubbs/model"
	"goyoubbs/util"
	"log"
)

func SetAvatar(db *mdb.DB, s5 string) {
	var uidB, uv []byte
	taskObj := model.AvatarTask{}
	_ = db.View(func(tx *bbolt.Tx) error {
		_ = db.HScanFunc(tx, "task_to_get_avatar", nil, 1, func(key, val []byte) bool {
			uidB = bytes.Clone(key)
			uv = bytes.Clone(val)
			return true
		})
		return nil
	})

	if len(uidB) == 0 || len(uv) == 0 {
		return
	}

	err := json.Unmarshal(uv, &taskObj)
	if err != nil {
		_ = db.Update(func(tx *bbolt.Tx) error {
			return db.HDel(tx, "task_to_get_avatar", uidB)
		})
		return
	}

	err = FetchAvatar(db, taskObj.Uid, taskObj.Avatar, taskObj.SavePath, taskObj.Agent, s5)
	if err != nil {
		_ = db.Update(func(tx *bbolt.Tx) error {
			err = util.GenAvatar(db, tx, taskObj.Uid, taskObj.Name)
			if err != nil {
				log.Println("GenAvatar err", err)
				return err
			}
			return nil
		})
	}

	_ = db.Update(func(tx *bbolt.Tx) error {
		return db.HDel(tx, "task_to_get_avatar", uidB)
	})
}
