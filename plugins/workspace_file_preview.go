package plugins

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	collectUtils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	defaultWorkspacePreviewTokenTTL = 2 * time.Hour
	maxWorkspacePreviewTokenTTL     = 24 * time.Hour
)

type workspacePreviewTokenRequest struct {
	ProjectCode string `json:"project_code"`
	Path        string `json:"path"`
	TTLSeconds  int64  `json:"ttl_seconds"`
}

type workspacePreviewAccessPayload struct {
	FileID      string `json:"file_id"`
	ProjectCode string `json:"project_code"`
	Path        string `json:"path"`
	UserID      string `json:"user_id"`
	Kind        string `json:"kind"`
	Exp         int64  `json:"exp"`
}

type workspacePreviewRemoteFileContext struct {
	ProjectRoot string
	TargetPath  string
	SSHClient   *ssh.Client
	SFTPClient  *sftp.Client
}

func RegisterWorkspaceFilePreviewRoutes(r gin.IRoutes) {
	r.POST("/workspace/file-preview/token", handleWorkspaceFilePreviewToken)
	r.GET("/workspace/file-preview/:file_id", handleWorkspaceFilePreview)
	r.HEAD("/workspace/file-preview/:file_id", handleWorkspaceFilePreview)
}

func handleWorkspaceFilePreviewToken(c *gin.Context) {
	userID := currentSessionUserID(c)
	if strings.EqualFold(collectUtils.GetAppKey("must_login"), "true") && userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "msg": "请先登录"})
		return
	}

	var req workspacePreviewTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": err.Error()})
		return
	}
	projectCode := strings.TrimSpace(req.ProjectCode)
	rawPath := strings.TrimSpace(req.Path)
	if projectCode == "" || rawPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "project_code/path不能为空"})
		return
	}

	gormDB := (&templateService.BaseHandler{}).GetGormDb()
	if gormDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "数据库未初始化"})
		return
	}
	project, err := getWorkspaceProject(gormDB, projectCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": err.Error()})
		return
	}
	projectRoot, err := normalizeWorkspaceProjectRoot(valueString(project.ProjectDir))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": err.Error()})
		return
	}
	targetPath, err := normalizeWorkspaceTargetPath(projectRoot, rawPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": err.Error()})
		return
	}
	if !isPDFTargetPath(targetPath) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "当前预览链接仅支持pdf文件"})
		return
	}

	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = defaultWorkspacePreviewTokenTTL
	}
	if ttl > maxWorkspacePreviewTokenTTL {
		ttl = maxWorkspacePreviewTokenTTL
	}
	fileID := makeWorkspacePreviewFileID(projectCode, targetPath)
	payload := workspacePreviewAccessPayload{
		FileID:      fileID,
		ProjectCode: projectCode,
		Path:        targetPath,
		UserID:      userID,
		Kind:        "pdf",
		Exp:         time.Now().Add(ttl).Unix(),
	}
	accessToken, err := signWorkspacePreviewAccessToken(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": err.Error()})
		return
	}
	previewURL := requestBaseURL(c) + "/workspace/file-preview/" + fileID + "?access_token=" + url.QueryEscape(accessToken)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "生成文件预览令牌成功",
		"data": gin.H{
			"file_id":          fileID,
			"access_token":     accessToken,
			"access_token_ttl": int64(ttl.Seconds()),
			"expires_at":       payload.Exp,
			"preview_url":      previewURL,
			"mime":             "application/pdf",
		},
	})
}

func handleWorkspaceFilePreview(c *gin.Context) {
	fileID := strings.TrimSpace(c.Param("file_id"))
	payload, err := parseWorkspacePreviewAccessToken(workspacePreviewAccessToken(c))
	if err != nil {
		workspacePreviewError(c, http.StatusUnauthorized, err)
		return
	}
	if payload.FileID == "" || !hmac.Equal([]byte(payload.FileID), []byte(fileID)) {
		workspacePreviewError(c, http.StatusUnauthorized, fmt.Errorf("file_id与access_token不匹配"))
		return
	}
	if payload.Kind != "pdf" || !isPDFTargetPath(payload.Path) {
		workspacePreviewError(c, http.StatusBadRequest, fmt.Errorf("当前预览链接仅支持pdf文件"))
		return
	}

	fileCtx, err := openWorkspacePreviewRemoteFile(payload.ProjectCode, payload.Path)
	if err != nil {
		workspacePreviewError(c, http.StatusBadRequest, err)
		return
	}
	defer fileCtx.Close()
	if err := ensureRemotePathChainSafe(fileCtx.SFTPClient, fileCtx.ProjectRoot, fileCtx.TargetPath, false); err != nil {
		workspacePreviewError(c, http.StatusBadRequest, err)
		return
	}

	srcFile, err := fileCtx.SFTPClient.Open(fileCtx.TargetPath)
	if err != nil {
		workspacePreviewError(c, http.StatusNotFound, err)
		return
	}
	defer srcFile.Close()
	info, err := srcFile.Stat()
	if err != nil {
		workspacePreviewError(c, http.StatusNotFound, err)
		return
	}
	if info.IsDir() {
		workspacePreviewError(c, http.StatusBadRequest, fmt.Errorf("目录不支持预览"))
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", contentDispositionInline(path.Base(fileCtx.TargetPath)))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "private, max-age=0, must-revalidate")
	http.ServeContent(c.Writer, c.Request, path.Base(fileCtx.TargetPath), info.ModTime(), srcFile)
	c.Abort()
}

func workspacePreviewAccessToken(c *gin.Context) string {
	if token := strings.TrimSpace(c.Query("access_token")); token != "" {
		return token
	}
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func signWorkspacePreviewAccessToken(payload workspacePreviewAccessPayload) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	bodyText := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, workspacePreviewSecret())
	mac.Write([]byte(bodyText))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return bodyText + "." + signature, nil
}

