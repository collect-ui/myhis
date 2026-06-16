package plugins

import (
	"encoding/base64"
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
	"github.com/pkg/sftp"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

type WorkspaceFileAccessService struct {
	templateService.BaseHandler
}

func (s *WorkspaceFileAccessService) Result(template *config.Template, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	operation := strings.ToLower(strings.TrimSpace(gocast.ToString(params["operation"])))
	if operation == "" {
		return common.NotOk("operation不能为空")
	}

	projectCode := strings.TrimSpace(gocast.ToString(params["project_code"]))
	if projectCode == "" {
		return common.NotOk("项目编码不能为空")
	}

	gormDB := s.GetGormDb()
	if gormDB == nil {
		return common.NotOk("数据库未初始化")
	}

	project, err := getWorkspaceProject(gormDB, projectCode)
	if err != nil {
		return common.NotOk(err.Error())
	}
	projectRoot, err := normalizeWorkspaceProjectRoot(gocast.ToString(project.ProjectDir))
	if err != nil {
		return common.NotOk(err.Error())
	}

	serverUserID := strings.TrimSpace(gocast.ToString(project.ServerOsUsersID))
	if serverUserID == "" {
		return common.NotOk("项目未配置服务器用户")
	}

	serverUser, err := getServerUser(gormDB, serverUserID)
	if err != nil {
		return common.NotOk(err.Error())
	}
	serverID := strings.TrimSpace(gocast.ToString(serverUser.ServerID))
	if serverID == "" {
		return common.NotOk("服务器用户缺少服务器ID")
	}

	serverInstance, err := getServerInstance(gormDB, serverID)
	if err != nil {
		return common.NotOk(err.Error())
	}

	serverIP := strings.TrimSpace(serverInstance.ServerIP)
	if serverIP == "" {
		return common.NotOk("服务器IP不能为空")
	}
	userName := strings.TrimSpace(serverUser.UserName)
	if userName == "" {
		return common.NotOk("服务器用户名不能为空")
	}
	password, err := decryptServerPassword(gocast.ToString(serverUser.UserPwd))
	if err != nil {
		return common.NotOk(fmt.Sprintf("服务器密码解密失败: %s", err.Error()))
	}
	port := strings.TrimSpace(serverInstance.ServerPort)
	if port == "" {
		port = "22"
	}

	sshClient, err := dialSSH(serverIP, port, userName, password)
	if err != nil {
		return common.NotOk(err.Error())
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return common.NotOk(err.Error())
	}
	defer sftpClient.Close()

	switch operation {
	case "content":
		return s.readSingle(params, sftpClient, projectRoot)
	case "content_batch":
		return s.readBatch(params, sftpClient, projectRoot)
	case "save":
		return s.writeSingle(params, sftpClient, projectRoot)
	default:
		return common.NotOk("operation仅支持 content/content_batch/save")
	}
}

func (s *WorkspaceFileAccessService) readSingle(params map[string]interface{}, sftpClient *sftp.Client, projectRoot string) *common.Result {
	targetPath, err := normalizeWorkspaceTargetPath(projectRoot, gocast.ToString(params["path"]))
	if err != nil {
		return common.NotOk(err.Error())
	}
	if targetPath == projectRoot {
		return common.NotOk("项目根目录不支持直接读取")
	}
	if err := ensureRemotePathChainSafe(sftpClient, projectRoot, targetPath, false); err != nil {
		return common.NotOk(err.Error())
	}

	srcFile, err := sftpClient.Open(targetPath)
	if err != nil {
		return common.NotOk(err.Error())
	}
	defer srcFile.Close()

	stat, err := srcFile.Stat()
	if err != nil {
		return common.NotOk(err.Error())
	}
	if stat.IsDir() {
		return common.NotOk("目录不支持直接读取")
	}

	fileName := path.Base(targetPath)
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(fileName)), ".")
	mimeType := detectMimeByExt(ext)
	if isPdfExtension(ext) || strings.Contains(mimeType, "application/pdf") {
		if mimeType == "" {
			mimeType = "application/pdf"
		}
		result := map[string]interface{}{
			"name":           fileName,
			"path":           targetPath,
			"ext":            ext,
			"size":           stat.Size(),
			"mime":           mimeType,
			"kind":           "pdf",
			"modify_time":    utils.DateFormat(stat.ModTime(), ""),
			"create_time":    "",
			"truncated":      false,
			"content_text":   "",
			"content_base64": "",
			"preview_mode":   "stream",
		}
		return common.Ok(result, "读取文件成功")
	}

	maxBytes := gocast.ToInt64(params["max_bytes"])
	if maxBytes <= 0 {
		maxBytes = 2 * 1024 * 1024
	}
	contentBytes, err := io.ReadAll(io.LimitReader(srcFile, maxBytes+1))
	if err != nil {
		return common.NotOk(err.Error())
	}
	truncated := false
	if int64(len(contentBytes)) > maxBytes {
		truncated = true
		contentBytes = contentBytes[:maxBytes]
	}

	if mimeType == "" {
		mimeType = "application/octet-stream"
		if len(contentBytes) > 0 {
			mimeType = http.DetectContentType(contentBytes)
		}
	}
	isImage := isImageExtension(ext) || strings.HasPrefix(mimeType, "image/")
	isDocx := isDocxExtension(ext) || strings.Contains(mimeType, "wordprocessingml.document")
	isPDF := isPdfExtension(ext) || strings.Contains(mimeType, "application/pdf")
	isText := !isImage && !isDocx && !isPDF && isLikelyTextContent(contentBytes)
	kind := "binary"
	if isImage {
		kind = "image"
	} else if isDocx {
		kind = "docx"
	} else if isPDF {
		kind = "pdf"
	} else if isText {
		kind = "text"
	}

	contentText := ""
	contentBase64 := ""
	if kind == "text" {
		contentText = string(contentBytes)
	} else if kind == "image" || kind == "docx" {
		contentBase64 = base64.StdEncoding.EncodeToString(contentBytes)
	}
	if kind == "pdf" {
		truncated = false
	}

	result := map[string]interface{}{
		"name":           fileName,
		"path":           targetPath,
		"ext":            ext,
		"size":           stat.Size(),
		"mime":           mimeType,
		"kind":           kind,
		"modify_time":    utils.DateFormat(stat.ModTime(), ""),
		"create_time":    "",
		"truncated":      truncated,
		"content_text":   contentText,
		"content_base64": contentBase64,
	}
	if kind == "pdf" {
		result["preview_mode"] = "stream"
	}
	return common.Ok(result, "读取文件成功")
}

