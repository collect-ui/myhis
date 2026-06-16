package plugins

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/pkg/sftp"
)

func normalizeWorkspaceProjectRoot(raw string) (string, error) {
	root := strings.TrimSpace(raw)
	if root == "" {
		return "", fmt.Errorf("项目目录不能为空")
	}
	if strings.ContainsRune(root, '\x00') {
		return "", fmt.Errorf("项目目录包含非法字符")
	}
	root = path.Clean(root)
	if root == "." || root == "" || root == "/" {
		return "", fmt.Errorf("项目目录[%s]不合法", strings.TrimSpace(raw))
	}
	if !strings.HasPrefix(root, "/") {
		return "", fmt.Errorf("项目目录[%s]必须为绝对路径", root)
	}
	return root, nil
}

func containsUnsafePathSegment(raw string) bool {
	if raw == "" {
		return false
	}
	for _, segment := range strings.Split(strings.ReplaceAll(raw, "\\", "/"), "/") {
		if strings.TrimSpace(segment) == ".." {
			return true
		}
	}
	return false
}

func normalizeWorkspaceTargetPath(projectRoot, rawPath string) (string, error) {
	value := strings.TrimSpace(rawPath)
	if value == "" {
		return "", fmt.Errorf("path不能为空")
	}
	if strings.ContainsRune(value, '\x00') {
		return "", fmt.Errorf("path包含非法字符")
	}
	if containsUnsafePathSegment(value) {
		return "", fmt.Errorf("path包含非法路径穿越片段")
	}
	targetPath := path.Clean(value)
	if targetPath == "." || targetPath == "" {
		return "", fmt.Errorf("path不能为空")
	}
	if !strings.HasPrefix(targetPath, "/") {
		return "", fmt.Errorf("path必须为绝对路径")
	}

	root, err := normalizeWorkspaceProjectRoot(projectRoot)
	if err != nil {
		return "", err
	}
	if targetPath != root && !strings.HasPrefix(targetPath, root+"/") {
		return "", fmt.Errorf("路径[%s]超出项目目录[%s]限制", targetPath, root)
	}
	return targetPath, nil
}

func normalizeWorkspaceTargetPaths(projectRoot string, rawPaths []string) ([]string, error) {
	list := make([]string, 0, len(rawPaths))
	for _, raw := range rawPaths {
		targetPath, err := normalizeWorkspaceTargetPath(projectRoot, raw)
		if err != nil {
			return nil, err
		}
		list = append(list, targetPath)
	}
	return list, nil
}

func ensureRemotePathChainSafe(sftpClient *sftp.Client, projectRoot, targetPath string, allowMissingTail bool) error {
	root, err := normalizeWorkspaceProjectRoot(projectRoot)
	if err != nil {
		return err
	}
	target, err := normalizeWorkspaceTargetPath(root, targetPath)
	if err != nil {
		return err
	}

	current := root
	rootInfo, err := sftpClient.Lstat(current)
	if err != nil {
		if isRemoteNotExist(err) {
			return fmt.Errorf("项目目录[%s]不存在", current)
		}
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("项目目录[%s]为软链接，禁止操作", current)
	}

	if target == root {
		return nil
	}

	relativePath := strings.TrimPrefix(target, root+"/")
	for _, segment := range strings.Split(relativePath, "/") {
		if segment == "" {
			continue
		}
		current = path.Join(current, segment)
		info, statErr := sftpClient.Lstat(current)
		if statErr != nil {
			if allowMissingTail && isRemoteNotExist(statErr) {
				return nil
			}
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("路径[%s]包含软链接，禁止操作", current)
		}
	}
	return nil
}
