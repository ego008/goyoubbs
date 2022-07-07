package util

import "os"

func AutoCreateDir(dir string) (err error) {
	if _, err = os.Stat(dir); err != nil {
		return os.MkdirAll(dir, 0711)
	}
	return
}
