package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/xiao-ren-wu/restgo"
)

type Layer struct {
	MediaType string   `json:"mediaType"`
	Size      float64  `json:"size"`
	Digest    string   `json:"digest"`
	Urls      []string `json:"urls"`
}

type Layers []Layer

type Config struct {
	MediaType string  `json:"mediaType"`
	Size      float64 `json:"size"`
	Digest    string  `json:"digest"`
}

type ManifestsData struct {
	Layers Layers `json:"layers"`
	Config Config `json:"config"`
}

func getToken(repository string) (token string, err error) {
	var resp restgo.Response
	var data map[string]string
	url := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", repository)
	resp, err = restgo.NewRestGoBuilder().Send(restgo.GET, url)
	if err == nil {
		resp.BodyUnmarshal(&data)
		token = data["token"]
	} else {
		fmt.Println("get token err: ", err.Error())
	}
	return
}

func GetFsLayers(registry, repository, tag string) (data ManifestsData, err error) {
	var res *http.Response
	var req *http.Request
	var body []byte
	var token = ""
	if tag == "" {
		tag = "latest"
	}
	client := &http.Client{}
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repository, tag)
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("get fsLayers err: ", err.Error())
		return
	}
	token, err = getToken(repository)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	res, err = client.Do(req)
	if err != nil {
		fmt.Println("get fsLayers err: ", err.Error())
		return
	}
	defer res.Body.Close()
	body, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("get fsLayers err: ", err.Error())
		return
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println("get fsLayers err: ", res.StatusCode)
		err = errors.New("get fsLayers err: " + strconv.Itoa(res.StatusCode))
		return
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("get fsLayers err: ", err.Error())
		return
	}
	if data.Config.Digest == "" {
		err = errors.New("get fsLayers err: " + "Digest 不存在")
		return
	}
	return data, nil
}

func GetDockerConfig(registry, repository, digest, dir string) (data map[string]interface{}, err error) {
	var res *http.Response
	var req *http.Request
	var body []byte
	client := &http.Client{}
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, repository, digest)
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("GetDockerConfig err: ", err.Error())
		return
	}
	token := ""
	token, err = getToken(repository)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	res, err = client.Do(req)
	if err != nil {
		fmt.Println("GetDockerConfig err: ", err.Error())
		return
	}
	defer res.Body.Close()
	body, err = io.ReadAll(res.Body)
	if err != nil {
		return
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println("GetDockerConfig err: ", res.StatusCode)
		err = errors.New("GetDockerConfig err: " + strconv.Itoa(res.StatusCode))
		return
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("GetDockerConfig err: ", err.Error())
		return
	}
	os.Mkdir(dir, 0755)
	// 构建文件路径
	filePath := dir + "/" + digest[7:] + ".json"
	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("GetDockerConfig err: ", err.Error())
		return
	}
	defer file.Close()
	// 将内容写入文件
	formattedJSON, _ := json.MarshalIndent(data, "", "  ")
	_, err = file.Write(formattedJSON)
	if err != nil {
		fmt.Println("GetDockerConfig err: ", err.Error())
		return
	}
	return data, nil
}

func DownloadFile(registry, imgValue, ublob, fileName string, layer Layer) error {
	var req *http.Request
	var resp *http.Response
	client := &http.Client{}
	out, err := os.Create(fileName)
	if err != nil {
		fmt.Println("DownloadFile err: ", err.Error())
		return err
	}
	defer out.Close()
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, imgValue, ublob)
	req, _ = http.NewRequest("GET", url, nil)
	token := ""
	token, err = getToken(imgValue)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println("DownloadFile err: ", err.Error())
		return err
	}
	if resp.StatusCode != http.StatusOK {
		if len(layer.Urls) == 0 {
			fmt.Println("DownloadFile err: ", "下载文件失败")
			return errors.New("DownloadFile err: 下载文件失败")
		} else {
			req, _ = http.NewRequest("GET", layer.Urls[0], nil)
			req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
			resp, err = client.Do(req)
			if err != nil {
				fmt.Println("DownloadFile err: ", err.Error())
				return err
			}
			if resp.StatusCode != http.StatusOK {
				fmt.Println("DownloadFile err: ", "下载文件失败")
				return errors.New("DownloadFile err: 下载文件失败")
			}
		}
	}

	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("DownloadFile err: ", err.Error())
		return err
	}
	return nil
}
