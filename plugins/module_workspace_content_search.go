package plugins

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	"moon/model/devops"
)

type WorkspaceContentSearchService struct {
	templateService.BaseHandler
}

type workspaceContentSearchItem struct {
	FilePath   string `json:"file_path"`
	FileName   string `json:"file_name"`
	Path       string `json:"path"`
	LineNo     int    `json:"line_no"`
	ColumnNo   int    `json:"column_no"`
	BeforeText string `json:"before_text"`
	MatchText  string `json:"match_text"`
	AfterText  string `json:"after_text"`
}

type workspaceContentSearchSummary struct {
	Engine       string `json:"engine"`
	DurationMs   int64  `json:"duration_ms"`
	ScannedFiles int    `json:"scanned_files"`
	MatchedFiles int    `json:"matched_files"`
	HitCount     int    `json:"hit_count"`
	FailedFiles  int    `json:"failed_files"`
	Truncated    bool   `json:"truncated"`
	ScanStopped  bool   `json:"scan_stopped"`
}

func (s *WorkspaceContentSearchService) Result(template *config.Template, ts *templateService.TemplateService) *common.Result {
	startAt := time.Now()
	params := template.GetParams()

	projectCode := strings.TrimSpace(gocast.ToString(params["project_code"]))
	if projectCode == "" {
		return common.NotOk("项目编码不能为空")
	}
	keyword := strings.TrimSpace(gocast.ToString(params["keyword"]))
	if keyword == "" {
		return common.NotOk("关键字不能为空")
	}
	if utf8.RuneCountInString(keyword) < 2 {
		return common.NotOk("关键字至少2个字符")
	}

	matchCase := gocast.ToBool(params["match_case"])
	maxResults := clampInt(gocast.ToInt(params["max_results"]), 1, 200, 100)
	contextLines := clampInt(gocast.ToInt(params["context_lines"]), 0, 2, 1)
	maxScanFiles := clampInt(gocast.ToInt(params["max_scan_files"]), 1, 5000, 1200)
	maxFileBytes := clampInt64(gocast.ToInt64(params["max_file_bytes"]), 4096, 2*1024*1024, 256*1024)
	maxMatchesPerFile := clampInt(gocast.ToInt(params["max_matches_per_file"]), 1, 50, 20)
	includePatterns := parseIncludePatterns(gocast.ToString(params["include_glob"]))

	gormDB := s.GetGormDb()
	if gormDB == nil {
		return common.NotOk("数据库未初始化")
	}

	project, err := getWorkspaceProject(gormDB, projectCode)
	if err != nil {
		return common.NotOk(err.Error())
	}
	projectDir := strings.TrimSpace(gocast.ToString(project.ProjectDir))
	if projectDir == "" {
		return common.NotOk("项目目录为空，无法执行内容搜索")
	}
	serverUserID := strings.TrimSpace(gocast.ToString(project.ServerOsUsersID))
	if serverUserID == "" {
		return common.NotOk("项目未配置服务器用户，无法执行内容搜索")
	}

	serverUser, err := getServerUser(gormDB, serverUserID)
	if err != nil {
		return common.NotOk(err.Error())
	}
	serverID := strings.TrimSpace(gocast.ToString(serverUser.ServerID))
	if serverID == "" {
		return common.NotOk("服务器用户缺少服务器ID，无法执行内容搜索")
	}

	serverInstance, err := getServerInstance(gormDB, serverID)
	if err != nil {
		return common.NotOk(err.Error())
	}

	serverIP := strings.TrimSpace(serverInstance.ServerIP)
	if serverIP == "" {
		return common.NotOk("服务器IP为空，无法执行内容搜索")
	}
	userName := strings.TrimSpace(serverUser.UserName)
	if userName == "" {
		return common.NotOk("服务器用户名为空，无法执行内容搜索")
	}
	password, err := decryptServerPassword(gocast.ToString(serverUser.UserPwd))
	if err != nil {
		return common.NotOk(fmt.Sprintf("服务器密码解密失败: %s", err.Error()))
	}

	port := strings.TrimSpace(serverInstance.ServerPort)
	if port == "" {
		port = "22"
	}

	excludedSegments := contentSearchExcludeSegments(project.ExcludeDirs, params["exclude_dirs"])
	files, err := getWorkspaceProjectFiles(gormDB, projectCode)
	if err != nil {
		return common.NotOk(err.Error())
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

	items := make([]workspaceContentSearchItem, 0, minInt(maxResults, 32))
	matchedFiles := make(map[string]struct{})
	summary := workspaceContentSearchSummary{
		Engine: "sftp_scan",
	}
	targetKeyword := keyword
	if !matchCase {
		targetKeyword = strings.ToLower(keyword)
	}

	stopped := false
	for _, file := range files {
		if summary.ScannedFiles >= maxScanFiles {
			stopped = true
			break
		}
		fullPath := strings.TrimSpace(gocast.ToString(file.Path))
		if fullPath == "" {
			continue
		}
		fileName := strings.TrimSpace(gocast.ToString(file.Name))
		displayPath := trimProjectPrefix(fullPath, projectDir)
		if shouldSkipContentSearchPath(displayPath, excludedSegments) {
			continue
		}
		if !matchesIncludePatterns(fullPath, displayPath, fileName, includePatterns) {
			continue
		}

		summary.ScannedFiles++
		srcFile, openErr := sftpClient.Open(fullPath)
		if openErr != nil {
			summary.FailedFiles++
			continue
		}
		contentBytes, readErr := readLimitedBytes(srcFile, maxFileBytes)
		_ = srcFile.Close()
		if readErr != nil {
			summary.FailedFiles++
			continue
		}
		if !isLikelyTextContent(contentBytes) {
			continue
		}

		contentText := strings.ReplaceAll(string(contentBytes), "\r\n", "\n")
		lines := strings.Split(contentText, "\n")
		fileMatched := false
		fileHitCount := 0
		for idx, line := range lines {
			candidate := line
			if !matchCase {
				candidate = strings.ToLower(candidate)
			}
			col := strings.Index(candidate, targetKeyword)
			if col < 0 {
				continue
			}
			beforeText := joinContextLines(lines, idx-contextLines, idx)
			afterText := joinContextLines(lines, idx+1, idx+1+contextLines)
			item := workspaceContentSearchItem{
				FilePath:   fullPath,
				FileName:   resolveFileName(fileName, fullPath),
				Path:       displayPath,
				LineNo:     idx + 1,
				ColumnNo:   col + 1,
				BeforeText: beforeText,
				MatchText:  line,
				AfterText:  afterText,
			}
			items = append(items, item)
			summary.HitCount++
			fileHitCount++
			fileMatched = true

			if fileHitCount >= maxMatchesPerFile {
				break
			}
			if len(items) >= maxResults {
				summary.Truncated = true
				stopped = true
				break
			}
		}
		if fileMatched {
			matchedFiles[fullPath] = struct{}{}
		}
		if stopped {
			break
		}
	}

	summary.MatchedFiles = len(matchedFiles)
	summary.ScanStopped = stopped
	summary.DurationMs = time.Since(startAt).Milliseconds()

	result := map[string]interface{}{
		"summary": summary,
		"items":   items,
	}
	return common.Ok(result, "搜索完成")
}

func clampInt(value, min, max, def int) int {
	if value <= 0 {
		value = def
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampInt64(value, min, max, def int64) int64 {
	if value <= 0 {
		value = def
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func parseIncludePatterns(raw string) []string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil
	}
	parts := strings.Split(text, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		pattern := strings.TrimSpace(part)
		if pattern == "" {
			continue
		}
		out = append(out, pattern)
	}
	return out
}

func joinContextLines(lines []string, start, end int) string {
	if len(lines) == 0 {
		return ""
	}
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start >= end {
		return ""
	}
	return strings.Join(lines[start:end], "\n")
}

func resolveFileName(fileName, fullPath string) string {
	if strings.TrimSpace(fileName) != "" {
		return fileName
	}
	return path.Base(strings.TrimSpace(fullPath))
}

func trimProjectPrefix(fullPath, projectDir string) string {
	full := normalizeContentSearchPath(fullPath)
	base := strings.TrimRight(normalizeContentSearchPath(projectDir), "/")
	if full == "" {
		return ""
	}
	if base == "" {
		return full
	}
	withSlash := base + "/"
	if full == base {
		return path.Base(full)
	}
	if strings.HasPrefix(full, withSlash) {
		return strings.TrimPrefix(full, withSlash)
	}
	lowerFull := strings.ToLower(full)
	lowerBase := strings.ToLower(base)
	lowerWithSlash := lowerBase + "/"
	if lowerFull == lowerBase {
		return path.Base(full)
	}
	if strings.HasPrefix(lowerFull, lowerWithSlash) {
		return full[len(withSlash):]
	}
	return full
}

func matchesIncludePatterns(fullPath, displayPath, fileName string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	fullPath = normalizeContentSearchPath(fullPath)
	displayPath = normalizeContentSearchPath(displayPath)
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		pattern = normalizeContentSearchPath(pattern)
		if matched, _ := path.Match(pattern, fileName); matched {
			return true
		}
		if matched, _ := path.Match(pattern, displayPath); matched {
			return true
		}
		if strings.Contains(displayPath, pattern) || strings.Contains(fullPath, pattern) {
			return true
		}
	}
	return false
}

func shouldSkipContentSearchPath(displayPath string, excludedSegments map[string]struct{}) bool {
	pathValue := strings.Trim(strings.ToLower(normalizeContentSearchPath(displayPath)), "/")
	if pathValue == "" {
		return false
	}
	normalized := "/" + pathValue + "/"
	for segment := range excludedSegments {
		if strings.Contains(normalized, "/"+segment+"/") {
			return true
		}
	}
	fileName := path.Base(pathValue)
	skippedNames := []string{
		"bun.lockb",
		"package-lock.json",
		"pnpm-lock.yaml",
		"yarn.lock",
	}
	for _, name := range skippedNames {
		if fileName == name {
			return true
		}
	}
	skippedSuffixes := []string{
		".map",
		".min.css",
		".min.js",
		".7z",
		".avi",
		".avif",
		".bmp",
		".class",
		".dll",
		".docx",
		".dylib",
		".eot",
		".exe",
		".flac",
		".gif",
		".gz",
		".ico",
		".jar",
		".jpeg",
		".jpg",
		".mkv",
		".mov",
		".mp3",
		".mp4",
		".otf",
		".pdf",
		".png",
		".rar",
		".so",
		".svg",
		".tar",
		".tgz",
		".ttf",
		".war",
		".wasm",
		".wav",
		".webp",
		".woff",
		".woff2",
		".zip",
	}
	for _, suffix := range skippedSuffixes {
		if strings.HasSuffix(fileName, suffix) {
			return true
		}
	}
	return false
}

func contentSearchExcludeSegments(values ...interface{}) map[string]struct{} {
	segments := map[string]struct{}{}
	for _, segment := range []string{
		".git",
		".hg",
		".svn",
		".cache",
		".next",
		".nuxt",
		".parcel-cache",
		".svelte-kit",
		".turbo",
		".vite",
		"node_modules",
		"bower_components",
		"dist",
		"build",
		"out",
		"coverage",
		"vendor",
		".tools",
		".tmp-playwright",
		".opencode",
		".codex",
		".idea",
		".vscode",
		"__pycache__",
	} {
		addContentSearchExcludeSegment(segments, segment)
	}
	for _, value := range values {
		for segment := range parseExcludeDirSet(value) {
			addContentSearchExcludeSegment(segments, segment)
		}
	}
	return segments
}

func addContentSearchExcludeSegment(segments map[string]struct{}, segment string) {
	segment = strings.Trim(strings.ToLower(normalizeContentSearchPath(segment)), "/")
	if segment == "" {
		return
	}
	segments[segment] = struct{}{}
}

func normalizeContentSearchPath(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
}

func getWorkspaceProject(gormDB *gorm.DB, projectCode string) (*devops.WebshellWorkspaceProject, error) {
	var project devops.WebshellWorkspaceProject
	err := gormDB.
		Model(&devops.WebshellWorkspaceProject{}).
		Select("project_code", "project_dir", "server_os_users_id", "exclude_dirs").
		Where("project_code = ? AND ifnull(is_delete, '0') = '0'", projectCode).
		Order("modify_time desc").
		First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("项目[%s]不存在", projectCode)
		}
		return nil, err
	}
	return &project, nil
}

