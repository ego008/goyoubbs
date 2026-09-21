package model

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"
)

type EmailInfo struct {
	Key     uint64
	ToEmail string
	Subject string
	Body    string
}

func EmailInfoUpdate(db *mdb.DB, tx *bbolt.Tx, obj EmailInfo) {
	jb, err := json.Marshal(obj)
	if err != nil {
		return
	}
	_ = db.HSet(tx, "mail_queue", mdb.I2b(obj.Key), jb)
}
