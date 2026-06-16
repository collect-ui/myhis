package plugins

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/demdxx/gocast"

	"moon/model/base"
)

const (
	defaultAgentFileReadMaxBytes     = int64(64 * 1024)
	hardAgentFileReadMaxBytes        = int64(256 * 1024)
	defaultAgentFileEditMaxBytes     = int64(256 * 1024)
	hardAgentFileEditMaxBytes        = int64(1024 * 1024)
	defaultAgentSearchMaxResults     = 100
	hardAgentSearchMaxResults        = 500
	hardAgentSearchFileMaxBytes      = int64(8 * 1024 * 1024)
	defaultAgentToolMaxRounds        = 16
	hardAgentToolMaxRounds           = 48
	defaultAgentCommandTimeoutMS     = 30000
	hardAgentCommandTimeoutMS        = 180000
	defaultAgentCommandMaxOutput     = 96 * 1024
	hardAgentCommandMaxOutput        = 512 * 1024
	hardAgentProviderToolOutputBytes = 96 * 1024
	hardAgentProviderToolInputBytes  = 256 * 1024
	minAgentProviderToolOutputBytes  = 4 * 1024
	agentToolOutputPreviewBytes      = 24 * 1024
	agentGrepLineMaxBytes            = 4096
	agentGrepContextLineMaxBytes     = 2048
	agentFileReadToolName            = "read_project_file"
	agentCodexCLIGlobToolName        = "glob"
	agentCodexCLIGrepToolName        = "grep"
	agentCodexCLIEditToolName        = "edit"
	agentCodexCLIDeleteToolName      = "delete_file"
	agentRunCommandToolName          = "run_command"
	agentBrowserCheckToolName        = "browser_check"
	agentImageInspectToolName        = "inspect_image"

	legacyAgentCodexCLIGlobToolName   = "codexcli_glob"
	legacyAgentCodexCLIGrepToolName   = "codexcli_grep"
	legacyAgentCodexCLIEditToolName   = "codexcli_edit"
	legacyAgentCodexCLIDeleteToolName = "codexcli_delete_file"
)

type agentToolPolicy struct {
	Enabled                  bool
	LocalProjectFileRead     bool
	LocalProjectFileSearch   bool
	LocalProjectFileMutation bool
	LocalProjectFileDelete   bool
	LocalProjectCommand      bool
	BrowserValidation        bool
	ImageInspection          bool
	DeleteRequiresConfirm    bool
	AllowedRoots             []string
	MaxBytes                 int64
	MaxEditBytes             int64
	MaxSearchResults         int
	MaxToolRounds            int
	MaxCommandOutputBytes    int
}

type agentToolCall struct {
	ID        string `json:"id,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type agentToolResult struct {
	CallID     string `json:"call_id,omitempty"`
	Name       string `json:"name,omitempty"`
	Arguments  string `json:"arguments,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

type agentCodexCLIEditOperation struct {
	Op            string `json:"op"`
	OldText       string `json:"old_text"`
	NewText       string `json:"new_text"`
	Text          string `json:"text"`
	Anchor        string `json:"anchor"`
	StartLine     int    `json:"start_line"`
	EndLine       int    `json:"end_line"`
	Line          int    `json:"line"`
	ReplaceAll    bool   `json:"replace_all"`
	AllowMultiple bool   `json:"allow_multiple"`
}

func enableAgentCodexCLIToolset(policy *agentToolPolicy) {
	if policy == nil {
		return
	}
	policy.LocalProjectFileRead = true
	policy.LocalProjectFileSearch = true
	policy.LocalProjectFileMutation = true
	policy.LocalProjectFileDelete = true
}

func enableAgentValidationToolset(policy *agentToolPolicy) {
	if policy == nil {
		return
	}
	policy.LocalProjectCommand = true
	policy.BrowserValidation = true
	policy.ImageInspection = true
}

func resolveAgentToolPolicy(session *base.AgentSession, requestData map[string]interface{}) agentToolPolicy {
	policy := agentToolPolicy{
		MaxBytes:              defaultAgentFileReadMaxBytes,
		MaxEditBytes:          defaultAgentFileEditMaxBytes,
		MaxSearchResults:      defaultAgentSearchMaxResults,
		MaxCommandOutputBytes: defaultAgentCommandMaxOutput,
		DeleteRequiresConfirm: true,
	}

	rawPolicy := strings.TrimSpace(gocast.ToString(requestData["tool_policy_json"]))
	if rawPolicy == "" && session != nil {
		rawPolicy = strings.TrimSpace(session.ToolPolicyJSON)
	}
	if rawPolicy == "" {
		if session != nil && normalizeScene(session.SceneCode) == "agent_regression_page" {
			policy.Enabled = true
			enableAgentCodexCLIToolset(&policy)
			enableAgentValidationToolset(&policy)
			policy.AllowedRoots = defaultAgentWorkspaceRoots()
		}
		return normalizeAgentToolPolicy(policy)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(rawPolicy), &data); err != nil {
		return policy
	}

	policy.Enabled = true
	if value, ok := data["enable"]; ok {
		policy.Enabled = gocast.ToBool(value)
	}
	if value, ok := data["enabled"]; ok {
		policy.Enabled = gocast.ToBool(value)
	}
	if value, ok := data["local_project_file_read"]; ok {
		policy.LocalProjectFileRead = gocast.ToBool(value)
	}
	if value, ok := data[agentFileReadToolName]; ok {
		policy.LocalProjectFileRead = gocast.ToBool(value)
	}
	for _, key := range []string{"codexcli", "codex_cli", "enable_codexcli"} {
		if value, ok := data[key]; ok && gocast.ToBool(value) {
			enableAgentCodexCLIToolset(&policy)
		}
	}
	for _, key := range []string{"validation", "enable_validation", "test_tools", "browser_tools", "playwright"} {
		if value, ok := data[key]; ok && gocast.ToBool(value) {
			enableAgentValidationToolset(&policy)
		}
	}
	for _, key := range []string{
		"local_project_file_search",
		"file_search",
		agentCodexCLIGlobToolName,
		agentCodexCLIGrepToolName,
		legacyAgentCodexCLIGlobToolName,
		legacyAgentCodexCLIGrepToolName,
		"codex_cli_glob",
		"codex_cli_grep",
	} {
		if value, ok := data[key]; ok {
			policy.LocalProjectFileSearch = gocast.ToBool(value)
		}
	}
	for _, key := range []string{
		"local_project_file_mutation",
		"file_mutation",
		"file_edit",
		agentCodexCLIEditToolName,
		legacyAgentCodexCLIEditToolName,
		"codex_cli_edit",
	} {
		if value, ok := data[key]; ok {
			policy.LocalProjectFileMutation = gocast.ToBool(value)
		}
	}
	for _, key := range []string{
		"local_project_file_delete",
		"file_delete",
		"allow_file_delete",
		agentCodexCLIDeleteToolName,
		legacyAgentCodexCLIDeleteToolName,
		"codex_cli_delete_file",
	} {
		if value, ok := data[key]; ok {
			policy.LocalProjectFileDelete = gocast.ToBool(value)
		}
	}
	for _, key := range []string{
		"local_project_command",
		"project_command",
		"run_command",
		"command",
		"shell",
		agentRunCommandToolName,
	} {
		if value, ok := data[key]; ok {
			policy.LocalProjectCommand = gocast.ToBool(value)
		}
	}
	for _, key := range []string{
		"browser_validation",
		"headless_browser",
		"browser_check",
		"run_playwright",
		agentBrowserCheckToolName,
	} {
		if value, ok := data[key]; ok {
			policy.BrowserValidation = gocast.ToBool(value)
		}
	}
	for _, key := range []string{
		"image_inspection",
		"image_validation",
		"inspect_image",
		"screenshot_check",
		agentImageInspectToolName,
	} {
		if value, ok := data[key]; ok {
			policy.ImageInspection = gocast.ToBool(value)
		}
	}
	if value, ok := data["delete_requires_confirmation"]; ok {
		policy.DeleteRequiresConfirm = gocast.ToBool(value)
	}
	if tools, ok := data["tools"].([]interface{}); ok {
		for _, item := range tools {
			toolName := normalizeAgentToolName(gocast.ToString(item))
			switch toolName {
			case normalizeAgentToolName(agentFileReadToolName), "read_file":
				policy.LocalProjectFileRead = true
			case "codexcli", "codex_cli":
				enableAgentCodexCLIToolset(&policy)
			case agentCodexCLIGlobToolName, "file_glob", "find", "file_find", legacyAgentCodexCLIGlobToolName, "codex_cli_glob":
				policy.LocalProjectFileSearch = true
			case agentCodexCLIGrepToolName, "search", "content_search", legacyAgentCodexCLIGrepToolName, "codex_cli_grep":
				policy.LocalProjectFileSearch = true
			case agentCodexCLIEditToolName, "file_edit", "modify", "mutation", legacyAgentCodexCLIEditToolName, "codex_cli_edit":
				policy.LocalProjectFileMutation = true
			case "delete", agentCodexCLIDeleteToolName, legacyAgentCodexCLIDeleteToolName, "codex_cli_delete_file":
				policy.LocalProjectFileDelete = true
			case agentRunCommandToolName, "project_command", "shell", "terminal":
				policy.LocalProjectCommand = true
			case agentBrowserCheckToolName, "browser", "headless_browser", "playwright", "run_playwright":
				policy.BrowserValidation = true
			case agentImageInspectToolName, "image", "image_check", "screenshot_check":
				policy.ImageInspection = true
			}
		}
	}
	if roots := readStringList(data["allowed_roots"]); len(roots) > 0 {
		policy.AllowedRoots = roots
	} else if roots := readStringList(data["project_roots"]); len(roots) > 0 {
		policy.AllowedRoots = roots
	} else if roots := readStringList(data["local_roots"]); len(roots) > 0 {
		policy.AllowedRoots = roots
	}
	if maxBytes := gocast.ToInt64(data["max_bytes"]); maxBytes > 0 {
		policy.MaxBytes = maxBytes
	}
	if maxEditBytes := gocast.ToInt64(data["max_edit_bytes"]); maxEditBytes > 0 {
		policy.MaxEditBytes = maxEditBytes
	}
	if maxSearchResults := gocast.ToInt(data["max_search_results"]); maxSearchResults > 0 {
		policy.MaxSearchResults = maxSearchResults
	}
	for _, key := range []string{"max_tool_rounds", "maxToolRounds", "tool_rounds"} {
		if maxToolRounds := gocast.ToInt(data[key]); maxToolRounds > 0 {
			policy.MaxToolRounds = maxToolRounds
			break
		}
	}
	for _, key := range []string{"max_command_output_bytes", "max_output_bytes", "command_output_bytes"} {
		if maxOutputBytes := gocast.ToInt(data[key]); maxOutputBytes > 0 {
			policy.MaxCommandOutputBytes = maxOutputBytes
			break
		}
	}
	if session != nil && normalizeScene(session.SceneCode) == "agent_regression_page" && policy.Enabled {
		enableAgentCodexCLIToolset(&policy)
		enableAgentValidationToolset(&policy)
		if len(policy.AllowedRoots) == 0 {
			policy.AllowedRoots = defaultAgentWorkspaceRoots()
		}
	}
	return normalizeAgentToolPolicy(policy)
}

func adaptAgentToolPolicyForPrompt(policy agentToolPolicy, inputText string) agentToolPolicy {
	if !policy.Enabled {
		return policy
	}
	text := strings.ToLower(strings.TrimSpace(inputText))
	if text == "" {
		return policy
	}
	if agentPromptRequestsReadOnly(text) {
		policy.LocalProjectFileMutation = false
		policy.LocalProjectFileDelete = false
		if !agentPromptRequestsCommandExecution(text) {
			policy.LocalProjectCommand = false
		}
		if !agentPromptRequestsBrowserValidation(text) {
			policy.BrowserValidation = false
			policy.ImageInspection = false
		}
		return normalizeAgentToolPolicy(policy)
	}
	return policy
}

func agentPromptRequestsReadOnly(text string) bool {
	for _, keyword := range []string{
		"只读", "只查", "不要修改", "不修改", "不要改", "无需修改", "禁止修改",
		"read-only", "readonly", "do not modify", "don't modify", "no edits", "without editing",
	} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func agentPromptRequestsCommandExecution(text string) bool {
	for _, keyword := range []string{
		"执行命令", "运行命令", "跑命令", "跑测试", "运行测试", "执行测试", "跑一下", "运行一下",
		"复测", "自测", "验证一下", "启动服务", "重启", "go test", "go vet", "npm ", "pnpm ", "yarn ",
		"curl ", "sqlite3 ", "playwright", "browser_check", "run_command",
	} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func agentPromptRequestsBrowserValidation(text string) bool {
	for _, keyword := range []string{
		"浏览器", "页面验证", "截图", "视觉", "ui", "dom", "playwright", "browser", "console.error", "pageerror",
	} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func normalizeAgentToolPolicy(policy agentToolPolicy) agentToolPolicy {
	if !policy.Enabled {
		return policy
	}
	if !policy.LocalProjectFileRead &&
		!policy.LocalProjectFileSearch &&
		!policy.LocalProjectFileMutation &&
		!policy.LocalProjectFileDelete &&
		!policy.LocalProjectCommand &&
		!policy.BrowserValidation &&
		!policy.ImageInspection {
		policy.Enabled = false
		return policy
	}
	if policy.MaxBytes <= 0 {
		policy.MaxBytes = defaultAgentFileReadMaxBytes
	}
	if policy.MaxBytes > hardAgentFileReadMaxBytes {
		policy.MaxBytes = hardAgentFileReadMaxBytes
	}
	if policy.MaxEditBytes <= 0 {
		policy.MaxEditBytes = defaultAgentFileEditMaxBytes
	}
	if policy.MaxEditBytes > hardAgentFileEditMaxBytes {
		policy.MaxEditBytes = hardAgentFileEditMaxBytes
	}
	if policy.MaxSearchResults <= 0 {
		policy.MaxSearchResults = defaultAgentSearchMaxResults
	}
	if policy.MaxSearchResults > hardAgentSearchMaxResults {
		policy.MaxSearchResults = hardAgentSearchMaxResults
	}
	if policy.MaxToolRounds <= 0 {
		policy.MaxToolRounds = defaultAgentToolMaxRounds
	}
	if policy.MaxToolRounds > hardAgentToolMaxRounds {
		policy.MaxToolRounds = hardAgentToolMaxRounds
	}
	if policy.MaxCommandOutputBytes <= 0 {
		policy.MaxCommandOutputBytes = defaultAgentCommandMaxOutput
	}
	if policy.MaxCommandOutputBytes > hardAgentCommandMaxOutput {
		policy.MaxCommandOutputBytes = hardAgentCommandMaxOutput
	}
	if policy.anyLocalToolEnabled() && len(policy.AllowedRoots) == 0 {
		policy.AllowedRoots = defaultAgentWorkspaceRoots()
	}
	policy.AllowedRoots = normalizeAgentAllowedRoots(policy.AllowedRoots)
	if policy.anyLocalToolEnabled() && len(policy.AllowedRoots) == 0 {
		policy.Enabled = false
	}
	return policy
}

func (policy agentToolPolicy) anyLocalToolEnabled() bool {
	return policy.LocalProjectFileRead ||
		policy.LocalProjectFileSearch ||
		policy.LocalProjectFileMutation ||
		policy.LocalProjectFileDelete ||
		policy.LocalProjectCommand ||
		policy.BrowserValidation ||
		policy.ImageInspection
}

func normalizeAgentToolName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

func canonicalAgentToolName(name string) string {
	switch normalizeAgentToolName(name) {
	case normalizeAgentToolName(agentFileReadToolName), "read_file":
		return agentFileReadToolName
	case agentCodexCLIGlobToolName, legacyAgentCodexCLIGlobToolName, "codex_cli_glob":
		return agentCodexCLIGlobToolName
	case agentCodexCLIGrepToolName, legacyAgentCodexCLIGrepToolName, "codex_cli_grep":
		return agentCodexCLIGrepToolName
	case agentCodexCLIEditToolName, legacyAgentCodexCLIEditToolName, "codex_cli_edit":
		return agentCodexCLIEditToolName
	case agentCodexCLIDeleteToolName, legacyAgentCodexCLIDeleteToolName, "codex_cli_delete_file":
		return agentCodexCLIDeleteToolName
	case agentRunCommandToolName, "project_command", "shell", "terminal":
		return agentRunCommandToolName
	case agentBrowserCheckToolName, "browser", "headless_browser", "playwright", "run_playwright":
		return agentBrowserCheckToolName
	case agentImageInspectToolName, "image", "image_check", "screenshot_check":
		return agentImageInspectToolName
	default:
		return strings.TrimSpace(name)
	}
}

func readStringList(value interface{}) []string {
	switch list := value.(type) {
	case []interface{}:
		result := make([]string, 0, len(list))
		for _, item := range list {
			if text := strings.TrimSpace(gocast.ToString(item)); text != "" {
				result = append(result, text)
			}
		}
		return result
	case []string:
		result := make([]string, 0, len(list))
		for _, item := range list {
			if text := strings.TrimSpace(item); text != "" {
				result = append(result, text)
			}
		}
		return result
	case string:
		return splitAgentRootList(list)
	default:
		return nil
	}
}

func splitAgentRootList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == '\n' || r == '\t'
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if text := strings.TrimSpace(part); text != "" {
			result = append(result, text)
		}
	}
	return result
}

func defaultAgentFileReadRoots() []string {
	if roots := splitAgentRootList(os.Getenv("AGENT_FILE_READ_ROOTS")); len(roots) > 0 {
		return roots
	}
	if _, err := os.Stat("/data/project/sport"); err == nil {
		return []string{"/data/project/sport"}
	}
	if cwd, err := os.Getwd(); err == nil {
		return []string{cwd}
	}
	return nil
}

func defaultAgentWorkspaceRoots() []string {
	if roots := splitAgentRootList(os.Getenv("AGENT_FILE_READ_ROOTS")); len(roots) > 0 {
		return roots
	}
	candidates := []string{
		"/data/project/sport",
		"/data/project/ai-study",
		"/data/project/ai-study/backend",
		"/data/project/auto-check",
		"/data/project/collect",
		"/data/project/sport-ui",
		"/data/project/collect-ui",
		"/data/project/moongod-frontend",
		"/data/project/moongod-backend",
		"/data/project/tianshu-platform",
	}
	roots := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			roots = append(roots, candidate)
		}
	}
	if len(roots) > 0 {
		return roots
	}
	return defaultAgentFileReadRoots()
}

