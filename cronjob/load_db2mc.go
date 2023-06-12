package cronjob

import (
	"github.com/ego008/sdb"
	"goyoubbs/model"
)

func loadDb2Mc(db *sdb.DB) {
	model.ConfLoad2MC(db)
	model.UpdateBadBotName(db)
	model.UpdateBadIpPrefix(db)
	model.UpdateAllowIpPrefix(db)
}
