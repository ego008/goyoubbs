package model

import "sync/atomic"

var (
	BadIpPrefixLst   = NewConStrSlice() // 17.121
	AllowIpPrefixLst = NewConStrSlice() // Manual input, select from all ipInfo list
	BadBotNameMap    atomic.Value       //map[string]interface{}{} // key: string, value: name. DataForSeoBot,SeznamBot,GrapeshotCrawler,
)

func init() {
	BadBotNameMap.Store(Map{})
}