func mergeAgentRoots(primary []string, extra []string) []string {
	result := make([]string, 0, len(primary)+len(extra))
	result = append(result, primary...)
	result = append(result, extra...)
	return result
}

func normalizeAgentAllowedRoots(roots []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		absRoot = filepath.Clean(absRoot)
		info, err := os.Stat(absRoot)
		if err != nil || !info.IsDir() {
			continue
		}
		if seen[absRoot] {
			continue
		}
		seen[absRoot] = true
		result = append(result, absRoot)
	}
	return result
}

func (policy agentToolPolicy) toolDefinitions() []interface{} {
	if !policy.Enabled {
		return nil
	}
	tools := make([]interface{}, 0, 8)
	if policy.LocalProjectFileRead {
		tools = append(tools, map[string]interface{}{
			"type":        "function",
			"name":        agentFileReadToolName,
			"description": "Read a UTF-8 text file from the allowed local project roots. Use this when the user asks to inspect source, config, logs, or other project files.",
			"parameters": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path under an allowed root, or a path relative to the first allowed root.",
					},
					"max_bytes": map[string]interface{}{
						"type":        "integer",
						"description": "Optional maximum bytes to return. The server caps this value.",
						"minimum":     1,
						"maximum":     hardAgentFileReadMaxBytes,
					},
					"start_line": map[string]interface{}{
						"type":        "integer",
						"description": "Optional 1-based first line to return.",
						"minimum":     1,
					},
					"end_line": map[string]interface{}{
						"type":        "integer",
						"description": "Optional 1-based last line to return.",
						"minimum":     1,
					},
				},
				"required": []string{"path"},
			},
		})
	}
	if policy.LocalProjectFileSearch {
		tools = append(tools,
			map[string]interface{}{
				"type":        "function",
				"name":        agentCodexCLIGlobToolName,
				"description": "Discover files and directories under the allowed local project roots using glob filename/path patterns. Use this when you know the filename shape but not the exact location.",
				"parameters": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]interface{}{
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "Glob pattern matched against slash-style relative paths. Supports *, ?, character groups, braces, and ** recursion. Example: plugins/**/*.go",
						},
						"root_dir": map[string]interface{}{
							"type":        "string",
							"description": "Optional root directory under an allowed root. Defaults to the first allowed root.",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Compatibility alias for root_dir.",
						},
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Optional compatibility filename/path fuzzy filter. It does not search file contents.",
						},
						"recursive": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to recurse into subdirectories. Defaults to true.",
						},
						"include_dirs": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to include directories in results. Defaults to false.",
						},
						"include_files": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to include files in results. Defaults to true.",
						},
						"include_hidden": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to search hidden directories/files. Defaults to false; .git is always skipped.",
						},
						"follow_symlinks": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to follow symbolic links. Defaults to false.",
						},
						"max_depth": map[string]interface{}{
							"type":        "integer",
							"description": "Optional maximum traversal depth relative to root_dir.",
							"minimum":     1,
						},
						"exclude_dirs": map[string]interface{}{
							"type":        "array",
							"description": "Directory names or glob patterns to exclude.",
							"items":       map[string]interface{}{"type": "string"},
						},
						"exclude_patterns": map[string]interface{}{
							"type":        "array",
							"description": "Glob patterns to exclude.",
							"items":       map[string]interface{}{"type": "string"},
						},
						"max_results": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum result count.",
							"minimum":     1,
							"maximum":     hardAgentSearchMaxResults,
						},
					},
				},
			},
			map[string]interface{}{
				"type":        "function",
				"name":        agentCodexCLIGrepToolName,
				"description": "Search UTF-8 text file contents under the allowed local project roots. Supports fixed strings, regex, whole-word matching, context lines, file type filters, and glob exclusions.",
				"parameters": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]interface{}{
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "Search text or regex pattern.",
						},
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Alias for pattern.",
						},
						"paths": map[string]interface{}{
							"type":        "array",
							"description": "Optional files/directories under allowed roots. Defaults to the first allowed root.",
							"items":       map[string]interface{}{"type": "string"},
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Compatibility single path under an allowed root.",
						},
						"glob": map[string]interface{}{
							"type":        "string",
							"description": "Optional file glob filter such as **/*.go or collect/**/*.json. Leading ! entries are treated as exclusions.",
						},
						"literal": map[string]interface{}{
							"type":        "boolean",
							"description": "Compatibility alias for regex=false.",
						},
						"regex": map[string]interface{}{
							"type":        "boolean",
							"description": "Treat pattern as a regular expression. Defaults to false.",
						},
						"case_sensitive": map[string]interface{}{
							"type":        "boolean",
							"description": "Use case-sensitive matching. Defaults to false.",
						},
						"whole_word": map[string]interface{}{
							"type":        "boolean",
							"description": "Require whole-word matches.",
						},
						"multiline": map[string]interface{}{
							"type":        "boolean",
							"description": "Allow matches to span multiple lines.",
						},
						"fuzzy": map[string]interface{}{
							"type":        "boolean",
							"description": "Compatibility mode for fuzzy content matching.",
						},
						"context_before": map[string]interface{}{
							"type":        "integer",
							"description": "Number of lines before each match.",
							"minimum":     0,
							"maximum":     5,
						},
						"context_after": map[string]interface{}{
							"type":        "integer",
							"description": "Number of lines after each match.",
							"minimum":     0,
							"maximum":     5,
						},
						"context_lines": map[string]interface{}{
							"type":        "integer",
							"description": "Compatibility shortcut for context_before and context_after.",
							"minimum":     0,
							"maximum":     5,
						},
						"max_files": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of files to scan.",
							"minimum":     1,
						},
						"file_types": map[string]interface{}{
							"type":        "array",
							"description": "Limit search to language/file type aliases such as py, js, ts, go, java, md, json, yml.",
							"items":       map[string]interface{}{"type": "string"},
						},
						"exclude_patterns": map[string]interface{}{
							"type":        "array",
							"description": "Glob patterns to exclude.",
							"items":       map[string]interface{}{"type": "string"},
						},
						"include_hidden": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to search hidden files/directories. Defaults to false; .git is always skipped.",
						},
						"follow_symlinks": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to follow symbolic links. Defaults to false.",
						},
						"binary_skip": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to skip binary files. Defaults to true.",
						},
						"max_results": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum result count.",
							"minimum":     1,
							"maximum":     hardAgentSearchMaxResults,
						},
					},
				},
			},
		)
	}
	if policy.LocalProjectFileMutation {
		tools = append(tools, map[string]interface{}{
			"type":        "function",
			"name":        agentCodexCLIEditToolName,
			"description": "Incrementally edit a UTF-8 text file under allowed roots using exact replace, anchor insertion, line range replacement, or line range deletion. Prefer small exact edits.",
			"parameters": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path under an allowed root, or path relative to the first allowed root.",
					},
					"dry_run": map[string]interface{}{
						"type":        "boolean",
						"description": "Preview the edit without writing the file.",
					},
					"create": map[string]interface{}{
						"type":        "boolean",
						"description": "Allow creating a new file when the path does not exist.",
					},
					"operations": map[string]interface{}{
						"type":        "array",
						"description": "Ordered edit operations. Supported op values: replace, insert_before, insert_after, append, prepend, delete_range, replace_range.",
						"items": map[string]interface{}{
							"type":                 "object",
							"additionalProperties": true,
						},
					},
				},
				"required": []string{"path", "operations"},
			},
		})
	}
	if policy.LocalProjectFileDelete {
		tools = append(tools, map[string]interface{}{
			"type":        "function",
			"name":        agentCodexCLIDeleteToolName,
			"description": "Delete a single non-critical file under allowed roots. Deletion requires explicit confirmation text returned by a prior dry run or failed delete attempt. Critical project/system files are refused.",
			"parameters": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path under an allowed root, or path relative to the first allowed root.",
					},
					"dry_run": map[string]interface{}{
						"type":        "boolean",
						"description": "Return deletion risk and required confirmation without deleting.",
					},
					"confirmation": map[string]interface{}{
						"type":        "string",
						"description": "Required exact confirmation phrase. The tool returns the required phrase when confirmation is missing.",
					},
				},
				"required": []string{"path"},
			},
		})
	}
	if policy.LocalProjectCommand {
		tools = append(tools, map[string]interface{}{
			"type":        "function",
			"name":        agentRunCommandToolName,
			"description": "Run a non-interactive test, build, validation, or diagnostic command in an allowed local workspace. Use it for go test, npm/node Playwright scripts, curl checks, and repository validation after edits. The server enforces workspace, timeout, output, and destructive-command guards.",
			"parameters": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"cmd": map[string]interface{}{
						"type":        "string",
						"description": "Shell command to run non-interactively. Keep it focused; do not start long-lived servers here.",
					},
					"workdir": map[string]interface{}{
						"type":        "string",
						"description": "Directory under an allowed root. Defaults to the first allowed root.",
					},
					"timeout_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Command timeout in milliseconds. The server caps this value.",
						"minimum":     1000,
						"maximum":     hardAgentCommandTimeoutMS,
					},
					"max_output_bytes": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum stdout/stderr bytes returned. The server caps this value.",
						"minimum":     1024,
						"maximum":     hardAgentCommandMaxOutput,
					},
					"env": map[string]interface{}{
						"type":                 "object",
						"description":          "Optional simple environment overrides, for example NO_PROXY or VERIFY_BASE_URL.",
						"additionalProperties": map[string]interface{}{"type": "string"},
					},
					"allow_nonzero": map[string]interface{}{
						"type":        "boolean",
						"description": "Return success even when the process exits non-zero. Defaults to false.",
					},
				},
				"required": []string{"cmd"},
			},
		})
	}
	if policy.BrowserValidation {
		tools = append(tools, map[string]interface{}{
			"type":        "function",
			"name":        agentBrowserCheckToolName,
			"description": "Run a built-in Playwright Chromium headless browser check for a URL: optional login, console/page/request error capture, DOM text assertions, screenshot capture, and screenshot image sanity metrics.",
			"parameters": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "Target URL to open and verify.",
					},
					"login_url": map[string]interface{}{
						"type":        "string",
						"description": "Optional login URL opened before the target URL.",
					},
					"username": map[string]interface{}{
						"type":        "string",
						"description": "Optional username for login form.",
					},
					"password": map[string]interface{}{
						"type":        "string",
						"description": "Optional password for login form.",
					},
					"expected_texts": map[string]interface{}{
						"type":        "array",
						"description": "Visible text snippets that must appear after navigation.",
						"items":       map[string]interface{}{"type": "string"},
					},
					"forbidden_texts": map[string]interface{}{
						"type":        "array",
						"description": "Visible text snippets that must not appear.",
						"items":       map[string]interface{}{"type": "string"},
					},
					"selectors": map[string]interface{}{
						"type":        "array",
						"description": "CSS selectors whose match counts should be returned.",
						"items":       map[string]interface{}{"type": "string"},
					},
					"screenshot_path": map[string]interface{}{
						"type":        "string",
						"description": "Optional PNG path under an allowed root. Defaults to .tmp-agent/browser-check-*.png.",
					},
					"viewport_width": map[string]interface{}{
						"type":        "integer",
						"description": "Viewport width. Defaults to 1440.",
						"minimum":     320,
					},
					"viewport_height": map[string]interface{}{
						"type":        "integer",
						"description": "Viewport height. Defaults to 980.",
						"minimum":     320,
					},
					"wait_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Extra wait after navigation. Defaults to 2500.",
						"minimum":     0,
						"maximum":     30000,
					},
					"timeout_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Navigation/action timeout. Defaults to 30000 and is capped by the server.",
						"minimum":     1000,
						"maximum":     hardAgentCommandTimeoutMS,
					},
				},
				"required": []string{"url"},
			},
		})
	}
	if policy.ImageInspection {
		tools = append(tools, map[string]interface{}{
			"type":        "function",
			"name":        agentImageInspectToolName,
			"description": "Inspect a local screenshot/image under allowed roots and return dimensions, blankness/variance, dominant color ratio, SHA-256, and pass/fail sanity checks.",
			"parameters": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Image path under an allowed root.",
					},
					"min_width": map[string]interface{}{
						"type":        "integer",
						"description": "Optional minimum expected width.",
						"minimum":     1,
					},
					"min_height": map[string]interface{}{
						"type":        "integer",
						"description": "Optional minimum expected height.",
						"minimum":     1,
					},
					"max_dominant_color_ratio": map[string]interface{}{
						"type":        "number",
						"description": "Optional fail threshold for the largest quantized color bucket. Defaults to 0.995.",
						"minimum":     0,
						"maximum":     1,
					},
				},
				"required": []string{"path"},
			},
		})
	}
	return tools
}

