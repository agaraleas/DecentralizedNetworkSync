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