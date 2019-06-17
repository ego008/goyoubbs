package util

import (
	"regexp"
)

var (
	nicknameRegexp    = regexp.MustCompile(`^[a-z0-9A-Z\p{Han}]+(_[a-z0-9A-Z\p{Han}]+)*$`)
	usernameRegexp    = regexp.MustCompile(`^[a-zA-Z][a-z0-9A-Z]*(_[a-z0-9A-Z]+)*$`)
	regUserNameRegexp = regexp.MustCompile(`[^a-z0-9A-Z\p{Han}]+`)
	mailRegexp        = regexp.MustCompile(`^[a-zA-Z][a-z0-9A-Z]*(_[a-z0-9A-Z]+)*$`)
)

func IsNickname(str string) bool {
	if len(str) == 0 {
		return false
	}
	return nicknameRegexp.MatchString(str)
}

func IsUserName(str string) bool {
	if len(str) == 0 {
		return false
	}
	return nicknameRegexp.MatchString(str) // 支持中文
	//return usernameRegexp.MatchString(str)
}

func IsMail(str string) bool {
	if len(str) < 6 { // x@x.xx
		return false
	}
	return mailRegexp.MatchString(str)
}

func RemoveCharacter(str string) string {
	return regUserNameRegexp.ReplaceAllString(str, "")
}