func executeAgentToolCall(call agentToolCall, policy agentToolPolicy) agentToolResult {
	start := time.Now()
	toolName := canonicalAgentToolName(call.Name)
	result := agentToolResult{
		CallID:    call.CallID,
		Name:      toolName,
		Arguments: call.Arguments,
	}
	if result.CallID == "" {
		result.CallID = call.ID
	}

	var output map[string]interface{}
	var err error
	switch toolName {
	case agentFileReadToolName:
		output, err = executeAgentReadProjectFile(call.Arguments, policy)
	case agentCodexCLIGlobToolName:
		output, err = executeAgentCodexCLIGlob(call.Arguments, policy)
	case agentCodexCLIGrepToolName:
		output, err = executeAgentCodexCLIGrep(call.Arguments, policy)
	case agentCodexCLIEditToolName:
		output, err = executeAgentCodexCLIEdit(call.Arguments, policy)
	case agentCodexCLIDeleteToolName:
		output, err = executeAgentCodexCLIDeleteFile(call.Arguments, policy)
	case agentRunCommandToolName:
		output, err = executeAgentRunCommand(call.Arguments, policy)
	case agentBrowserCheckToolName:
		output, err = executeAgentBrowserCheck(call.Arguments, policy)
	case agentImageInspectToolName:
		output, err = executeAgentInspectImage(call.Arguments, policy)
	default:
		err = fmt.Errorf("未知工具: %s", call.Name)
	}
	if err != nil {
		result.Error = err.Error()
		if output == nil {
			output = map[string]interface{}{
				"success": false,
				"tool":    toolName,
				"error":   err.Error(),
			}
		} else {
			output["success"] = false
			output["tool"] = toolName
			if _, ok := output["error"]; !ok {
				output["error"] = err.Error()
			}
		}
	}
	result.DurationMS = time.Since(start).Milliseconds()
	result.Output = marshalAgentToolOutput(output)
	return result
}

func marshalAgentToolOutput(output map[string]interface{}) string {
	data, err := json.Marshal(output)
	if err != nil {
		return `{"success":false,"error":"工具结果序列化失败"}`
	}
	if len(data) > hardAgentProviderToolOutputBytes {
		return compactAgentToolOutputForProvider(output, data, hardAgentProviderToolOutputBytes)
	}
	return string(data)
}

func compactAgentToolOutputForProvider(output map[string]interface{}, raw []byte, maxBytes int) string {
	if output == nil {
		output = map[string]interface{}{}
	}
	if maxBytes <= 0 {
		maxBytes = hardAgentProviderToolOutputBytes
	}
	previewBytes := agentToolOutputPreviewBytes
	if previewBytes > maxBytes/2 {
		previewBytes = maxBytes / 2
	}
	if previewBytes < 1024 {
		previewBytes = 1024
	}
	compact := map[string]interface{}{
		"success":          output["success"],
		"tool":             output["tool"],
		"output_truncated": true,
		"original_bytes":   len(raw),
		"message":          "工具输出超过模型单条 function_call_output 安全上限，已保留关键字段和预览；请缩小查询范围或读取更具体的文件/行号。",
		"preview_json":     truncateAgentToolText(string(raw), previewBytes),
	}
	for _, key := range []string{
		"error", "run_error", "cmd", "workdir", "exit_code", "timed_out",
		"stdout_truncated", "stderr_truncated", "path", "relative_path",
		"count", "total", "truncated", "scanned_files", "candidate_files",
		"skipped_binary", "skipped_large", "summary", "duration_ms",
		"url", "final_url", "status", "goto_error", "login", "render_state",
		"expected_texts", "forbidden_texts", "selector_counts",
		"screenshot_path", "artifact_url", "markdown_image", "image",
	} {
		if value, ok := output[key]; ok {
			compact[key] = value
		}
	}
	if preview := agentToolResultsPreview(output["results"]); len(preview) > 0 {
		compact["results_preview"] = preview
	}
	data, err := json.Marshal(compact)
	if err != nil {
		return `{"success":false,"output_truncated":true,"error":"工具结果压缩失败"}`
	}
	if len(data) <= maxBytes {
		return string(data)
	}
	delete(compact, "results_preview")
	compact["preview_json"] = truncateAgentToolText(string(raw), previewBytes/2)
	data, err = json.Marshal(compact)
	if err != nil {
		return `{"success":false,"output_truncated":true,"error":"工具结果压缩失败"}`
	}
	if len(data) <= maxBytes {
		return string(data)
	}
	compact["preview_json"] = truncateAgentToolText(string(raw), 1024)
	data, err = json.Marshal(compact)
	if err != nil {
		return `{"success":false,"output_truncated":true,"error":"工具结果压缩失败"}`
	}
	if len(data) <= maxBytes {
		return string(data)
	}
	delete(compact, "preview_json")
	data, err = json.Marshal(compact)
	if err != nil {
		return `{"success":false,"output_truncated":true,"error":"工具结果压缩失败"}`
	}
	return string(data)
}

func agentToolOutputForProvider(output string) string {
	return agentToolOutputForProviderWithLimit(output, hardAgentProviderToolOutputBytes)
}

func agentToolOutputForProviderWithLimit(output string, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = hardAgentProviderToolOutputBytes
	}
	if maxBytes < minAgentProviderToolOutputBytes {
		return agentToolOutputBudgetNotice(output)
	}
	if len([]byte(output)) <= maxBytes {
		return output
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err == nil && parsed != nil {
		return compactAgentToolOutputForProvider(parsed, []byte(output), maxBytes)
	}
	previewBytes := agentToolOutputPreviewBytes
	if previewBytes > maxBytes/2 {
		previewBytes = maxBytes / 2
	}
	if previewBytes < 1024 {
		previewBytes = 1024
	}
	compact := map[string]interface{}{
		"output_truncated": true,
		"original_bytes":   len([]byte(output)),
		"message":          "工具输出超过模型单条 function_call_output 安全上限，已截断为预览。",
		"preview":          truncateAgentToolText(output, previewBytes),
	}
	data, err := json.Marshal(compact)
	if err != nil {
		return truncateAgentToolText(output, maxBytes)
	}
	return string(data)
}

func agentToolOutputBudgetNotice(output string) string {
	data, err := json.Marshal(map[string]interface{}{
		"output_truncated": true,
		"original_bytes":   len([]byte(output)),
		"message":          "本轮模型请求的工具输出总量已达到安全预算，已省略该工具的详细输出；请缩小查询范围或读取更具体的文件/行号。",
	})
	if err != nil {
		return `{"output_truncated":true,"message":"工具输出已省略"}`
	}
	return string(data)
}

func agentToolResultsPreview(raw interface{}) []map[string]interface{} {
	rows, ok := raw.([]map[string]interface{})
	if !ok || len(rows) == 0 {
		return nil
	}
	limit := len(rows)
	if limit > 5 {
		limit = 5
	}
	preview := make([]map[string]interface{}, 0, limit)
	for _, row := range rows[:limit] {
		item := map[string]interface{}{}
		for _, key := range []string{"relative_path", "file", "path", "line", "line_number", "column", "status", "success"} {
			if value, ok := row[key]; ok {
				item[key] = value
			}
		}
		if text := gocast.ToString(row["text"]); text != "" {
			item["text"] = truncateAgentToolText(text, agentGrepLineMaxBytes)
		}
		if content := gocast.ToString(row["content"]); content != "" {
			item["content"] = truncateAgentToolText(content, agentGrepLineMaxBytes)
		}
		preview = append(preview, item)
	}
	return preview
}

type agentCappedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
	discarded int
}

func newAgentCappedBuffer(limit int) *agentCappedBuffer {
	if limit <= 0 {
		limit = defaultAgentCommandMaxOutput
	}
	return &agentCappedBuffer{limit: limit}
}

func (buf *agentCappedBuffer) Write(p []byte) (int, error) {
	if buf == nil {
		return len(p), nil
	}
	remaining := buf.limit - buf.buffer.Len()
	if remaining > 0 {
		if len(p) <= remaining {
			_, _ = buf.buffer.Write(p)
		} else {
			_, _ = buf.buffer.Write(p[:remaining])
			buf.truncated = true
			buf.discarded += len(p) - remaining
		}
	} else {
		buf.truncated = true
		buf.discarded += len(p)
	}
	return len(p), nil
}

func (buf *agentCappedBuffer) String() string {
	if buf == nil {
		return ""
	}
	text := strings.ToValidUTF8(buf.buffer.String(), "")
	if buf.truncated {
		text += fmt.Sprintf("\n\n[output truncated: discarded %d bytes]", buf.discarded)
	}
	return text
}

func (buf *agentCappedBuffer) Truncated() bool {
	return buf != nil && buf.truncated
}

