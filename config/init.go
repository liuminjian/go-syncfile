package config

import (
	"github.com/tietang/props/ini"
	"github.com/tietang/props/kvs"
)

var ConfigMgr *ini.IniFileConfigSource

func init() {
	//dir, _ := os.Getwd()
	dir := kvs.GetCurrentFilePath("config.ini", 1)
	ConfigMgr = ini.NewIniFileConfigSource(dir)
	debug := ConfigMgr.GetBoolDefault("server.debug", false)
	InitLog(debug)
}
