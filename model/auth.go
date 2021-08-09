package model

type AuthInfo struct {
	Uid    uint64 `json:"uid"`
	Name   string `json:"name"`
	Openid string `json:"openid"`
}

type AuthProfileInfo struct {
	LoginBy string // qq|weibo|github
	OpenId  string
	Name    string
	Avatar  string
	Agent   string
	Url     string
	About   string
}

type AvatarTask struct {
	Uid      uint64
	Name     string
	Avatar   string
	SavePath string
	Agent    string
	Try      int
}