func executeAgentRunCommand(arguments string, policy agentToolPolicy) (map[string]interface{}, error) {
	if !policy.Enabled || !policy.LocalProjectCommand {
		return nil, fmt.Errorf("本地命令执行工具未启用")
	}
	var args struct {
		Cmd            string            `json:"cmd"`
		Workdir        string            `json:"workdir"`
		TimeoutMS      int               `json:"timeout_ms"`
		MaxOutputBytes int               `json:"max_output_bytes"`
		Env            map[string]string `json:"env"`
		AllowNonzero   bool              `json:"allow_nonzero"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(arguments)), &args); err != nil {
		return nil, fmt.Errorf("工具参数解析失败: %w", err)
	}
	cmdText := strings.TrimSpace(args.Cmd)
	if cmdText == "" {
		return nil, fmt.Errorf("cmd 不能为空")
	}
	if err := validateAgentCommand(cmdText); err != nil {
		return nil, err
	}
	workdir, root, err := resolveAgentCommandWorkdir(args.Workdir, policy.AllowedRoots)
	if err != nil {
		return nil, err
	}
	timeoutMS := clampAgentToolInt(args.TimeoutMS, 1000, hardAgentCommandTimeoutMS, defaultAgentCommandTimeoutMS)
	maxOutputBytes := clampAgentToolInt(args.MaxOutputBytes, 1024, hardAgentCommandMaxOutput, policy.MaxCommandOutputBytes)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	start := time.Now()
	command := exec.CommandContext(ctx, "/bin/bash", "-lc", cmdText)
	command.Dir = workdir
	command.Env = os.Environ()
	for key, value := range args.Env {
		key = strings.TrimSpace(key)
		if key == "" || strings.Contains(key, "=") || strings.ContainsRune(key, '\x00') {
			continue
		}
		command.Env = append(command.Env, key+"="+value)
	}
	stdoutBuf := newAgentCappedBuffer(maxOutputBytes)
	stderrBuf := newAgentCappedBuffer(maxOutputBytes)
	command.Stdout = stdoutBuf
	command.Stderr = stderrBuf
	err = command.Run()
	durationMS := time.Since(start).Milliseconds()
	timedOut := ctx.Err() == context.DeadlineExceeded
	exitCode := 0
	if err != nil {
		exitCode = -1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	success := err == nil || args.AllowNonzero
	if timedOut {
		success = false
	}
	output := map[string]interface{}{
		"success":          success,
		"tool":             agentRunCommandToolName,
		"cmd":              cmdText,
		"workdir":          workdir,
		"root":             root,
		"exit_code":        exitCode,
		"timed_out":        timedOut,
		"duration_ms":      durationMS,
		"stdout":           stdoutBuf.String(),
		"stderr":           stderrBuf.String(),
		"stdout_truncated": stdoutBuf.Truncated(),
		"stderr_truncated": stderrBuf.Truncated(),
	}
	if err != nil {
		output["error"] = err.Error()
	}
	if !success {
		return output, fmt.Errorf("命令执行失败: exit_code=%d timed_out=%v", exitCode, timedOut)
	}
	return output, nil
}

func validateAgentCommand(cmdText string) error {
	if strings.ContainsRune(cmdText, '\x00') {
		return fmt.Errorf("cmd 包含非法字符")
	}
	if len(cmdText) > 6000 {
		return fmt.Errorf("cmd 过长")
	}
	lower := strings.ToLower(cmdText)
	for _, pattern := range []string{
		"rm -rf", "rm -fr", "git reset --hard", "git clean",
		"mkfs", "dd if=", "shutdown", "reboot", "poweroff",
		"sudo ", " su ", "nohup ", "setsid ", "disown",
		"chmod -r", "chown -r",
	} {
		if strings.Contains(lower, pattern) {
			return fmt.Errorf("命令包含高风险片段，已拒绝: %s", pattern)
		}
	}
	if matched, _ := regexp.MatchString(`(?m)(^|[\s;])&($|\s)`, cmdText); matched {
		return fmt.Errorf("命令不能启动后台进程")
	}
	return nil
}

func executeAgentBrowserCheck(arguments string, policy agentToolPolicy) (map[string]interface{}, error) {
	if !policy.Enabled || !policy.BrowserValidation {
		return nil, fmt.Errorf("无头浏览器验证工具未启用")
	}
	var args struct {
		URL            string   `json:"url"`
		LoginURL       string   `json:"login_url"`
		Username       string   `json:"username"`
		Password       string   `json:"password"`
		ExpectedTexts  []string `json:"expected_texts"`
		ForbiddenTexts []string `json:"forbidden_texts"`
		Selectors      []string `json:"selectors"`
		ScreenshotPath string   `json:"screenshot_path"`
		ViewportWidth  int      `json:"viewport_width"`
		ViewportHeight int      `json:"viewport_height"`
		WaitMS         int      `json:"wait_ms"`
		TimeoutMS      int      `json:"timeout_ms"`
		MaxOutputBytes int      `json:"max_output_bytes"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(arguments)), &args); err != nil {
		return nil, fmt.Errorf("工具参数解析失败: %w", err)
	}
	args.URL = strings.TrimSpace(args.URL)
	if args.URL == "" {
		return nil, fmt.Errorf("url 不能为空")
	}
	screenshotPath := strings.TrimSpace(args.ScreenshotPath)
	if screenshotPath == "" {
		screenshotPath = defaultAgentBrowserScreenshotPath(policy.AllowedRoots)
	}
	absScreenshotPath, screenshotRoot, err := resolveAgentProjectPath(screenshotPath, policy.AllowedRoots)
	if err != nil {
		return nil, fmt.Errorf("screenshot_path 无效: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absScreenshotPath), 0755); err != nil {
		return nil, fmt.Errorf("创建截图目录失败: %w", err)
	}
	workdir, _, err := resolveAgentCommandWorkdir(agentPlaywrightWorkdir(policy.AllowedRoots), policy.AllowedRoots)
	if err != nil {
		return nil, err
	}
	timeoutMS := clampAgentToolInt(args.TimeoutMS, 1000, hardAgentCommandTimeoutMS, defaultAgentCommandTimeoutMS)
	waitMS := clampAgentToolInt(args.WaitMS, 0, 30000, 2500)
	viewportWidth := clampAgentToolInt(args.ViewportWidth, 320, 7680, 1440)
	viewportHeight := clampAgentToolInt(args.ViewportHeight, 320, 4320, 980)
	maxOutputBytes := clampAgentToolInt(args.MaxOutputBytes, 1024, hardAgentCommandMaxOutput, policy.MaxCommandOutputBytes)
	payload := map[string]interface{}{
		"url":             args.URL,
		"login_url":       strings.TrimSpace(args.LoginURL),
		"username":        args.Username,
		"password":        args.Password,
		"expected_texts":  args.ExpectedTexts,
		"forbidden_texts": args.ForbiddenTexts,
		"selectors":       args.Selectors,
		"screenshot_path": absScreenshotPath,
		"viewport_width":  viewportWidth,
		"viewport_height": viewportHeight,
		"wait_ms":         waitMS,
		"timeout_ms":      timeoutMS,
	}
	payloadBytes, _ := json.Marshal(payload)
	scriptPath, cleanup, err := writeAgentBrowserCheckScript(workdir)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMS+10000)*time.Millisecond)
	defer cancel()
	command := exec.CommandContext(ctx, "node", scriptPath)
	command.Dir = workdir
	command.Env = append(os.Environ(), "NO_PROXY=*", "NODE_PATH="+filepath.Join(workdir, "node_modules"))
	command.Stdin = bytes.NewReader(payloadBytes)
	stdoutBuf := newAgentCappedBuffer(maxOutputBytes)
	stderrBuf := newAgentCappedBuffer(maxOutputBytes)
	command.Stdout = stdoutBuf
	command.Stderr = stderrBuf
	start := time.Now()
	runErr := command.Run()
	durationMS := time.Since(start).Milliseconds()
	if ctx.Err() == context.DeadlineExceeded {
		runErr = fmt.Errorf("browser_check timeout")
	}
	result := map[string]interface{}{}
	if text := strings.TrimSpace(stdoutBuf.String()); text != "" {
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			result = map[string]interface{}{
				"success": false,
				"error":   "Playwright 输出不是 JSON: " + err.Error(),
				"stdout":  text,
			}
		}
	}
	if len(result) == 0 {
		result = map[string]interface{}{"success": false}
	}
	result["tool"] = agentBrowserCheckToolName
	result["duration_ms"] = durationMS
	result["playwright_workdir"] = workdir
	result["screenshot_path"] = absScreenshotPath
	result["screenshot_root"] = screenshotRoot
	result["artifact_url"] = agentArtifactURL(absScreenshotPath)
	result["markdown_image"] = agentMarkdownImageForPath(absScreenshotPath, "browser-check screenshot")
	result["stderr"] = stderrBuf.String()
	result["stderr_truncated"] = stderrBuf.Truncated()
	if runErr != nil {
		result["run_error"] = runErr.Error()
	}
	if info, statErr := os.Stat(absScreenshotPath); statErr == nil && !info.IsDir() {
		if imageStats, inspectErr := inspectAgentImagePath(absScreenshotPath, policy.AllowedRoots, 0, 0, 0.995); inspectErr == nil {
			result["image"] = imageStats
			if looksBlank, _ := imageStats["looks_blank"].(bool); looksBlank {
				result["success"] = false
			}
		} else {
			result["image_error"] = inspectErr.Error()
		}
	}
	if ok, _ := result["success"].(bool); !ok {
		if runErr != nil {
			return result, runErr
		}
		return result, fmt.Errorf("browser_check 验证失败")
	}
	return result, nil
}

func executeAgentInspectImage(arguments string, policy agentToolPolicy) (map[string]interface{}, error) {
	if !policy.Enabled || !policy.ImageInspection {
		return nil, fmt.Errorf("图片检查工具未启用")
	}
	var args struct {
		Path                  string  `json:"path"`
		MinWidth              int     `json:"min_width"`
		MinHeight             int     `json:"min_height"`
		MaxDominantColorRatio float64 `json:"max_dominant_color_ratio"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(arguments)), &args); err != nil {
		return nil, fmt.Errorf("工具参数解析失败: %w", err)
	}
	result, err := inspectAgentImagePath(args.Path, policy.AllowedRoots, args.MinWidth, args.MinHeight, args.MaxDominantColorRatio)
	if err != nil {
		return nil, err
	}
	if ok, _ := result["success"].(bool); !ok {
		return result, fmt.Errorf("图片检查未通过")
	}
	return result, nil
}

func executeAgentReadProjectFile(arguments string, policy agentToolPolicy) (map[string]interface{}, error) {
	if !policy.Enabled || !policy.LocalProjectFileRead {
		return nil, fmt.Errorf("本地项目文件读取工具未启用")
	}
	var args struct {
		Path      string `json:"path"`
		MaxBytes  int64  `json:"max_bytes"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(arguments)), &args); err != nil {
		return nil, fmt.Errorf("工具参数解析失败: %w", err)
	}
	targetPath, root, err := resolveAgentReadPath(args.Path, policy.AllowedRoots)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, fmt.Errorf("读取文件状态失败: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("目标是目录，不能按文件读取: %s", targetPath)
	}

	maxBytes := policy.MaxBytes
	if args.MaxBytes > 0 && args.MaxBytes < maxBytes {
		maxBytes = args.MaxBytes
	}
	if maxBytes <= 0 {
		maxBytes = defaultAgentFileReadMaxBytes
	}
	if maxBytes > hardAgentFileReadMaxBytes {
		maxBytes = hardAgentFileReadMaxBytes
	}

	file, err := os.Open(targetPath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	if err := ensureAgentTextFile(file, info.Size(), targetPath); err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("重置文件读取位置失败: %w", err)
	}

	startLine := args.StartLine
	endLine := args.EndLine
	content := ""
	returnedBytes := 0
	returnedLines := 0
	totalLines := 0
	totalLinesKnown := true
	truncated := false
	hasMore := false
	nextStartLine := 0
	readMode := "full"

	if startLine > 0 || endLine > 0 {
		readMode = "line_range"
		var err error
		content, returnedBytes, returnedLines, totalLines, truncated, hasMore, nextStartLine, err = readAgentFileLineRange(file, startLine, endLine, maxBytes)
		if err != nil {
			return nil, fmt.Errorf("读取文件分片失败: %w", err)
		}
		totalLinesKnown = !hasMore
	} else {
		contentBytes, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
		if err != nil {
			return nil, fmt.Errorf("读取文件失败: %w", err)
		}
		truncated = int64(len(contentBytes)) > maxBytes
		if truncated {
			contentBytes = contentBytes[:maxBytes]
			readMode = "head_chunk"
		}
		content = string(contentBytes)
		returnedBytes = len([]byte(content))
		returnedLines = countAgentLines(content)
		totalLines = returnedLines
		totalLinesKnown = !truncated
		hasMore = truncated
		if hasMore {
			nextStartLine = returnedLines + 1
		}
	}
	relPath, _ := filepath.Rel(root, targetPath)
	return map[string]interface{}{
		"success":             true,
		"tool":                agentFileReadToolName,
		"path":                targetPath,
		"relative_path":       relPath,
		"root":                root,
		"size_bytes":          info.Size(),
		"requested_max_bytes": maxBytes,
		"returned_bytes":      returnedBytes,
		"returned_lines":      returnedLines,
		"line_count":          returnedLines,
		"truncated":           truncated,
		"has_more":            hasMore,
		"next_start_line":     nextStartLine,
		"read_mode":           readMode,
		"start_line":          startLine,
		"end_line":            endLine,
		"total_lines":         totalLines,
		"total_lines_known":   totalLinesKnown,
		"content":             content,
	}, nil
}

func ensureAgentTextFile(file *os.File, size int64, targetPath string) error {
	sampleSize := int64(4096)
	if size < sampleSize {
		sampleSize = size
	}
	if sampleSize <= 0 {
		return nil
	}
	sample := make([]byte, sampleSize)
	n, err := file.Read(sample)
	if err != nil && err != io.EOF {
		return err
	}
	if !looksTextContent(sample[:n]) {
		return fmt.Errorf("文件不是可直接返回的 UTF-8 文本: %s", targetPath)
	}
	return nil
}

