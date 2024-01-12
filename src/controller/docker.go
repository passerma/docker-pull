package controller

import (
	"compress/gzip"
	"crypto/sha256"
	"docker-pull/src/util"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
)

func DockerPull(guid, img, tag string) {
	registry := util.AllConf["registry"]
	dir := path.Join(util.DistPath, guid)
	tarName := path.Join(util.DistPath, guid+".tar")

	imgArr := strings.Split(img, "/")
	imgValue := ""
	if len(imgArr) == 2 {
		imgValue = img
	} else {
		imgValue = "library/" + img
	}
	if tag == "" {
		tag = "latest"
	}

	fmt.Println("docker pull ", imgValue+":"+tag)
	sendInfoMsg(guid, "docker pull "+imgValue+":"+tag)

	var data util.ManifestsData
	var err error

	data, err = util.GetFsLayers(registry, imgValue, tag)

	// 阿里云镜像没获取到，再从官网获取
	if err != nil {
		registry = "registry-1.docker.io"
		data, err = util.GetFsLayers(registry, imgValue, tag)
	}

	if err == nil {
		sendInfoMsg(guid, "GetFsLayers: "+imgValue)
		if config, err := util.GetDockerConfig(registry, imgValue, data.Config.Digest, dir); err == nil {
			sendInfoMsg(guid, "Creating image structure in: "+imgValue)
			content := []map[string]interface{}{
				{
					"Config":   data.Config.Digest[7:] + ".json",
					"RepoTags": []string{},
					"Layers":   []string{},
				},
			}
			content[0]["RepoTags"] = append(content[0]["RepoTags"].([]string), imgValue+":"+tag)

			empty_json := `{"created":"1970-01-01T00:00:00Z","container_config":{"Hostname":"","Domainname":"",			"User":"","AttachStdin":false,
			"AttachStdout":false,"AttachStderr":false,"Tty":false,"OpenStdin":false, "StdinOnce":false,"Env":null,"Cmd":null,"Image":"",
			"Volumes":null,"WorkingDir":"","Entrypoint":null,"OnBuild":null,"Labels":null}}`

			parentid := ""
			fakeLayerID := ""
			for _, layer := range data.Layers {
				ublob := layer.Digest
				ublobByte := []byte(parentid + "-" + ublob)
				ublobHash := sha256.Sum256(ublobByte)
				fakeLayerID = hex.EncodeToString(ublobHash[:])
				layerdir := dir + "/" + fakeLayerID
				os.Mkdir(layerdir, 0755)

				versionFile, _ := os.Create(layerdir + "/VERSION")
				versionFile.Write([]byte("1.0"))
				versionFile.Close()

				sendInfoMsg(guid, ublob[7:19]+": Downloading...")

				if err := util.DownloadFile(registry, imgValue, ublob, layerdir+"/layer_gzip.tar", layer); err != nil {
					os.RemoveAll(dir)
					sendErrMsg(guid, err.Error())
					break
				} else {
					sendInfoMsg(guid, ublob[7:19]+": Downloaded")
					sendInfoMsg(guid, ublob[7:19]+": Extracting...")

					gzipFile, _ := os.Open(layerdir + "/layer_gzip.tar")
					defer gzipFile.Close()
					gunzipReader, _ := gzip.NewReader(gzipFile)
					defer gunzipReader.Close()
					tarFile, _ := os.Create(layerdir + "/layer.tar")
					defer tarFile.Close()
					io.Copy(tarFile, gunzipReader)

					os.Remove(layerdir + "/layer_gzip.tar")

					sendInfoMsg(guid, ublob[7:19]+": Extracted")

					content[0]["Layers"] = append(content[0]["Layers"].([]string), fakeLayerID+"/layer.tar")

					file, _ := os.Create(layerdir + "/json")
					defer file.Close()

					jsonObj := map[string]interface{}{}
					// 获取最后一层的信息
					lastLayer := data.Layers[len(data.Layers)-1]
					// 检查最后一层的摘要是否与当前层的摘要相同
					if lastLayer.Digest == layer.Digest {
						jsonObj = config
						// 删除history和rootfs字段（或rootfS字段）
						delete(jsonObj, "history")
						delete(jsonObj, "rootfs") // 尝试删除rootfs字段
						delete(jsonObj, "rootfS") // 如果rootfs字段不存在，则删除rootfS字段
					} else {
						// 其他层的JSON对象为空
						json.Unmarshal([]byte(empty_json), &jsonObj)
					}

					// 设置id和parent字段（如果存在）
					jsonObj["id"] = fakeLayerID
					if parentid != "" {
						jsonObj["parent"] = parentid
					} else {
						parentid = jsonObj["id"].(string) // 更新parentID的值
					}

					jsonMarshal, _ := json.Marshal(&jsonObj)
					// 将JSON对象序列化为字符串并写入文件
					os.WriteFile(file.Name(), jsonMarshal, 0644)
				}
			}
			file, _ := os.Create(dir + "/manifest.json")
			contentMars, _ := json.Marshal(content)
			file.Write(contentMars)
			file.Close()

			file, _ = os.Create(dir + "/repositories")
			contentMars, _ = json.Marshal(map[string]interface{}{
				imgValue: map[string]string{
					tag: fakeLayerID,
				},
			})
			file.Write(contentMars)
			file.Close()

			cmd := exec.Command("tar", "-cf", tarName, "-C", dir+"/", ".")
			cmd.Start()
			cmd.Wait()
			os.RemoveAll(dir)
			sendSusMsg(guid, "/dist/down/"+guid+".tar")
			fmt.Println("docker save ", "/dist/"+guid+".tar")
		} else {
			// 删除文件夹
			os.RemoveAll(dir)
			sendErrMsg(guid, err.Error())
		}
	} else {
		sendErrMsg(guid, err.Error())
	}
}
