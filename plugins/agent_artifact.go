package plugins

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const agentArtifactRoute = "/template_data/agent/artifact"

func RegisterAgentArtifactRoutes(r *gin.Engine) {
	r.GET(agentArtifactRoute, handleAgentArtifact)
	r.HEAD(agentArtifactRoute, handleAgentArtifact)
}

func agentArtifactURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return agentArtifactRoute + "?path=" + url.QueryEscape(path)
}

func agentMarkdownImageForPath(path string, alt string) string {
	artifactURL := agentArtifactURL(path)
	if artifactURL == "" {
		return ""
	}
	alt = strings.TrimSpace(alt)
	if alt == "" {
		alt = filepath.Base(path)
	}
	alt = strings.NewReplacer("[", "(", "]", ")", "\n", " ", "\r", " ").Replace(alt)
	return fmt.Sprintf("![%s](%s)", alt, artifactURL)
}

func handleAgentArtifact(c *gin.Context) {
	rawPath := strings.TrimSpace(c.Query("path"))
	if rawPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "path 不能为空"})
		return
	}
	targetPath, _, err := resolveAgentProjectPath(rawPath, normalizeAgentAllowedRoots(defaultAgentWorkspaceRoots()))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "msg": err.Error()})
		return
	}
	if !isAgentImageArtifactPath(targetPath) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "当前 artifact 预览仅支持图片文件"})
		return
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "msg": err.Error()})
		return
	}
	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "path 不能是目录"})
		return
	}
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	http.ServeFile(c.Writer, c.Request, targetPath)
}

func isAgentImageArtifactPath(path string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	default:
		return false
	}
}