func (s *WorkspaceFileAccessService) readBatch(params map[string]interface{}, sftpClient *sftp.Client, projectRoot string) *common.Result {
	rawPaths := parsePathList(params["paths"])
	if len(rawPaths) == 0 {
		return common.NotOk("paths不能为空")
	}
	pathList, err := normalizeWorkspaceTargetPaths(projectRoot, rawPaths)
	if err != nil {
		return common.NotOk(err.Error())
	}

	maxBytes := gocast.ToInt64(params["max_bytes"])
	if maxBytes <= 0 {
		maxBytes = 2 * 1024 * 1024
	}

	items := make([]map[string]interface{}, 0, len(pathList))
	for _, targetPath := range pathList {
		item := map[string]interface{}{"path": targetPath, "ok": false, "content": "", "size": 0, "truncated": false}
		if targetPath == projectRoot {
			item["error"] = "项目根目录不支持直接读取"
			items = append(items, item)
			continue
		}
		if err := ensureRemotePathChainSafe(sftpClient, projectRoot, targetPath, false); err != nil {
			item["error"] = err.Error()
			items = append(items, item)
			continue
		}

		srcFile, openErr := sftpClient.Open(targetPath)
		if openErr != nil {
			item["error"] = openErr.Error()
			items = append(items, item)
			continue
		}

		stat, statErr := srcFile.Stat()
		if statErr != nil {
			_ = srcFile.Close()
			item["error"] = statErr.Error()
			items = append(items, item)
			continue
		}
		if stat.IsDir() {
			_ = srcFile.Close()
			item["error"] = "目录不支持直接读取"
			items = append(items, item)
			continue
		}

		contentBytes, readErr := io.ReadAll(io.LimitReader(srcFile, maxBytes+1))
		_ = srcFile.Close()
		if readErr != nil {
			item["error"] = readErr.Error()
			items = append(items, item)
			continue
		}
		truncated := false
		if int64(len(contentBytes)) > maxBytes {
			truncated = true
			contentBytes = contentBytes[:maxBytes]
		}
		item["ok"] = true
		item["content"] = string(contentBytes)
		item["size"] = len(contentBytes)
		item["truncated"] = truncated
		items = append(items, item)
	}

	return common.Ok(map[string]interface{}{"items": items, "count": len(items)}, "批量读取文件成功")
}

func (s *WorkspaceFileAccessService) writeSingle(params map[string]interface{}, sftpClient *sftp.Client, projectRoot string) *common.Result {
	targetPath, err := normalizeWorkspaceTargetPath(projectRoot, gocast.ToString(params["path"]))
	if err != nil {
		return common.NotOk(err.Error())
	}
	if targetPath == projectRoot {
		return common.NotOk("项目根目录不支持直接写入")
	}
	if err := ensureRemotePathChainSafe(sftpClient, projectRoot, targetPath, true); err != nil {
		return common.NotOk(err.Error())
	}

	content := gocast.ToString(params["content"])
	maxWriteBytes := gocast.ToInt64(params["max_write_bytes"])
	if maxWriteBytes <= 0 {
		maxWriteBytes = 5 * 1024 * 1024
	}
	if int64(len(content)) > maxWriteBytes {
		return common.NotOk(fmt.Sprintf("内容超过限制，最大 %d 字节", maxWriteBytes))
	}

	if exists, err := remotePathExists(sftpClient, targetPath); err != nil {
		return common.NotOk(err.Error())
	} else if exists {
		info, statErr := sftpClient.Stat(targetPath)
		if statErr != nil {
			return common.NotOk(statErr.Error())
		}
		if info.IsDir() {
			return common.NotOk("目录不支持直接写入")
		}
	}

	targetDir := path.Dir(targetPath)
	if targetDir != "" && targetDir != "." {
		if err := sftpClient.MkdirAll(targetDir); err != nil {
			return common.NotOk(err.Error())
		}
	}

	dstFile, err := sftpClient.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return common.NotOk(err.Error())
	}
	defer dstFile.Close()

	if _, err := dstFile.Write([]byte(content)); err != nil {
		return common.NotOk(err.Error())
	}

	return common.Ok(map[string]interface{}{
		"path":    targetPath,
		"size":    len(content),
		"success": true,
	}, "保存文件成功")
}
