package util

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
)

func AutoCreateDir(dir string) (err error) {
	if _, err = os.Stat(dir); err != nil {
		return os.MkdirAll(dir, 0711)
	}
	return
}

// CmdExists Golang check if command exists
func CmdExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// FindInPs find bin name in processes
// ex: FindInPs("ffmpeg", "ffmpeg -i")
func FindInPs(cmd, s string) (bool, string) {
	first := exec.Command("ps", "-ef")
	second := exec.Command("grep", cmd)

	reader, writer := io.Pipe()
	first.Stdout = writer
	second.Stdin = reader

	var buff bytes.Buffer
	second.Stdout = &buff

	first.Start()
	second.Start()
	first.Wait()
	writer.Close()
	second.Wait()

	out := buff.String() // convert output to string
	return strings.Contains(out, s), out
}

func HashFile(filePath string) (hashStr string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	var fileData bytes.Buffer
	if _, err = io.Copy(&fileData, file); err != nil {
		return
	}
	hashValue := Xxhash(fileData.Bytes())
	fileData.Reset()
	return TenTo62(hashValue)
}
