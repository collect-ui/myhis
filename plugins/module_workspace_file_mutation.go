package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
	"github.com/pkg/sftp"
	"gorm.io/gorm"

	"moon/model/devops"
)

type WorkspaceFileMutationService struct {
	templateService.BaseHandler
}

type workspaceFileRow struct {
	FileID     string `gorm:"column:file_id"`
	Name       string `gorm:"column:name"`
	Path       string `gorm:"column:path"`
	ParentID   string `gorm:"column:parent_id"`
	IsDir      string `gorm:"column:is_dir"`
	OrderIndex int    `gorm:"column:order_index"`
}

type workspaceFileMutationTarget struct {
	FileID   string
	Name     string
	Path     string
	ParentID string
	IsDir    string
}

func (s *WorkspaceFileMutationService) Result(template *config.Template, ts *templateService.TemplateService) *common.Result {
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
	params["project_dir"] = gocast.ToString(project.ProjectDir)
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

	result := map[string]interface{}{
		"operation":    operation,
		"project_code": projectCode,
	}

	switch operation {
	case "add":
		if err := s.remoteAdd(gormDB, sftpClient, projectCode, params, result); err != nil {
			return common.NotOk(err.Error())
		}
	case "update":
		if err := s.remoteRename(gormDB, sftpClient, params, result); err != nil {
			return common.NotOk(err.Error())
		}
	case "delete":
		if err := s.remoteDelete(gormDB, sftpClient, params, result); err != nil {
			return common.NotOk(err.Error())
		}
	case "move":
		if err := s.remoteMove(gormDB, sftpClient, params, result); err != nil {
			return common.NotOk(err.Error())
		}
	case "copy":
		if err := s.remoteCopy(gormDB, sftpClient, params, result); err != nil {
			return common.NotOk(err.Error())
		}
	default:
		return common.NotOk("operation仅支持 add/update/delete/move/copy")
	}

	return common.Ok(result, "远程文件操作成功")
}

func (s *WorkspaceFileMutationService) remoteAdd(gormDB *gorm.DB, sftpClient *sftp.Client, projectCode string, params map[string]interface{}, result map[string]interface{}) error {
	projectRoot, err := normalizeWorkspaceProjectRoot(gocast.ToString(params["project_dir"]))
	if err != nil {
		return err
	}
	targetPath, err := normalizeWorkspaceTargetPath(projectRoot, gocast.ToString(params["path"]))
	if err != nil {
		return err
	}
	if targetPath == projectRoot {
		return fmt.Errorf("项目根目录不支持重复新增")
	}
	if err := ensureWorkspacePathAvailable(gormDB, projectCode, targetPath, ""); err != nil {
		return err
	}
	if err := ensureRemotePathChainSafe(sftpClient, projectRoot, targetPath, true); err != nil {
		return err
	}

	exists, err := remotePathExists(sftpClient, targetPath)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("路径[%s]已存在", targetPath)
	}

	isDir := toBoolStringValue(params["is_dir"], true)
	if isDir {
		if err := sftpClient.MkdirAll(targetPath); err != nil {
			return err
		}
		result["is_dir"] = "1"
	} else {
		parentDir := path.Dir(targetPath)
		if parentDir != "" && parentDir != "." {
			if err := sftpClient.MkdirAll(parentDir); err != nil {
				return err
			}
		}
		file, err := sftpClient.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
		if err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		result["is_dir"] = "0"
	}

	fileID := stableNodeID(targetPath)
	if fileID == "" {
		return fmt.Errorf("生成文件主键失败")
	}
	parentID := normalizeWorkspaceParentID(projectRoot, targetPath, gocast.ToString(params["parent_id"]))
	fileName := strings.TrimSpace(gocast.ToString(params["name"]))
	if fileName == "" {
		fileName = path.Base(targetPath)
	}
	if err := upsertWorkspaceFileRecord(gormDB, projectCode, fileID, fileName, targetPath, parentID, boolToDirFlag(isDir), workspaceOpUser(params)); err != nil {
		return err
	}

	result["file_id"] = fileID
	result["name"] = fileName
	result["parent_id"] = parentID
	result["path"] = targetPath
	return nil
}

