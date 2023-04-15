package util

import (
	"regexp"
)

var (
	nicknameRegexp    = regexp.MustCompile(`^[a-z0-9A-Z\p{Han}]+(_[a-z0-9A-Z\p{Han}]+)*$`)
	regUserNameRegexp = regexp.MustCompile(`[^a-z0-9A-Z\p{Han}]+`)
)

func IsNickname(str string) bool {
	if len(str) == 0 {
		return false
	}
	return nicknameRegexp.MatchString(str)
}

func RemoveCharacter(str string) string {
	return regUserNameRegexp.ReplaceAllString(str, "")
}
