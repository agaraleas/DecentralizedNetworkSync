package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

func InitLogging() {
	Log.SetFormatter(&logrus.TextFormatter{})
	Log.SetOutput(os.Stdout)
	Log.SetLevel(logrus.InfoLevel)
}

type AbstractLogger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}