func (s *WorkspaceFileMutationService) remoteRename(gormDB *gorm.DB, sftpClient *sftp.Client, params map[string]interface{}, result map[string]interface{}) error {
	projectCode := strings.TrimSpace(gocast.ToString(params["project_code"]))
	if projectCode == "" {
		return fmt.Errorf("项目编码不能为空")
	}
	fileID := strings.TrimSpace(gocast.ToString(params["file_id"]))
	if fileID == "" {
		return fmt.Errorf("file_id不能为空")
	}

	currentFile, err := getWorkspaceFileByID(gormDB, projectCode, fileID)
	if err != nil {
		return err
	}
	projectRoot, err := normalizeWorkspaceProjectRoot(gocast.ToString(params["project_dir"]))
	if err != nil {
		return err
	}
	oldPath, err := normalizeWorkspaceTargetPath(projectRoot, currentFile.Path)
	if err != nil {
		return err
	}
	newPath, err := normalizeWorkspaceTargetPath(projectRoot, gocast.ToString(params["path"]))
	if err != nil {
		return err
	}
	if newPath == projectRoot {
		return fmt.Errorf("项目根目录不支持重命名覆盖")
	}
	if err := ensureWorkspacePathAvailable(gormDB, projectCode, newPath, fileID); err != nil {
		return err
	}
	if err := ensureRemotePathChainSafe(sftpClient, projectRoot, oldPath, false); err != nil {
		return err
	}
	if err := ensureRemotePathChainSafe(sftpClient, projectRoot, newPath, true); err != nil {
		return err
	}

	result["file_id"] = fileID
	result["old_path"] = oldPath
	result["new_path"] = newPath

	if oldPath == newPath {
		if err := updateWorkspaceFileRecord(gormDB, projectCode, fileID, path.Base(newPath), newPath, normalizeWorkspaceParentID(projectRoot, newPath, gocast.ToString(params["parent_id"])), currentFile.IsDir, workspaceOpUser(params)); err != nil {
			return err
		}
		result["skip"] = true
		result["skip_reason"] = "路径未变化"
		return nil
	}

	exists, err := remotePathExists(sftpClient, newPath)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("目标路径[%s]已存在", newPath)
	}

	parentDir := path.Dir(newPath)
	if parentDir != "" && parentDir != "." {
		if err := sftpClient.MkdirAll(parentDir); err != nil {
			return err
		}
	}

	if err := sftpClient.PosixRename(oldPath, newPath); err != nil {
		if err2 := sftpClient.Rename(oldPath, newPath); err2 != nil {
			return err
		}
	}

	if err := updateWorkspaceFileSubtree(gormDB, projectCode, projectRoot, oldPath, newPath, fileID, currentFile.IsDir, workspaceOpUser(params)); err != nil {
		return err
	}

	result["file_id"] = stableNodeID(newPath)
	result["parent_id"] = normalizeWorkspaceParentID(projectRoot, newPath, gocast.ToString(params["parent_id"]))
	result["name"] = path.Base(newPath)
	result["path"] = newPath

	return nil
}

