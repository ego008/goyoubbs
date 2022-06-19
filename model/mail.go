package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
)

type EmailInfo struct {
	Key     uint64
	ToEmail string
	Subject string
	Body    string
}

func EmailInfoUpdate(db *sdb.DB, obj EmailInfo) {
	jb, err := json.Marshal(obj)
	if err != nil {
		return
	}
	_ = db.Hset("mail_queue", sdb.I2b(obj.Key), jb)
}