func readAgentFileLineRange(file *os.File, startLine int, endLine int, maxBytes int64) (string, int, int, int, bool, bool, int, error) {
	if startLine <= 0 {
		startLine = 1
	}
	if maxBytes <= 0 {
		maxBytes = defaultAgentFileReadMaxBytes
	}
	reader := bufio.NewReader(file)
	var builder strings.Builder
	totalLines := 0
	truncated := false
	hasMore := false
	nextStartLine := 0
	for {
		line, readErr := reader.ReadString('\n')
		if len(line) > 0 {
			totalLines++
			inRange := totalLines >= startLine && (endLine <= 0 || totalLines <= endLine)
			if inRange {
				nextSize := int64(builder.Len() + len([]byte(line)))
				if nextSize > maxBytes {
					truncated = true
					hasMore = true
					nextStartLine = totalLines
					break
				}
				builder.WriteString(line)
			}
			if endLine > 0 && totalLines >= endLine {
				if readErr == nil {
					if _, peekErr := reader.Peek(1); peekErr == nil {
						hasMore = true
						nextStartLine = endLine + 1
					}
				}
				break
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return "", 0, 0, totalLines, truncated, hasMore, nextStartLine, readErr
		}
	}
	content := builder.String()
	returnedLines := countAgentLines(content)
	return content, len([]byte(content)), returnedLines, totalLines, truncated, hasMore, nextStartLine, nil
}

func executeAgentCodexCLIGlob(arguments string, policy agentToolPolicy) (map[string]interface{}, error) {
	if !policy.Enabled || !policy.LocalProjectFileSearch {
		return nil, fmt.Errorf("本地文件查找工具未启用")
	}
	start := time.Now()
	var args struct {
		Pattern         string   `json:"pattern"`
		Query           string   `json:"query"`
		RootDir         string   `json:"root_dir"`
		Path            string   `json:"path"`
		MaxResults      int      `json:"max_results"`
		IncludeDirs     bool     `json:"include_dirs"`
		IncludeFiles    *bool    `json:"include_files"`
		IncludeHidden   bool     `json:"include_hidden"`
		Recursive       *bool    `json:"recursive"`
		FollowSymlinks  bool     `json:"follow_symlinks"`
		MaxDepth        int      `json:"max_depth"`
		ExcludeDirs     []string `json:"exclude_dirs"`
		ExcludePatterns []string `json:"exclude_patterns"`
		Exclude         []string `json:"exclude"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(arguments)), &args); err != nil {
		return nil, fmt.Errorf("工具参数解析失败: %w", err)
	}
	pattern := strings.TrimSpace(args.Pattern)
	if pattern == "" {
		return nil, fmt.Errorf("pattern 不能为空")
	}
	includeFiles := true
	if args.IncludeFiles != nil {
		includeFiles = *args.IncludeFiles
	}
	recursive := true
	if args.Recursive != nil {
		recursive = *args.Recursive
	}
	if !recursive && args.MaxDepth <= 0 {
		args.MaxDepth = 1
	}
	maxResults := clampAgentToolInt(args.MaxResults, 1, hardAgentSearchMaxResults, policy.MaxSearchResults)
	searchRoot, root, err := resolveAgentGlobSearchRoot(args.RootDir, args.Path, policy.AllowedRoots)
	if err != nil {
		return nil, err
	}
	matcher, err := makeAgentGlobMatcher(pattern)
	if err != nil {
		return nil, err
	}
	excludePatterns := append([]string{}, args.ExcludePatterns...)
	excludePatterns = append(excludePatterns, args.Exclude...)
	excludeMatcher, err := makeAgentGlobExcludeMatcher(strings.Join(excludePatterns, "\n"))
	if err != nil {
		return nil, err
	}
	excludeDirMatcher, err := makeAgentGlobExcludeMatcher(strings.Join(args.ExcludeDirs, "\n"))
	if err != nil {
		return nil, err
	}
	query := strings.TrimSpace(args.Query)

	results := make([]map[string]interface{}, 0, maxResults)
	truncated := false
	err = walkAgentSearch(searchRoot, args.IncludeHidden, args.FollowSymlinks, args.MaxDepth, func(path string, info fs.FileInfo, depth int) error {
		relRoot, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		relRoot = filepath.ToSlash(relRoot)
		relSearch, relSearchErr := filepath.Rel(searchRoot, path)
		if relSearchErr != nil {
			relSearch = relRoot
		}
		relSearch = filepath.ToSlash(relSearch)
		isDir := info != nil && info.IsDir()
		if isDir && shouldSkipAgentToolPath(relRoot, true, args.IncludeHidden) {
			return filepath.SkipDir
		}
		if !isDir && shouldSkipAgentToolPath(relRoot, false, args.IncludeHidden) {
			return nil
		}
		if isDir && matchesAnyAgentGlob(excludeDirMatcher, relRoot, relSearch, filepath.Base(relRoot)) {
			return filepath.SkipDir
		}
		if matchesAnyAgentGlob(excludeMatcher, relRoot, relSearch, filepath.Base(relRoot)) {
			if isDir {
				return filepath.SkipDir
			}
			return nil
		}
		if isDir && !args.IncludeDirs {
			return nil
		}
		if !isDir && !includeFiles {
			return nil
		}
		if !matchesAnyAgentGlob(matcher, relRoot, relSearch, filepath.Base(relRoot)) {
			return nil
		}
		if query != "" && !agentFuzzyContains(relRoot, query) && !agentFuzzyContains(filepath.Base(relRoot), query) {
			return nil
		}
		item := map[string]interface{}{
			"path":          path,
			"relative_path": relRoot,
			"is_directory":  isDir,
			"extension":     filepath.Ext(path),
		}
		if isDir {
			item["type"] = "directory"
		} else if info != nil {
			item["type"] = "file"
			item["size_bytes"] = info.Size()
		}
		results = append(results, item)
		if len(results) >= maxResults {
			truncated = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("文件查找失败: %w", err)
	}
	sort.SliceStable(results, func(i, j int) bool {
		return gocast.ToString(results[i]["relative_path"]) < gocast.ToString(results[j]["relative_path"])
	})
	output := map[string]interface{}{
		"success":     true,
		"status":      "success",
		"tool":        agentCodexCLIGlobToolName,
		"root":        root,
		"search_root": searchRoot,
		"pattern":     pattern,
		"query":       args.Query,
		"match_mode":  "path_glob",
		"total":       len(results),
		"count":       len(results),
		"truncated":   truncated,
		"matches":     results,
		"results":     results,
		"summary": map[string]interface{}{
			"total_matches":   len(results),
			"truncated":       truncated,
			"search_time_ms":  agentSearchTimeMS(start),
			"search_root":     searchRoot,
			"include_hidden":  args.IncludeHidden,
			"follow_symlinks": args.FollowSymlinks,
		},
		"search_time_ms": agentSearchTimeMS(start),
	}
	if len(results) == 0 {
		output["message"] = fmt.Sprintf("No files matching pattern %q", pattern)
	}
	return output, nil
}

func executeAgentCodexCLIGrep(arguments string, policy agentToolPolicy) (map[string]interface{}, error) {
	if !policy.Enabled || !policy.LocalProjectFileSearch {
		return nil, fmt.Errorf("本地文件内容搜索工具未启用")
	}
	start := time.Now()
	var args struct {
		Pattern         string   `json:"pattern"`
		Query           string   `json:"query"`
		Paths           []string `json:"paths"`
		Path            string   `json:"path"`
		Glob            string   `json:"glob"`
		Literal         bool     `json:"literal"`
		Regex           bool     `json:"regex"`
		CaseSensitive   bool     `json:"case_sensitive"`
		WholeWord       bool     `json:"whole_word"`
		Multiline       bool     `json:"multiline"`
		Fuzzy           bool     `json:"fuzzy"`
		ContextBefore   int      `json:"context_before"`
		ContextAfter    int      `json:"context_after"`
		ContextLines    int      `json:"context_lines"`
		MaxResults      int      `json:"max_results"`
		MaxFiles        int      `json:"max_files"`
		FileTypes       []string `json:"file_types"`
		ExcludePatterns []string `json:"exclude_patterns"`
		Exclude         []string `json:"exclude"`
		IncludeHidden   bool     `json:"include_hidden"`
		FollowSymlinks  bool     `json:"follow_symlinks"`
		BinarySkip      *bool    `json:"binary_skip"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(arguments)), &args); err != nil {
		return nil, fmt.Errorf("工具参数解析失败: %w", err)
	}
	pattern := strings.TrimSpace(args.Pattern)
	if pattern == "" {
		pattern = strings.TrimSpace(args.Query)
	}
	if pattern == "" {
		return nil, fmt.Errorf("pattern 或 query 不能为空")
	}
	maxResults := clampAgentToolInt(args.MaxResults, 1, hardAgentSearchMaxResults, policy.MaxSearchResults)
	if args.ContextLines > 0 {
		if args.ContextBefore <= 0 {
			args.ContextBefore = args.ContextLines
		}
		if args.ContextAfter <= 0 {
			args.ContextAfter = args.ContextLines
		}
	}
	contextBefore := clampAgentToolInt(args.ContextBefore, 0, 5, 0)
	contextAfter := clampAgentToolInt(args.ContextAfter, 0, 5, 0)
	binarySkip := true
	if args.BinarySkip != nil {
		binarySkip = *args.BinarySkip
	}
	searchRoots, root, err := resolveAgentGrepSearchRoots(args.Paths, args.Path, policy.AllowedRoots)
	if err != nil {
		return nil, err
	}
	globMatcher, err := makeAgentGlobIncludeExcludeMatcher(args.Glob, append(args.ExcludePatterns, args.Exclude...))
	if err != nil {
		return nil, err
	}
	fileTypeMatcher := makeAgentGrepFileTypeMatcher(args.FileTypes)
	textMatcher, err := makeAgentGrepTextMatcher(agentGrepMatcherOptions{
		Pattern:       pattern,
		Regex:         args.Regex && !args.Literal,
		CaseSensitive: args.CaseSensitive,
		WholeWord:     args.WholeWord,
		Fuzzy:         args.Fuzzy,
		Multiline:     args.Multiline,
	})
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0, maxResults)
	filesWithMatches := map[string]bool{}
	candidateFiles := 0
	scannedFiles := 0
	skippedBinary := 0
	skippedLarge := 0
	truncated := false
	addMatch := func(path string, relPath string, lineNumber int, column int, absoluteOffset int, lineText string, before []map[string]interface{}, after []map[string]interface{}, submatches []map[string]interface{}) error {
		lineText = truncateAgentToolText(lineText, agentGrepLineMaxBytes)
		item := map[string]interface{}{
			"path":            path,
			"file":            relPath,
			"relative_path":   relPath,
			"line":            lineNumber,
			"line_number":     lineNumber,
			"column":          column,
			"absolute_offset": absoluteOffset,
			"text":            lineText,
			"content":         lineText,
			"submatches":      submatches,
			"context": map[string]interface{}{
				"before": agentContextTextList(before),
				"after":  agentContextTextList(after),
			},
		}
		if before != nil {
			item["before"] = before
		}
		if after != nil {
			item["after"] = after
		}
		results = append(results, item)
		filesWithMatches[relPath] = true
		if len(results) >= maxResults {
			truncated = true
			return filepath.SkipAll
		}
		return nil
	}
	searchFile := func(path string, info fs.FileInfo, searchRoot string) error {
		relPath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		relSearch, relSearchErr := filepath.Rel(searchRoot, path)
		if relSearchErr != nil {
			relSearch = relPath
		}
		relSearch = filepath.ToSlash(relSearch)
		if shouldSkipAgentToolPath(relPath, false, args.IncludeHidden) || shouldSkipAgentGrepFile(relPath) {
			return nil
		}
		if !matchesAnyAgentGlob(globMatcher, relPath, relSearch, filepath.Base(relPath)) || !fileTypeMatcher(relPath) {
			return nil
		}
		if info == nil || info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		if args.MaxFiles > 0 && candidateFiles >= args.MaxFiles {
			truncated = true
			return filepath.SkipAll
		}
		candidateFiles++
		if info.Size() > hardAgentSearchFileMaxBytes {
			skippedLarge++
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if binarySkip && !looksTextContent(data) {
			skippedBinary++
			return nil
		}
		scannedFiles++
		content := string(data)
		if args.Multiline {
			for _, match := range textMatcher.FindAll(content) {
				lineNumber, column := agentLineColumnAtOffset(content, match.Start)
				lines := splitAgentLinesKeepEnd(content)
				lineText := agentLineText(lines, lineNumber)
				before := collectAgentContextLines(lines, lineNumber-1-contextBefore, lineNumber-1)
				after := collectAgentContextLines(lines, lineNumber, lineNumber+contextAfter)
				if err := addMatch(path, relPath, lineNumber, column, match.Start, lineText, before, after, []map[string]interface{}{match.Submatch()}); err != nil {
					return err
				}
			}
			return nil
		}
		lines := splitAgentLinesKeepEnd(content)
		absoluteOffset := 0
		for i, line := range lines {
			lineText := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
			matches := textMatcher.FindAll(lineText)
			if len(matches) == 0 {
				absoluteOffset += len(line)
				continue
			}
			first := matches[0]
			column := -1
			if first.Start >= 0 {
				column = first.Start + 1
			}
			submatches := make([]map[string]interface{}, 0, len(matches))
			for _, match := range matches {
				submatches = append(submatches, match.Submatch())
			}
			before := collectAgentContextLines(lines, i-contextBefore, i)
			after := collectAgentContextLines(lines, i+1, i+1+contextAfter)
			if err := addMatch(path, relPath, i+1, column, absoluteOffset+maxAgentInt(first.Start, 0), lineText, before, after, submatches); err != nil {
				return err
			}
			absoluteOffset += len(line)
		}
		return nil
	}

	for _, searchRoot := range searchRoots {
		info, statErr := os.Stat(searchRoot)
		if statErr != nil {
			return nil, fmt.Errorf("读取搜索路径失败: %w", statErr)
		}
		if !info.IsDir() {
			if err := searchFile(searchRoot, info, filepath.Dir(searchRoot)); err != nil && err != filepath.SkipAll {
				return nil, err
			}
			if truncated {
				break
			}
			continue
		}
		err = walkAgentSearch(searchRoot, args.IncludeHidden, args.FollowSymlinks, 0, func(path string, info fs.FileInfo, depth int) error {
			relPath, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			relPath = filepath.ToSlash(relPath)
			if info != nil && info.IsDir() {
				if shouldSkipAgentToolPath(relPath, true, args.IncludeHidden) {
					return filepath.SkipDir
				}
				return nil
			}
			return searchFile(path, info, searchRoot)
		})
		if err != nil && err != filepath.SkipAll {
			return nil, fmt.Errorf("内容搜索失败: %w", err)
		}
		if truncated {
			break
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		left := gocast.ToString(results[i]["relative_path"])
		right := gocast.ToString(results[j]["relative_path"])
		if left == right {
			return gocast.ToInt(results[i]["line"]) < gocast.ToInt(results[j]["line"])
		}
		return left < right
	})
	groupedResults := buildAgentGrepGroupedResults(results)
	return map[string]interface{}{
		"success":         true,
		"status":          "success",
		"tool":            agentCodexCLIGrepToolName,
		"root":            root,
		"search_roots":    searchRoots,
		"search_root":     strings.Join(searchRoots, ";"),
		"pattern":         pattern,
		"query":           pattern,
		"glob":            args.Glob,
		"match_mode":      "content",
		"count":           len(results),
		"total":           len(results),
		"truncated":       truncated,
		"scanned_files":   scannedFiles,
		"candidate_files": candidateFiles,
		"skipped_binary":  skippedBinary,
		"skipped_large":   skippedLarge,
		"matches":         results,
		"results":         results,
		"grouped_results": groupedResults,
		"files":           groupedResults,
		"summary": map[string]interface{}{
			"total_matches":      len(results),
			"files_with_matches": len(filesWithMatches),
			"search_time_ms":     agentSearchTimeMS(start),
			"scanned_files":      scannedFiles,
			"candidate_files":    candidateFiles,
			"skipped_binary":     skippedBinary,
			"skipped_large":      skippedLarge,
			"truncated":          truncated,
		},
		"search_time_ms": agentSearchTimeMS(start),
	}, nil
}

func executeAgentCodexCLIContentSearch(arguments string, policy agentToolPolicy) (map[string]interface{}, error) {
	return executeAgentCodexCLIGrep(arguments, policy)
}

func executeAgentCodexCLIEdit(arguments string, policy agentToolPolicy) (map[string]interface{}, error) {
	if !policy.Enabled || !policy.LocalProjectFileMutation {
		return nil, fmt.Errorf("本地文件编辑工具未启用")
	}
	var args struct {
		Path       string                       `json:"path"`
		DryRun     bool                         `json:"dry_run"`
		Create     bool                         `json:"create"`
		Operations []agentCodexCLIEditOperation `json:"operations"`
		Edits      []agentCodexCLIEditOperation `json:"edits"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(arguments)), &args); err != nil {
		return nil, fmt.Errorf("工具参数解析失败: %w", err)
	}
	if len(args.Operations) == 0 {
		args.Operations = args.Edits
	}
	if len(args.Operations) == 0 {
		return nil, fmt.Errorf("operations 不能为空")
	}
	targetPath, root, err := resolveAgentProjectPath(args.Path, policy.AllowedRoots)
	if err != nil {
		return nil, err
	}
	info, statErr := os.Stat(targetPath)
	exists := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("读取文件状态失败: %w", statErr)
	}
	if !exists && !args.Create {
		return nil, fmt.Errorf("文件不存在，若要创建请设置 create=true: %s", targetPath)
	}
	if exists && info.IsDir() {
		return nil, fmt.Errorf("目标是目录，不能按文件编辑: %s", targetPath)
	}
	if exists && info.Size() > policy.MaxEditBytes {
		return nil, fmt.Errorf("文件超过可编辑大小限制: %d > %d", info.Size(), policy.MaxEditBytes)
	}

	oldContent := ""
	fileMode := fs.FileMode(0644)
	if exists {
		data, err := os.ReadFile(targetPath)
		if err != nil {
			return nil, fmt.Errorf("读取文件失败: %w", err)
		}
		if !looksTextContent(data) {
			return nil, fmt.Errorf("文件不是可编辑的 UTF-8 文本: %s", targetPath)
		}
		oldContent = string(data)
		fileMode = info.Mode().Perm()
	}
	newContent, summaries, err := applyAgentEditOperations(oldContent, args.Operations)
	if err != nil {
		return nil, err
	}
	if int64(len([]byte(newContent))) > policy.MaxEditBytes {
		return nil, fmt.Errorf("编辑后文件超过大小限制: %d > %d", len([]byte(newContent)), policy.MaxEditBytes)
	}
	if !args.DryRun && newContent != oldContent {
		if !exists {
			parent := filepath.Dir(targetPath)
			if parentInfo, err := os.Stat(parent); err != nil || !parentInfo.IsDir() {
				return nil, fmt.Errorf("父目录不存在，不能创建文件: %s", parent)
			}
		}
		if err := os.WriteFile(targetPath, []byte(newContent), fileMode); err != nil {
			return nil, fmt.Errorf("写入文件失败: %w", err)
		}
	}
	relPath, _ := filepath.Rel(root, targetPath)
	return map[string]interface{}{
		"success":         true,
		"tool":            agentCodexCLIEditToolName,
		"path":            targetPath,
		"relative_path":   filepath.ToSlash(relPath),
		"root":            root,
		"dry_run":         args.DryRun,
		"created":         !args.DryRun && !exists,
		"would_create":    !exists,
		"changed":         newContent != oldContent,
		"written":         !args.DryRun && newContent != oldContent,
		"before_bytes":    len([]byte(oldContent)),
		"after_bytes":     len([]byte(newContent)),
		"operation_count": len(args.Operations),
		"operations":      summaries,
		"preview":         truncateAgentToolText(newContent, 4096),
	}, nil
}

func executeAgentCodexCLIDeleteFile(arguments string, policy agentToolPolicy) (map[string]interface{}, error) {
	if !policy.Enabled || !policy.LocalProjectFileDelete {
		return nil, fmt.Errorf("本地文件删除工具未启用")
	}
	var args struct {
		Path         string `json:"path"`
		DryRun       bool   `json:"dry_run"`
		Confirmation string `json:"confirmation"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(arguments)), &args); err != nil {
		return nil, fmt.Errorf("工具参数解析失败: %w", err)
	}
	targetPath, root, err := resolveAgentProjectPath(args.Path, policy.AllowedRoots)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, fmt.Errorf("读取文件状态失败: %w", err)
	}
	relPath, _ := filepath.Rel(root, targetPath)
	relPath = filepath.ToSlash(relPath)
	base := map[string]interface{}{
		"tool":                  agentCodexCLIDeleteToolName,
		"path":                  targetPath,
		"relative_path":         relPath,
		"root":                  root,
		"size_bytes":            info.Size(),
		"confirmation_required": policy.DeleteRequiresConfirm,
		"required_confirmation": "DELETE " + relPath,
	}
	if info.IsDir() {
		base["success"] = false
		base["refused"] = true
		base["message"] = "目录删除不支持，请人工确认后在宿主环境处理"
		return base, nil
	}
	if isAgentCriticalDeletePath(relPath) {
		base["success"] = false
		base["refused"] = true
		base["critical"] = true
		base["message"] = "关键项目/系统文件禁止通过 delete_file 删除"
		return base, nil
	}
	if args.DryRun || (policy.DeleteRequiresConfirm && strings.TrimSpace(args.Confirmation) != "DELETE "+relPath) {
		base["success"] = false
		base["dry_run"] = args.DryRun
		base["deleted"] = false
		base["message"] = "未删除文件；如用户明确确认删除，请再次调用并传入 required_confirmation 的精确文本"
		return base, nil
	}
	if err := os.Remove(targetPath); err != nil {
		return nil, fmt.Errorf("删除文件失败: %w", err)
	}
	base["success"] = true
	base["deleted"] = true
	base["message"] = "文件已删除"
	return base, nil
}

