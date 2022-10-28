package images

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"studygolang/docker/xdocker/model"
)

func DoSearchImageRequest(keyword string) ([]string, error) {
	url := fmt.Sprintf("%s%s?keyword=%s", model.DefaultServerUrl, model.SearchUrl, keyword)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	list := make([]string, 0)
	err = json.Unmarshal(respBytes, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func FetchImageList(imageName string) ([]string, error) {
	url := fmt.Sprintf("%s%s?imagename=%s", model.DefaultServerUrl, model.ListUrl, imageName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	list := make([]string, 0)
	err = json.Unmarshal(respBytes, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func DownloadImage(imageName, tag string) (err error) {
	url := fmt.Sprintf("%s%s", model.DefaultServerUrl, model.PullUrl)

	postData := make(map[string]interface{})
	postData["imagename"] = imageName
	postData["tag"] = tag
	postBytes, err := json.Marshal(postData)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(postBytes))
	if err != nil {
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// 文件的数据长度
	length := resp.ContentLength

	fileBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if len(fileBytes) != int(length) {
		fmt.Println(len(fileBytes), length)
		return errors.New("file length is wrong")
	}

	// 返回的数据长度小于512 则说明返回的是错误信息，而不是镜像文件数据
	if length < 512 {
		return fmt.Errorf(string(fileBytes))
	}

	imageStorePath := fmt.Sprintf("%s%s@%s.tar", model.DefaultImagePath, imageName, tag)
	dstFile, err := os.OpenFile(imageStorePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		return
	}
	defer dstFile.Close()

	n, err := dstFile.Write(fileBytes)
	if err != nil {
		return
	}
	if n != len(fileBytes) {
		return errors.New("write file length is wrong")
	}

	return nil
}

func UploadImage(tarPath, imageName, tag string) (err error) {
	url := fmt.Sprintf("%s%s", model.DefaultServerUrl, model.PushUrl)

	imageFile, err := os.Open(tarPath)
	if err != nil {
		panic(err)
	}

	// 模拟客户端提交表单
	values := map[string]io.Reader {
		"file": imageFile,
		"imagename": strings.NewReader(imageName),
		"tag": strings.NewReader(tag),
	}

	// 构建multipart，然后post给服务端
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	for key, reader := range values {
		var fw io.Writer
		if x, ok := reader.(io.Closer); ok {
			defer x.Close()
		}

		if x, ok := reader.(*os.File); ok {
			// 添加文件
			if fw, err = writer.CreateFormFile(key, x.Name()); err != nil {
				return
			}
		} else {
			// 添加字符串
			if fw, err = writer.CreateFormField(key); err != nil {
				return
			}
		}

		if _, err = io.Copy(fw, reader); err != nil {
			return
		}
	}
	// close动作会在末端写入 boundary
	writer.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}

	// form-data格式，自动生成分隔符
	// 例如 Content-Type: multipart/form-data; boundary=d76.....d29
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("response code is not 200")
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if string(respBytes) != "ok" {
		return fmt.Errorf(string(respBytes))
	}

	return nil
}