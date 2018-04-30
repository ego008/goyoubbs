/*
MIT License

Copyright (c) 2017

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

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
	return usernameRegexp.MatchString(str)
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