func resolveAgentReadPath(rawPath string, roots []string) (string, string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", "", fmt.Errorf("path 不能为空")
	}
	if len(roots) == 0 {
		return "", "", fmt.Errorf("没有配置允许读取的项目根目录")
	}

	var candidates []string
	if filepath.IsAbs(rawPath) {
		candidates = append(candidates, filepath.Clean(rawPath))
	} else {
		for _, root := range roots {
			candidates = append(candidates, filepath.Clean(filepath.Join(root, rawPath)))
		}
	}
	for _, candidate := range candidates {
		for _, root := range roots {
			if isPathInsideRoot(candidate, root) {
				return candidate, root, nil
			}
		}
	}
	return "", "", fmt.Errorf("路径不在允许读取的项目根目录内: %s", rawPath)
}

func resolveAgentProjectPath(rawPath string, roots []string) (string, string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", "", fmt.Errorf("path 不能为空")
	}
	if len(roots) == 0 {
		return "", "", fmt.Errorf("没有配置允许访问的项目根目录")
	}

	var candidates []string
	if filepath.IsAbs(rawPath) {
		candidates = append(candidates, filepath.Clean(rawPath))
	} else {
		for _, root := range roots {
			candidates = append(candidates, filepath.Clean(filepath.Join(root, rawPath)))
		}
	}
	for _, candidate := range candidates {
		absCandidate, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		absCandidate = filepath.Clean(absCandidate)
		for _, root := range roots {
			if isPathInsideRoot(absCandidate, root) {
				return absCandidate, root, nil
			}
		}
	}
	return "", "", fmt.Errorf("路径不在允许访问的项目根目录内: %s", rawPath)
}

func resolveAgentSearchRoot(rawPath string, roots []string) (string, string, error) {
	if strings.TrimSpace(rawPath) == "" {
		if len(roots) == 0 {
			return "", "", fmt.Errorf("没有配置允许搜索的项目根目录")
		}
		return roots[0], roots[0], nil
	}
	targetPath, root, err := resolveAgentProjectPath(rawPath, roots)
	if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(targetPath); err != nil {
		return "", "", fmt.Errorf("读取搜索路径失败: %w", err)
	}
	return targetPath, root, nil
}

func resolveAgentGlobSearchRoot(rootDir string, path string, roots []string) (string, string, error) {
	searchPath := strings.TrimSpace(rootDir)
	if searchPath == "" {
		searchPath = strings.TrimSpace(path)
	}
	searchRoot, root, err := resolveAgentSearchRoot(searchPath, roots)
	if err != nil {
		return "", "", err
	}
	return searchRoot, root, nil
}

func resolveAgentCommandWorkdir(rawPath string, roots []string) (string, string, error) {
	if len(roots) == 0 {
		return "", "", fmt.Errorf("没有配置允许执行命令的项目根目录")
	}
	if strings.TrimSpace(rawPath) == "" {
		return roots[0], roots[0], nil
	}
	targetPath, root, err := resolveAgentProjectPath(rawPath, roots)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return "", "", fmt.Errorf("读取命令工作目录失败: %w", err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("workdir 必须是目录: %s", targetPath)
	}
	return targetPath, root, nil
}

func agentPlaywrightWorkdir(roots []string) string {
	if override := strings.TrimSpace(os.Getenv("AGENT_PLAYWRIGHT_WORKDIR")); override != "" {
		if _, err := os.Stat(filepath.Join(override, "node_modules", "playwright")); err == nil {
			return override
		}
	}
	candidates := []string{"/data/project/sport", "/data/project/sport-ui", "/data/project/collect-ui"}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "node_modules", "playwright")); err == nil {
			return candidate
		}
	}
	for _, candidate := range []string{"/data/project/sport", "/data/project/sport-ui", "/data/project/collect-ui"} {
		if _, err := os.Stat(filepath.Join(candidate, "package.json")); err == nil {
			return candidate
		}
	}
	for _, root := range roots {
		if _, err := os.Stat(filepath.Join(root, "node_modules", "playwright")); err == nil {
			return root
		}
	}
	return ""
}

func defaultAgentBrowserScreenshotPath(roots []string) string {
	root := ""
	if len(roots) > 0 {
		root = roots[0]
	}
	if root == "" {
		root = "."
	}
	return filepath.Join(root, ".tmp-agent", fmt.Sprintf("browser-check-%d.png", time.Now().UnixNano()))
}

func writeAgentBrowserCheckScript(workdir string) (string, func(), error) {
	script := `
const fs = require('fs');

async function main() {
  let chromium;
  try {
    chromium = require('playwright').chromium;
  } catch (error) {
    console.log(JSON.stringify({success:false, error:'playwright require failed: '+String(error && error.message || error)}));
    return;
  }
  const input = JSON.parse(fs.readFileSync(0, 'utf8') || '{}');
  const timeout = Number(input.timeout_ms || 30000);
  const waitMs = Number(input.wait_ms || 2500);
  const consoleErrors = [];
  const pageErrors = [];
  const requestFailed = [];
  const badResponses = [];
  const criticalTypes = new Set(['document', 'script', 'stylesheet', 'xhr', 'fetch']);
  const browser = await chromium.launch({headless: true});
  const context = await browser.newContext({
    ignoreHTTPSErrors: true,
    viewport: {
      width: Number(input.viewport_width || 1440),
      height: Number(input.viewport_height || 980),
    },
  });
  const page = await context.newPage();
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });
  page.on('pageerror', (err) => pageErrors.push(String(err)));
  page.on('requestfailed', (req) => {
    if (!criticalTypes.has(req.resourceType())) return;
    const failure = req.failure();
    requestFailed.push(req.resourceType()+' '+req.method()+' '+req.url()+' => '+(failure && failure.errorText ? failure.errorText : 'failed'));
  });
  page.on('response', (res) => {
    const req = res.request();
    if (!criticalTypes.has(req.resourceType())) return;
    const status = res.status();
    if (status >= 400) badResponses.push(req.resourceType()+' '+req.method()+' '+res.url()+' => '+status);
  });

  async function gotoUrl(url) {
    let status = 'no_response';
    let error = '';
    try {
      const resp = await page.goto(url, {waitUntil: 'domcontentloaded', timeout});
      status = resp ? String(resp.status()) : 'no_response';
      await page.waitForLoadState('networkidle', {timeout: Math.min(timeout, 10000)}).catch(() => {});
    } catch (err) {
      error = String(err);
    }
    return {status, error};
  }

  async function firstVisible(selectors) {
    for (const selector of selectors) {
      const loc = page.locator(selector).first();
      if (await loc.count().catch(() => 0)) {
        if (await loc.isVisible().catch(() => false)) return loc;
      }
    }
    return null;
  }

  async function performLogin() {
    if (!input.login_url && !input.username && !input.password) return {attempted:false};
    let loginUrl = input.login_url;
    if (!loginUrl) {
      const u = new URL(input.url);
      loginUrl = u.origin + '/collect-ui/#/collect-ui/login';
    }
    const nav = await gotoUrl(loginUrl);
    await page.waitForTimeout(500);
    const username = await firstVisible([
      'input[placeholder="用户名"]',
      'input[name="username"]',
      'input[id*="username"]',
      'input[type="text"]'
    ]);
    const password = await firstVisible([
      'input[placeholder="密码"]',
      'input[type="password"]',
      'input[name="password"]',
      'input[id*="password"]'
    ]);
    let filled = false;
    if (username && input.username !== undefined) {
      await username.fill(String(input.username));
      filled = true;
    }
    if (password && input.password !== undefined) {
      await password.fill(String(input.password));
      filled = true;
    }
    let clicked = false;
    if (filled) {
      for (const selector of ['button:has-text("登 陆")', 'button:has-text("登录")', 'button[type="submit"]', '.login-btn']) {
        const loc = page.locator(selector).first();
        if (await loc.count().catch(() => 0)) {
          await loc.click({timeout}).catch(() => {});
          clicked = true;
          break;
        }
      }
    }
    await page.waitForTimeout(1200);
    return {attempted:true, login_url:loginUrl, status:nav.status, error:nav.error, filled, clicked, final_url: page.url()};
  }

  let login = {attempted:false};
  let nav = {status:'no_response', error:''};
  let result;
  try {
    login = await performLogin();
    nav = await gotoUrl(input.url);
    if (waitMs > 0) await page.waitForTimeout(waitMs);
    const renderState = await page.evaluate(() => {
      const root = document.querySelector('#root');
      const rootHtml = root ? (root.innerHTML || '') : '';
      const visibleText = (document.body && document.body.innerText ? document.body.innerText : '').trim();
      const bodyRect = document.body ? document.body.getBoundingClientRect() : null;
      return {
        root_exists: !!root,
        root_child_count: root ? root.childElementCount : 0,
        root_inner_html_length: rootHtml.length,
        unresolved_router_tag_count: document.querySelectorAll('router').length,
        visible_text_length: visibleText.length,
        visible_text_sample: visibleText.replace(/\s+/g, ' ').slice(0, 1200),
        body_width: bodyRect ? Math.round(bodyRect.width) : 0,
        body_height: bodyRect ? Math.round(bodyRect.height) : 0,
      };
    });
    const bodyText = await page.locator('body').innerText({timeout: 5000}).catch(() => '');
    const expected = (Array.isArray(input.expected_texts) ? input.expected_texts : []).map((text) => ({
      text: String(text),
      found: bodyText.includes(String(text)),
    }));
    const forbidden = (Array.isArray(input.forbidden_texts) ? input.forbidden_texts : []).map((text) => ({
      text: String(text),
      found: bodyText.includes(String(text)),
    }));
    const selectorCounts = {};
    for (const selector of (Array.isArray(input.selectors) ? input.selectors : [])) {
      selectorCounts[String(selector)] = await page.locator(String(selector)).count().catch(() => -1);
    }
    if (input.screenshot_path) {
      await page.screenshot({path: input.screenshot_path, fullPage: true});
    }
    const blankRenderFail = !renderState.root_exists || renderState.root_child_count === 0 || renderState.root_inner_html_length === 0 || renderState.unresolved_router_tag_count > 0;
    const missingExpected = expected.filter((item) => !item.found);
    const foundForbidden = forbidden.filter((item) => item.found);
    const failed = !!nav.error || blankRenderFail || missingExpected.length > 0 || foundForbidden.length > 0 || consoleErrors.length > 0 || pageErrors.length > 0 || requestFailed.length > 0 || badResponses.length > 0;
    result = {
      success: !failed,
      url: input.url,
      status: nav.status,
      goto_error: nav.error,
      final_url: page.url(),
      login,
      render_state: renderState,
      expected_texts: expected,
      forbidden_texts: forbidden,
      selector_counts: selectorCounts,
      blank_render_fail: blankRenderFail,
      console_errors: consoleErrors.slice(0, 20),
      page_errors: pageErrors.slice(0, 20),
      request_failed: requestFailed.slice(0, 20),
      bad_responses: badResponses.slice(0, 20),
      missing_expected_count: missingExpected.length,
      found_forbidden_count: foundForbidden.length,
    };
  } catch (error) {
    result = {success:false, error:String(error && error.stack || error), url: input.url, status: nav.status, goto_error: nav.error, final_url: page.url(), login};
  } finally {
    await context.close().catch(() => {});
    await browser.close().catch(() => {});
  }
  console.log(JSON.stringify(result));
}

main().catch((error) => {
  console.log(JSON.stringify({success:false, error:String(error && error.stack || error)}));
});
`
	scriptDir := strings.TrimSpace(workdir)
	if scriptDir == "" {
		scriptDir = os.TempDir()
	} else {
		scriptDir = filepath.Join(scriptDir, ".tmp-agent")
	}
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return "", func() {}, fmt.Errorf("创建浏览器验证脚本目录失败: %w", err)
	}
	file, err := os.CreateTemp(scriptDir, "agent-browser-check-*.cjs")
	if err != nil {
		return "", func() {}, fmt.Errorf("创建浏览器验证脚本失败: %w", err)
	}
	path := file.Name()
	if _, err := file.WriteString(script); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", func() {}, fmt.Errorf("写入浏览器验证脚本失败: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", func() {}, fmt.Errorf("关闭浏览器验证脚本失败: %w", err)
	}
	cleanup := func() {
		_ = os.Remove(path)
	}
	return path, cleanup, nil
}

