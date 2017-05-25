package main

import (
	"fmt"
	"io"
	Log "log"
	"os"
)

var log *logManage

type logManage struct {
	file   *os.File
	writes map[string]io.Writer
	Log.Logger
}

func (this *logManage) Write(p []byte) (int, error) {
	fmt.Print(string(p))
	for _, write := range this.writes {
		write.Write(p)
	}
	return this.file.Write(p)
}

func logInit() {
	logm := logManage{}
	file, err := os.OpenFile("chat_group_push.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		panic("open log file error")
	}
	logm.file = file
	logger := Log.New(&logm, "", Log.LstdFlags)
	logm.Logger = *logger
	log = &logm
}