func (s *WorkspaceFileMutationService) remoteMove(gormDB *gorm.DB, sftpClient *sftp.Client, params map[string]interface{}, result map[string]interface{}) error {
	projectCode := strings.TrimSpace(gocast.ToString(params["project_code"]))
	if projectCode == "" {
		return fmt.Errorf("项目编码不能为空")
	}
	fileID := strings.TrimSpace(gocast.ToString(params["file_id"]))
	if fileID == "" {
		return fmt.Errorf("file_id不能为空")
	}

	currentFile, err := getWorkspaceFileByID(gormDB, projectCode, fileID)
	if err != nil {
		return err
	}
	projectRoot, err := normalizeWorkspaceProjectRoot(gocast.ToString(params["project_dir"]))
	if err != nil {
		return err
	}
	oldPath, err := normalizeWorkspaceTargetPath(projectRoot, currentFile.Path)
	if err != nil {
		return err
	}

	targetParentID := strings.TrimSpace(gocast.ToString(params["target_parent_id"]))
	if targetParentID == "" {
		targetParentID = strings.TrimSpace(gocast.ToString(params["parent_file_id"]))
	}
	if targetParentID == "0" {
		targetParentID = ""
	}

	targetParentPath := projectRoot
	if targetParentID != "" {
		parentFile, err := getWorkspaceFileByID(gormDB, projectCode, targetParentID)
		if err != nil {
			return err
		}
		if parentFile.IsDir != "1" {
			return fmt.Errorf("目标父节点不是目录")
		}
		targetParentPath, err = normalizeWorkspaceTargetPath(projectRoot, parentFile.Path)
		if err != nil {
			return err
		}
		if currentFile.IsDir == "1" && (targetParentPath == oldPath || strings.HasPrefix(targetParentPath, oldPath+"/")) {
			return fmt.Errorf("目录不能移动到自身或子目录下")
		}
	}

	fileName := strings.TrimSpace(gocast.ToString(params["name"]))
	if fileName == "" {
		fileName = currentFile.Name
	}
	if fileName == "" {
		fileName = path.Base(oldPath)
	}
	newPath, err := normalizeWorkspaceTargetPath(projectRoot, path.Join(targetParentPath, fileName))
	if err != nil {
		return err
	}
	if newPath == projectRoot {
		return fmt.Errorf("项目根目录不支持移动覆盖")
	}
	if currentFile.IsDir == "1" && strings.HasPrefix(newPath, oldPath+"/") {
		return fmt.Errorf("目录不能移动到自身或子目录下")
	}

	result["file_id"] = fileID
	result["old_path"] = oldPath
	result["new_path"] = newPath
	result["target_parent_id"] = targetParentID

	newFileID := fileID
	user := workspaceOpUser(params)
	if oldPath != newPath {
		if err := ensureWorkspacePathAvailable(gormDB, projectCode, newPath, fileID); err != nil {
			return err
		}
		if err := ensureRemotePathChainSafe(sftpClient, projectRoot, oldPath, false); err != nil {
			return err
		}
		if err := ensureRemotePathChainSafe(sftpClient, projectRoot, newPath, true); err != nil {
			return err
		}
		exists, err := remotePathExists(sftpClient, newPath)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("目标路径[%s]已存在", newPath)
		}
		parentDir := path.Dir(newPath)
		if parentDir != "" && parentDir != "." {
			if err := sftpClient.MkdirAll(parentDir); err != nil {
				return err
			}
		}
		if err := sftpClient.PosixRename(oldPath, newPath); err != nil {
			if err2 := sftpClient.Rename(oldPath, newPath); err2 != nil {
				return err
			}
		}
		if err := updateWorkspaceFileSubtree(gormDB, projectCode, projectRoot, oldPath, newPath, fileID, currentFile.IsDir, user); err != nil {
			return err
		}
		newFileID = stableNodeID(newPath)
		result["file_id"] = newFileID
		result["moved"] = true
	} else if currentFile.ParentID != targetParentID {
		if err := updateWorkspaceFileRecord(gormDB, projectCode, fileID, path.Base(newPath), newPath, targetParentID, currentFile.IsDir, user); err != nil {
			return err
		}
	}

	rows := normalizeWorkspaceOrderRows(params["rows"])
	if len(rows) > 0 {
		if err := applyWorkspaceFileOrder(gormDB, projectCode, targetParentID, fileID, newFileID, rows, user); err != nil {
			return err
		}
		result["order_updated"] = true
		result["order_count"] = len(rows)
	}

	result["name"] = path.Base(newPath)
	result["path"] = newPath
	result["parent_id"] = targetParentID
	return nil
}

func (s *WorkspaceFileMutationService) remoteDelete(gormDB *gorm.DB, sftpClient *sftp.Client, params map[string]interface{}, result map[string]interface{}) error {
	projectCode := strings.TrimSpace(gocast.ToString(params["project_code"]))
	if projectCode == "" {
		return fmt.Errorf("项目编码不能为空")
	}

	projectRoot, err := normalizeWorkspaceProjectRoot(gocast.ToString(params["project_dir"]))
	if err != nil {
		return err
	}
	targets, err := resolveWorkspaceFileMutationTargets(gormDB, projectCode, projectRoot, params)
	if err != nil {
		return err
	}
	targets = compactWorkspaceMutationTargets(targets)
	if len(targets) == 0 {
		return fmt.Errorf("path或file_id不能为空")
	}
	for _, target := range targets {
		if target.Path == projectRoot {
			return fmt.Errorf("禁止删除项目根目录")
		}
		if err := ensureRemotePathChainSafe(sftpClient, projectRoot, target.Path, false); err != nil {
			return err
		}
	}

	removedCount := 0
	skippedCount := 0
	user := workspaceOpUser(params)
	for _, target := range targets {
		exists, err := remotePathExists(sftpClient, target.Path)
		if err != nil {
			return err
		}
		if !exists {
			skippedCount++
			if err := softDeleteWorkspaceFileSubtree(gormDB, projectCode, target.Path, user); err != nil {
				return err
			}
			continue
		}

		if err := sftpClient.RemoveAll(target.Path); err != nil {
			return err
		}

		if err := softDeleteWorkspaceFileSubtree(gormDB, projectCode, target.Path, user); err != nil {
			return err
		}
		removedCount++
	}

	result["count"] = len(targets)
	result["removed_count"] = removedCount
	result["skipped_count"] = skippedCount
	result["removed"] = removedCount > 0
	return nil
}