func inspectAgentImagePath(rawPath string, roots []string, minWidth int, minHeight int, maxDominantColorRatio float64) (map[string]interface{}, error) {
	targetPath, root, err := resolveAgentReadPath(rawPath, roots)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, fmt.Errorf("读取图片状态失败: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("图片路径是目录: %s", targetPath)
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("读取图片失败: %w", err)
	}
	hash := sha256.Sum256(data)
	reader := bytes.NewReader(data)
	config, format, err := image.DecodeConfig(reader)
	if err != nil {
		return nil, fmt.Errorf("图片格式解析失败: %w", err)
	}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("重置图片读取失败: %w", err)
	}
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("图片解码失败: %w", err)
	}
	if maxDominantColorRatio <= 0 || maxDominantColorRatio > 1 {
		maxDominantColorRatio = 0.995
	}
	stats := sampleAgentImageStats(img)
	looksBlank := stats.DominantColorRatio >= maxDominantColorRatio && stats.LumaStdDev < 8
	success := true
	failures := make([]string, 0, 3)
	if minWidth > 0 && config.Width < minWidth {
		success = false
		failures = append(failures, fmt.Sprintf("width %d < min_width %d", config.Width, minWidth))
	}
	if minHeight > 0 && config.Height < minHeight {
		success = false
		failures = append(failures, fmt.Sprintf("height %d < min_height %d", config.Height, minHeight))
	}
	if looksBlank {
		success = false
		failures = append(failures, "image looks blank or single-color")
	}
	relPath, _ := filepath.Rel(root, targetPath)
	artifactURL := agentArtifactURL(targetPath)
	return map[string]interface{}{
		"success":                   success,
		"tool":                      agentImageInspectToolName,
		"path":                      targetPath,
		"relative_path":             filepath.ToSlash(relPath),
		"root":                      root,
		"artifact_url":              artifactURL,
		"markdown_image":            agentMarkdownImageForPath(targetPath, filepath.Base(targetPath)),
		"format":                    format,
		"size_bytes":                info.Size(),
		"sha256":                    fmt.Sprintf("%x", hash[:]),
		"width":                     config.Width,
		"height":                    config.Height,
		"sample_count":              stats.SampleCount,
		"avg_luma":                  stats.AvgLuma,
		"luma_stddev":               stats.LumaStdDev,
		"dominant_color_ratio":      stats.DominantColorRatio,
		"approx_unique_color_count": stats.UniqueColorCount,
		"dark_pixel_ratio":          stats.DarkPixelRatio,
		"light_pixel_ratio":         stats.LightPixelRatio,
		"looks_blank":               looksBlank,
		"max_dominant_color_ratio":  maxDominantColorRatio,
		"failures":                  failures,
	}, nil
}

type agentImageStats struct {
	SampleCount        int
	AvgLuma            float64
	LumaStdDev         float64
	DominantColorRatio float64
	UniqueColorCount   int
	DarkPixelRatio     float64
	LightPixelRatio    float64
}

func sampleAgentImageStats(img image.Image) agentImageStats {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	totalPixels := width * height
	step := 1
	if totalPixels > 40000 {
		step = int(math.Ceil(math.Sqrt(float64(totalPixels) / 40000.0)))
	}
	buckets := map[uint32]int{}
	count := 0
	darkCount := 0
	lightCount := 0
	sum := 0.0
	sumSq := 0.0
	maxBucket := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r16, g16, b16, a16 := img.At(x, y).RGBA()
			if a16 == 0 {
				r16, g16, b16 = 0xffff, 0xffff, 0xffff
			}
			r8 := float64(r16 >> 8)
			g8 := float64(g16 >> 8)
			b8 := float64(b16 >> 8)
			luma := 0.2126*r8 + 0.7152*g8 + 0.0722*b8
			sum += luma
			sumSq += luma * luma
			if luma < 24 {
				darkCount++
			}
			if luma > 232 {
				lightCount++
			}
			bucket := uint32((r16>>12)&0xf)<<8 | uint32((g16>>12)&0xf)<<4 | uint32((b16>>12)&0xf)
			buckets[bucket]++
			if buckets[bucket] > maxBucket {
				maxBucket = buckets[bucket]
			}
			count++
		}
	}
	if count == 0 {
		return agentImageStats{}
	}
	avg := sum / float64(count)
	variance := sumSq/float64(count) - avg*avg
	if variance < 0 {
		variance = 0
	}
	return agentImageStats{
		SampleCount:        count,
		AvgLuma:            math.Round(avg*100) / 100,
		LumaStdDev:         math.Round(math.Sqrt(variance)*100) / 100,
		DominantColorRatio: math.Round((float64(maxBucket)/float64(count))*10000) / 10000,
		UniqueColorCount:   len(buckets),
		DarkPixelRatio:     math.Round((float64(darkCount)/float64(count))*10000) / 10000,
		LightPixelRatio:    math.Round((float64(lightCount)/float64(count))*10000) / 10000,
	}
}

func resolveAgentGrepSearchRoots(paths []string, singlePath string, roots []string) ([]string, string, error) {
	if strings.TrimSpace(singlePath) != "" {
		paths = append([]string{singlePath}, paths...)
	}
	if len(paths) == 0 {
		paths = []string{""}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(paths))
	selectedRoot := ""
	for _, rawPath := range paths {
		searchRoot, root, err := resolveAgentSearchRoot(rawPath, roots)
		if err != nil {
			return nil, "", err
		}
		if selectedRoot == "" {
			selectedRoot = root
		}
		if seen[searchRoot] {
			continue
		}
		seen[searchRoot] = true
		result = append(result, searchRoot)
	}
	if len(result) == 0 {
		return nil, "", fmt.Errorf("没有可搜索路径")
	}
	return result, selectedRoot, nil
}

func clampAgentToolInt(value int, minValue int, maxValue int, defaultValue int) int {
	if value <= 0 {
		value = defaultValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func splitAgentGlobList(pattern string) []string {
	var result []string
	var builder strings.Builder
	braceDepth := 0
	flush := func() {
		part := strings.TrimSpace(filepath.ToSlash(builder.String()))
		builder.Reset()
		part = strings.TrimPrefix(part, "./")
		part = strings.TrimPrefix(part, "/")
		if part != "" {
			result = append(result, part)
		}
	}
	for _, r := range pattern {
		switch r {
		case '{':
			braceDepth++
			builder.WriteRune(r)
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			builder.WriteRune(r)
		case '\n', '\t', ';', ',':
			if braceDepth == 0 {
				flush()
			} else {
				builder.WriteRune(r)
			}
		default:
			builder.WriteRune(r)
		}
	}
	flush()
	return result
}

func makeAgentGlobMatcher(pattern string) (func(string) bool, error) {
	patterns := splitAgentGlobList(pattern)
	if len(patterns) == 0 {
		return func(string) bool { return true }, nil
	}
	type compiledGlob struct {
		re       *regexp.Regexp
		baseOnly bool
	}
	compiled := make([]compiledGlob, 0, len(patterns))
	for _, pattern := range patterns {
		candidates := []string{pattern}
		if strings.HasPrefix(pattern, "**/") {
			candidates = append(candidates, strings.TrimPrefix(pattern, "**/"))
		}
		for _, candidate := range candidates {
			expr, err := agentGlobPatternToRegexp(candidate)
			if err != nil {
				return nil, fmt.Errorf("glob pattern 无效: %s: %w", candidate, err)
			}
			re, err := regexp.Compile(expr)
			if err != nil {
				return nil, fmt.Errorf("glob pattern 无效: %s: %w", candidate, err)
			}
			compiled = append(compiled, compiledGlob{
				re:       re,
				baseOnly: !strings.Contains(candidate, "/"),
			})
		}
	}
	return func(relPath string) bool {
		relPath = filepath.ToSlash(strings.TrimPrefix(relPath, "./"))
		base := filepath.Base(relPath)
		for _, item := range compiled {
			target := relPath
			if item.baseOnly {
				target = base
			}
			if item.re.MatchString(target) {
				return true
			}
		}
		return false
	}, nil
}

func makeAgentGlobIncludeExcludeMatcher(pattern string, excludePatterns []string) (func(string) bool, error) {
	includePatterns := make([]string, 0)
	excludes := append([]string{}, excludePatterns...)
	for _, item := range splitAgentGlobList(pattern) {
		if strings.HasPrefix(item, "!") {
			excludes = append(excludes, strings.TrimPrefix(item, "!"))
			continue
		}
		includePatterns = append(includePatterns, item)
	}
	includeMatcher, err := makeAgentGlobMatcher(strings.Join(includePatterns, "\n"))
	if err != nil {
		return nil, err
	}
	excludeMatcher, err := makeAgentGlobExcludeMatcher(strings.Join(excludes, "\n"))
	if err != nil {
		return nil, err
	}
	return func(relPath string) bool {
		return includeMatcher(relPath) && !excludeMatcher(relPath)
	}, nil
}

func makeAgentGlobExcludeMatcher(pattern string) (func(string) bool, error) {
	if len(splitAgentGlobList(pattern)) == 0 {
		return func(string) bool { return false }, nil
	}
	return makeAgentGlobMatcher(pattern)
}

func matchesAnyAgentGlob(matcher func(string) bool, relRoot string, relSearch string, base string) bool {
	if matcher == nil {
		return true
	}
	if matcher(relRoot) {
		return true
	}
	if relSearch != "" && relSearch != "." && relSearch != relRoot && matcher(relSearch) {
		return true
	}
	return false
}

func agentGlobPatternToRegexp(pattern string) (string, error) {
	body, err := agentGlobPatternBodyToRegexp(pattern)
	if err != nil {
		return "", err
	}
	return "^" + body + "$", nil
}

func agentGlobPatternBodyToRegexp(pattern string) (string, error) {
	var builder strings.Builder
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					builder.WriteString("(?:.*/)?")
					i += 2
				} else {
					builder.WriteString(".*")
					i++
				}
			} else {
				builder.WriteString("[^/]*")
			}
		case '?':
			builder.WriteString("[^/]")
		case '[':
			end := i + 1
			for end < len(pattern) && pattern[end] != ']' {
				end++
			}
			if end >= len(pattern) {
				return "", fmt.Errorf("unmatched bracket")
			}
			classText := pattern[i+1 : end]
			if classText == "" {
				return "", fmt.Errorf("empty character group")
			}
			builder.WriteString("[")
			if strings.HasPrefix(classText, "!") {
				builder.WriteString("^")
				classText = strings.TrimPrefix(classText, "!")
			}
			for j := 0; j < len(classText); j++ {
				switch classText[j] {
				case '\\', ']':
					builder.WriteByte('\\')
					builder.WriteByte(classText[j])
				default:
					builder.WriteByte(classText[j])
				}
			}
			builder.WriteString("]")
			i = end
		case '{':
			end := findAgentGlobBraceEnd(pattern, i)
			if end < 0 {
				return "", fmt.Errorf("unmatched brace")
			}
			parts := splitAgentGlobBraceAlternatives(pattern[i+1 : end])
			if len(parts) == 0 {
				return "", fmt.Errorf("empty brace alternatives")
			}
			builder.WriteString("(?:")
			for index, part := range parts {
				if index > 0 {
					builder.WriteString("|")
				}
				expr, err := agentGlobPatternBodyToRegexp(part)
				if err != nil {
					return "", err
				}
				builder.WriteString(expr)
			}
			builder.WriteString(")")
			i = end
		default:
			builder.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}
	return builder.String(), nil
}

func findAgentGlobBraceEnd(pattern string, start int) int {
	depth := 0
	for i := start; i < len(pattern); i++ {
		switch pattern[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitAgentGlobBraceAlternatives(value string) []string {
	var result []string
	var builder strings.Builder
	depth := 0
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '{':
			depth++
			builder.WriteByte(value[i])
		case '}':
			if depth > 0 {
				depth--
			}
			builder.WriteByte(value[i])
		case ',':
			if depth == 0 {
				result = append(result, builder.String())
				builder.Reset()
			} else {
				builder.WriteByte(value[i])
			}
		default:
			builder.WriteByte(value[i])
		}
	}
	result = append(result, builder.String())
	return result
}

func shouldSkipAgentToolPath(relPath string, isDir bool, includeHidden bool) bool {
	relPath = filepath.ToSlash(filepath.Clean(relPath))
	if relPath == "." || relPath == "" {
		return false
	}
	parts := strings.Split(relPath, "/")
	for _, part := range parts {
		switch part {
		case ".git", ".svn", ".hg":
			return true
		}
		if !includeHidden && strings.HasPrefix(part, ".") {
			return true
		}
	}
	if !isDir {
		return false
	}
	switch parts[len(parts)-1] {
	case "node_modules", "vendor", "__pycache__", "dist", "build", "target", "bin", "test-results", "coverage":
		return true
	default:
		return false
	}
}

func walkAgentSearch(root string, includeHidden bool, followSymlinks bool, maxDepth int, visit func(path string, info fs.FileInfo, depth int) error) error {
	root = filepath.Clean(root)
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if followSymlinks {
		if info, statErr := os.Stat(root); statErr == nil {
			rootInfo = info
		}
	}
	if !rootInfo.IsDir() {
		return visit(root, rootInfo, 0)
	}
	seenDirs := map[string]bool{}
	if realRoot, err := filepath.EvalSymlinks(root); err == nil {
		seenDirs[filepath.Clean(realRoot)] = true
	}
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			return nil
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			info, infoErr := entry.Info()
			if infoErr != nil {
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				if !followSymlinks {
					continue
				}
				statInfo, statErr := os.Stat(path)
				if statErr != nil {
					continue
				}
				info = statInfo
			}
			nextDepth := depth + 1
			if maxDepth > 0 && nextDepth > maxDepth {
				continue
			}
			visitErr := visit(path, info, nextDepth)
			if visitErr == filepath.SkipAll {
				return filepath.SkipAll
			}
			if info.IsDir() {
				if visitErr == filepath.SkipDir {
					continue
				}
				if followSymlinks {
					realPath, realErr := filepath.EvalSymlinks(path)
					if realErr != nil {
						continue
					}
					realPath = filepath.Clean(realPath)
					if seenDirs[realPath] {
						continue
					}
					seenDirs[realPath] = true
				}
				if err := walk(path, nextDepth); err == filepath.SkipAll {
					return filepath.SkipAll
				}
			}
		}
		return nil
	}
	return walk(root, 0)
}

func shouldSkipAgentGrepFile(relPath string) bool {
	name := strings.ToLower(filepath.Base(filepath.ToSlash(relPath)))
	switch name {
	case ".env":
		return true
	}
	return strings.HasSuffix(name, ".pem") ||
		strings.HasSuffix(name, ".key") ||
		strings.HasSuffix(name, ".exe") ||
		strings.HasSuffix(name, ".dll") ||
		strings.HasSuffix(name, ".so") ||
		strings.HasSuffix(name, ".class") ||
		strings.HasSuffix(name, ".pyc")
}

