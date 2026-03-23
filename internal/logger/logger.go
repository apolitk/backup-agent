package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

var levelNames = [...]string{"DEBUG", "INFO", "WARN", "ERROR"}

type Logger struct {
	level  Level
	logger *log.Logger
}

func New(level string) *Logger {
	levelMap := map[string]Level{
		"debug": Debug,
		"info":  Info,
		"warn":  Warn,
		"error": Error,
	}
	l, ok := levelMap[level]
	if !ok {
		l = Info
	}
	return &Logger{
		level:  l,
		logger: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	prefix := fmt.Sprintf("[%s] %s: ", timestamp, levelNames[level])
	l.logger.Printf(prefix+msg, args...)
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(Debug, msg, args...)
}

func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(Info, msg, args...)
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(Warn, msg, args...)
}

func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(Error, msg, args...)
}