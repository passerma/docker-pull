package controller

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/robfig/cron"

	"github.com/gin-gonic/gin"
	goutils "github.com/typa01/go-utils"
)

var ifChannelsMapInit = false

var channelsMap = map[chan string]string{}

func initChannelsMap() {
	channelsMap = make(map[chan string]string)
	c := cron.New()
	c.AddFunc("*/30 * * * * *", func() {
		for k := range channelsMap {
			k <- "event:live\ndata: live\n\n"
		}
	})
	c.Start()
}

func AddChannel(channel chan string, uid string) {
	if !ifChannelsMapInit {
		initChannelsMap()
		ifChannelsMapInit = true
	}
	channelsMap[channel] = uid
	fmt.Println("建立sse连接成功", uid)
}

func sendInfoMsg(uid, msg string) {
	for k, v := range channelsMap {
		if uid == v {
			k <- fmt.Sprintf("event:info\ndata: %s\n\n", msg)
		}
	}
}

func sendErrMsg(uid, msg string) {
	for k, v := range channelsMap {
		if uid == v {
			k <- fmt.Sprintf("event:err\ndata: %s\n\n", msg)
			k <- "close"
		}
	}
}

func sendSusMsg(uid, msg string) {
	for k, v := range channelsMap {
		if uid == v {
			k <- fmt.Sprintf("event:sus\ndata: %s\n\n", msg)
			k <- "close"
		}
	}
}

func Pull(ctx *gin.Context) {
	uid := goutils.GUID()
	channel := make(chan string)
	img := ctx.Query("img")
	tag := ctx.Query("tag")
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	w := ctx.Writer
	flusher, _ := w.(http.Flusher)
	closeNotify := ctx.Request.Context().Done()
	AddChannel(channel, uid)
	go func() {
		<-closeNotify
		close(channel)
		delete(channelsMap, channel)
		fmt.Println("sse连接断开", uid)
	}()
	ok := true
	if img == "" || tag == "" {
		ok = false
	}
	// 校验镜像名
	pattern := "^[a-zA-Z0-9/._-]+$"
	regex := regexp.MustCompile(pattern)
	imgMatch := regex.MatchString(img)
	// 校验tag
	pattern = "^[a-zA-Z0-9._-]+$"
	regex = regexp.MustCompile(pattern)
	tagMatch := regex.MatchString(tag)
	if !imgMatch || !tagMatch {
		ok = false
	}
	if ok {
		fmt.Fprintf(w, "event:start\ndata: %s\n\n", "ok")
		flusher.Flush()
		go DockerPull(uid, img, tag)
		for msg := range channel {
			if msg == "close" {
				break
			} else {
				w.WriteString(msg)
				flusher.Flush()
			}
		}
	} else {
		fmt.Fprintf(w, "event:err\ndata: %s\n\n", "参数错误")
		flusher.Flush()
	}
}
