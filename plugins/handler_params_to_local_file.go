package plugins

import (
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	collect "github.com/collect-ui/collect/src/collect/filters"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

type ToLocalFile struct {
	templateService.BaseHandler
}

func removeDotSlashPrefix(path string) string {
	// 检查路径是否以 "./" 开头
	if strings.HasPrefix(path, "./") {
		// 去掉 "./" 前缀
		return path[2:]
	}
	// 如果路径不以 "./" 开头，直接返回原路径
	return path
}

// 缩短文件名并添加时间戳后缀
func shortenFileName(filename string, maxLength int) string {
	// 获取文件扩展名
	ext := filepath.Ext(filename)
	// 获取文件名（不包括扩展名）
	name := strings.TrimSuffix(filename, ext)

	// 生成时间戳后缀
	timestamp := time.Now().Format("15-04-05") // 时分秒格式
	suffix := "_" + timestamp

	// 检查文件名是否包含中文
	if containsChinese(name) {
		// 如果包含中文，直接添加时间戳后缀，不截取文件名
		return name + suffix + ext
	}

	// 如果不包含中文，计算文件名最大允许长度（减去时间戳后缀和扩展名的长度）
	maxNameLength := maxLength - len(suffix) - len(ext)

	// 如果文件名长度超过最大允许长度，截取文件名
	if len(name) > maxNameLength {
		name = name[:maxNameLength]
	}

	// 拼接文件名、时间戳后缀和扩展名
	return name + suffix + ext
}

// 检查字符串是否包含中文
func containsChinese(s string) bool {
	for _, r := range s {
		if utf8.RuneLen(r) > 1 {
			return true
		}
	}
	return false
}

func (si *ToLocalFile) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	file := ts.File

	if file == nil {
		return common.NotOk("上传文件不能为空")
	}

	targetDir, err := getTargetPath(handlerParam.Field, params)
	if err != nil {
		return common.NotOk(err.Error())
	}
	// 检查目标文件夹是否存在
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		// 目标文件夹不存在，创建它
		err = os.MkdirAll(targetDir, os.ModePerm)
		if err != nil {
			return common.NotOk(fmt.Sprintf("创建文件夹失败: %s", err.Error()))
		}
	}
	shortenedFileName := shortenFileName(ts.FileHeader.Filename, 50)
	realFilePath := targetDir + shortenedFileName
	httpFilePath := calculateHttpPath(realFilePath, collect.GetKey("file_prefix"), collect.GetKey("local_file_dir"))

	// 创建目标文件
	out, err := os.Create(realFilePath)
	if err != nil {
		return common.NotOk(fmt.Sprintf("创建文件失败: %s", err.Error()))
	}
	defer out.Close()

	// 将文件内容复制到目标文件
	_, err = io.Copy(out, file)
	if err != nil {
		return common.NotOk(fmt.Sprintf("写入文件失败: %s", err.Error()))
	}
	// 获取文件大小
	fileInfo, err := os.Stat(realFilePath)
	if err != nil {
		return common.NotOk(fmt.Sprintf("获取文件大小失败: %s", err.Error()))
	}
	fileSize := fileInfo.Size()

	// 转换文件大小为易读格式
	fileSizeReadable := formatFileSize(fileSize)

	fileData := make(map[string]interface{})
	fileData["path"] = httpFilePath
	fileData["real_path"] = realFilePath
	fileData["size"] = fileSizeReadable
	fileData["filename"] = shortenedFileName
	fileData["filetype"] = getFileType(shortenedFileName)
	r := common.Ok(fileData, "处理参数成功")
	return r
}

// getFileType 根据文件扩展名返回文件类型
func getFileType(shortenedFileName string) string {
	// 找到最后一个 '.' 的位置
	lastDotIndex := strings.LastIndex(shortenedFileName, ".")
	if lastDotIndex == -1 {
		// 如果没有找到 '.'，返回 "unknown"
		return "unknown"
	}

	// 提取文件扩展名
	fileExtension := shortenedFileName[lastDotIndex+1:]

	// 根据文件扩展名判断文件类型
	switch strings.ToLower(fileExtension) {
	case "jpg", "jpeg", "png", "gif", "svg", "bmp", "tiff", "tif", "webp", "ico", "heic", "heif":
		return "image"
	case "mp4", "avi", "mov", "mkv", "flv", "wmv", "webm":
		return "video"
	case "mp3", "wav", "ogg", "flac", "aac", "m4a":
		return "audio"
	case "txt", "md", "rtf", "log", "csv", "tsv":
		return "text"
	case "pdf":
		return "pdf"
	case "doc", "docx":
		return "word"
	case "xls", "xlsx":
		return "excel"
	case "ppt", "pptx":
		return "powerpoint"
	default:
		return fileExtension
	}
}

func formatFileSize(size int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func calculateHttpPath(realFilePath, filePrefix, localFileDir string) string {
	// 去掉本地文件路径前缀
	if strings.HasPrefix(realFilePath, localFileDir) {
		realFilePath = realFilePath[len(localFileDir):]
	}
	// 将路径中的反斜杠转换为正斜杠
	realFilePath = filepath.ToSlash(realFilePath)

	// 拼接文件前缀
	httpPath := filePrefix + realFilePath
	// 确保路径以 "/" 开头
	if !strings.HasPrefix(httpPath, "/") {
		httpPath = "/" + httpPath
	}
	return httpPath
}