func makeAgentGrepFileTypeMatcher(types []string) func(string) bool {
	if len(types) == 0 {
		return func(string) bool { return true }
	}
	extensionsByType := map[string][]string{
		"py":         {".py", ".pyi", ".pyx"},
		"python":     {".py", ".pyi", ".pyx"},
		"js":         {".js", ".jsx", ".mjs"},
		"javascript": {".js", ".jsx", ".mjs"},
		"ts":         {".ts", ".tsx"},
		"typescript": {".ts", ".tsx"},
		"go":         {".go"},
		"java":       {".java"},
		"c":          {".c", ".h"},
		"cpp":        {".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"},
		"rust":       {".rs"},
		"rs":         {".rs"},
		"md":         {".md", ".markdown"},
		"markdown":   {".md", ".markdown"},
		"json":       {".json"},
		"yaml":       {".yaml", ".yml"},
		"yml":        {".yaml", ".yml"},
		"sql":        {".sql"},
		"sh":         {".sh", ".bash", ".zsh"},
		"shell":      {".sh", ".bash", ".zsh"},
		"html":       {".html", ".htm"},
		"css":        {".css"},
	}
	allowed := map[string]bool{}
	for _, rawType := range types {
		fileType := strings.TrimSpace(strings.ToLower(rawType))
		if fileType == "" {
			continue
		}
		if strings.HasPrefix(fileType, ".") {
			allowed[fileType] = true
			continue
		}
		if exts, ok := extensionsByType[fileType]; ok {
			for _, ext := range exts {
				allowed[ext] = true
			}
			continue
		}
		allowed["."+fileType] = true
	}
	if len(allowed) == 0 {
		return func(string) bool { return true }
	}
	return func(relPath string) bool {
		return allowed[strings.ToLower(filepath.Ext(relPath))]
	}
}

type agentGrepMatcherOptions struct {
	Pattern       string
	Regex         bool
	CaseSensitive bool
	WholeWord     bool
	Fuzzy         bool
	Multiline     bool
}

type agentGrepTextMatch struct {
	Start int
	End   int
	Text  string
}

func (match agentGrepTextMatch) Submatch() map[string]interface{} {
	return map[string]interface{}{
		"match": map[string]interface{}{
			"text": truncateAgentToolText(match.Text, agentGrepLineMaxBytes),
		},
		"start": match.Start,
		"end":   match.End,
	}
}

type agentGrepTextMatcher struct {
	findAll func(string) []agentGrepTextMatch
}

func (matcher agentGrepTextMatcher) FindAll(text string) []agentGrepTextMatch {
	if matcher.findAll == nil {
		return nil
	}
	return matcher.findAll(text)
}

func makeAgentGrepTextMatcher(options agentGrepMatcherOptions) (agentGrepTextMatcher, error) {
	pattern := options.Pattern
	if options.Fuzzy {
		return agentGrepTextMatcher{findAll: func(text string) []agentGrepTextMatch {
			if agentFuzzyContains(text, pattern) {
				return []agentGrepTextMatch{{Start: -1, End: -1, Text: pattern}}
			}
			return nil
		}}, nil
	}
	if options.Regex {
		expr := pattern
		if options.WholeWord {
			expr = `\b(?:` + expr + `)\b`
		}
		flags := ""
		if !options.CaseSensitive {
			flags += "i"
		}
		if options.Multiline {
			flags += "s"
		}
		if flags != "" {
			expr = "(?" + flags + ":" + expr + ")"
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return agentGrepTextMatcher{}, fmt.Errorf("grep regex 无效: %w", err)
		}
		return agentGrepTextMatcher{findAll: func(text string) []agentGrepTextMatch {
			indices := re.FindAllStringIndex(text, -1)
			if len(indices) == 0 {
				return nil
			}
			matches := make([]agentGrepTextMatch, 0, len(indices))
			for _, pair := range indices {
				if len(pair) != 2 {
					continue
				}
				matches = append(matches, agentGrepTextMatch{
					Start: pair[0],
					End:   pair[1],
					Text:  text[pair[0]:pair[1]],
				})
			}
			return matches
		}}, nil
	}
	if pattern == "" {
		return agentGrepTextMatcher{}, fmt.Errorf("grep pattern 不能为空")
	}
	needle := pattern
	if !options.CaseSensitive {
		needle = strings.ToLower(needle)
	}
	return agentGrepTextMatcher{findAll: func(text string) []agentGrepTextMatch {
		haystack := text
		if !options.CaseSensitive {
			haystack = strings.ToLower(haystack)
		}
		var matches []agentGrepTextMatch
		offset := 0
		for offset <= len(haystack) {
			index := strings.Index(haystack[offset:], needle)
			if index < 0 {
				break
			}
			start := offset + index
			end := start + len(needle)
			if !options.WholeWord || isAgentWholeWordMatch(text, start, end) {
				matchText := ""
				if start >= 0 && end <= len(text) {
					matchText = text[start:end]
				}
				matches = append(matches, agentGrepTextMatch{Start: start, End: end, Text: matchText})
			}
			if end <= offset {
				offset++
			} else {
				offset = end
			}
		}
		return matches
	}}, nil
}

func isAgentWholeWordMatch(text string, start int, end int) bool {
	beforeOK := start <= 0
	afterOK := end >= len(text)
	if !beforeOK {
		r, _ := utf8.DecodeLastRuneInString(text[:start])
		beforeOK = !isAgentWordRune(r)
	}
	if !afterOK {
		r, _ := utf8.DecodeRuneInString(text[end:])
		afterOK = !isAgentWordRune(r)
	}
	return beforeOK && afterOK
}

func isAgentWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func agentContextTextList(lines []map[string]interface{}) []string {
	if len(lines) == 0 {
		return nil
	}
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, gocast.ToString(line["text"]))
	}
	return result
}

func agentLineColumnAtOffset(content string, offset int) (int, int) {
	if offset < 0 {
		return 1, -1
	}
	if offset > len(content) {
		offset = len(content)
	}
	line := 1
	lineStart := 0
	for index, r := range content[:offset] {
		if r == '\n' {
			line++
			lineStart = index + 1
		}
	}
	return line, offset - lineStart + 1
}

func agentLineText(lines []string, lineNumber int) string {
	if lineNumber <= 0 || lineNumber > len(lines) {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSuffix(lines[lineNumber-1], "\n"), "\r")
}

func buildAgentGrepGroupedResults(results []map[string]interface{}) []map[string]interface{} {
	indexByFile := map[string]int{}
	grouped := make([]map[string]interface{}, 0)
	for _, item := range results {
		relPath := gocast.ToString(item["relative_path"])
		groupIndex, ok := indexByFile[relPath]
		if !ok {
			groupIndex = len(grouped)
			indexByFile[relPath] = groupIndex
			grouped = append(grouped, map[string]interface{}{
				"file":    relPath,
				"path":    item["path"],
				"matches": []map[string]interface{}{},
			})
		}
		match := map[string]interface{}{
			"line":        item["line"],
			"line_number": item["line_number"],
			"column":      item["column"],
			"content":     item["content"],
			"text":        item["text"],
			"context":     item["context"],
		}
		if value, ok := item["before"]; ok {
			match["before"] = value
		}
		if value, ok := item["after"]; ok {
			match["after"] = value
		}
		if value, ok := item["submatches"]; ok {
			match["submatches"] = value
		}
		matches := grouped[groupIndex]["matches"].([]map[string]interface{})
		matches = append(matches, match)
		grouped[groupIndex]["matches"] = matches
	}
	return grouped
}

func agentSearchTimeMS(start time.Time) float64 {
	return float64(time.Since(start).Microseconds()) / 1000
}

func maxAgentInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func agentFuzzyContains(value string, query string) bool {
	value = strings.ToLower(value)
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	if strings.Contains(value, query) {
		return true
	}
	index := 0
	queryRunes := []rune(query)
	for _, r := range value {
		if index < len(queryRunes) && r == queryRunes[index] {
			index++
		}
	}
	return index == len(queryRunes)
}

func makeAgentLineMatcher(pattern string, literal bool, caseSensitive bool, fuzzy bool) (func(string) (bool, int), error) {
	if fuzzy {
		return func(line string) (bool, int) {
			if agentFuzzyContains(line, pattern) {
				return true, -1
			}
			return false, -1
		}, nil
	}
	if literal {
		needle := pattern
		if !caseSensitive {
			needle = strings.ToLower(needle)
		}
		return func(line string) (bool, int) {
			haystack := line
			if !caseSensitive {
				haystack = strings.ToLower(haystack)
			}
			index := strings.Index(haystack, needle)
			if index < 0 {
				return false, -1
			}
			return true, index + 1
		}, nil
	}
	expr := pattern
	if !caseSensitive {
		expr = "(?i:" + pattern + ")"
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("grep regex 无效: %w", err)
	}
	return func(line string) (bool, int) {
		index := re.FindStringIndex(line)
		if len(index) == 0 {
			return false, -1
		}
		return true, index[0] + 1
	}, nil
}

func splitAgentLinesKeepEnd(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.SplitAfter(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func collectAgentContextLines(lines []string, start int, end int) []map[string]interface{} {
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start >= end {
		return nil
	}
	result := make([]map[string]interface{}, 0, end-start)
	for i := start; i < end; i++ {
		text := strings.TrimSuffix(strings.TrimSuffix(lines[i], "\n"), "\r")
		result = append(result, map[string]interface{}{
			"line": i + 1,
			"text": truncateAgentToolText(text, agentGrepContextLineMaxBytes),
		})
	}
	return result
}

func applyAgentEditOperations(content string, operations []agentCodexCLIEditOperation) (string, []map[string]interface{}, error) {
	summaries := make([]map[string]interface{}, 0, len(operations))
	current := content
	for index, op := range operations {
		opName := normalizeAgentToolName(op.Op)
		if opName == "" {
			return "", nil, fmt.Errorf("第 %d 个编辑操作缺少 op", index+1)
		}
		before := current
		summary := map[string]interface{}{
			"index": index + 1,
			"op":    opName,
		}
		switch opName {
		case "replace":
			if op.OldText == "" {
				return "", nil, fmt.Errorf("第 %d 个 replace 操作缺少 old_text", index+1)
			}
			count := strings.Count(current, op.OldText)
			if count == 0 {
				return "", nil, fmt.Errorf("第 %d 个 replace 操作未找到 old_text", index+1)
			}
			if count > 1 && !op.ReplaceAll && !op.AllowMultiple {
				return "", nil, fmt.Errorf("第 %d 个 replace 操作匹配 %d 处，请缩小 old_text 或设置 replace_all=true", index+1, count)
			}
			if op.ReplaceAll || op.AllowMultiple {
				current = strings.ReplaceAll(current, op.OldText, op.NewText)
			} else {
				current = strings.Replace(current, op.OldText, op.NewText, 1)
			}
			summary["matches"] = count
		case "insert_before", "insert_after":
			if op.Anchor == "" {
				return "", nil, fmt.Errorf("第 %d 个 %s 操作缺少 anchor", index+1, opName)
			}
			count := strings.Count(current, op.Anchor)
			if count == 0 {
				return "", nil, fmt.Errorf("第 %d 个 %s 操作未找到 anchor", index+1, opName)
			}
			if count > 1 && !op.AllowMultiple {
				return "", nil, fmt.Errorf("第 %d 个 %s 操作匹配 %d 处，请缩小 anchor 或设置 allow_multiple=true", index+1, opName, count)
			}
			replacement := op.Text + op.Anchor
			if opName == "insert_after" {
				replacement = op.Anchor + op.Text
			}
			replaceCount := 1
			if op.AllowMultiple {
				replaceCount = -1
			}
			current = strings.Replace(current, op.Anchor, replacement, replaceCount)
			summary["matches"] = count
		case "append":
			current += op.Text
		case "prepend":
			current = op.Text + current
		case "delete_range", "replace_range":
			startLine := op.StartLine
			if startLine == 0 {
				startLine = op.Line
			}
			endLine := op.EndLine
			if endLine == 0 {
				endLine = startLine
			}
			replacement := ""
			if opName == "replace_range" {
				replacement = op.Text
			}
			next, err := replaceAgentLineRange(current, startLine, endLine, replacement)
			if err != nil {
				return "", nil, fmt.Errorf("第 %d 个 %s 操作失败: %w", index+1, opName, err)
			}
			current = next
			summary["start_line"] = startLine
			summary["end_line"] = endLine
		default:
			return "", nil, fmt.Errorf("不支持的编辑操作: %s", op.Op)
		}
		summary["changed"] = current != before
		summaries = append(summaries, summary)
	}
	return current, summaries, nil
}

func replaceAgentLineRange(content string, startLine int, endLine int, replacement string) (string, error) {
	if startLine <= 0 || endLine <= 0 || endLine < startLine {
		return "", fmt.Errorf("行范围无效: %d-%d", startLine, endLine)
	}
	lines := splitAgentLinesKeepEnd(content)
	if startLine > len(lines) || endLine > len(lines) {
		return "", fmt.Errorf("行范围超出文件总行数 %d: %d-%d", len(lines), startLine, endLine)
	}
	var builder strings.Builder
	for _, line := range lines[:startLine-1] {
		builder.WriteString(line)
	}
	builder.WriteString(replacement)
	for _, line := range lines[endLine:] {
		builder.WriteString(line)
	}
	return builder.String(), nil
}

func truncateAgentToolText(content string, maxBytes int) string {
	if len([]byte(content)) <= maxBytes {
		return content
	}
	count := 0
	for index, r := range content {
		count += len(string(r))
		if count > maxBytes {
			return content[:index] + "\n...[truncated]"
		}
	}
	return content
}

func isAgentCriticalDeletePath(relPath string) bool {
	relPath = filepath.ToSlash(filepath.Clean(relPath))
	if relPath == "." || relPath == "" || strings.HasPrefix(relPath, "../") {
		return true
	}
	parts := strings.Split(relPath, "/")
	switch parts[0] {
	case ".git", "database", "conf":
		return true
	}
	switch relPath {
	case "AGENTS.md", "go.mod", "go.sum", "main.go", "build.sh", "build-windows.bat", "linux-startup", "linux-shutdown", "shutdown.sh", "run-dev-main":
		return true
	}
	lower := strings.ToLower(relPath)
	return strings.HasSuffix(lower, ".db") ||
		strings.HasSuffix(lower, ".sqlite") ||
		strings.HasSuffix(lower, ".sqlite3") ||
		strings.HasSuffix(lower, ".exe") ||
		strings.HasSuffix(lower, ".dll") ||
		strings.HasSuffix(lower, ".so")
}

func isPathInsideRoot(targetPath string, root string) bool {
	targetPath = filepath.Clean(targetPath)
	root = filepath.Clean(root)
	rel, err := filepath.Rel(root, targetPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func looksTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	if strings.Contains(string(data), "\x00") {
		return false
	}
	return utf8.Valid(data)
}

func countAgentLines(content string) int {
	if content == "" {
		return 0
	}
	lines := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		lines++
	}
	return lines
}

func sliceAgentLines(content string, startLine int, endLine int) string {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if startLine <= 0 {
		startLine = 1
	}
	if endLine <= 0 || endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine || startLine > len(lines) {
		return ""
	}
	return strings.Join(lines[startLine-1:endLine], "")
}