func (s *WorkspaceFileMutationService) remoteCopy(gormDB *gorm.DB, sftpClient *sftp.Client, params map[string]interface{}, result map[string]interface{}) error {
	projectCode := strings.TrimSpace(gocast.ToString(params["project_code"]))
	if projectCode == "" {
		return fmt.Errorf("项目编码不能为空")
	}
	projectRoot, err := normalizeWorkspaceProjectRoot(gocast.ToString(params["project_dir"]))
	if err != nil {
		return err
	}

	targetParentID := strings.TrimSpace(gocast.ToString(params["target_parent_id"]))
	if targetParentID == "0" {
		targetParentID = ""
	}
	targetParentPath := projectRoot
	if targetParentID != "" {
		parentFile, err := getWorkspaceFileByID(gormDB, projectCode, targetParentID)
		if err != nil {
			return err
		}
		if parentFile.IsDir != "1" {
			return fmt.Errorf("目标父节点不是目录")
		}
		targetParentPath, err = normalizeWorkspaceTargetPath(projectRoot, parentFile.Path)
		if err != nil {
			return err
		}
	}
	if err := ensureRemotePathChainSafe(sftpClient, projectRoot, targetParentPath, false); err != nil {
		return err
	}
	parentInfo, err := sftpClient.Stat(targetParentPath)
	if err != nil {
		return err
	}
	if !parentInfo.IsDir() {
		return fmt.Errorf("目标路径不是目录")
	}

	sources, err := resolveWorkspaceFileMutationTargets(gormDB, projectCode, projectRoot, params)
	if err != nil {
		return err
	}
	sources = compactWorkspaceMutationTargets(sources)
	if len(sources) == 0 {
		return fmt.Errorf("复制源不能为空")
	}

	user := workspaceOpUser(params)
	copied := make([]map[string]interface{}, 0, len(sources))
	for _, source := range sources {
		if source.Path == projectRoot {
			return fmt.Errorf("项目根目录不支持复制")
		}
		if err := ensureRemotePathChainSafe(sftpClient, projectRoot, source.Path, false); err != nil {
			return err
		}
		sourceInfo, err := sftpClient.Stat(source.Path)
		if err != nil {
			return err
		}
		isDir := sourceInfo.IsDir()
		if isDir && (targetParentPath == source.Path || strings.HasPrefix(targetParentPath, source.Path+"/")) {
			return fmt.Errorf("目录不能复制到自身或子目录下")
		}

		sourceName := strings.TrimSpace(source.Name)
		if sourceName == "" {
			sourceName = path.Base(source.Path)
		}
		targetPath, err := nextWorkspaceCopyPath(gormDB, sftpClient, projectCode, targetParentPath, sourceName, isDir)
		if err != nil {
			return err
		}
		if err := ensureRemotePathChainSafe(sftpClient, projectRoot, targetPath, true); err != nil {
			return err
		}
		if isDir {
			if err := copyRemoteDir(sftpClient, source.Path, targetPath); err != nil {
				return err
			}
		} else {
			if err := copyRemoteFile(sftpClient, source.Path, targetPath); err != nil {
				return err
			}
		}
		if err := syncWorkspaceCopiedSubtree(gormDB, sftpClient, projectCode, projectRoot, targetPath, user); err != nil {
			return err
		}
		copied = append(copied, map[string]interface{}{
			"source_path": source.Path,
			"target_path": targetPath,
			"file_id":     stableNodeID(targetPath),
			"name":        path.Base(targetPath),
			"is_dir":      boolToDirFlag(isDir),
			"parent_id":   normalizeWorkspaceParentID(projectRoot, targetPath, ""),
		})
	}

	result["target_parent_id"] = targetParentID
	result["target_parent_path"] = targetParentPath
	result["copied"] = copied
	result["copied_count"] = len(copied)
	return nil
}