func getServerUser(gormDB *gorm.DB, serverUserID string) (*devops.ServerOsUsers, error) {
	var user devops.ServerOsUsers
	err := gormDB.
		Model(&devops.ServerOsUsers{}).
		Select("server_os_users_id", "user_name", "user_pwd", "server_id").
		Where("server_os_users_id = ?", serverUserID).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("服务器用户[%s]不存在", serverUserID)
		}
		return nil, err
	}
	return &user, nil
}

func getServerInstance(gormDB *gorm.DB, serverID string) (*devops.ServerInstance, error) {
	var server devops.ServerInstance
	err := gormDB.
		Model(&devops.ServerInstance{}).
		Select("server_id", "server_ip", "server_port").
		Where("server_id = ?", serverID).
		Order("modify_time desc").
		First(&server).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("服务器[%s]不存在", serverID)
		}
		return nil, err
	}
	return &server, nil
}

func getWorkspaceProjectFiles(gormDB *gorm.DB, projectCode string) ([]devops.WebshellWorkspaceFile, error) {
	list := make([]devops.WebshellWorkspaceFile, 0)
	err := gormDB.
		Model(&devops.WebshellWorkspaceFile{}).
		Select("path", "name").
		Where("project_code = ? AND ifnull(is_delete, '0') = '0' AND ifnull(is_dir, '0') != '1'", projectCode).
		Order("path asc").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func dialSSH(serverIP, port, userName, password string) (*ssh.Client, error) {
	configValue := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		Timeout: 15 * time.Second,
		Config: ssh.Config{
			Ciphers: []string{
				"aes128-ctr", "aes192-ctr", "aes256-ctr",
				"aes128-gcm@openssh.com",
				"arcfour256", "arcfour128",
				"aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc",
			},
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := fmt.Sprintf("%s:%s", serverIP, port)
	client, err := ssh.Dial("tcp", addr, configValue)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func decryptServerPassword(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("服务器密码为空")
	}
	aseTag := utils.GetAppKey("ase_tag")
	if aseTag == "" {
		return value, nil
	}
	if !(strings.HasPrefix(value, aseTag) && strings.HasSuffix(value, aseTag)) {
		return value, nil
	}
	value = strings.ReplaceAll(value, aseTag, "")
	aseWay := utils.GetAppKey("ase_way")
	if aseWay == "company" {
		companyKey := utils.GetAppKey("company_key")
		if companyKey == "" {
			return "", fmt.Errorf("company_key 为空")
		}
		return AESDecryptECBStr(value, companyKey), nil
	}

	aseKey := utils.GetAppKey("ase_key")
	aseIv := utils.GetAppKey("ase_iv")
	if aseKey == "" || aseIv == "" {
		return "", fmt.Errorf("ase_key 或 ase_iv 为空")
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return "", err
	}
	decrypted, err := decrypt([]byte(aseKey), []byte(aseIv), string(decoded))
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

func readLimitedBytes(srcFile io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = 256 * 1024
	}
	contentBytes, err := io.ReadAll(io.LimitReader(srcFile, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(contentBytes)) > maxBytes {
		contentBytes = contentBytes[:maxBytes]
	}
	return contentBytes, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
