package cronjob

import (
	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"

	"goyoubbs/model"
)

func loadDb2Mc(db *mdb.DB) {
	_ = db.View(func(tx *bbolt.Tx) error {
		model.ConfLoad2MC(db, tx)
		model.UpdateBadBotName(db, tx)
		model.UpdateBadIpPrefix(db, tx)
		model.UpdateAllowIpPrefix(db, tx)
		return nil
	})

}