func resolveWorkspaceFileMutationTargets(gormDB *gorm.DB, projectCode, projectRoot string, params map[string]interface{}) ([]workspaceFileMutationTarget, error) {
	fileIDs := make([]string, 0)
	for _, key := range []string{"file_id_list", "file_ids"} {
		fileIDs = append(fileIDs, workspaceStringList(params[key])...)
	}
	if fileID := strings.TrimSpace(gocast.ToString(params["file_id"])); fileID != "" {
		fileIDs = append(fileIDs, fileID)
	}

	paths := make([]string, 0)
	for _, key := range []string{"path_list", "paths"} {
		paths = append(paths, workspaceStringList(params[key])...)
	}
	if targetPath := strings.TrimSpace(gocast.ToString(params["path"])); targetPath != "" {
		paths = append(paths, targetPath)
	}

	targets := make([]workspaceFileMutationTarget, 0, len(fileIDs)+len(paths))
	seenPaths := map[string]struct{}{}
	for _, fileID := range workspaceStringList(fileIDs) {
		row, err := getWorkspaceFileByID(gormDB, projectCode, fileID)
		if err != nil {
			return nil, err
		}
		normalizedPath, err := normalizeWorkspaceTargetPath(projectRoot, row.Path)
		if err != nil {
			return nil, err
		}
		if _, ok := seenPaths[normalizedPath]; ok {
			continue
		}
		seenPaths[normalizedPath] = struct{}{}
		targets = append(targets, workspaceFileMutationTarget{
			FileID:   row.FileID,
			Name:     row.Name,
			Path:     normalizedPath,
			ParentID: row.ParentID,
			IsDir:    row.IsDir,
		})
	}
	for _, rawPath := range workspaceStringList(paths) {
		normalizedPath, err := normalizeWorkspaceTargetPath(projectRoot, rawPath)
		if err != nil {
			return nil, err
		}
		if _, ok := seenPaths[normalizedPath]; ok {
			continue
		}
		seenPaths[normalizedPath] = struct{}{}
		targets = append(targets, workspaceFileMutationTarget{
			Name: path.Base(normalizedPath),
			Path: normalizedPath,
		})
	}
	sort.Slice(targets, func(i, j int) bool {
		return len(targets[i].Path) < len(targets[j].Path)
	})
	return targets, nil
}

func workspaceStringList(raw interface{}) []string {
	items := make([]string, 0)
	switch value := raw.(type) {
	case nil:
		return items
	case []string:
		items = append(items, value...)
	case []interface{}:
		for _, item := range value {
			items = append(items, gocast.ToString(item))
		}
	case []map[string]interface{}:
		for _, item := range value {
			items = append(items, gocast.ToString(item))
		}
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return items
		}
		var stringList []string
		if err := json.Unmarshal([]byte(text), &stringList); err == nil {
			items = append(items, stringList...)
			break
		}
		var anyList []interface{}
		if err := json.Unmarshal([]byte(text), &anyList); err == nil {
			for _, item := range anyList {
				items = append(items, gocast.ToString(item))
			}
			break
		}
		if strings.Contains(text, ",") {
			items = append(items, strings.Split(text, ",")...)
		} else {
			items = append(items, text)
		}
	default:
		text := strings.TrimSpace(gocast.ToString(raw))
		if text != "" {
			items = append(items, text)
		}
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(item)
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		result = append(result, text)
	}
	return result
}

func compactWorkspaceMutationTargets(targets []workspaceFileMutationTarget) []workspaceFileMutationTarget {
	if len(targets) <= 1 {
		return targets
	}
	sort.Slice(targets, func(i, j int) bool {
		return len(targets[i].Path) < len(targets[j].Path)
	})
	result := make([]workspaceFileMutationTarget, 0, len(targets))
	for _, target := range targets {
		if strings.TrimSpace(target.Path) == "" {
			continue
		}
		nested := false
		for _, current := range result {
			if target.Path == current.Path || strings.HasPrefix(target.Path, current.Path+"/") {
				nested = true
				break
			}
		}
		if !nested {
			result = append(result, target)
		}
	}
	return result
}

func nextWorkspaceCopyPath(gormDB *gorm.DB, sftpClient *sftp.Client, projectCode, targetParentPath, baseName string, isDir bool) (string, error) {
	name := path.Base(strings.TrimSpace(baseName))
	if name == "." || name == "/" || name == "" {
		return "", fmt.Errorf("复制源名称不能为空")
	}
	for index := 0; index < 1000; index++ {
		candidateName := workspaceCopyNameCandidate(name, isDir, index)
		candidatePath := path.Join(targetParentPath, candidateName)
		recordExists, err := workspacePathRecordExists(gormDB, projectCode, candidatePath)
		if err != nil {
			return "", err
		}
		if recordExists {
			continue
		}
		remoteExists, err := remotePathExists(sftpClient, candidatePath)
		if err != nil {
			return "", err
		}
		if remoteExists {
			continue
		}
		return candidatePath, nil
	}
	return "", fmt.Errorf("无法生成可用复制路径")
}

