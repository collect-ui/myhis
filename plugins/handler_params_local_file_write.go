package plugins

import (
	"encoding/json"
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalFileWrite struct {
	templateService.BaseHandler
}

func expandHomePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}

func (si *LocalFileWrite) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()

	targetPath := gocast.ToString(utils.RenderVar(handlerParam.Field, params))
	targetPath = expandHomePath(targetPath)
	if strings.TrimSpace(targetPath) == "" {
		return common.NotOk("目标文件路径不能为空")
	}

	content := gocast.ToString(utils.RenderVar(handlerParam.Value, params))
	if strings.TrimSpace(content) == "" {
		return common.NotOk("写入内容不能为空")
	}

	var jsonObj interface{}
	if err := json.Unmarshal([]byte(content), &jsonObj); err != nil {
		return common.NotOk("auth.json 内容不是合法 JSON: " + err.Error())
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0700); err != nil {
		return common.NotOk("创建目录失败: " + err.Error())
	}

	backupEnabled := true
	if _, ok := params["backup"]; ok {
		backupEnabled = gocast.ToBool(params["backup"])
	}

	backupPath := ""
	if backupEnabled {
		if _, err := os.Stat(targetPath); err == nil {
			originContent, readOk := utils.ReadFileBytes(targetPath)
			if !readOk {
				return common.NotOk("读取原文件失败")
			}
			backupPath = fmt.Sprintf("%s.bak.%s", targetPath, time.Now().Format("20060102150405"))
			if err := os.WriteFile(backupPath, originContent, 0600); err != nil {
				return common.NotOk("写入备份文件失败: " + err.Error())
			}
		}
	}

	tmpPath := fmt.Sprintf("%s.tmp.%d", targetPath, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, []byte(content), 0600); err != nil {
		return common.NotOk("写入临时文件失败: " + err.Error())
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return common.NotOk("替换目标文件失败: " + err.Error())
	}

	result := map[string]interface{}{
		"target_path": targetPath,
		"backup_path": backupPath,
		"bytes":       len(content),
		"updated_at":  time.Now().Format("2006-01-02 15:04:05"),
	}
	return common.Ok(result, "本地文件写入成功")
}
