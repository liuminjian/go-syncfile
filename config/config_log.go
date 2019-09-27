package config

import (
	"github.com/mattn/go-colorable"
	"github.com/sirupsen/logrus"
	"github.com/tietang/go-utils"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var formatter *prefixed.TextFormatter

var lfh *utils.LineNumLogrusHook

func InitLog(debug bool) {
	formatter = &prefixed.TextFormatter{}
	formatter.ForceColors = true
	formatter.DisableColors = false
	formatter.ForceFormatting = true
	formatter.SetColorScheme(&prefixed.ColorScheme{
		InfoLevelStyle:  "green",
		WarnLevelStyle:  "yellow",
		ErrorLevelStyle: "red",
		FatalLevelStyle: "41",
		PanicLevelStyle: "41",
		DebugLevelStyle: "blue",
		PrefixStyle:     "cyan",
		TimestampStyle:  "37",
	})
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006-01-02.15:04:05.000000"
	logrus.SetFormatter(formatter)
	logrus.SetOutput(colorable.NewColorableStdout())
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.SetReportCaller(true)
	SetLineNumLogrusHook()
}

func SetLineNumLogrusHook() {
	lfh = utils.NewLineNumLogrusHook()
	lfh.EnableFileNameLog = true
	lfh.EnableFuncNameLog = true
	logrus.AddHook(lfh)
}