func workspaceCopyNameCandidate(baseName string, isDir bool, index int) string {
	if index <= 0 {
		return baseName
	}
	suffix := " copy"
	if index > 1 {
		suffix = fmt.Sprintf(" copy %d", index)
	}
	if isDir {
		return baseName + suffix
	}
	ext := path.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)
	if stem == "" {
		return baseName + suffix
	}
	return stem + suffix + ext
}

func workspacePathRecordExists(gormDB *gorm.DB, projectCode, targetPath string) (bool, error) {
	var count int64
	if err := gormDB.Model(&devops.WebshellWorkspaceFile{}).
		Where("project_code = ? AND path = ? AND ifnull(is_delete, '0') = '0'", projectCode, targetPath).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func copyRemoteFile(sftpClient *sftp.Client, sourcePath, targetPath string) error {
	parentDir := path.Dir(targetPath)
	if parentDir != "" && parentDir != "." {
		if err := sftpClient.MkdirAll(parentDir); err != nil {
			return err
		}
	}
	sourceFile, err := sftpClient.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := sftpClient.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return err
	}
	defer targetFile.Close()
	_, err = io.Copy(targetFile, sourceFile)
	return err
}

func copyRemoteDir(sftpClient *sftp.Client, sourcePath, targetPath string) error {
	if err := sftpClient.MkdirAll(targetPath); err != nil {
		return err
	}
	entries, err := sftpClient.ReadDir(sourcePath)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, entry := range entries {
		sourceChild := path.Join(sourcePath, entry.Name())
		targetChild := path.Join(targetPath, entry.Name())
		if entry.IsDir() {
			if err := copyRemoteDir(sftpClient, sourceChild, targetChild); err != nil {
				return err
			}
			continue
		}
		if err := copyRemoteFile(sftpClient, sourceChild, targetChild); err != nil {
			return err
		}
	}
	return nil
}

func syncWorkspaceCopiedSubtree(gormDB *gorm.DB, sftpClient *sftp.Client, projectCode, projectRoot, targetPath, user string) error {
	info, err := sftpClient.Stat(targetPath)
	if err != nil {
		return err
	}
	isDir := boolToDirFlag(info.IsDir())
	if err := upsertWorkspaceFileRecord(
		gormDB,
		projectCode,
		stableNodeID(targetPath),
		path.Base(targetPath),
		targetPath,
		normalizeWorkspaceParentID(projectRoot, targetPath, ""),
		isDir,
		user,
	); err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	entries, err := sftpClient.ReadDir(targetPath)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, entry := range entries {
		if err := syncWorkspaceCopiedSubtree(gormDB, sftpClient, projectCode, projectRoot, path.Join(targetPath, entry.Name()), user); err != nil {
			return err
		}
	}
	return nil
}

func getWorkspaceFileByID(gormDB *gorm.DB, projectCode, fileID string) (*workspaceFileRow, error) {
	row := &workspaceFileRow{}
	err := gormDB.
		Model(&devops.WebshellWorkspaceFile{}).
		Select("file_id", "name", "path", "parent_id", "is_dir", "order_index").
		Where("project_code = ? AND file_id = ? AND ifnull(is_delete, '0') = '0'", projectCode, fileID).
		First(row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("文件记录[%s]不存在", fileID)
		}
		return nil, err
	}
	return row, nil
}

