package utils

import (
	"os"
	"path/filepath"
)

var errLogFile *os.File

func ErrorWrite(logPath, content string) {
	var err error
	err = os.MkdirAll(
		filepath.Dir(logPath),
		0755,
	)
	if err != nil {
		panic(err)
	}
	if errLogFile == nil {
		errLogFile, err = os.OpenFile(logPath+"error.txt",
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644,
		)
		if err != nil {
			panic(err)
		}
	}
	// defer errLogFile.Close()
	_, err = errLogFile.WriteString(content + "\n")
	if err != nil {
		panic(err)
	}
}