func parseWorkspacePreviewAccessToken(raw string) (*workspacePreviewAccessPayload, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, fmt.Errorf("access_token不能为空")
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("access_token格式不正确")
	}
	mac := hmac.New(sha256.New, workspacePreviewSecret())
	mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(expected, actual) {
		return nil, fmt.Errorf("access_token签名无效")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("access_token载荷无效")
	}
	var payload workspacePreviewAccessPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Exp <= time.Now().Unix() {
		return nil, fmt.Errorf("access_token已过期")
	}
	return &payload, nil
}

func workspacePreviewSecret() []byte {
	if secret := strings.TrimSpace(collectUtils.GetAppKey("workspace_preview_secret")); secret != "" {
		return []byte(secret)
	}
	if secret := strings.TrimSpace(collectUtils.GetAppKey("company_key")); secret != "" {
		return []byte(secret)
	}
	return []byte("moon-workspace-preview-dev-secret")
}

func makeWorkspacePreviewFileID(projectCode, targetPath string) string {
	sum := sha256.Sum256([]byte("workspace-preview\x00" + strings.TrimSpace(projectCode) + "\x00" + strings.TrimSpace(targetPath)))
	return hex.EncodeToString(sum[:])
}

func openWorkspacePreviewRemoteFile(projectCode, rawPath string) (*workspacePreviewRemoteFileContext, error) {
	gormDB := (&templateService.BaseHandler{}).GetGormDb()
	if gormDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	project, err := getWorkspaceProject(gormDB, projectCode)
	if err != nil {
		return nil, err
	}
	projectRoot, err := normalizeWorkspaceProjectRoot(valueString(project.ProjectDir))
	if err != nil {
		return nil, err
	}
	targetPath, err := normalizeWorkspaceTargetPath(projectRoot, rawPath)
	if err != nil {
		return nil, err
	}
	if !isPDFTargetPath(targetPath) {
		return nil, fmt.Errorf("当前预览链接仅支持pdf文件")
	}
	serverUserID := strings.TrimSpace(valueString(project.ServerOsUsersID))
	if serverUserID == "" {
		return nil, fmt.Errorf("项目未配置服务器用户")
	}
	serverUser, err := getServerUser(gormDB, serverUserID)
	if err != nil {
		return nil, err
	}
	serverID := strings.TrimSpace(valueString(serverUser.ServerID))
	if serverID == "" {
		return nil, fmt.Errorf("服务器用户缺少服务器ID")
	}
	serverInstance, err := getServerInstance(gormDB, serverID)
	if err != nil {
		return nil, err
	}
	serverIP := strings.TrimSpace(serverInstance.ServerIP)
	if serverIP == "" {
		return nil, fmt.Errorf("服务器IP不能为空")
	}
	userName := strings.TrimSpace(serverUser.UserName)
	if userName == "" {
		return nil, fmt.Errorf("服务器用户名不能为空")
	}
	password, err := decryptServerPassword(valueString(serverUser.UserPwd))
	if err != nil {
		return nil, fmt.Errorf("服务器密码解密失败: %w", err)
	}
	port := strings.TrimSpace(serverInstance.ServerPort)
	if port == "" {
		port = "22"
	}
	sshClient, err := dialSSH(serverIP, port, userName, password)
	if err != nil {
		return nil, err
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, err
	}
	return &workspacePreviewRemoteFileContext{
		ProjectRoot: projectRoot,
		TargetPath:  targetPath,
		SSHClient:   sshClient,
		SFTPClient:  sftpClient,
	}, nil
}

func (c *workspacePreviewRemoteFileContext) Close() {
	if c == nil {
		return
	}
	if c.SFTPClient != nil {
		_ = c.SFTPClient.Close()
	}
	if c.SSHClient != nil {
		_ = c.SSHClient.Close()
	}
}

func isPDFTargetPath(targetPath string) bool {
	return strings.EqualFold(filepath.Ext(targetPath), ".pdf")
}

func contentDispositionInline(fileName string) string {
	fallbackName := asciiHeaderFileName(fileName)
	quoted := strings.ReplaceAll(fallbackName, "\\", "\\\\")
	quoted = strings.ReplaceAll(quoted, `"`, `\"`)
	return fmt.Sprintf("inline; filename=\"%s\"; filename*=UTF-8''%s", quoted, url.PathEscape(fileName))
}

func asciiHeaderFileName(fileName string) string {
	var builder strings.Builder
	for _, r := range fileName {
		if r < 32 || r > 126 || r == '"' || r == '\\' {
			builder.WriteByte('_')
			continue
		}
		builder.WriteRune(r)
	}
	value := strings.TrimSpace(builder.String())
	if value == "" || value == "." || value == ".." {
		return "preview.pdf"
	}
	return value
}

func workspacePreviewError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"success": false,
		"code":    strconv.Itoa(status),
		"msg":     err.Error(),
	})
}

func currentSessionUserID(c *gin.Context) string {
	key := strings.TrimSpace(collectUtils.GetAppKey("user_id_key"))
	if key == "" {
		key = "user_id"
	}
	session := sessions.Default(c)
	value := session.Get(key)
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func requestBaseURL(c *gin.Context) string {
	scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := c.Request.Host
	if forwardedHost := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	return scheme + "://" + host
}

func valueString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