func ensureWorkspacePathAvailable(gormDB *gorm.DB, projectCode, targetPath, excludeFileID string) error {
	if gormDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	query := gormDB.
		Model(&devops.WebshellWorkspaceFile{}).
		Where("project_code = ? AND path = ? AND ifnull(is_delete, '0') = '0'", projectCode, targetPath)
	if strings.TrimSpace(excludeFileID) != "" {
		query = query.Where("file_id != ?", excludeFileID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("路径[%s]已存在", targetPath)
	}
	return nil
}

func upsertWorkspaceFileRecord(gormDB *gorm.DB, projectCode, fileID, name, targetPath, parentID, isDir, user string) error {
	now := utils.DateFormat(time.Now(), "")
	values := map[string]interface{}{
		"project_code": projectCode,
		"name":         name,
		"path":         targetPath,
		"parent_id":    parentID,
		"is_dir":       isDir,
		"is_delete":    "0",
		"modify_time":  now,
		"modify_user":  user,
	}

	var count int64
	if err := gormDB.Model(&devops.WebshellWorkspaceFile{}).Where("file_id = ?", fileID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return gormDB.Model(&devops.WebshellWorkspaceFile{}).
			Where("file_id = ?", fileID).
			Updates(values).Error
	}

	projectCodeValue := projectCode
	nameValue := name
	pathValue := targetPath
	parentIDValue := parentID
	isDirValue := isDir
	isDeleteValue := "0"
	createTimeValue := now
	createUserValue := user
	modifyTimeValue := now
	modifyUserValue := user
	return gormDB.Create(&devops.WebshellWorkspaceFile{
		FileID:      fileID,
		ProjectCode: &projectCodeValue,
		Name:        &nameValue,
		Path:        &pathValue,
		ParentID:    &parentIDValue,
		IsDir:       &isDirValue,
		IsDelete:    &isDeleteValue,
		CreateTime:  &createTimeValue,
		CreateUser:  &createUserValue,
		ModifyTime:  &modifyTimeValue,
		ModifyUser:  &modifyUserValue,
	}).Error
}

func updateWorkspaceFileRecord(gormDB *gorm.DB, projectCode, fileID, name, targetPath, parentID, isDir, user string) error {
	now := utils.DateFormat(time.Now(), "")
	return gormDB.Model(&devops.WebshellWorkspaceFile{}).
		Where("project_code = ? AND file_id = ?", projectCode, fileID).
		Updates(map[string]interface{}{
			"name":        name,
			"path":        targetPath,
			"parent_id":   parentID,
			"is_dir":      isDir,
			"is_delete":   "0",
			"modify_time": now,
			"modify_user": user,
		}).Error
}

func updateWorkspaceFileSubtree(gormDB *gorm.DB, projectCode, projectRoot, oldPath, newPath, fileID, isDir, user string) error {
	rows := make([]workspaceFileRow, 0)
	query := gormDB.Model(&devops.WebshellWorkspaceFile{}).
		Select("file_id", "name", "path", "parent_id", "is_dir", "order_index").
		Where("project_code = ? AND ifnull(is_delete, '0') = '0'", projectCode)
	if isDir == "1" {
		query = query.Where("(path = ? OR path LIKE ?)", oldPath, oldPath+"/%")
	} else {
		query = query.Where("file_id = ?", fileID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return updateWorkspaceFileRecord(gormDB, projectCode, fileID, path.Base(newPath), newPath, normalizeWorkspaceParentID(projectRoot, newPath, ""), isDir, user)
	}

	sort.Slice(rows, func(i, j int) bool {
		return len(rows[i].Path) < len(rows[j].Path)
	})

	newIDs := make([]string, 0, len(rows))
	rowUpdates := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		rowPath := strings.TrimSpace(row.Path)
		nextPath := newPath
		if rowPath != oldPath {
			nextPath = newPath + strings.TrimPrefix(rowPath, oldPath)
		}
		nextID := stableNodeID(nextPath)
		if nextID == "" {
			return fmt.Errorf("生成文件主键失败: %s", nextPath)
		}
		newIDs = append(newIDs, nextID)
		rowUpdates = append(rowUpdates, map[string]string{
			"old_id":    row.FileID,
			"file_id":   nextID,
			"name":      path.Base(nextPath),
			"path":      nextPath,
			"parent_id": normalizeWorkspaceParentID(projectRoot, nextPath, ""),
			"is_dir":    row.IsDir,
		})
	}

	var conflictCount int64
	if err := gormDB.Model(&devops.WebshellWorkspaceFile{}).
		Where("project_code = ? AND file_id IN ? AND ifnull(is_delete, '0') = '0' AND path NOT LIKE ?", projectCode, newIDs, oldPath+"%").
		Count(&conflictCount).Error; err != nil {
		return err
	}
	if conflictCount > 0 {
		return fmt.Errorf("目标路径存在冲突记录")
	}

	now := utils.DateFormat(time.Now(), "")
	return gormDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_code = ? AND file_id IN ? AND ifnull(is_delete, '0') != '0'", projectCode, newIDs).
			Delete(&devops.WebshellWorkspaceFile{}).Error; err != nil {
			return err
		}
		for _, item := range rowUpdates {
			if err := tx.Model(&devops.WebshellWorkspaceFile{}).
				Where("project_code = ? AND file_id = ?", projectCode, item["old_id"]).
				Updates(map[string]interface{}{
					"file_id":     item["file_id"],
					"name":        item["name"],
					"path":        item["path"],
					"parent_id":   item["parent_id"],
					"is_dir":      item["is_dir"],
					"is_delete":   "0",
					"modify_time": now,
					"modify_user": user,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func normalizeWorkspaceOrderRows(raw interface{}) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0)
	switch value := raw.(type) {
	case []map[string]interface{}:
		return value
	case []interface{}:
		for _, item := range value {
			if row, ok := item.(map[string]interface{}); ok {
				rows = append(rows, row)
			}
		}
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return rows
		}
		if err := json.Unmarshal([]byte(text), &rows); err == nil {
			return rows
		}
		var anyRows []interface{}
		if err := json.Unmarshal([]byte(text), &anyRows); err != nil {
			return rows
		}
		for _, item := range anyRows {
			if row, ok := item.(map[string]interface{}); ok {
				rows = append(rows, row)
			}
		}
	}
	return rows
}

func workspaceOrderRowFileID(row map[string]interface{}) string {
	for _, key := range []string{"file_id", "id", "key"} {
		value := strings.TrimSpace(gocast.ToString(row[key]))
		if value != "" {
			return value
		}
	}
	return ""
}

func workspaceOrderRowIndex(row map[string]interface{}, fallback int) int {
	raw := row["order_index"]
	switch value := raw.(type) {
	case int:
		if value > 0 {
			return value
		}
	case int32:
		if value > 0 {
			return int(value)
		}
	case int64:
		if value > 0 {
			return int(value)
		}
	case float32:
		if value > 0 {
			return int(value)
		}
	case float64:
		if value > 0 {
			return int(value)
		}
	case json.Number:
		if n, err := value.Int64(); err == nil && n > 0 {
			return int(n)
		}
	default:
		text := strings.TrimSpace(gocast.ToString(raw))
		if text != "" {
			if n, err := strconv.Atoi(text); err == nil && n > 0 {
				return n
			}
		}
	}
	return fallback
}

func applyWorkspaceFileOrder(gormDB *gorm.DB, projectCode, targetParentID, oldFileID, newFileID string, rows []map[string]interface{}, user string) error {
	now := utils.DateFormat(time.Now(), "")
	seen := make(map[string]struct{})
	return gormDB.Transaction(func(tx *gorm.DB) error {
		for index, row := range rows {
			rowID := workspaceOrderRowFileID(row)
			if rowID == "" {
				continue
			}
			if rowID == oldFileID {
				rowID = newFileID
			}
			if _, ok := seen[rowID]; ok {
				continue
			}
			seen[rowID] = struct{}{}
			updates := map[string]interface{}{
				"order_index": workspaceOrderRowIndex(row, (index+1)*10),
				"modify_time": now,
				"modify_user": user,
			}
			if rowID == newFileID {
				updates["parent_id"] = targetParentID
			}
			if err := tx.Model(&devops.WebshellWorkspaceFile{}).
				Where("project_code = ? AND file_id = ? AND ifnull(is_delete, '0') = '0'", projectCode, rowID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func softDeleteWorkspaceFileSubtree(gormDB *gorm.DB, projectCode, targetPath, user string) error {
	now := utils.DateFormat(time.Now(), "")
	return gormDB.Model(&devops.WebshellWorkspaceFile{}).
		Where("project_code = ? AND ifnull(is_delete, '0') = '0' AND (path = ? OR path LIKE ?)", projectCode, targetPath, targetPath+"/%").
		Updates(map[string]interface{}{
			"is_delete":   "1",
			"modify_time": now,
			"modify_user": user,
		}).Error
}

func normalizeWorkspaceParentID(projectRoot, targetPath, rawParentID string) string {
	parentID := strings.TrimSpace(rawParentID)
	if parentID != "" {
		return parentID
	}
	parentPath := path.Dir(targetPath)
	if parentPath == "." || parentPath == "/" || parentPath == projectRoot || parentPath == targetPath {
		return ""
	}
	return stableNodeID(parentPath)
}

func workspaceOpUser(params map[string]interface{}) string {
	user := strings.TrimSpace(gocast.ToString(params["session_user_id"]))
	if user == "" {
		user = strings.TrimSpace(gocast.ToString(params["op_user"]))
	}
	return user
}

func boolToDirFlag(isDir bool) string {
	if isDir {
		return "1"
	}
	return "0"
}

func remotePathExists(client *sftp.Client, targetPath string) (bool, error) {
	_, err := client.Stat(targetPath)
	if err == nil {
		return true, nil
	}
	if isRemoteNotExist(err) {
		return false, nil
	}
	return false, err
}

func isRemoteNotExist(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") || strings.Contains(msg, "does not exist")
}

func toBoolStringValue(raw interface{}, defaultValue bool) bool {
	if raw == nil {
		return defaultValue
	}
	value := strings.TrimSpace(strings.ToLower(gocast.ToString(raw)))
	if value == "" {
		return defaultValue
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}
