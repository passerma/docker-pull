package main

import (
	"docker-pull/src/controller"
	"docker-pull/src/util"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"
)

func initClearDist() {
	c := cron.New()
	timer, err := strconv.Atoi(util.AllConf["clearDistTime"])
	if err != nil {
		timer = 6
	}
	c.AddFunc("0 */1 * * *", func() {
		files, errDir := os.ReadDir(util.DistPath)
		if errDir != nil {
			return
		}
		for _, file := range files {
			if !file.IsDir() {
				fileName := file.Name()
				filePath := path.Join(util.DistPath, fileName)
				fileinfo, err := os.Stat(filePath)
				if err == nil && filepath.Ext(filePath) == ".tar" {
					modTime := fileinfo.ModTime()
					duration := time.Since(modTime)
					if duration.Hours() > float64(timer) {
						fmt.Println("删除过期文件：", filePath)
						os.Remove(filePath)
					}
				}
			}
		}
	})
	c.Start()
}

func main() {
	if util.AllConf["registry"] == "" {
		panic("registry is empty")
	}

	if _, err := os.Stat(util.DistPath); os.IsNotExist(err) {
		os.Mkdir(util.DistPath, 0755)
	}

	gin.SetMode(gin.ReleaseMode)

	// 禁用控制台颜色，将日志写入文件时不需要控制台颜色。
	gin.DisableConsoleColor()
	// 记录到文件。
	f, _ := os.Create(path.Join(util.LogPath, "gin.log"))
	gin.DefaultWriter = io.MultiWriter(f)

	router := gin.Default()
	staticFile := path.Join(util.StaticPath)

	router.GET("/", func(c *gin.Context) {
		c.File(staticFile + "/" + "index.html")
	})
	router.GET("/:fileName", func(c *gin.Context) {
		fileName := c.Param("fileName")
		if fileName == "" {
			fileName = "index.html"
		}
		c.File(staticFile + "/" + fileName)
	})
	router.GET("/api/pull", func(c *gin.Context) {
		controller.Pull(c)
	})
	router.Static("/dist/down", util.DistPath)

	fmt.Println("registry: ", util.AllConf["registry"])
	initClearDist()

	router.Run(":" + util.AllConf["port"])
}
