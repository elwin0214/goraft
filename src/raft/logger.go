package raft

import (
	"log"
	"os"
)

type Logger struct {
	Trace *log.Logger
	Debug *log.Logger
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
}

func (logger *Logger) Init() {
	logger.Trace = log.New(os.Stdout, "[TRACE]", log.LstdFlags)
	logger.Debug = log.New(os.Stdout, "[DEBUG]", log.LstdFlags)
	logger.Info = log.New(os.Stdout, "[INFO ]", log.LstdFlags)
	logger.Warn = log.New(os.Stdout, "[WARN ]", log.LstdFlags)
	logger.Error = log.New(os.Stdout, "[ERROR]", log.LstdFlags)
}
