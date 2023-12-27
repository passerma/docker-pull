package util

import (
	"io"
	"os"
	"path"

	"gopkg.in/yaml.v2"
)

var Pwd, _ = os.Getwd()
var StaticPath = ""
var DistPath = ""
var LogPath = ""
var AllConf map[string]string

func init() {
	dir, _ := os.Getwd()
	file, _ := os.Open(dir + "/app.yml")
	bytes, _ := io.ReadAll(file)
	err := yaml.Unmarshal(bytes, &AllConf)
	if err != nil {
		panic(err)
	}
	StaticPath = path.Join(Pwd, AllConf["staticDir"])
	DistPath = path.Join(Pwd, AllConf["distDir"])
	LogPath = path.Join(Pwd, AllConf["logDir"])
}
