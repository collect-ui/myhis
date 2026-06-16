package plugins

import (
	"archive/zip"
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type RenderDoc struct {
	templateService.BaseHandler
}

// 替换document.xml中的变量
func replaceVariables(content string, variables map[string]interface{}) string {
	for key, value := range variables {
		placeholder := fmt.Sprintf("\\${%s}", key)
		//fmt.Println(placeholder)
		re := regexp.MustCompile(placeholder)
		content = re.ReplaceAllString(content, gocast.ToString(value))

	}
	return content
}

// 解压ZIP文件
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

// 压缩文件夹为ZIP文件
func zipFolder(source, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		// 跳过根目录
		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// 设置ZIP文件中的路径
		header.Name = relPath
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

func (si *RenderDoc) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	arr, _ := utils.RenderVarToArrMap(handlerParam.Foreach, params)
	detail, _ := utils.RenderVarToMap(handlerParam.Field, params)
	localFileDir := utils.GetAppKey("local_file_dir")
	parts := strings.Split(localFileDir, "/")
	localFileDirParent := strings.Join(parts[:len(parts)-1], "/")
	for index, v := range arr {
		tempDir := gocast.ToString("tb_project_id") + "_" + gocast.ToString(index+1)
		path := v["value"].(string)
		srcDocx := localFileDirParent + path
		ext := filepath.Ext(path) // 返回 ".docx"
		// 去掉扩展名
		base := strings.TrimSuffix(srcDocx, ext) // 去掉 ".docx"
		// 在文件名末尾添加 "_output"
		now := time.Now()

		// 格式化时间为时分秒字符串
		timeStr := now.Format("_15_04_05")
		suffix := timeStr + "_output"
		destDocx := base + suffix + ext
		baseHttp := strings.TrimSuffix(path, ext) // 去掉 ".docx"
		httpPath := baseHttp + suffix + ext
		v["link_value"] = httpPath
		// 解压ZIP文件
		err := unzip(srcDocx, tempDir)
		if err != nil {

			continue
		}
		// 解压ZIP文件
		err = unzip(srcDocx, tempDir)
		if err != nil {

			return common.NotOk(err.Error())
		}

		// 读取document.xml
		documentPath := filepath.Join(tempDir, "word", "document.xml")
		content, err := ioutil.ReadFile(documentPath)
		if err != nil {
			return common.NotOk(err.Error())
		}

		// 替换变量
		newContent := replaceVariables(string(content), detail)

		// 写回document.xml
		err = ioutil.WriteFile(documentPath, []byte(newContent), 0644)
		if err != nil {
			return common.NotOk(err.Error())
		}

		// 重新打包为ZIP文件
		err = zipFolder(tempDir, destDocx)
		if err != nil {
			return common.NotOk(err.Error())
		}
		// 清理临时目录
		err = os.RemoveAll(tempDir)
		if err != nil {
			return common.NotOk(err.Error())
		}

	}
	r := common.Ok(arr, "处理参数成功")
	return r
}
