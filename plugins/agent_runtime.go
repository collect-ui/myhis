package plugins

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	common "github.com/collect-ui/collect/src/collect/common"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	"github.com/demdxx/gocast"
	"github.com/google/uuid"

	"moon/model/base"
)

const (
	agentStatusActive           = "active"
	runStatusQueued             = "queued"
	runStatusRunning            = "running"
	runStatusCompleted          = "completed"
	runStatusFailed             = "failed"
	runStatusCancelled          = "cancelled"
	defaultAgentModel           = "gpt-5-mini"
	defaultChatGPTAuthModel     = "gpt-5.4-mini"
	defaultAgentScene           = "default"
	defaultTriggerType          = "user"
	defaultWorkerID             = "local-agent-worker"
	defaultInstructions         = "You are a Codex CLI style coding agent. Work from evidence in the current repository: inspect files before answering code/config questions, follow repository instructions, keep changes scoped, and report validation results."
	agentProjectContextMaxBytes = 24000
	agentProviderMaxAttempts    = 2
	agentProviderDebugToolName  = "model_request_debug"
	agentProviderDebugMaxBytes  = 512 * 1024
	agentProviderInputMaxBytes  = 16 * 1024
	agentProviderTextMaxBytes   = 2048
	agentProviderDebugMaxItems  = 24
	agentPreflightMaxToolCalls  = 24
	agentPreflightMaxBytes      = 48 * 1024
)

var (
	agentProviderResponseHeaderTimeout   = 30 * time.Second
	agentProviderStreamHeaderTimeout     = 30 * time.Second
	agentProviderNonStreamRequestTimeout = 45 * time.Second
	agentProviderStreamIdleTimeout       = 45 * time.Second
	agentProviderRetryWaitBase           = time.Second
	agentProviderRateLimitRetryWaitMin   = 10 * time.Second
	agentProviderRateLimitRetryWaitMax   = 15 * time.Second
)

type agentProviderResult struct {
	ResponseID    string                   `json:"response_id"`
	OutputText    string                   `json:"output_text"`
	RawJSON       map[string]interface{}   `json:"raw_json,omitempty"`
	Usage         map[string]interface{}   `json:"usage,omitempty"`
	ToolCalls     []agentToolCall          `json:"tool_calls,omitempty"`
	ToolResults   []agentToolResult        `json:"tool_results,omitempty"`
	DebugRequests []map[string]interface{} `json:"debug_requests,omitempty"`
	Mocked        bool                     `json:"mocked"`
}

type agentToolRunError struct {
	Err         error
	ToolResults []agentToolResult
}

func (err *agentToolRunError) Error() string {
	if err == nil || err.Err == nil {
		return ""
	}
	return err.Err.Error()
}

func (err *agentToolRunError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

type agentProviderRequestDebugError struct {
	Err   error
	Debug map[string]interface{}
}

func (err *agentProviderRequestDebugError) Error() string {
	if err == nil || err.Err == nil {
		return ""
	}
	return err.Err.Error()
}

func (err *agentProviderRequestDebugError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

type openAIResponse struct {
	ID         string                     `json:"id"`
	OutputText string                     `json:"output_text"`
	Output     []openAIResponseOutputItem `json:"output,omitempty"`
	Usage      map[string]interface{}     `json:"usage,omitempty"`
	Error      *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type openAIResponseOutputItem struct {
	ID        string                      `json:"id,omitempty"`
	Type      string                      `json:"type,omitempty"`
	CallID    string                      `json:"call_id,omitempty"`
	Name      string                      `json:"name,omitempty"`
	Arguments string                      `json:"arguments,omitempty"`
	Status    string                      `json:"status,omitempty"`
	Content   []openAIResponseContentItem `json:"content,omitempty"`
	Raw       map[string]interface{}      `json:"-"`
}

type openAIResponseContentItem struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type chatGPTCodexEvent struct {
	Type     string                 `json:"type"`
	Delta    string                 `json:"delta"`
	Text     string                 `json:"text"`
	ItemID   string                 `json:"item_id"`
	Item     json.RawMessage        `json:"item,omitempty"`
	Usage    map[string]interface{} `json:"usage,omitempty"`
	Response *struct {
		ID    string                 `json:"id"`
		Usage map[string]interface{} `json:"usage,omitempty"`
	} `json:"response,omitempty"`
}

type agentProviderStreamFunc func(delta string)
type agentToolEventFunc func(result agentToolResult)
type agentProviderLogFunc func(event map[string]interface{})
type agentProviderDebugFunc func(debug map[string]interface{})

type codexAuthFile struct {
	AuthMode     string  `json:"auth_mode"`
	OPENAIAPIKey *string `json:"OPENAI_API_KEY"`
	Tokens       struct {
		AccessToken string `json:"access_token"`
		AccountID   string `json:"account_id"`
	} `json:"tokens"`
}

type agentCredential struct {
	Token     string `json:"token"`
	AccountID string `json:"account_id"`
	Mode      string `json:"mode"`
	Source    string `json:"source"`
}

var (
	agentRuntimeInitOnce       sync.Once
	agentCredentialCacheMu     sync.Mutex
	agentCredentialCache       agentCredential
	agentCredentialCachePath   string
	agentCredentialCacheLoaded bool
	agentRunCancelMu           sync.Mutex
	agentRunCancelMap          = map[string]context.CancelFunc{}
)

type agentProviderHTTPError struct {
	StatusCode int
	Message    string
	RetryAfter string
	RequestID  string
}

func (err *agentProviderHTTPError) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

type agentProviderHTTPStatusError struct {
	Err            error
	StatusCode     int
	StatusText     string
	StatusReceived bool
}

func (err *agentProviderHTTPStatusError) Error() string {
	if err == nil || err.Err == nil {
		return ""
	}
	return err.Err.Error()
}

func (err *agentProviderHTTPStatusError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

type agentProviderResponseHeaderTimeoutError struct {
	Err     error
	Timeout time.Duration
}

func (err *agentProviderResponseHeaderTimeoutError) Error() string {
	if err == nil || err.Err == nil {
		return ""
	}
	return err.Err.Error()
}

func (err *agentProviderResponseHeaderTimeoutError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

func callAgentLowcodeService(service string, params map[string]interface{}) (*common.Result, error) {
	service = strings.TrimSpace(service)
	if service == "" {
		return nil, fmt.Errorf("service 不能为空")
	}
	callParams := make(map[string]interface{}, len(params)+1)
	for key, value := range params {
		callParams[key] = value
	}
	callParams["service"] = service
	ts := templateService.TemplateService{OpUser: "agent_runtime"}
	result := ts.ResultInner(callParams)
	if result == nil {
		return nil, fmt.Errorf("%s 执行失败", service)
	}
	if !result.Success {
		msg := strings.TrimSpace(result.Msg)
		if msg == "" {
			msg = service + " 执行失败"
		}
		return result, fmt.Errorf("%s", msg)
	}
	return result, nil
}

func decodeAgentLowcodeData(data interface{}, target interface{}) error {
	if target == nil {
		return fmt.Errorf("target 不能为空")
	}
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func agentLowcodeAffected(result *common.Result) int64 {
	if result == nil {
		return 0
	}
	if result.Count != 0 {
		return result.Count
	}
	if data, ok := result.Data.(map[string]interface{}); ok {
		return gocast.ToInt64(data["affected"])
	}
	return 0
}

func queryAgentSession(params map[string]interface{}) (*base.AgentSession, bool, error) {
	queryParams := make(map[string]interface{}, len(params)+1)
	for key, value := range params {
		queryParams[key] = value
	}
	queryParams["to_obj"] = true
	result, err := callAgentLowcodeService("agent.session_query", queryParams)
	if err != nil {
		return nil, false, err
	}
	var session base.AgentSession
	if err := decodeAgentLowcodeData(result.Data, &session); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(session.AgentSessionID) == "" {
		return nil, false, nil
	}
	return &session, true, nil
}

func queryAgentRun(params map[string]interface{}) (*base.AgentRun, bool, error) {
	queryParams := make(map[string]interface{}, len(params)+1)
	for key, value := range params {
		queryParams[key] = value
	}
	queryParams["to_obj"] = true
	result, err := callAgentLowcodeService("agent.run_query", queryParams)
	if err != nil {
		return nil, false, err
	}
	var run base.AgentRun
	if err := decodeAgentLowcodeData(result.Data, &run); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(run.AgentRunID) == "" {
		return nil, false, nil
	}
	return &run, true, nil
}

func queryAgentMessageList(params map[string]interface{}) ([]base.AgentMessage, error) {
	result, err := callAgentLowcodeService("agent.message_query", params)
	if err != nil {
		return nil, err
	}
	var messageList []base.AgentMessage
	if err := decodeAgentLowcodeData(result.Data, &messageList); err != nil {
		return nil, err
	}
	return messageList, nil
}

func queryLatestAssistantMessage(agentRunID string) (*base.AgentMessage, bool, error) {
	result, err := callAgentLowcodeService("agent.message_latest_assistant_query", map[string]interface{}{
		"agent_run_id": agentRunID,
	})
	if err != nil {
		return nil, false, err
	}
	var message base.AgentMessage
	if err := decodeAgentLowcodeData(result.Data, &message); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(message.AgentMessageID) == "" {
		return nil, false, nil
	}
	return &message, true, nil
}

func ensureAgentRuntime() {
	agentRuntimeInitOnce.Do(func() {
		go startAgentRunWorker()
	})
}

func startAgentRunWorker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		processQueuedRuns()
	}
}

func processQueuedRuns() {
	markExpiredAgentRuns()
	result, err := callAgentLowcodeService("agent.run_queued_query", nil)
	if err != nil {
		return
	}
	var runList []base.AgentRun
	if err := decodeAgentLowcodeData(result.Data, &runList); err != nil {
		return
	}
	for _, run := range runList {
		if !shouldProcessQueuedAgentRun(run) {
			continue
		}
		_ = executeAgentRun(run.AgentRunID)
	}
}

func shouldProcessQueuedAgentRun(run base.AgentRun) bool {
	requestData := decodeAgentRunRequest(run.RequestJSON)
	if gocast.ToBool(requestData["run_sync"]) {
		return false
	}
	return !isAgentStreamResponse(requestData)
}

func claimAgentRun(agentRunID string) (*base.AgentRun, bool, error) {
	agentRunID = strings.TrimSpace(agentRunID)
	if agentRunID == "" {
		return nil, false, fmt.Errorf("agent_run_id 不能为空")
	}
	result, err := callAgentLowcodeService("agent.run_claim_update", map[string]interface{}{
		"agent_run_id":      agentRunID,
		"queued_status":     runStatusQueued,
		"lease_expire_time": time.Now().Add(2 * time.Minute).Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		return nil, false, err
	}
	if agentLowcodeAffected(result) == 0 {
		return nil, false, nil
	}
	run, ok, err := queryAgentRun(map[string]interface{}{"agent_run_id": agentRunID})
	if err != nil {
		return nil, true, err
	}
	if !ok {
		return nil, true, fmt.Errorf("agent_run 不存在: %s", agentRunID)
	}
	return run, true, nil
}

func markExpiredAgentRuns() {
	result, err := callAgentLowcodeService("agent.run_expired_query", nil)
	if err != nil {
		return
	}
	var runList []base.AgentRun
	if err := decodeAgentLowcodeData(result.Data, &runList); err != nil {
		return
	}
	for _, run := range runList {
		_ = updateAgentRunExpiredByLowcode(run.AgentRunID)
	}
}

func updateAgentRunExpiredByLowcode(agentRunID string) error {
	for _, service := range []string{"agent.run_expired_fail_update", "agent.message_run_failed_update"} {
		if _, err := callAgentLowcodeService(service, map[string]interface{}{
			"agent_run_id": agentRunID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func startAgentRunHeartbeat(ctx context.Context, agentRunID string) context.CancelFunc {
	if ctx == nil {
		ctx = context.Background()
	}
	hbCtx, cancel := context.WithCancel(ctx)
	if strings.TrimSpace(agentRunID) == "" {
		return cancel
	}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				_, _ = callAgentLowcodeService("agent.run_heartbeat_update", map[string]interface{}{
					"agent_run_id":      agentRunID,
					"running_status":    runStatusRunning,
					"lease_expire_time": time.Now().Add(2 * time.Minute).Format("2006-01-02 15:04:05"),
				})
			}
		}
	}()
	return cancel
}

func nowText() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func agentParamText(params map[string]interface{}, key string, fallback string) string {
	if params != nil {
		if value := strings.TrimSpace(gocast.ToString(params[key])); value != "" {
			return value
		}
	}
	return fallback
}

func agentParamNow(params map[string]interface{}, key string) string {
	return agentParamText(params, key, nowText())
}

func normalizeModel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return defaultAgentModel
	}
	return model
}

func resolveProviderModel(model string, credential agentCredential) string {
	model = normalizeModel(model)
	if credential.Mode != "chatgpt_access_token" {
		return model
	}
	switch model {
	case "", defaultAgentModel:
		return defaultChatGPTAuthModel
	default:
		return model
	}
}

func normalizeScene(scene string) string {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return defaultAgentScene
	}
	return scene
}

func codexAuthFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".codex", "auth.json"), nil
}

func loadCodexAuthFileFromPath(authPath string) (*codexAuthFile, error) {
	data, err := os.ReadFile(authPath)
	if err != nil {
		return nil, err
	}
	var authData codexAuthFile
	if err := json.Unmarshal(data, &authData); err != nil {
		return nil, err
	}
	return &authData, nil
}

func loadCodexAuthFile() (*codexAuthFile, error) {
	authPath, err := codexAuthFilePath()
	if err != nil {
		return nil, err
	}
	return loadCodexAuthFileFromPath(authPath)
}

func credentialFromCodexAuthData(authData *codexAuthFile) agentCredential {
	if authData == nil {
		return agentCredential{}
	}
	if authData.OPENAIAPIKey != nil && strings.TrimSpace(*authData.OPENAIAPIKey) != "" {
		return agentCredential{
			Token:     strings.TrimSpace(*authData.OPENAIAPIKey),
			AccountID: strings.TrimSpace(authData.Tokens.AccountID),
			Mode:      "platform_api_key",
			Source:    "codex_auth_openai_api_key",
		}
	}
	if strings.TrimSpace(authData.Tokens.AccessToken) != "" {
		return agentCredential{
			Token:     strings.TrimSpace(authData.Tokens.AccessToken),
			AccountID: strings.TrimSpace(authData.Tokens.AccountID),
			Mode:      "chatgpt_access_token",
			Source:    "codex_auth_access_token",
		}
	}
	return agentCredential{}
}

func resolveAgentCredentialWithReload(forceReload bool) agentCredential {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	accountID := strings.TrimSpace(os.Getenv("OPENAI_ACCOUNT_ID"))
	if apiKey != "" {
		return agentCredential{
			Token:     apiKey,
			AccountID: accountID,
			Mode:      "platform_api_key",
			Source:    "env_openai_api_key",
		}
	}

	agentCredentialCacheMu.Lock()
	defer agentCredentialCacheMu.Unlock()
	authPath, pathErr := codexAuthFilePath()
	if pathErr != nil {
		if agentCredentialCacheLoaded {
			return agentCredentialCache
		}
		return agentCredential{}
	}
	if !forceReload && agentCredentialCacheLoaded && agentCredentialCachePath == authPath {
		return agentCredentialCache
	}
	authData, err := loadCodexAuthFileFromPath(authPath)
	if err != nil || authData == nil {
		if agentCredentialCacheLoaded && agentCredentialCachePath == authPath {
			return agentCredentialCache
		}
		return agentCredential{}
	}
	agentCredentialCache = credentialFromCodexAuthData(authData)
	agentCredentialCachePath = authPath
	agentCredentialCacheLoaded = true
	return agentCredentialCache
}

func resolveAgentCredential() agentCredential {
	return resolveAgentCredentialWithReload(false)
}

func agentCredentialChanged(prev, next agentCredential) bool {
	return prev.Token != next.Token ||
		prev.AccountID != next.AccountID ||
		prev.Mode != next.Mode ||
		prev.Source != next.Source
}

func agentCredentialValueFingerprint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("sha256:%x", sum[:6])
}

func agentCredentialTokenFingerprint(credential agentCredential) string {
	return agentCredentialValueFingerprint(credential.Token)
}

func agentCredentialAccountFingerprint(credential agentCredential) string {
	return agentCredentialValueFingerprint(credential.AccountID)
}

func formatAgentCredentialFingerprintForLog(credential agentCredential) string {
	if fingerprint := agentCredentialTokenFingerprint(credential); fingerprint != "" {
		return "token指纹 " + fingerprint
	}
	return "token指纹 空"
}

func formatAgentCredentialFingerprintValue(fingerprint string) string {
	if strings.TrimSpace(fingerprint) == "" {
		return "空"
	}
	return fingerprint
}

func formatAgentCredentialChangeForLog(prev, next agentCredential) string {
	parts := make([]string, 0, 4)
	if prev.Token != next.Token {
		parts = append(parts, fmt.Sprintf("token指纹 %s -> %s",
			formatAgentCredentialFingerprintValue(agentCredentialTokenFingerprint(prev)),
			formatAgentCredentialFingerprintValue(agentCredentialTokenFingerprint(next))))
	}
	if prev.AccountID != next.AccountID {
		parts = append(parts, fmt.Sprintf("account_id指纹 %s -> %s",
			formatAgentCredentialFingerprintValue(agentCredentialAccountFingerprint(prev)),
			formatAgentCredentialFingerprintValue(agentCredentialAccountFingerprint(next))))
	}
	if prev.Mode != next.Mode {
		parts = append(parts, fmt.Sprintf("mode %s -> %s", prev.Mode, next.Mode))
	}
	if prev.Source != next.Source {
		parts = append(parts, fmt.Sprintf("source %s -> %s", prev.Source, next.Source))
	}
	if len(parts) == 0 {
		return formatAgentCredentialFingerprintForLog(next)
	}
	return strings.Join(parts, "，")
}

func attachAgentCredentialFingerprintFields(event map[string]interface{}, credential agentCredential) {
	if event == nil {
		return
	}
	if fingerprint := agentCredentialTokenFingerprint(credential); fingerprint != "" {
		event["token_fingerprint"] = fingerprint
	}
	if fingerprint := agentCredentialAccountFingerprint(credential); fingerprint != "" {
		event["account_id_fingerprint"] = fingerprint
	}
}

func shouldReloadCredentialAfterProviderError(err error, credential agentCredential) bool {
	if credential.Source != "codex_auth_access_token" && credential.Source != "codex_auth_openai_api_key" {
		return false
	}
	var httpErr *agentProviderHTTPError
	if !errors.As(err, &httpErr) || httpErr == nil {
		return false
	}
	switch httpErr.StatusCode {
	case http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}

func shouldRetryProviderAfterTransientError(err error) bool {
	if err == nil {
		return false
	}
	if isAgentProviderTimeoutError(err) {
		return true
	}
	if code, _, received, ok := agentProviderHTTPStatusFromError(err); ok {
		if !received {
			return true
		}
		switch code {
		case http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		}
	}
	var httpErr *agentProviderHTTPError
	if errors.As(err, &httpErr) && httpErr != nil {
		switch httpErr.StatusCode {
		case http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		}
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "eof") ||
		strings.Contains(message, "connection reset") ||
		strings.Contains(message, "connection refused") ||
		strings.Contains(message, "unexpected end of file")
}

func shouldStopAuthReloadRetryWhenCredentialUnchanged(err error) bool {
	if err == nil {
		return false
	}
	var httpErr *agentProviderHTTPError
	if !errors.As(err, &httpErr) || httpErr == nil {
		return false
	}
	switch httpErr.StatusCode {
	case http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}

func isAgentProviderRateLimitError(err error) bool {
	code, _, received, ok := agentProviderHTTPStatusFromError(err)
	return ok && received && code == http.StatusTooManyRequests
}

func agentProviderRetryWaitDuration(err error, attempt int) time.Duration {
	if isAgentProviderRateLimitError(err) {
		minWait := agentProviderRateLimitRetryWaitMin
		maxWait := agentProviderRateLimitRetryWaitMax
		if minWait < 0 {
			minWait = 0
		}
		if maxWait < minWait {
			maxWait = minWait
		}
		jitter := maxWait - minWait
		if jitter <= 0 {
			return minWait
		}
		return minWait + time.Duration(rand.Int63n(int64(jitter)+1))
	}
	wait := time.Duration(attempt) * agentProviderRetryWaitBase
	if wait < 0 {
		return 0
	}
	return wait
}

func agentProviderDurationFromEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	if seconds, err := strconv.ParseFloat(raw, 64); err == nil && seconds > 0 {
		return time.Duration(seconds * float64(time.Second))
	}
	return fallback
}

func formatAgentProviderDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	seconds := d.Seconds()
	if seconds < 10 {
		return fmt.Sprintf("%.1f秒", seconds)
	}
	return fmt.Sprintf("%.0f秒", seconds)
}

func formatAgentProviderHTTPStatus(code int, text string) string {
	text = strings.TrimSpace(text)
	if code <= 0 {
		return ""
	}
	if text == "" {
		text = strings.TrimSpace(http.StatusText(code))
	}
	if text == "" {
		return fmt.Sprintf("HTTP %d", code)
	}
	if strings.HasPrefix(text, fmt.Sprintf("%d ", code)) {
		return "HTTP " + text
	}
	return fmt.Sprintf("HTTP %d %s", code, text)
}

func agentProviderHTTPStatusForLog(resp *http.Response) string {
	if resp == nil {
		return "未收到HTTP状态码"
	}
	status := formatAgentProviderHTTPStatus(resp.StatusCode, resp.Status)
	if status == "" {
		return "未收到HTTP状态码"
	}
	return status
}

func wrapAgentProviderHTTPStatusError(err error, resp *http.Response) error {
	if err == nil {
		return nil
	}
	var existing *agentProviderHTTPStatusError
	if errors.As(err, &existing) {
		return err
	}
	statusErr := &agentProviderHTTPStatusError{Err: err}
	if resp != nil {
		statusErr.StatusReceived = true
		statusErr.StatusCode = resp.StatusCode
		statusErr.StatusText = strings.TrimSpace(resp.Status)
	}
	return statusErr
}

func agentProviderHTTPStatusFromError(err error) (int, string, bool, bool) {
	if err == nil {
		return 0, "", false, false
	}
	var statusErr *agentProviderHTTPStatusError
	if errors.As(err, &statusErr) && statusErr != nil {
		return statusErr.StatusCode, statusErr.StatusText, statusErr.StatusReceived, true
	}
	var httpErr *agentProviderHTTPError
	if errors.As(err, &httpErr) && httpErr != nil && httpErr.StatusCode > 0 {
		return httpErr.StatusCode, http.StatusText(httpErr.StatusCode), true, true
	}
	return 0, "", false, false
}

func agentProviderResponseHeaderTimeoutFromError(err error) (time.Duration, bool) {
	var timeoutErr *agentProviderResponseHeaderTimeoutError
	if errors.As(err, &timeoutErr) && timeoutErr != nil && timeoutErr.Timeout > 0 {
		return timeoutErr.Timeout, true
	}
	return 0, false
}

func agentProviderHTTPStatusSummary(err error) string {
	code, text, received, ok := agentProviderHTTPStatusFromError(err)
	if !ok {
		return ""
	}
	if !received {
		return "HTTP状态码未收到（响应头尚未返回，SSE尚未开始）"
	}
	return formatAgentProviderHTTPStatus(code, text)
}

func agentProviderElapsedFields(startedAt time.Time, attemptStartedAt time.Time) map[string]interface{} {
	now := time.Now()
	total := now.Sub(startedAt)
	attempt := now.Sub(attemptStartedAt)
	return map[string]interface{}{
		"elapsed_ms":               total.Milliseconds(),
		"elapsed_text":             formatAgentProviderDuration(total),
		"attempt_elapsed_ms":       attempt.Milliseconds(),
		"attempt_elapsed_text":     formatAgentProviderDuration(attempt),
		"provider_elapsed_ms":      total.Milliseconds(),
		"provider_elapsed_text":    formatAgentProviderDuration(total),
		"provider_attempt_ms":      attempt.Milliseconds(),
		"provider_attempt_elapsed": formatAgentProviderDuration(attempt),
	}
}

func isAgentProviderTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "context deadline exceeded") ||
		strings.Contains(message, "client.timeout") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "awaiting headers")
}

func isAgentProviderResponseHeaderTimeout(err error) bool {
	if !isAgentProviderTimeoutError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "awaiting response headers") ||
		strings.Contains(message, "awaiting headers")
}

func wrapAgentProviderResponseHeaderTimeoutError(err error, timeout time.Duration) error {
	if err == nil || timeout <= 0 || !isAgentProviderResponseHeaderTimeout(err) {
		return err
	}
	var existing *agentProviderResponseHeaderTimeoutError
	if errors.As(err, &existing) {
		return err
	}
	return &agentProviderResponseHeaderTimeoutError{Err: err, Timeout: timeout}
}

func trimAgentProviderErrorText(text string, maxBytes int) string {
	text = strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if maxBytes <= 0 || len(text) <= maxBytes {
		return text
	}
	for maxBytes > 0 && !utf8.ValidString(text[:maxBytes]) {
		maxBytes--
	}
	if maxBytes <= 0 {
		return ""
	}
	return strings.TrimSpace(text[:maxBytes]) + "..."
}

func agentProviderFailureSummary(err error) string {
	if err == nil {
		return ""
	}
	var httpErr *agentProviderHTTPError
	if errors.As(err, &httpErr) && httpErr != nil {
		statusText := strings.TrimSpace(http.StatusText(httpErr.StatusCode))
		if httpErr.StatusCode == http.StatusTooManyRequests {
			statusText = "Too Many Requests/限流"
		}
		parts := []string{fmt.Sprintf("HTTP %d", httpErr.StatusCode)}
		if statusText != "" {
			parts = append(parts, statusText)
		}
		if httpErr.RetryAfter != "" {
			parts = append(parts, "Retry-After="+httpErr.RetryAfter)
		}
		message := trimAgentProviderErrorText(httpErr.Message, 180)
		if message != "" {
			parts = append(parts, message)
		}
		return strings.Join(parts, "，")
	}
	statusSummary := agentProviderHTTPStatusSummary(err)
	if isAgentProviderTimeoutError(err) {
		message := strings.ToLower(err.Error())
		var timeoutSummary string
		switch {
		case strings.Contains(message, "sse"):
			timeoutSummary = fmt.Sprintf("请求超时：%s 内未收到流式数据", formatAgentProviderDuration(agentProviderStreamIdleTimeout))
		case strings.Contains(message, "awaiting response headers") || strings.Contains(message, "awaiting headers"):
			headerTimeout := agentProviderResponseHeaderTimeout
			if actual, ok := agentProviderResponseHeaderTimeoutFromError(err); ok {
				headerTimeout = actual
			}
			timeoutSummary = fmt.Sprintf("请求超时：%s 内未收到模型响应头", formatAgentProviderDuration(headerTimeout))
		default:
			timeoutSummary = "请求超时：" + trimAgentProviderErrorText(err.Error(), 180)
		}
		if statusSummary != "" {
			return statusSummary + "，" + timeoutSummary
		}
		return timeoutSummary
	}
	summary := trimAgentProviderErrorText(err.Error(), 220)
	if statusSummary != "" && summary != "" {
		return statusSummary + "，" + summary
	}
	if statusSummary != "" {
		return statusSummary
	}
	return summary
}

func attachAgentProviderErrorFields(event map[string]interface{}, err error) {
	if event == nil || err == nil {
		return
	}
	event["error"] = err.Error()
	if summary := agentProviderFailureSummary(err); summary != "" {
		event["error_summary"] = summary
	}
	if statusSummary := agentProviderHTTPStatusSummary(err); statusSummary != "" {
		event["provider_status"] = statusSummary
		if mode, _ := event["mode"].(string); mode == "chatgpt_access_token" {
			event["chatgpt_status"] = statusSummary
		}
		code, text, received, _ := agentProviderHTTPStatusFromError(err)
		event["http_status_received"] = received
		if received {
			event["status_code"] = code
			if text != "" {
				event["status_text"] = text
			}
		} else {
			event["status_code_text"] = "未收到"
		}
	}
	if isAgentProviderTimeoutError(err) {
		event["timeout"] = true
	}
	var httpErr *agentProviderHTTPError
	if errors.As(err, &httpErr) && httpErr != nil {
		event["status_code"] = httpErr.StatusCode
		if httpErr.RetryAfter != "" {
			event["retry_after"] = httpErr.RetryAfter
		}
		if httpErr.RequestID != "" {
			event["request_id"] = httpErr.RequestID
		}
	}
}

func newAgentProviderHTTPClient(stream bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	headerTimeout := agentProviderHeaderTimeout(stream)
	transport.ResponseHeaderTimeout = headerTimeout
	if transport.TLSHandshakeTimeout <= 0 {
		transport.TLSHandshakeTimeout = 10 * time.Second
	}
	if transport.ExpectContinueTimeout <= 0 {
		transport.ExpectContinueTimeout = time.Second
	}
	client := &http.Client{Transport: transport}
	if !stream {
		client.Timeout = agentProviderNonStreamRequestTimeout
	}
	return client
}

func agentProviderHeaderTimeout(stream bool) time.Duration {
	if stream && agentProviderStreamHeaderTimeout > 0 {
		return agentProviderDurationFromEnv("AGENT_PROVIDER_STREAM_HEADER_TIMEOUT", agentProviderStreamHeaderTimeout)
	}
	return agentProviderDurationFromEnv("AGENT_PROVIDER_RESPONSE_HEADER_TIMEOUT", agentProviderResponseHeaderTimeout)
}

func agentProviderCodexUserAgent() string {
	return fmt.Sprintf("codex_cli_rs/0.133.0 (%s; %s) agent-runtime", runtime.GOOS, runtime.GOARCH)
}

func agentProviderSessionTraceID(session *base.AgentSession) string {
	if session == nil {
		return ""
	}
	if id := strings.TrimSpace(session.AgentSessionID); id != "" {
		return id
	}
	return strings.TrimSpace(session.SessionKey)
}

func applyAgentProviderCodexHeaders(req *http.Request, session *base.AgentSession, credential agentCredential) {
	if req == nil {
		return
	}
	req.Header.Set("originator", "codex_cli_rs")
	req.Header.Set("User-Agent", agentProviderCodexUserAgent())
	if credential.Mode != "chatgpt_access_token" {
		return
	}
	traceID := agentProviderSessionTraceID(session)
	if traceID != "" {
		req.Header.Set("session-id", traceID)
		req.Header.Set("thread-id", traceID)
		req.Header.Set("x-client-request-id", traceID)
		req.Header.Set("x-codex-window-id", traceID+":0")
		// Keep the old local header during rollout; the ChatGPT backend uses session-id/thread-id.
		req.Header.Set("session_id", traceID)
	}
	req.Header.Set("x-codex-installation-id", "agent-runtime")
}

func buildAgentProviderRequestHeadersDebug(req *http.Request) map[string]interface{} {
	if req == nil {
		return nil
	}
	debug := map[string]interface{}{}
	if strings.TrimSpace(req.Header.Get("Authorization")) != "" {
		debug["authorization_attached"] = true
		debug["Authorization"] = "Bearer ***"
	}
	for _, key := range []string{
		"Accept",
		"Content-Type",
		"originator",
		"User-Agent",
		"session-id",
		"thread-id",
		"x-client-request-id",
		"x-codex-installation-id",
		"x-codex-window-id",
		"session_id",
	} {
		if value := strings.TrimSpace(req.Header.Get(key)); value != "" {
			debug[key] = value
		}
	}
	for _, key := range []string{"ChatGPT-Account-ID", "OpenAI-Account-ID"} {
		if value := strings.TrimSpace(req.Header.Get(key)); value != "" {
			debug[key+"_fingerprint"] = agentCredentialValueFingerprint(value)
		}
	}
	return debug
}

func resolveAgentBaseURL(credential agentCredential) string {
	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if baseURL != "" {
		return strings.TrimRight(baseURL, "/")
	}
	if credential.Mode == "chatgpt_access_token" {
		return "https://chatgpt.com/backend-api/codex"
	}
	return "https://api.openai.com/v1"
}

func buildAgentProviderError(statusCode int, rawBody []byte, openaiResp *openAIResponse, credential agentCredential, header http.Header) error {
	bodyText := strings.TrimSpace(string(rawBody))
	var message string
	if openaiResp != nil && openaiResp.Error != nil && strings.TrimSpace(openaiResp.Error.Message) != "" {
		message = fmt.Sprintf("Agent provider 状态异常: status_code=%d, mode=%s, source=%s, message=%s", statusCode, credential.Mode, credential.Source, strings.TrimSpace(openaiResp.Error.Message))
	} else if bodyText != "" {
		if len(bodyText) > 400 {
			bodyText = bodyText[:400]
		}
		message = fmt.Sprintf("Agent provider 状态异常: status_code=%d, mode=%s, source=%s, body=%s", statusCode, credential.Mode, credential.Source, bodyText)
	} else {
		message = fmt.Sprintf("Agent provider 状态异常: status_code=%d, mode=%s, source=%s", statusCode, credential.Mode, credential.Source)
	}
	retryAfter := strings.TrimSpace(header.Get("Retry-After"))
	requestID := strings.TrimSpace(header.Get("x-request-id"))
	if requestID == "" {
		requestID = strings.TrimSpace(header.Get("openai-request-id"))
	}
	return &agentProviderHTTPError{StatusCode: statusCode, Message: message, RetryAfter: retryAfter, RequestID: requestID}
}

func agentConversationTextForProvider(role string, text string) (string, bool) {
	role = strings.TrimSpace(role)
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	if shouldDropAgentConversationTextForProvider(role, text) {
		return "", false
	}
	if role == "user" && looksLikeAgentRequestDebugDump(text) {
		return "用户曾提供模型请求调试记录；完整请求体、历史 input 预览、内部限流/失败日志已从模型上下文中省略。", true
	}
	return text, true
}

func shouldDropAgentConversationTextForProvider(role string, text string) bool {
	if role != "assistant" {
		return false
	}
	compact := strings.ToLower(strings.TrimSpace(text))
	if compact == "" {
		return true
	}
	if strings.Contains(compact, "agent provider 状态异常") ||
		strings.Contains(compact, "agent provider 请求失败") ||
		strings.Contains(compact, "status_code=429") ||
		strings.Contains(compact, "http 429") ||
		strings.Contains(compact, "too many requests") ||
		strings.Contains(compact, "usage limit has been reached") ||
		strings.Contains(compact, "服务重启或模型请求超过租约未更新") {
		return true
	}
	return looksLikeAgentRequestDebugDump(text)
}

func looksLikeAgentRequestDebugDump(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	lower := strings.ToLower(text)
	debugMarkers := []string{
		"模型请求摘要",
		"发送给模型的 input 预览",
		"完整 request body 预览",
		"input items 摘要",
		"request_json_preview",
		agentProviderDebugToolName,
	}
	for _, marker := range debugMarkers {
		if strings.Contains(lower, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func loadAgentConversationInput(agentSessionID string) []map[string]interface{} {
	messageList, err := queryAgentMessageList(map[string]interface{}{
		"agent_session_id": agentSessionID,
		"pagination":       false,
	})
	if err != nil {
		return nil
	}
	inputList := make([]map[string]interface{}, 0, len(messageList))
	for _, message := range messageList {
		role := strings.TrimSpace(message.Role)
		text, ok := agentConversationTextForProvider(role, message.ContentText)
		if !ok {
			continue
		}
		contentType := ""
		switch role {
		case "user":
			contentType = "input_text"
		case "assistant":
			contentType = "output_text"
		default:
			continue
		}
		inputList = append(inputList, map[string]interface{}{
			"type": "message",
			"role": role,
			"content": []map[string]interface{}{
				{
					"type": contentType,
					"text": text,
				},
			},
		})
	}
	return inputList
}

func newAgentUserInputMessage(inputText string) map[string]interface{} {
	return map[string]interface{}{
		"type": "message",
		"role": "user",
		"content": []map[string]interface{}{
			{
				"type": "input_text",
				"text": inputText,
			},
		},
	}
}

func replaceLatestAgentUserInput(inputList []map[string]interface{}, inputText string) []map[string]interface{} {
	if strings.TrimSpace(inputText) == "" {
		return inputList
	}
	for i := len(inputList) - 1; i >= 0; i-- {
		if gocast.ToString(inputList[i]["type"]) != "message" || gocast.ToString(inputList[i]["role"]) != "user" {
			continue
		}
		switch content := inputList[i]["content"].(type) {
		case []map[string]interface{}:
			for j := range content {
				if gocast.ToString(content[j]["type"]) == "input_text" {
					content[j]["text"] = inputText
					inputList[i]["content"] = content
					return inputList
				}
			}
			content = append(content, map[string]interface{}{"type": "input_text", "text": inputText})
			inputList[i]["content"] = content
			return inputList
		case []interface{}:
			for j, raw := range content {
				item, _ := raw.(map[string]interface{})
				if gocast.ToString(item["type"]) == "input_text" {
					item["text"] = inputText
					content[j] = item
					inputList[i]["content"] = content
					return inputList
				}
			}
			content = append(content, map[string]interface{}{"type": "input_text", "text": inputText})
			inputList[i]["content"] = content
			return inputList
		default:
			inputList[i]["content"] = []map[string]interface{}{
				{
					"type": "input_text",
					"text": inputText,
				},
			}
			return inputList
		}
	}
	return append(inputList, newAgentUserInputMessage(inputText))
}

func buildAgentInitialInputList(session *base.AgentSession, inputText string) []map[string]interface{} {
	if session == nil {
		if strings.TrimSpace(inputText) == "" {
			return nil
		}
		return []map[string]interface{}{newAgentUserInputMessage(inputText)}
	}
	inputList := loadAgentConversationInput(session.AgentSessionID)
	if len(inputList) == 0 {
		if strings.TrimSpace(inputText) == "" {
			return inputList
		}
		return []map[string]interface{}{newAgentUserInputMessage(inputText)}
	}
	return replaceLatestAgentUserInput(inputList, inputText)
}

func buildAgentPromptCacheKey(session *base.AgentSession, credential agentCredential, model string, instructions string, tools []interface{}) string {
	if session == nil {
		return ""
	}
	toolBytes, _ := json.Marshal(tools)
	seed := strings.Join([]string{
		"agent-runtime-v1",
		"mode=" + credential.Mode,
		"model=" + model,
		"scene=" + normalizeScene(session.SceneCode),
		"instructions=" + instructions,
		"tools=" + string(toolBytes),
	}, "\n")
	sum := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("sport-agent-%x", sum[:12])
}

func buildAgentResponsesRequestBody(session *base.AgentSession, inputText string, credential agentCredential, policy agentToolPolicy, previousResponseID string, toolOutputs []map[string]interface{}, stream bool) map[string]interface{} {
	var input interface{} = inputText
	if len(toolOutputs) > 0 {
		input = toolOutputs
	} else if credential.Mode == "chatgpt_access_token" {
		input = buildAgentInitialInputList(session, inputText)
	}
	input = compactAgentProviderInputToolOutputs(input)

	model := resolveProviderModel(session.Model, credential)
	tools := policy.toolDefinitions()
	instructions := buildAgentInstructions(session, policy)
	requestBody := map[string]interface{}{
		"model": model,
		"input": input,
	}
	if previousResponseID != "" {
		requestBody["previous_response_id"] = previousResponseID
	} else if credential.Mode != "chatgpt_access_token" && strings.TrimSpace(session.LastResponseID) != "" {
		requestBody["previous_response_id"] = session.LastResponseID
	}
	if stream {
		requestBody["stream"] = true
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
		requestBody["parallel_tool_calls"] = true
	} else if credential.Mode == "chatgpt_access_token" {
		requestBody["tools"] = []interface{}{}
		requestBody["tool_choice"] = "auto"
		requestBody["parallel_tool_calls"] = true
		requestBody["store"] = false
	}
	if credential.Mode == "chatgpt_access_token" {
		requestBody["store"] = false
		requestBody["include"] = []interface{}{}
		requestBody["client_metadata"] = map[string]string{
			"x-codex-installation-id": "agent-runtime",
		}
	}

	if instructions != "" {
		requestBody["instructions"] = instructions
	}
	if cacheKey := buildAgentPromptCacheKey(session, credential, model, instructions, tools); cacheKey != "" {
		requestBody["prompt_cache_key"] = cacheKey
	}
	return requestBody
}

func compactAgentProviderInputToolOutputs(input interface{}) interface{} {
	switch items := input.(type) {
	case []map[string]interface{}:
		remaining := hardAgentProviderToolInputBytes
		next := make([]map[string]interface{}, 0, len(items))
		for _, item := range items {
			copied := make(map[string]interface{}, len(item))
			for key, value := range item {
				copied[key] = value
			}
			if gocast.ToString(copied["type"]) == "function_call_output" {
				output := gocast.ToString(copied["output"])
				copied["output"] = compactAgentProviderToolOutputWithBudget(output, &remaining)
			}
			next = append(next, copied)
		}
		return next
	case []interface{}:
		remaining := hardAgentProviderToolInputBytes
		next := make([]interface{}, 0, len(items))
		for _, raw := range items {
			item, ok := raw.(map[string]interface{})
			if !ok {
				next = append(next, raw)
				continue
			}
			copied := make(map[string]interface{}, len(item))
			for key, value := range item {
				copied[key] = value
			}
			if gocast.ToString(copied["type"]) == "function_call_output" {
				output := gocast.ToString(copied["output"])
				copied["output"] = compactAgentProviderToolOutputWithBudget(output, &remaining)
			}
			next = append(next, copied)
		}
		return next
	default:
		return input
	}
}

func compactAgentProviderToolOutputWithBudget(output string, remaining *int) string {
	if remaining == nil {
		return agentToolOutputForProvider(output)
	}
	if *remaining <= 0 {
		return agentToolOutputBudgetNotice(output)
	}
	maxBytes := hardAgentProviderToolOutputBytes
	if *remaining < maxBytes {
		maxBytes = *remaining
	}
	compacted := agentToolOutputForProviderWithLimit(output, maxBytes)
	*remaining -= len([]byte(compacted))
	return compacted
}

func marshalAgentDebugJSON(value interface{}) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		data, err = json.Marshal(value)
	}
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func agentDebugStringBytes(value string) int {
	return len([]byte(value))
}

func agentDebugPreviewText(value string, maxBytes int) string {
	return truncateAgentToolText(value, maxBytes)
}

func agentDebugJSONBytes(value interface{}) (int, string) {
	data, err := json.Marshal(value)
	if err != nil {
		return 0, ""
	}
	return len(data), string(data)
}

func agentDebugPreviewJSON(value interface{}, maxBytes int) (string, bool) {
	if maxBytes <= 0 {
		return "", true
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		data, err = json.Marshal(value)
	}
	if err != nil {
		text := fmt.Sprintf("%v", value)
		return agentDebugPreviewText(text, maxBytes), len([]byte(text)) > maxBytes
	}
	text := string(data)
	return agentDebugPreviewText(text, maxBytes), len(data) > maxBytes
}

func buildAgentProviderContentDebug(raw interface{}) []map[string]interface{} {
	var contentList []interface{}
	switch typed := raw.(type) {
	case []interface{}:
		contentList = typed
	case []map[string]interface{}:
		for _, item := range typed {
			contentList = append(contentList, item)
		}
	default:
		return nil
	}
	debugList := make([]map[string]interface{}, 0, len(contentList))
	for idx, rawItem := range contentList {
		item, _ := rawItem.(map[string]interface{})
		if len(item) == 0 {
			continue
		}
		text := gocast.ToString(item["text"])
		entry := map[string]interface{}{
			"index": idx,
			"type":  gocast.ToString(item["type"]),
		}
		if text != "" {
			entry["text_bytes"] = agentDebugStringBytes(text)
			entry["text_omitted"] = true
		}
		debugList = append(debugList, entry)
	}
	return debugList
}

func buildAgentProviderInputItemDebug(raw interface{}, idx int) map[string]interface{} {
	item, _ := raw.(map[string]interface{})
	if len(item) == 0 {
		return map[string]interface{}{
			"index": idx,
			"kind":  fmt.Sprintf("%T", raw),
		}
	}
	entry := map[string]interface{}{
		"index": idx,
		"type":  gocast.ToString(item["type"]),
	}
	for _, key := range []string{"role", "name", "call_id", "id", "status"} {
		if value := strings.TrimSpace(gocast.ToString(item[key])); value != "" {
			entry[key] = value
		}
	}
	if arguments := gocast.ToString(item["arguments"]); arguments != "" {
		entry["arguments_bytes"] = agentDebugStringBytes(arguments)
		entry["arguments_omitted"] = true
	}
	if output := gocast.ToString(item["output"]); output != "" {
		entry["output_bytes"] = agentDebugStringBytes(output)
		entry["output_omitted"] = true
	}
	if content := buildAgentProviderContentDebug(item["content"]); len(content) > 0 {
		entry["content"] = content
	}
	return entry
}

func buildAgentProviderInputDebug(input interface{}) map[string]interface{} {
	debug := map[string]interface{}{}
	inputBytes, _ := agentDebugJSONBytes(input)
	debug["input_bytes"] = inputBytes
	if preview, truncated := agentDebugPreviewJSON(input, agentProviderInputMaxBytes); preview != "" {
		debug["input_preview"] = preview
		debug["input_truncated"] = truncated
		debug["input_preview_omitted"] = false
	} else {
		debug["input_preview_omitted"] = true
	}
	switch typed := input.(type) {
	case string:
		debug["kind"] = "text"
		debug["text_bytes"] = agentDebugStringBytes(typed)
		debug["text_omitted"] = true
	case []map[string]interface{}:
		items := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return buildAgentProviderInputDebug(items)
	case []interface{}:
		debug["kind"] = "items"
		debug["item_count"] = len(typed)
		debugLimit := len(typed)
		if debugLimit > agentProviderDebugMaxItems*4 {
			debugLimit = agentProviderDebugMaxItems * 4
			debug["items_truncated"] = true
			debug["items_omitted"] = len(typed) - debugLimit
		}
		itemDebug := make([]map[string]interface{}, 0, debugLimit)
		functionOutputCount := 0
		functionOutputBytes := 0
		messageCount := 0
		for idx, rawItem := range typed {
			item := buildAgentProviderInputItemDebug(rawItem, idx)
			if item["type"] == "function_call_output" {
				functionOutputCount++
				functionOutputBytes += gocast.ToInt(item["output_bytes"])
			}
			if item["type"] == "message" {
				messageCount++
			}
			if idx < debugLimit {
				itemDebug = append(itemDebug, item)
			}
		}
		debug["message_count"] = messageCount
		debug["function_call_output_count"] = functionOutputCount
		debug["function_call_output_bytes"] = functionOutputBytes
		debug["items"] = itemDebug
	default:
		debug["kind"] = fmt.Sprintf("%T", input)
	}
	return debug
}

func buildAgentProviderRequestDebug(requestBody map[string]interface{}, body []byte, credential agentCredential, baseURL string) map[string]interface{} {
	debug := map[string]interface{}{
		"success":        true,
		"tool":           agentProviderDebugToolName,
		"kind":           "model_provider_request",
		"mode":           credential.Mode,
		"source":         credential.Source,
		"base_url":       baseURL,
		"request_bytes":  len(body),
		"request_sha256": fmt.Sprintf("sha256:%x", sha256.Sum256(body)),
		"token_redacted": true,
		"message":        "模型请求调试记录：展示请求结构、字节数、哈希、凭据指纹，以及脱敏后的 input/request body 预览；超出预览上限会标记 truncated。",
	}
	if fingerprint := agentCredentialTokenFingerprint(credential); fingerprint != "" {
		debug["token_fingerprint"] = fingerprint
	}
	if fingerprint := agentCredentialAccountFingerprint(credential); fingerprint != "" {
		debug["account_id_fingerprint"] = fingerprint
	}
	if model := strings.TrimSpace(gocast.ToString(requestBody["model"])); model != "" {
		debug["model"] = model
	}
	if stream, ok := requestBody["stream"]; ok {
		debug["stream"] = stream
	}
	if previous := strings.TrimSpace(gocast.ToString(requestBody["previous_response_id"])); previous != "" {
		debug["previous_response_id_present"] = true
		debug["previous_response_id_preview"] = agentDebugPreviewText(previous, 80)
	}
	if cacheKey := strings.TrimSpace(gocast.ToString(requestBody["prompt_cache_key"])); cacheKey != "" {
		debug["prompt_cache_key"] = cacheKey
	}
	if instructions := gocast.ToString(requestBody["instructions"]); instructions != "" {
		debug["instructions_bytes"] = agentDebugStringBytes(instructions)
		debug["instructions_omitted"] = true
	}
	if tools, ok := requestBody["tools"].([]map[string]interface{}); ok {
		debug["tools_count"] = len(tools)
	} else if tools, ok := requestBody["tools"].([]interface{}); ok {
		debug["tools_count"] = len(tools)
	}
	if input, ok := requestBody["input"]; ok {
		inputDebug := buildAgentProviderInputDebug(input)
		debug["input"] = inputDebug
		debug["input_kind"] = inputDebug["kind"]
		debug["input_bytes"] = inputDebug["input_bytes"]
		debug["function_call_output_count"] = inputDebug["function_call_output_count"]
		debug["function_call_output_bytes"] = inputDebug["function_call_output_bytes"]
		debug["message_count"] = inputDebug["message_count"]
	}
	if preview, truncated := agentDebugPreviewJSON(requestBody, agentProviderDebugMaxBytes); preview != "" {
		debug["request_json_preview"] = preview
		debug["request_json_truncated"] = truncated
		debug["request_json_omitted"] = false
	} else {
		debug["request_json_omitted"] = true
	}
	return debug
}

func wrapAgentProviderRequestDebugError(err error, debug map[string]interface{}) error {
	if err == nil {
		return nil
	}
	if len(debug) == 0 {
		return err
	}
	var existing *agentProviderRequestDebugError
	if errors.As(err, &existing) {
		return err
	}
	return &agentProviderRequestDebugError{Err: err, Debug: debug}
}

func agentProviderRequestDebugFromError(err error) map[string]interface{} {
	var debugErr *agentProviderRequestDebugError
	if errors.As(err, &debugErr) && len(debugErr.Debug) > 0 {
		return debugErr.Debug
	}
	return nil
}

func newAgentProviderDebugToolResult(seq int, debug map[string]interface{}) agentToolResult {
	if debug == nil {
		debug = map[string]interface{}{}
	}
	debug["debug_seq"] = seq
	args := map[string]interface{}{
		"debug_seq":     seq,
		"request_bytes": debug["request_bytes"],
		"input_kind":    debug["input_kind"],
		"model":         debug["model"],
		"mode":          debug["mode"],
	}
	argBytes, _ := json.Marshal(args)
	outBytes, err := json.Marshal(debug)
	if err != nil {
		outBytes = []byte(`{"success":false,"tool":"model_request_debug","error":"模型请求调试记录序列化失败"}`)
	}
	return agentToolResult{
		CallID:    fmt.Sprintf("provider_request_debug_%d", seq),
		Name:      agentProviderDebugToolName,
		Arguments: string(argBytes),
		Output:    string(outBytes),
	}
}

func mergeAgentProviderDebugToolResults(debugRequests []map[string]interface{}, toolResults []agentToolResult) []agentToolResult {
	merged := make([]agentToolResult, 0, len(debugRequests)+len(toolResults))
	for idx, debug := range debugRequests {
		merged = append(merged, newAgentProviderDebugToolResult(idx+1, debug))
	}
	for _, result := range toolResults {
		if result.Name == agentProviderDebugToolName {
			continue
		}
		merged = append(merged, result)
	}
	return merged
}

func buildAgentInstructions(session *base.AgentSession, policy agentToolPolicy) string {
	instructions := strings.TrimSpace(session.SystemPrompt)
	if instructions == "" {
		instructions = defaultInstructions
	}
	if policy.Enabled && len(policy.AllowedRoots) > 0 {
		if !strings.Contains(instructions, "Codex CLI 对齐规则") {
			instructions += "\n\n" + buildAgentCodexLikeCoreInstructions()
		}
		if projectContext := buildAgentProjectContext(policy); projectContext != "" {
			instructions += "\n\n" + projectContext
		}
	}
	if policy.Enabled && policy.LocalProjectFileRead && len(policy.AllowedRoots) > 0 {
		instructions += "\n\n工具能力：当用户要求查看本地项目文件时，使用 read_project_file 工具读取文件内容，不要编造文件内容。允许读取根目录：" + strings.Join(policy.AllowedRoots, ", ")
		instructions += "\n\n文件读取输出规则：如果用户要求“全部读取”“完整读取”“读出来”“输出文件内容”或类似意图，必须把工具返回的 content 原样完整输出到代码块中，并按文件扩展名写代码块语言标识，例如 .go 使用 ```go。不要只总结文件结构、不要只输出开头、不要询问是否继续。只有当工具结果 truncated=true 或内容超过单条回复能力时，才说明已截断并继续用 start_line/end_line 分段读取和输出。"
	}
	if policy.Enabled && len(policy.AllowedRoots) > 0 {
		instructions += "\n\n代码定位工作流：当用户只说“继续”时，必须基于当前会话上下文继续上一个定位任务，不要从头泛泛解释。定位问题时按“先查找文件/符号 -> 读取关键片段 -> 必要时再查引用 -> 输出已确认链路和下一步验证命令”的顺序推进。避免重复同一个搜索；一旦工具结果足够支撑结论，就停止调用工具并直接回答。"
		instructions += "\n\n本地项目分析要求：当用户询问本仓库里的代码逻辑、配置链路、webshell、登录流程、接口调用、错误根因或“帮我定位/梳理/分析”时，必须在同一轮主动调用可用的查找/读取工具取得证据后再回答。不要让用户自己执行 grep/rg/cat，不要只给待执行命令；只有工具不可用或权限不足时才说明限制。"
	}
	if policy.Enabled && policy.LocalProjectFileSearch && len(policy.AllowedRoots) > 0 {
		instructions += "\n\n本地查找工具：使用 " + agentCodexCLIGlobToolName + " 按 glob 或模糊路径查找文件，使用 " + agentCodexCLIGrepToolName + " 按正则、literal 或 fuzzy 匹配搜索文件内容。定位代码时先查找再读取，避免猜测路径。不要用 " + agentRunCommandToolName + " 包 grep/cat/sed/awk 来读取文件，除非专用查找/读取工具无法表达。"
	}
	if policy.Enabled && policy.LocalProjectFileMutation && len(policy.AllowedRoots) > 0 {
		instructions += "\n\n本地编辑工具：使用 " + agentCodexCLIEditToolName + " 做小范围增量修改，优先 replace、insert_before、insert_after 或行范围替换。old_text/anchor 必须足够精确；多处命中时不要扩大修改，先重新 grep/read 定位。高风险编辑先 dry_run。"
	}
	if policy.Enabled && policy.LocalProjectFileDelete && len(policy.AllowedRoots) > 0 {
		instructions += "\n\n本地删除工具：删除文件必须先向用户确认精确相对路径；未获得明确确认前不要调用 " + agentCodexCLIDeleteToolName + " 执行删除。关键项目/系统文件、目录、数据库和配置文件禁止通过工具删除。"
	}
	if policy.Enabled && policy.LocalProjectCommand && len(policy.AllowedRoots) > 0 {
		instructions += "\n\n本地验证命令工具：使用 " + agentRunCommandToolName + " 运行非交互测试、构建、curl、node/npm/Playwright 脚本和诊断命令。命令必须在 allowed_roots 内的工作目录执行，优先跑最小有效验证；不要启动长期后台服务或执行破坏性命令。"
	}
	if policy.Enabled && policy.BrowserValidation && len(policy.AllowedRoots) > 0 {
		instructions += "\n\n无头浏览器工具：使用 " + agentBrowserCheckToolName + " 做 Playwright Chromium 自测，覆盖登录、打开真实 URL/hash 路由、console.error/pageerror/requestfailed、DOM 文本断言、截图和截图非空检查。页面或脚本验证失败时，先根据工具返回的错误和截图路径定位，再改代码/配置并复跑。工具返回截图时，最终回复必须把 markdown_image 字段原样输出为 Markdown 图片，不要只给本地路径。"
		instructions += "\n\n浏览器工具环境事实：" + agentBrowserCheckToolName + " 是后端内置工具，会在允许工作区内解析已有 Playwright；不要把“安装 Playwright”作为第一轮用户确认项。若工具返回 Cannot find module 'playwright'，先用 " + agentRunCommandToolName + " 检查工作区 node_modules/playwright 和 node -e \"require.resolve('playwright')\"，这是工具运行路径/依赖配置问题，应继续定位并复跑，而不是让用户提供安装步骤。"
		instructions += "\n\nWebshell 浏览器验收规则：当用户要求登录 webshell、执行命令并留下截图证据时，先打开真实 webshell URL 并通过页面/服务/本地库证据判断已有服务器和系统用户，不要把 SSH 账号密码作为默认必问项；本项目 webshell 登录会通过 project.server_os_users_query/webshell.gen_token 使用后端已保存的服务器用户。页面出现 webshell 登录弹窗时，优先选择现有/默认用户并点击登录，再在终端执行用户命令、截图、inspect_image 复核。"
	}
	if policy.Enabled && policy.ImageInspection && len(policy.AllowedRoots) > 0 {
		instructions += "\n\n图片检查工具：使用 " + agentImageInspectToolName + " 检查截图文件是否存在、尺寸是否正确、是否空白/单色、颜色和亮度统计是否正常。涉及视觉验收时必须把浏览器截图和图片检查结果作为证据，并在最终回复中展示 markdown_image 字段对应的图片。"
	}
	if policy.Enabled && policy.LocalProjectFileSearch && policy.LocalProjectFileMutation && policy.LocalProjectFileDelete && len(policy.AllowedRoots) > 0 {
		toolList := agentFileReadToolName + " 用于读取文件；" + agentCodexCLIGlobToolName + " 用于 glob/模糊查找文件；" + agentCodexCLIGrepToolName + " 用于 regex/literal/fuzzy 检索文件内容；" + agentCodexCLIEditToolName + " 用于增量修改/编辑文本文件；" + agentCodexCLIDeleteToolName + " 用于删除单个非关键文件且必须用户确认"
		if policy.LocalProjectCommand {
			toolList += "；" + agentRunCommandToolName + " 用于运行测试/构建/脚本/接口检查"
		}
		if policy.BrowserValidation {
			toolList += "；" + agentBrowserCheckToolName + " 用于无头浏览器登录、截图和页面验证"
		}
		if policy.ImageInspection {
			toolList += "；" + agentImageInspectToolName + " 用于截图/图片非空和尺寸检查"
		}
		instructions += "\n\n当用户询问“你支持什么工具”“有没有文件查找/编辑/测试/浏览器/图片工具”时，必须明确回答：" + toolList + "。"
	}
	return instructions
}

func buildAgentCodexLikeCoreInstructions() string {
	return strings.Join([]string{
		"Codex CLI 对齐规则：",
		"- 你是当前仓库里的工程代理，不是泛泛聊天助手；优先完成用户目标。",
		"- 回答项目、代码、配置、低代码页面、接口链路、错误定位、实现或修改问题前，必须先用可用工具搜索/读取相关文件，不能凭记忆猜路径或结论。",
		"- 启动时等同已加载仓库规则；必须遵守下面自动注入的 AGENTS.md/项目上下文。",
		"- 用户只说“继续”时，沿用当前会话最近的任务继续推进，不要重新解释背景。",
		"- 输出要落到证据：关键文件、触发链路、已确认结论、修改点、验证命令或验证结果。信息不足时说明缺口并继续用工具补证据。",
		"- 需要改代码时保持小范围、匹配现有模式；修改后运行最小有效验证，并明确任何失败原因。",
		"- 做前端/低代码页面工作时，优先用无头浏览器打开真实 URL，收集 console/page/request 错误，截图并检查图片是否正常；失败就继续定位和修正，直到验证通过或说明具体阻塞。",
	}, "\n")
}

func buildAgentProjectContext(policy agentToolPolicy) string {
	if !policy.Enabled || len(policy.AllowedRoots) == 0 {
		return ""
	}
	remaining := agentProjectContextMaxBytes
	sections := make([]string, 0, 2)
	for _, root := range policy.AllowedRoots {
		if remaining <= 0 {
			break
		}
		if text := readAgentInstructionFile(root, "AGENTS.md", remaining); text != "" {
			sections = append(sections, fmt.Sprintf("[AGENTS.md @ %s]\n%s", root, text))
			remaining -= len(text)
		}
	}
	if len(sections) == 0 {
		return ""
	}
	return "自动加载的项目上下文（按 Codex CLI 工作区规则注入，回答时优先遵守，不要整段复述）：\n\n" + strings.Join(sections, "\n\n")
}

func readAgentInstructionFile(root string, relativePath string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	if cleanRoot == "" {
		return ""
	}
	path := filepath.Clean(filepath.Join(cleanRoot, relativePath))
	if path != cleanRoot && !strings.HasPrefix(path, cleanRoot+string(os.PathSeparator)) {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return ""
	}
	return trimAgentInstructionText(text, maxBytes)
}

func trimAgentInstructionText(text string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(text) <= maxBytes {
		return text
	}
	cut := text[:maxBytes]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return strings.TrimSpace(cut) + "\n...(truncated)"
}

func readAgentProviderSSE(body io.Reader, onDelta agentProviderStreamFunc) (*agentProviderResult, error) {
	return readAgentProviderSSEWithActivity(body, onDelta, nil)
}

func readAgentProviderSSEWithActivity(body io.Reader, onDelta agentProviderStreamFunc, onActivity func()) (*agentProviderResult, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	result := &agentProviderResult{}
	var outputBuilder strings.Builder
	lastDoneText := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && onActivity != nil {
			onActivity()
		}
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var event chatGPTCodexEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		if event.Response != nil && strings.TrimSpace(event.Response.ID) != "" {
			result.ResponseID = strings.TrimSpace(event.Response.ID)
		}
		if event.Response != nil && len(event.Response.Usage) > 0 {
			result.Usage = event.Response.Usage
		}
		if len(event.Usage) > 0 {
			result.Usage = event.Usage
		}
		switch event.Type {
		case "response.output_text.delta":
			outputBuilder.WriteString(event.Delta)
			if onDelta != nil && event.Delta != "" {
				onDelta(event.Delta)
			}
		case "response.output_text.done":
			lastDoneText = event.Text
		case "response.output_item.done":
			if call, ok := parseAgentToolCallItem(event.Item); ok {
				result.ToolCalls = append(result.ToolCalls, call)
			}
		case "response.completed":
			if lastDoneText != "" && outputBuilder.Len() == 0 {
				outputBuilder.WriteString(lastDoneText)
				if onDelta != nil {
					onDelta(lastDoneText)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	result.OutputText = strings.TrimSpace(outputBuilder.String())
	if result.OutputText == "" {
		result.OutputText = strings.TrimSpace(lastDoneText)
	}
	return result, nil
}

func readAgentProviderSSEWithIdleTimeout(ctx context.Context, body io.ReadCloser, onDelta agentProviderStreamFunc, idleTimeout time.Duration) (*agentProviderResult, error) {
	if body == nil {
		return nil, fmt.Errorf("Agent provider SSE 响应体为空")
	}
	if idleTimeout <= 0 {
		return readAgentProviderSSE(body, onDelta)
	}
	type readResult struct {
		result *agentProviderResult
		err    error
	}
	doneCh := make(chan readResult, 1)
	activityCh := make(chan struct{}, 1)
	onActivity := func() {
		select {
		case activityCh <- struct{}{}:
		default:
		}
	}
	go func() {
		result, err := readAgentProviderSSEWithActivity(body, onDelta, onActivity)
		doneCh <- readResult{result: result, err: err}
	}()
	timer := time.NewTimer(idleTimeout)
	defer timer.Stop()
	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(idleTimeout)
	}
	for {
		select {
		case result := <-doneCh:
			return result.result, result.err
		case <-activityCh:
			resetTimer()
		case <-timer.C:
			_ = body.Close()
			return nil, fmt.Errorf("Agent provider SSE 读取超时: %w", context.DeadlineExceeded)
		case <-ctx.Done():
			_ = body.Close()
			return nil, ctx.Err()
		}
	}
}

func parseAgentToolCallItem(raw json.RawMessage) (agentToolCall, bool) {
	if len(raw) == 0 {
		return agentToolCall{}, false
	}
	var item openAIResponseOutputItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return agentToolCall{}, false
	}
	return responseOutputItemToolCall(item)
}

func responseOutputItemToolCall(item openAIResponseOutputItem) (agentToolCall, bool) {
	if strings.TrimSpace(item.Type) != "function_call" {
		return agentToolCall{}, false
	}
	call := agentToolCall{
		ID:        strings.TrimSpace(item.ID),
		CallID:    strings.TrimSpace(item.CallID),
		Name:      strings.TrimSpace(item.Name),
		Arguments: strings.TrimSpace(item.Arguments),
	}
	if call.CallID == "" {
		call.CallID = call.ID
	}
	if call.Name == "" {
		return agentToolCall{}, false
	}
	if call.Arguments == "" {
		call.Arguments = "{}"
	}
	return call, true
}

func extractAgentToolCalls(openaiResp openAIResponse) []agentToolCall {
	var calls []agentToolCall
	for _, item := range openaiResp.Output {
		if call, ok := responseOutputItemToolCall(item); ok {
			calls = append(calls, call)
		}
	}
	return calls
}

func extractAgentOutputText(openaiResp openAIResponse) string {
	if text := strings.TrimSpace(openaiResp.OutputText); text != "" {
		return text
	}
	var builder strings.Builder
	for _, item := range openaiResp.Output {
		if item.Type != "message" {
			continue
		}
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				builder.WriteString(content.Text)
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

func mergeAgentUsage(dst map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = map[string]interface{}{}
	}
	for key, value := range src {
		switch typed := value.(type) {
		case map[string]interface{}:
			existing, _ := dst[key].(map[string]interface{})
			dst[key] = mergeAgentUsage(existing, typed)
		case float64:
			dst[key] = gocast.ToFloat64(dst[key]) + typed
		case float32:
			dst[key] = gocast.ToFloat64(dst[key]) + float64(typed)
		case int:
			dst[key] = gocast.ToFloat64(dst[key]) + float64(typed)
		case int64:
			dst[key] = gocast.ToFloat64(dst[key]) + float64(typed)
		case int32:
			dst[key] = gocast.ToFloat64(dst[key]) + float64(typed)
		case json.Number:
			if n, err := typed.Float64(); err == nil {
				dst[key] = gocast.ToFloat64(dst[key]) + n
			}
		default:
			if _, exists := dst[key]; !exists {
				dst[key] = value
			}
		}
	}
	return dst
}

func attachAgentUsage(result *agentProviderResult, usage map[string]interface{}) {
	if result == nil || len(usage) == 0 {
		return
	}
	result.Usage = usage
	if result.RawJSON == nil {
		result.RawJSON = map[string]interface{}{}
	}
	result.RawJSON["usage"] = usage
}

func callResponsesProviderOnce(ctx context.Context, session *base.AgentSession, inputText string, credential agentCredential, policy agentToolPolicy, previousResponseID string, toolOutputs []map[string]interface{}, onDelta agentProviderStreamFunc, onDebug agentProviderDebugFunc) (*agentProviderResult, error) {
	stream := onDelta != nil || credential.Mode == "chatgpt_access_token"
	requestBody := buildAgentResponsesRequestBody(session, inputText, credential, policy, previousResponseID, toolOutputs, stream)
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	baseURL := resolveAgentBaseURL(credential)
	requestDebug := buildAgentProviderRequestDebug(requestBody, body, credential, baseURL)
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, wrapAgentProviderRequestDebugError(err, requestDebug)
	}
	req.Header.Set("Authorization", "Bearer "+credential.Token)
	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	applyAgentProviderCodexHeaders(req, session, credential)
	if credential.Mode == "chatgpt_access_token" && credential.AccountID != "" {
		req.Header.Set("ChatGPT-Account-ID", credential.AccountID)
	} else if credential.AccountID != "" {
		req.Header.Set("OpenAI-Account-ID", credential.AccountID)
	}
	requestDebug["request_headers"] = buildAgentProviderRequestHeadersDebug(req)
	if onDebug != nil {
		onDebug(requestDebug)
	}

	resp, err := newAgentProviderHTTPClient(stream).Do(req)
	if err != nil {
		err = wrapAgentProviderResponseHeaderTimeoutError(err, agentProviderHeaderTimeout(stream))
		providerErr := fmt.Errorf("Agent provider 请求失败: mode=%s, source=%s, base_url=%s, provider_status=%s, err=%w", credential.Mode, credential.Source, baseURL, agentProviderHTTPStatusForLog(resp), err)
		return nil, wrapAgentProviderRequestDebugError(wrapAgentProviderHTTPStatusError(providerErr, resp), requestDebug)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var buf bytes.Buffer
		if _, readErr := buf.ReadFrom(resp.Body); readErr != nil {
			return nil, wrapAgentProviderRequestDebugError(wrapAgentProviderHTTPStatusError(readErr, resp), requestDebug)
		}
		var openaiResp openAIResponse
		_ = json.Unmarshal(buf.Bytes(), &openaiResp)
		return nil, wrapAgentProviderRequestDebugError(buildAgentProviderError(resp.StatusCode, buf.Bytes(), &openaiResp, credential, resp.Header), requestDebug)
	}

	if stream {
		result, err := readAgentProviderSSEWithIdleTimeout(ctx, resp.Body, onDelta, agentProviderStreamIdleTimeout)
		if err != nil {
			providerErr := fmt.Errorf("Agent provider SSE 读取失败: mode=%s, source=%s, provider_status=%s, err=%w", credential.Mode, credential.Source, agentProviderHTTPStatusForLog(resp), err)
			return nil, wrapAgentProviderRequestDebugError(wrapAgentProviderHTTPStatusError(providerErr, resp), requestDebug)
		}
		if result.OutputText == "" && len(result.ToolCalls) == 0 {
			providerErr := fmt.Errorf("Agent provider 返回为空: mode=%s, source=%s, base_url=%s, provider_status=%s", credential.Mode, credential.Source, baseURL, agentProviderHTTPStatusForLog(resp))
			return nil, wrapAgentProviderRequestDebugError(wrapAgentProviderHTTPStatusError(providerErr, resp), requestDebug)
		}
		result.RawJSON = map[string]interface{}{
			"id":          result.ResponseID,
			"output_text": result.OutputText,
			"mode":        credential.Mode,
			"source":      credential.Source,
			"base_url":    baseURL,
			"transport":   "sse",
		}
		if len(result.Usage) > 0 {
			result.RawJSON["usage"] = result.Usage
		}
		result.DebugRequests = []map[string]interface{}{requestDebug}
		return result, nil
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, wrapAgentProviderRequestDebugError(wrapAgentProviderHTTPStatusError(err, resp), requestDebug)
	}
	var openaiResp openAIResponse
	if err := json.Unmarshal(buf.Bytes(), &openaiResp); err != nil {
		providerErr := fmt.Errorf("Agent provider 响应解析失败: mode=%s, source=%s, provider_status=%s, err=%w, body=%s", credential.Mode, credential.Source, agentProviderHTTPStatusForLog(resp), err, strings.TrimSpace(buf.String()))
		return nil, wrapAgentProviderRequestDebugError(wrapAgentProviderHTTPStatusError(providerErr, resp), requestDebug)
	}
	outputText := extractAgentOutputText(openaiResp)
	toolCalls := extractAgentToolCalls(openaiResp)
	if outputText == "" && len(toolCalls) == 0 {
		providerErr := fmt.Errorf("Agent provider 返回为空: mode=%s, source=%s, base_url=%s, provider_status=%s", credential.Mode, credential.Source, baseURL, agentProviderHTTPStatusForLog(resp))
		return nil, wrapAgentProviderRequestDebugError(wrapAgentProviderHTTPStatusError(providerErr, resp), requestDebug)
	}
	return &agentProviderResult{
		ResponseID:    openaiResp.ID,
		OutputText:    outputText,
		Usage:         openaiResp.Usage,
		ToolCalls:     toolCalls,
		DebugRequests: []map[string]interface{}{requestDebug},
		RawJSON: map[string]interface{}{
			"id":          openaiResp.ID,
			"output_text": outputText,
			"mode":        credential.Mode,
			"source":      credential.Source,
			"base_url":    baseURL,
			"usage":       openaiResp.Usage,
		},
	}, nil
}

func callResponsesProviderWithAuthReloadRetry(ctx context.Context, session *base.AgentSession, inputText string, credential agentCredential, policy agentToolPolicy, previousResponseID string, toolOutputs []map[string]interface{}, onDelta agentProviderStreamFunc, onLog agentProviderLogFunc, onDebug agentProviderDebugFunc) (*agentProviderResult, agentCredential, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	current := credential
	maxAttempts := agentProviderMaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	startedAt := time.Now()
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, current, err
		}
		attemptStartedAt := time.Now()
		if attempt > 1 {
			event := agentProviderElapsedFields(startedAt, attemptStartedAt)
			event["type"] = "provider_attempt"
			event["attempt"] = attempt
			event["max_attempts"] = maxAttempts
			event["mode"] = current.Mode
			event["source"] = current.Source
			attachAgentCredentialFingerprintFields(event, current)
			event["message"] = fmt.Sprintf("正在重试模型服务（第 %d/%d 次，%s，已耗时 %s）", attempt, maxAttempts, formatAgentCredentialFingerprintForLog(current), event["elapsed_text"])
			emitAgentProviderLog(onLog, event)
		}
		sawDelta := false
		deltaFunc := onDelta
		if onDelta != nil {
			deltaFunc = func(delta string) {
				sawDelta = true
				onDelta(delta)
			}
		}
		result, err := callResponsesProviderOnce(ctx, session, inputText, current, policy, previousResponseID, toolOutputs, deltaFunc, onDebug)
		if err == nil {
			return result, current, nil
		}
		failureSummary := agentProviderFailureSummary(err)
		if errors.Is(err, context.Canceled) || (ctx.Err() != nil && errors.Is(err, context.DeadlineExceeded)) {
			event := agentProviderElapsedFields(startedAt, attemptStartedAt)
			event["type"] = "provider_cancelled"
			event["attempt"] = attempt
			event["max_attempts"] = maxAttempts
			event["mode"] = current.Mode
			event["source"] = current.Source
			event["message"] = fmt.Sprintf("本次请求已终止：%s（本次耗时 %s，总耗时 %s）", failureSummary, event["attempt_elapsed_text"], event["elapsed_text"])
			attachAgentProviderErrorFields(event, err)
			emitAgentProviderLog(onLog, event)
			return nil, current, err
		}
		shouldReloadCredential := shouldReloadCredentialAfterProviderError(err, current)
		shouldRetryProvider := !shouldReloadCredential && shouldRetryProviderAfterTransientError(err)
		if sawDelta || attempt >= maxAttempts || (!shouldReloadCredential && !shouldRetryProvider) {
			event := agentProviderElapsedFields(startedAt, attemptStartedAt)
			event["type"] = "provider_failed"
			event["attempt"] = attempt
			event["max_attempts"] = maxAttempts
			event["mode"] = current.Mode
			event["source"] = current.Source
			attachAgentCredentialFingerprintFields(event, current)
			event["message"] = fmt.Sprintf("模型请求失败：%s，停止重试（%s，本次耗时 %s，总耗时 %s）", failureSummary, formatAgentCredentialFingerprintForLog(current), event["attempt_elapsed_text"], event["elapsed_text"])
			attachAgentProviderErrorFields(event, err)
			emitAgentProviderLog(onLog, event)
			return nil, current, err
		}

		if shouldReloadCredential {
			event := agentProviderElapsedFields(startedAt, attemptStartedAt)
			event["type"] = "auth_reload_retry"
			event["attempt"] = attempt + 1
			event["max_attempts"] = maxAttempts
			event["mode"] = current.Mode
			event["source"] = current.Source
			attachAgentCredentialFingerprintFields(event, current)
			event["message"] = fmt.Sprintf("模型请求失败：%s，正在重新读取 auth.json 后重试（第 %d/%d 次，当前%s，本次耗时 %s，总耗时 %s）", failureSummary, attempt+1, maxAttempts, formatAgentCredentialFingerprintForLog(current), event["attempt_elapsed_text"], event["elapsed_text"])
			attachAgentProviderErrorFields(event, err)
			emitAgentProviderLog(onLog, event)
			refreshedCredential := resolveAgentCredentialWithReload(true)
			if refreshedCredential.Token == "" {
				event := agentProviderElapsedFields(startedAt, attemptStartedAt)
				event["type"] = "auth_reload_failed"
				event["attempt"] = attempt + 1
				event["max_attempts"] = maxAttempts
				event["mode"] = current.Mode
				event["source"] = current.Source
				attachAgentCredentialFingerprintFields(event, current)
				event["message"] = fmt.Sprintf("auth.json 已重新读取，但没有可用 token，停止重试（原%s，总耗时 %s）", formatAgentCredentialFingerprintForLog(current), event["elapsed_text"])
				emitAgentProviderLog(onLog, event)
				return nil, current, err
			}
			credentialChanged := agentCredentialChanged(current, refreshedCredential)
			stopForUnchangedCredential := !credentialChanged && shouldStopAuthReloadRetryWhenCredentialUnchanged(err)
			if credentialChanged {
				event := agentProviderElapsedFields(startedAt, attemptStartedAt)
				event["type"] = "auth_reload_changed"
				event["attempt"] = attempt + 1
				event["max_attempts"] = maxAttempts
				event["mode"] = refreshedCredential.Mode
				event["source"] = refreshedCredential.Source
				attachAgentCredentialFingerprintFields(event, refreshedCredential)
				event["message"] = fmt.Sprintf("auth.json 已重新读取，凭据已更新：%s（总耗时 %s）", formatAgentCredentialChangeForLog(current, refreshedCredential), event["elapsed_text"])
				emitAgentProviderLog(onLog, event)
			} else {
				message := "auth.json 已重新读取，凭据未变化，继续重试请求，" + formatAgentCredentialFingerprintForLog(refreshedCredential)
				if stopForUnchangedCredential {
					message = "auth.json 已重新读取，凭据未变化，" + formatAgentCredentialFingerprintForLog(refreshedCredential)
				}
				event := agentProviderElapsedFields(startedAt, attemptStartedAt)
				event["type"] = "auth_reload_unchanged"
				event["attempt"] = attempt + 1
				event["max_attempts"] = maxAttempts
				event["mode"] = refreshedCredential.Mode
				event["source"] = refreshedCredential.Source
				attachAgentCredentialFingerprintFields(event, refreshedCredential)
				event["message"] = fmt.Sprintf("%s（总耗时 %s）", message, event["elapsed_text"])
				emitAgentProviderLog(onLog, event)
			}
			if stopForUnchangedCredential {
				event := agentProviderElapsedFields(startedAt, attemptStartedAt)
				event["type"] = "auth_reload_unchanged_stop"
				event["attempt"] = attempt + 1
				event["max_attempts"] = maxAttempts
				event["mode"] = refreshedCredential.Mode
				event["source"] = refreshedCredential.Source
				attachAgentCredentialFingerprintFields(event, refreshedCredential)
				event["message"] = fmt.Sprintf("auth.json 重新读取后凭据未变化，停止重复等待：%s（%s，总耗时 %s）", failureSummary, formatAgentCredentialFingerprintForLog(refreshedCredential), event["elapsed_text"])
				attachAgentProviderErrorFields(event, err)
				emitAgentProviderLog(onLog, event)
				return nil, current, err
			}
			current = refreshedCredential
		} else {
			event := agentProviderElapsedFields(startedAt, attemptStartedAt)
			event["type"] = "provider_transient_retry"
			event["attempt"] = attempt + 1
			event["max_attempts"] = maxAttempts
			event["mode"] = current.Mode
			event["source"] = current.Source
			attachAgentCredentialFingerprintFields(event, current)
			event["message"] = fmt.Sprintf("模型请求失败：%s，将直接重试模型服务（第 %d/%d 次，%s，临时错误不重新读取 auth.json，本次耗时 %s，总耗时 %s）", failureSummary, attempt+1, maxAttempts, formatAgentCredentialFingerprintForLog(current), event["attempt_elapsed_text"], event["elapsed_text"])
			attachAgentProviderErrorFields(event, err)
			emitAgentProviderLog(onLog, event)
		}
		wait := agentProviderRetryWaitDuration(err, attempt)
		event := agentProviderElapsedFields(startedAt, attemptStartedAt)
		event["type"] = "provider_retry_wait"
		event["attempt"] = attempt + 1
		event["max_attempts"] = maxAttempts
		event["mode"] = current.Mode
		event["source"] = current.Source
		event["wait_ms"] = wait.Milliseconds()
		event["wait_text"] = formatAgentProviderDuration(wait)
		attachAgentCredentialFingerprintFields(event, current)
		event["message"] = fmt.Sprintf("等待 %s 后继续重试（已耗时 %s）", event["wait_text"], event["elapsed_text"])
		emitAgentProviderLog(onLog, event)
		select {
		case <-ctx.Done():
			return nil, current, ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil, current, fmt.Errorf("Agent provider 重试失败")
}

func emitAgentProviderLog(onLog agentProviderLogFunc, event map[string]interface{}) {
	if onLog == nil {
		return
	}
	if event == nil {
		event = map[string]interface{}{}
	}
	event["create_time"] = nowText()
	onLog(event)
}

func registerAgentRunCancel(agentRunID string, cancel context.CancelFunc) {
	agentRunID = strings.TrimSpace(agentRunID)
	if agentRunID == "" || cancel == nil {
		return
	}
	agentRunCancelMu.Lock()
	agentRunCancelMap[agentRunID] = cancel
	agentRunCancelMu.Unlock()
}

func unregisterAgentRunCancel(agentRunID string) {
	agentRunID = strings.TrimSpace(agentRunID)
	if agentRunID == "" {
		return
	}
	agentRunCancelMu.Lock()
	delete(agentRunCancelMap, agentRunID)
	agentRunCancelMu.Unlock()
}

func cancelAgentRun(agentRunID string) bool {
	agentRunID = strings.TrimSpace(agentRunID)
	if agentRunID == "" {
		return false
	}
	agentRunCancelMu.Lock()
	cancel := agentRunCancelMap[agentRunID]
	agentRunCancelMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func markAgentRunCancelled(run *base.AgentRun, assistantMessage *base.AgentMessage, cause error) error {
	msg := "用户已终止本次请求"
	if cause != nil && !errors.Is(cause, context.Canceled) {
		msg += "：" + cause.Error()
	}
	if run != nil {
		run.Status = runStatusCancelled
		run.CurrentStep = runStatusCancelled
		run.ErrorMsg = msg
		run.FinishedAt = nowText()
		run.HeartbeatTime = run.FinishedAt
		run.LeaseExpireTime = run.FinishedAt
		run.ModifyTime = run.FinishedAt
		_, _ = callAgentLowcodeService("agent.run_cancel_update", map[string]interface{}{
			"agent_run_id": run.AgentRunID,
			"error_msg":    msg,
		})
	}
	if assistantMessage != nil {
		_ = updateAgentMessageContent(assistantMessage.AgentMessageID, msg, "", runStatusCancelled)
	} else if run != nil {
		_, _ = callAgentLowcodeService("agent.message_cancel_update", map[string]interface{}{
			"agent_run_id": run.AgentRunID,
			"content_text": msg,
		})
	}
	return errors.New(msg)
}

func buildAgentToolOutputs(results []agentToolResult) []map[string]interface{} {
	outputs := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		if strings.TrimSpace(result.CallID) == "" {
			continue
		}
		outputs = append(outputs, map[string]interface{}{
			"type":    "function_call_output",
			"call_id": result.CallID,
			"output":  agentToolOutputForProvider(result.Output),
		})
	}
	return outputs
}

func appendAgentToolContinuationInput(inputItems []map[string]interface{}, calls []agentToolCall, results []agentToolResult) []map[string]interface{} {
	resultByCallID := make(map[string]agentToolResult, len(results))
	for _, result := range results {
		resultByCallID[result.CallID] = result
	}
	for _, call := range calls {
		callID := strings.TrimSpace(call.CallID)
		if callID == "" {
			callID = strings.TrimSpace(call.ID)
		}
		if callID == "" {
			continue
		}
		callItem := map[string]interface{}{
			"type":      "function_call",
			"call_id":   callID,
			"name":      call.Name,
			"arguments": call.Arguments,
			"status":    "completed",
		}
		if strings.TrimSpace(call.ID) != "" {
			callItem["id"] = call.ID
		}
		inputItems = append(inputItems, callItem)
		result := resultByCallID[callID]
		inputItems = append(inputItems, map[string]interface{}{
			"type":    "function_call_output",
			"call_id": callID,
			"output":  agentToolOutputForProvider(result.Output),
		})
	}
	return inputItems
}

func appendAgentToolLimitMessage(inputItems []map[string]interface{}, maxToolRounds int) []map[string]interface{} {
	text := fmt.Sprintf("工具调用已达到本次上限（%d 轮）。不要继续调用工具；请只基于已经返回的工具结果，输出当前已确认的文件、调用链路、结论、剩余不确定点，以及可以由用户或下一次运行继续执行的命令。", maxToolRounds)
	return append(inputItems, map[string]interface{}{
		"type": "message",
		"role": "user",
		"content": []map[string]interface{}{
			{
				"type": "input_text",
				"text": text,
			},
		},
	})
}

func shouldRunAgentWebshellPreflight(inputText string, policy agentToolPolicy) bool {
	if !policy.Enabled || len(policy.AllowedRoots) == 0 {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(inputText))
	if !strings.Contains(text, "webshell") {
		return false
	}
	for _, keyword := range []string{"登录", "登錄", "login", "auth", "token", "逻辑", "邏輯", "分析", "梳理"} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func shouldRunAgentCodexLikePreflight(inputText string, policy agentToolPolicy) bool {
	if !policy.Enabled || len(policy.AllowedRoots) == 0 {
		return false
	}
	if !policy.LocalProjectFileRead && !policy.LocalProjectFileSearch {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(inputText))
	if text == "" {
		return false
	}
	for _, keyword := range []string{
		"codex", "cli", "agent", "agents.md", "system_prompt", "tool_policy",
		"聊天框", "提示词", "系统提示", "核心提示", "工具链", "工具调用", "工程代理",
		"agent_regression", "run_stream", "gpt-5.3-codex",
	} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func newAgentPreflightToolCall(name string, args map[string]interface{}, seq int) agentToolCall {
	data, _ := json.Marshal(args)
	callID := fmt.Sprintf("preflight_%s_%d", name, seq)
	return agentToolCall{
		ID:        callID,
		CallID:    callID,
		Name:      name,
		Arguments: string(data),
	}
}

func buildAgentWebshellPreflightCalls(policy agentToolPolicy) []agentToolCall {
	calls := make([]agentToolCall, 0, agentPreflightMaxToolCalls)
	appendCall := func(name string, args map[string]interface{}) {
		if len(args) == 0 {
			return
		}
		calls = append(calls, newAgentPreflightToolCall(name, args, len(calls)+1))
	}
	if policy.LocalProjectFileSearch {
		appendCall(agentCodexCLIGrepToolName, map[string]interface{}{
			"pattern":        "rightMenuAction|webshell-login-form|webshell\\.gen_token|gen_token_from_token|project\\.server_os_users_query|loginVisible|lastDblData|server_os_users_id|\"tag\"\\s*:\\s*\"ssh\"",
			"glob":           "collect/frontend/page_data/data/server/webshell_ssh_fragment.json",
			"regex":          true,
			"context_lines":  1,
			"max_results":    80,
			"case_sensitive": false,
		})
		appendCall(agentCodexCLIGrepToolName, map[string]interface{}{
			"pattern":        "gen_token|get_info_by_token|gen_token_from_token|update_invalid_token|update_finish_token|session_user_id|webshell_token",
			"glob":           "collect/webshell/token/index.yml",
			"regex":          true,
			"context_lines":  1,
			"max_results":    60,
			"case_sensitive": false,
		})
		appendCall(agentCodexCLIGrepToolName, map[string]interface{}{
			"pattern":        "module_ssh|shell_term|SshName|ssh\\.Dial|session_user_id",
			"paths":          []string{"plugins/module_ssh.go", "plugins/handler_params_shell_term.go"},
			"regex":          true,
			"context_lines":  1,
			"max_results":    50,
			"case_sensitive": false,
		})
	}
	if policy.LocalProjectFileRead {
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":      "collect/webshell/service.yml",
			"max_bytes": 12000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "collect/webshell/token/index.yml",
			"start_line": 248,
			"end_line":   357,
			"max_bytes":  22000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "collect/frontend/page_data/index.yml",
			"start_line": 184,
			"end_line":   222,
			"max_bytes":  12000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "collect/frontend/page_data/data/server/webshell.json",
			"start_line": 1,
			"end_line":   130,
			"max_bytes":  12000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "collect/frontend/page_data/data/server/webshell_ssh_fragment.json",
			"start_line": 180,
			"end_line":   235,
			"max_bytes":  16000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "collect/frontend/page_data/data/server/webshell_ssh_fragment.json",
			"start_line": 740,
			"end_line":   865,
			"max_bytes":  26000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "collect/frontend/page_data/data/server/webshell_ssh_fragment.json",
			"start_line": 1095,
			"end_line":   1120,
			"max_bytes":  10000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "collect/frontend/page_data/data/server/webshell_ssh_fragment.json",
			"start_line": 1220,
			"end_line":   1310,
			"max_bytes":  22000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "collect/project/server/index.yml",
			"start_line": 454,
			"end_line":   474,
			"max_bytes":  10000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":      "collect/project/server/connect_server.json",
			"max_bytes": 12000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":      "collect/webshell/token/get_info_by_token.sql",
			"max_bytes": 6000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "collect/project/server_os_users/index.yml",
			"start_line": 188,
			"end_line":   205,
			"max_bytes":  6000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":      "collect/project/server_os_users/server_os_users_query.sql",
			"max_bytes": 6000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "main.go",
			"start_line": 368,
			"end_line":   372,
			"max_bytes":  4000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "conf/application.properties",
			"start_line": 67,
			"end_line":   74,
			"max_bytes":  4000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "/data/project/sport-ui/src/components/ssh.tsx",
			"start_line": 424,
			"end_line":   434,
			"max_bytes":  6000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "/data/project/collect/src/collect/service_imp/service_template_service.go",
			"start_line": 303,
			"end_line":   325,
			"max_bytes":  8000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "/data/project/collect/src/collect/service_imp/service_template_service.go",
			"start_line": 513,
			"end_line":   560,
			"max_bytes":  12000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":       "/data/project/collect/src/collect/service_imp/service_before_plugin.go",
			"start_line": 130,
			"end_line":   149,
			"max_bytes":  8000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":      "plugins/module_ssh.go",
			"max_bytes": 12000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":      "plugins/handler_params_shell_term.go",
			"max_bytes": 12000,
		})
	}
	return calls
}

func buildAgentCodexLikePreflightCalls(policy agentToolPolicy) []agentToolCall {
	calls := make([]agentToolCall, 0, 8)
	appendCall := func(name string, args map[string]interface{}) {
		if len(args) == 0 {
			return
		}
		calls = append(calls, newAgentPreflightToolCall(name, args, len(calls)+1))
	}
	if policy.LocalProjectFileRead {
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":      "AGENTS.md",
			"max_bytes": 24000,
		})
		appendCall(agentFileReadToolName, map[string]interface{}{
			"path":      "go.mod",
			"max_bytes": 12000,
		})
	}
	if policy.LocalProjectFileSearch {
		appendCall(agentCodexCLIGrepToolName, map[string]interface{}{
			"pattern":        "buildAgentInstructions|defaultInstructions|system_prompt|tool_policy_json|run_stream|agent_regression|codexcli|AGENTS",
			"glob":           "plugins/**/*.go",
			"context_lines":  2,
			"max_results":    80,
			"case_sensitive": false,
		})
		appendCall(agentCodexCLIGrepToolName, map[string]interface{}{
			"pattern":        "system_prompt|tool_policy_json|agent_run|run_create|agent_regression",
			"glob":           "collect/agent/**/*.yml",
			"context_lines":  2,
			"max_results":    60,
			"case_sensitive": false,
		})
		appendCall(agentCodexCLIGrepToolName, map[string]interface{}{
			"pattern":        "system_prompt|tool_policy_json|agent_regression|Codex|提示词|工具调用",
			"glob":           "collect/frontend/page_data/data/system/agent_regression.json",
			"context_lines":  1,
			"max_results":    60,
			"case_sensitive": false,
		})
	}
	return calls
}

func runAgentPreflightToolCalls(inputText string, policy agentToolPolicy, onTool agentToolEventFunc) []agentToolResult {
	var calls []agentToolCall
	switch {
	case shouldRunAgentWebshellPreflight(inputText, policy):
		calls = buildAgentWebshellPreflightCalls(policy)
	case shouldRunAgentCodexLikePreflight(inputText, policy):
		calls = buildAgentCodexLikePreflightCalls(policy)
	default:
		return nil
	}
	results := make([]agentToolResult, 0, len(calls)+6)
	seenCalls := make(map[string]bool)
	seenReadRanges := make(map[string]bool)
	nextSeq := len(calls) + 1
	dynamicReadCount := 0
	fallbackSearchCount := 0
	for i := 0; i < len(calls) && i < agentPreflightMaxToolCalls; i++ {
		call := calls[i]
		callKey := call.Name + "\x00" + call.Arguments
		if seenCalls[callKey] {
			continue
		}
		seenCalls[callKey] = true
		result := executeAgentToolCall(call, policy)
		results = append(results, result)
		if onTool != nil {
			onTool(result)
		}
		if call.Name != agentCodexCLIGrepToolName {
			continue
		}
		if agentToolOutputCount(result) == 0 && fallbackSearchCount < 3 {
			fallbacks := buildAgentFallbackSearchCalls(call, &nextSeq)
			for _, fallback := range fallbacks {
				key := fallback.Name + "\x00" + fallback.Arguments
				if seenCalls[key] {
					continue
				}
				calls = append(calls, fallback)
				fallbackSearchCount++
			}
			continue
		}
		if policy.LocalProjectFileRead && dynamicReadCount < 6 {
			reads := buildAgentReadCallsFromGrepResult(result, &nextSeq, seenReadRanges, 6-dynamicReadCount)
			calls = append(calls, reads...)
			dynamicReadCount += len(reads)
		}
	}
	return results
}

func decodeAgentToolOutput(result agentToolResult) map[string]interface{} {
	data := map[string]interface{}{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(result.Output)), &data)
	return data
}

func agentToolOutputCount(result agentToolResult) int {
	return gocast.ToInt(decodeAgentToolOutput(result)["count"])
}

func buildAgentFallbackSearchCalls(call agentToolCall, nextSeq *int) []agentToolCall {
	args := map[string]interface{}{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(call.Arguments)), &args)
	glob := strings.TrimSpace(gocast.ToString(args["glob"]))
	if glob == "" {
		return nil
	}
	pattern := "login|auth|token|session_user_id|must_login|webshell|登录|鉴权"
	var fallbackGlob string
	switch {
	case strings.HasPrefix(glob, "collect/webshell/"):
		fallbackGlob = "collect/**/*.yml"
	case strings.HasPrefix(glob, "collect/frontend/"):
		fallbackGlob = "collect/frontend/page_data/**/*.json"
	case strings.HasPrefix(glob, "plugins/"):
		fallbackGlob = "**/*.go"
	default:
		return nil
	}
	callSeq := *nextSeq
	*nextSeq = *nextSeq + 1
	return []agentToolCall{
		newAgentPreflightToolCall(agentCodexCLIGrepToolName, map[string]interface{}{
			"pattern":        pattern,
			"glob":           fallbackGlob,
			"regex":          true,
			"context_lines":  1,
			"max_results":    80,
			"case_sensitive": false,
		}, callSeq),
	}
}

func buildAgentReadCallsFromGrepResult(result agentToolResult, nextSeq *int, seen map[string]bool, limit int) []agentToolCall {
	if limit <= 0 {
		return nil
	}
	output := decodeAgentToolOutput(result)
	rawResults, _ := output["results"].([]interface{})
	calls := make([]agentToolCall, 0, limit)
	for _, raw := range rawResults {
		item, _ := raw.(map[string]interface{})
		if len(item) == 0 {
			continue
		}
		path := strings.TrimSpace(gocast.ToString(item["relative_path"]))
		if path == "" {
			path = strings.TrimSpace(gocast.ToString(item["path"]))
		}
		if path == "" {
			continue
		}
		line := gocast.ToInt(item["line"])
		startLine := line - 40
		if startLine < 1 {
			startLine = 1
		}
		endLine := line + 90
		key := path
		if seen[key] {
			continue
		}
		seen[key] = true
		callSeq := *nextSeq
		*nextSeq = *nextSeq + 1
		calls = append(calls, newAgentPreflightToolCall(agentFileReadToolName, map[string]interface{}{
			"path":       path,
			"start_line": startLine,
			"end_line":   endLine,
			"max_bytes":  22000,
		}, callSeq))
		if len(calls) >= limit {
			break
		}
	}
	return calls
}

func appendAgentPreflightEvidence(inputText string, results []agentToolResult) string {
	if len(results) == 0 {
		return inputText
	}
	var builder strings.Builder
	builder.WriteString("系统已先执行本地项目只读工具调用，下面是压缩后的关键证据。回答时必须基于这些结果说明关键文件、接口链路和仍需验证的点；不要声称没有读取文件。若这些结果已覆盖用户要求的链路，直接总结，不要重复调用工具；确需补证据时优先用结构化 grep/read_project_file，不要用 run_command 包 grep/cat/sed/awk 做大范围读取。\n")
	remaining := agentPreflightMaxBytes
	if guide := buildAgentPreflightEvidenceGuide(results); guide != "" {
		builder.WriteString(guide)
		remaining -= len([]byte(guide))
	}
	for i, result := range results {
		if remaining <= 0 {
			builder.WriteString("\n[preflight truncated]\n预读证据已达到输入预算；完整工具结果仍保存在本次运行的 tool_results/debug 中。\n")
			break
		}
		rendered := renderAgentPreflightToolResult(i+1, result)
		if len([]byte(rendered)) > remaining {
			rendered = truncateAgentToolText(rendered, remaining)
		}
		builder.WriteString(rendered)
		remaining -= len([]byte(rendered))
	}
	builder.WriteString("\n用户最新问题：\n")
	builder.WriteString(inputText)
	return builder.String()
}

func renderAgentPreflightToolResult(seq int, result agentToolResult) string {
	budget := agentPreflightToolBudget(result)
	output := compactAgentPreflightToolOutput(result, budget)
	return fmt.Sprintf("\n[tool %d] %s\narguments: %s\noutput:\n%s\n", seq, result.Name, compactAgentPreflightArguments(result.Arguments), output)
}

func compactAgentPreflightArguments(raw string) string {
	args := map[string]interface{}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &args); err != nil || len(args) == 0 {
		return truncateAgentToolText(strings.TrimSpace(raw), 600)
	}
	compact := map[string]interface{}{}
	for _, key := range []string{"path", "glob", "pattern", "query", "start_line", "end_line", "max_bytes", "max_results", "regex", "context_lines", "case_sensitive"} {
		if value, ok := args[key]; ok {
			compact[key] = value
		}
	}
	if value, ok := args["paths"]; ok {
		compact["paths"] = value
	}
	data, err := json.Marshal(compact)
	if err != nil {
		return truncateAgentToolText(strings.TrimSpace(raw), 600)
	}
	return string(data)
}

func agentPreflightToolBudget(result agentToolResult) int {
	switch result.Name {
	case agentCodexCLIGrepToolName:
		return 1800
	case agentCodexCLIGlobToolName:
		return 1200
	case agentFileReadToolName:
		args := map[string]interface{}{}
		_ = json.Unmarshal([]byte(strings.TrimSpace(result.Arguments)), &args)
		path := strings.TrimSpace(gocast.ToString(args["path"]))
		startLine := gocast.ToInt(args["start_line"])
		switch {
		case strings.Contains(path, "webshell_ssh_fragment.json") && (startLine >= 740 && startLine <= 1310):
			return 4200
		case strings.Contains(path, "collect/webshell/token/index.yml"):
			return 3400
		case strings.Contains(path, "plugins/module_ssh.go"), strings.Contains(path, "plugins/handler_params_shell_term.go"):
			return 3000
		case strings.Contains(path, "service_template_service.go"), strings.Contains(path, "service_before_plugin.go"):
			return 2600
		default:
			return 1800
		}
	default:
		return 1600
	}
}

func compactAgentPreflightToolOutput(result agentToolResult, budget int) string {
	if result.Error != "" {
		return truncateAgentToolText("ERROR: "+result.Error, budget)
	}
	data := map[string]interface{}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Output)), &data); err != nil || len(data) == 0 {
		return truncateAgentToolText(strings.TrimSpace(result.Output), budget)
	}
	switch result.Name {
	case agentCodexCLIGrepToolName:
		return compactAgentPreflightGrepOutput(data, budget)
	case agentCodexCLIGlobToolName:
		return compactAgentPreflightGlobOutput(data, budget)
	case agentFileReadToolName:
		return compactAgentPreflightReadOutput(data, budget)
	default:
		return truncateAgentToolText(strings.TrimSpace(result.Output), budget)
	}
}

func compactAgentPreflightGrepOutput(data map[string]interface{}, budget int) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("success=%v count=%d total=%d truncated=%v scanned_files=%d\n",
		data["success"], gocast.ToInt(data["count"]), gocast.ToInt(data["total"]), data["truncated"], gocast.ToInt(data["scanned_files"])))
	rows := agentPreflightGrepRows(data)
	limit := len(rows)
	if limit > 8 {
		limit = 8
	}
	if len(rows) > limit {
		builder.WriteString(fmt.Sprintf("showing_first=%d, %d more matches omitted\n", limit, len(rows)-limit))
	}
	for i := 0; i < limit; i++ {
		row, _ := rows[i].(map[string]interface{})
		if len(row) == 0 {
			continue
		}
		path := gocast.ToString(row["relative_path"])
		if path == "" {
			path = gocast.ToString(row["file"])
		}
		text := compactAgentPreflightInlineText(gocast.ToString(row["text"]), 160)
		builder.WriteString(fmt.Sprintf("- %s:%d %s\n", path, gocast.ToInt(row["line"]), text))
	}
	return truncateAgentToolText(strings.TrimSpace(builder.String()), budget)
}

func compactAgentPreflightInlineText(text string, maxBytes int) string {
	text = truncateAgentToolText(strings.TrimSpace(text), maxBytes)
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.Join(strings.Fields(text), " ")
}

func agentPreflightGrepRows(data map[string]interface{}) []interface{} {
	for _, key := range []string{"results", "matches", "results_preview"} {
		if rows := agentPreflightRows(data[key]); len(rows) > 0 {
			return rows
		}
	}
	if rows := agentPreflightRowsFromGroupedFiles(data["files"]); len(rows) > 0 {
		return rows
	}
	if previewJSON := strings.TrimSpace(gocast.ToString(data["preview_json"])); previewJSON != "" {
		preview := map[string]interface{}{}
		if err := json.Unmarshal([]byte(previewJSON), &preview); err == nil {
			return agentPreflightGrepRows(preview)
		}
	}
	return nil
}

func agentPreflightRowsFromGroupedFiles(raw interface{}) []interface{} {
	files := agentPreflightRows(raw)
	if len(files) == 0 {
		return nil
	}
	rows := make([]interface{}, 0)
	for _, rawFile := range files {
		fileData, _ := rawFile.(map[string]interface{})
		if len(fileData) == 0 {
			continue
		}
		file := gocast.ToString(fileData["file"])
		for _, rawMatch := range agentPreflightRows(fileData["matches"]) {
			match, _ := rawMatch.(map[string]interface{})
			if len(match) == 0 {
				continue
			}
			if gocast.ToString(match["relative_path"]) == "" {
				match["relative_path"] = file
			}
			rows = append(rows, match)
			if len(rows) >= 12 {
				return rows
			}
		}
	}
	return rows
}

func compactAgentPreflightGlobOutput(data map[string]interface{}, budget int) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("success=%v count=%d total=%d truncated=%v\n",
		data["success"], gocast.ToInt(data["count"]), gocast.ToInt(data["total"]), data["truncated"]))
	rows := agentPreflightRows(data["matches"])
	if len(rows) == 0 {
		rows = agentPreflightRows(data["results"])
	}
	limit := len(rows)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		row, _ := rows[i].(map[string]interface{})
		if len(row) == 0 {
			continue
		}
		builder.WriteString(fmt.Sprintf("- %s size=%d\n", gocast.ToString(row["relative_path"]), gocast.ToInt(row["size_bytes"])))
	}
	if len(rows) > limit {
		builder.WriteString(fmt.Sprintf("- ... %d more paths omitted\n", len(rows)-limit))
	}
	return truncateAgentToolText(strings.TrimSpace(builder.String()), budget)
}

func compactAgentPreflightReadOutput(data map[string]interface{}, budget int) string {
	path := gocast.ToString(data["relative_path"])
	if path == "" {
		path = gocast.ToString(data["path"])
	}
	startLine := gocast.ToInt(data["start_line"])
	if startLine <= 0 {
		startLine = 1
	}
	header := fmt.Sprintf("success=%v path=%s lines=%d-%d returned_lines=%d total_lines=%d truncated=%v\n",
		data["success"], path, startLine, gocast.ToInt(data["end_line"]), gocast.ToInt(data["returned_lines"]), gocast.ToInt(data["total_lines"]), data["truncated"])
	content := strings.TrimRight(gocast.ToString(data["content"]), "\n")
	if content == "" {
		return truncateAgentToolText(strings.TrimSpace(header), budget)
	}
	contentBudget := budget - len([]byte(header)) - 24
	if contentBudget < 600 {
		contentBudget = 600
	}
	snippet := numberAgentPreflightContentLines(content, startLine, contentBudget)
	return truncateAgentToolText(header+"content:\n"+snippet, budget)
}

func numberAgentPreflightContentLines(content string, startLine int, budget int) string {
	content = truncateAgentToolText(content, budget)
	lines := strings.Split(content, "\n")
	var builder strings.Builder
	for i, line := range lines {
		builder.WriteString(fmt.Sprintf("%d | %s\n", startLine+i, line))
	}
	return strings.TrimRight(builder.String(), "\n")
}

func agentPreflightRows(raw interface{}) []interface{} {
	switch rows := raw.(type) {
	case []interface{}:
		return rows
	case []map[string]interface{}:
		result := make([]interface{}, 0, len(rows))
		for _, row := range rows {
			result = append(result, row)
		}
		return result
	default:
		return nil
	}
}

func shouldAnswerFromAgentPreflightOnly(inputText string, policy agentToolPolicy, results []agentToolResult) bool {
	if len(results) == 0 || !agentPromptRequestsReadOnly(strings.ToLower(strings.TrimSpace(inputText))) {
		return false
	}
	if shouldRunAgentWebshellPreflight(inputText, policy) && buildAgentPreflightEvidenceGuide(results) != "" {
		return true
	}
	return false
}

func disableAgentInteractiveToolDefinitions(policy agentToolPolicy) agentToolPolicy {
	policy.LocalProjectFileRead = false
	policy.LocalProjectFileSearch = false
	policy.LocalProjectFileMutation = false
	policy.LocalProjectFileDelete = false
	policy.LocalProjectCommand = false
	policy.BrowserValidation = false
	policy.ImageInspection = false
	return policy
}

func buildAgentPreflightEvidenceGuide(results []agentToolResult) string {
	if len(results) == 0 {
		return ""
	}
	combined := strings.Builder{}
	for _, result := range results {
		combined.WriteString(result.Arguments)
		combined.WriteString("\n")
		combined.WriteString(result.Output)
		combined.WriteString("\n")
	}
	text := combined.String()
	if !strings.Contains(text, "webshell_ssh_fragment.json") || !strings.Contains(text, "webshell.gen_token") {
		return ""
	}
	var lines []string
	if strings.Contains(text, "project.server_os_users_query") && strings.Contains(text, "webshell-login-form") {
		lines = append(lines, "- 前端右键/登录菜单：`collect/frontend/page_data/data/server/webshell_ssh_fragment.json:191-230` 打开 `loginVisible`、保存 `lastDblData`、重置 `webshell-login-form`，先调用 `project.server_os_users_query(server_id)` 获取 `userList` 并回填 `server_os_users_id`；不要把这一步写成 `webshell.gen_token`。")
	}
	if strings.Contains(text, "last_webshell_token") {
		lines = append(lines, "- 登录弹窗提交：`webshell_ssh_fragment.json:743-755` 提交 `webshell-login-form` 后 `post:/template_data/data?service=webshell.gen_token`，用 `appendFormFields` 传 `server_os_users_id`，返回值写入 `last_webshell_token`。")
	}
	if strings.Contains(text, "gen_token_from_token") {
		lines = append(lines, "- 分屏/复制终端：`webshell_ssh_fragment.json:1220-1310` 使用 `webshell.gen_token_from_token` 从已有 token 重新生成 token。")
	}
	if strings.Contains(text, "HandlerWsRequest") && strings.Contains(text, "ws_service=project.connect_server") {
		lines = append(lines, "- WebSocket：`main.go` 暴露 `/template_data/ws/:token` 到 `HandlerWsRequest`，`conf/application.properties` 指定 `ws_service=project.connect_server`、`msg_service=webshell.term_msg`。")
	}
	if strings.Contains(text, "ssh.Dial") && strings.Contains(text, "ReadMessage") {
		lines = append(lines, "- SSH 执行：`plugins/module_ssh.go` 用 token 反查出的主机/用户密码执行 `ssh.Dial`，`plugins/handler_params_shell_term.go` 读取 WS 消息并写入 SSH stdin，处理 text/resize/pwd。")
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n关键链路提纲（由上面的预读范围提取，用于避免把相邻动作混淆）：\n" + strings.Join(lines, "\n") + "\n"
}

func callAgentProviderWithTools(ctx context.Context, session *base.AgentSession, inputText string, credential agentCredential, policy agentToolPolicy, onDelta agentProviderStreamFunc, onTool agentToolEventFunc, onLog agentProviderLogFunc) (*agentProviderResult, error) {
	maxToolRounds := policy.MaxToolRounds
	if maxToolRounds <= 0 {
		maxToolRounds = defaultAgentToolMaxRounds
	}
	previousResponseID := ""
	var toolOutputs []map[string]interface{}
	var chatGPTInput []map[string]interface{}
	allToolResults := runAgentPreflightToolCalls(inputText, policy, onTool)
	providerPolicy := policy
	if shouldAnswerFromAgentPreflightOnly(inputText, policy, allToolResults) {
		providerPolicy = disableAgentInteractiveToolDefinitions(policy)
	}
	var totalUsage map[string]interface{}
	var debugRequests []map[string]interface{}
	debugRequestKeys := map[string]bool{}
	debugRequestKey := func(debug map[string]interface{}) string {
		return fmt.Sprintf("%s|%s|%s|%s|%v|%v|%v|%s", gocast.ToString(debug["model"]), gocast.ToString(debug["mode"]), gocast.ToString(debug["token_fingerprint"]), gocast.ToString(debug["account_id_fingerprint"]), debug["request_bytes"], debug["input_bytes"], debug["function_call_output_bytes"], gocast.ToString(debug["request_sha256"]))
	}
	emitDebugRequests := func(requests []map[string]interface{}) {
		for _, debug := range requests {
			if len(debug) == 0 {
				continue
			}
			key := debugRequestKey(debug)
			if debugRequestKeys[key] {
				continue
			}
			debugRequestKeys[key] = true
			debugRequests = append(debugRequests, debug)
			if onTool != nil {
				onTool(newAgentProviderDebugToolResult(len(debugRequests), debug))
			}
		}
	}
	emitDebugRequest := func(debug map[string]interface{}) {
		emitDebugRequests([]map[string]interface{}{debug})
	}
	emitDebugRequestFromError := func(err error) {
		if debug := agentProviderRequestDebugFromError(err); len(debug) > 0 {
			emitDebugRequests([]map[string]interface{}{debug})
		}
	}
	if len(allToolResults) > 0 {
		inputText = appendAgentPreflightEvidence(inputText, allToolResults)
	}

	for round := 0; round < maxToolRounds; round++ {
		requestPreviousResponseID := previousResponseID
		requestToolOutputs := toolOutputs
		if credential.Mode == "chatgpt_access_token" {
			requestPreviousResponseID = ""
			requestToolOutputs = chatGPTInput
		}
		result, refreshedCredential, err := callResponsesProviderWithAuthReloadRetry(ctx, session, inputText, credential, providerPolicy, requestPreviousResponseID, requestToolOutputs, onDelta, onLog, emitDebugRequest)
		credential = refreshedCredential
		if err != nil {
			emitDebugRequestFromError(err)
			if len(allToolResults) > 0 {
				return nil, &agentToolRunError{Err: err, ToolResults: mergeAgentProviderDebugToolResults(debugRequests, allToolResults)}
			}
			return nil, &agentToolRunError{Err: err, ToolResults: mergeAgentProviderDebugToolResults(debugRequests, nil)}
		}
		emitDebugRequests(result.DebugRequests)
		totalUsage = mergeAgentUsage(totalUsage, result.Usage)
		if result.ResponseID != "" && credential.Mode != "chatgpt_access_token" {
			previousResponseID = result.ResponseID
		}
		if len(result.ToolCalls) == 0 {
			result.DebugRequests = debugRequests
			result.ToolResults = mergeAgentProviderDebugToolResults(debugRequests, allToolResults)
			attachAgentUsage(result, totalUsage)
			return result, nil
		}

		roundResults := make([]agentToolResult, 0, len(result.ToolCalls))
		for _, call := range result.ToolCalls {
			toolResult := executeAgentToolCall(call, policy)
			roundResults = append(roundResults, toolResult)
			if onTool != nil {
				onTool(toolResult)
			}
		}
		allToolResults = append(allToolResults, roundResults...)
		if credential.Mode == "chatgpt_access_token" {
			if len(chatGPTInput) == 0 {
				chatGPTInput = buildAgentInitialInputList(session, inputText)
			}
			chatGPTInput = appendAgentToolContinuationInput(chatGPTInput, result.ToolCalls, roundResults)
			if len(chatGPTInput) == 0 {
				return nil, fmt.Errorf("Agent 工具调用缺少 call_id，无法回传工具结果")
			}
			continue
		}
		toolOutputs = buildAgentToolOutputs(roundResults)
		if len(toolOutputs) == 0 {
			return nil, fmt.Errorf("Agent 工具调用缺少 call_id，无法回传工具结果")
		}
	}

	finalPolicy := providerPolicy
	finalPolicy.Enabled = false
	finalPolicy.LocalProjectFileRead = false
	finalPolicy.LocalProjectFileSearch = false
	finalPolicy.LocalProjectFileMutation = false
	finalPolicy.LocalProjectFileDelete = false
	if credential.Mode == "chatgpt_access_token" {
		if len(chatGPTInput) == 0 {
			chatGPTInput = buildAgentInitialInputList(session, inputText)
		}
		result, refreshedCredential, err := callResponsesProviderWithAuthReloadRetry(ctx, session, inputText, credential, finalPolicy, "", appendAgentToolLimitMessage(chatGPTInput, maxToolRounds), onDelta, onLog, emitDebugRequest)
		credential = refreshedCredential
		if err == nil {
			emitDebugRequests(result.DebugRequests)
			totalUsage = mergeAgentUsage(totalUsage, result.Usage)
			result.DebugRequests = debugRequests
			result.ToolResults = mergeAgentProviderDebugToolResults(debugRequests, allToolResults)
			attachAgentUsage(result, totalUsage)
			return result, nil
		}
		emitDebugRequestFromError(err)
		return nil, &agentToolRunError{Err: fmt.Errorf("Agent 工具调用达到上限后收尾失败: %w", err), ToolResults: mergeAgentProviderDebugToolResults(debugRequests, allToolResults)}
	}
	if len(toolOutputs) > 0 || previousResponseID != "" {
		result, refreshedCredential, err := callResponsesProviderWithAuthReloadRetry(ctx, session, inputText, credential, finalPolicy, previousResponseID, appendAgentToolLimitMessage(toolOutputs, maxToolRounds), onDelta, onLog, emitDebugRequest)
		credential = refreshedCredential
		if err == nil {
			emitDebugRequests(result.DebugRequests)
			totalUsage = mergeAgentUsage(totalUsage, result.Usage)
			result.DebugRequests = debugRequests
			result.ToolResults = mergeAgentProviderDebugToolResults(debugRequests, allToolResults)
			attachAgentUsage(result, totalUsage)
			return result, nil
		}
		emitDebugRequestFromError(err)
		return nil, &agentToolRunError{Err: fmt.Errorf("Agent 工具调用达到上限后收尾失败: %w", err), ToolResults: mergeAgentProviderDebugToolResults(debugRequests, allToolResults)}
	}
	return nil, fmt.Errorf("Agent 工具调用超过最大轮次(%d)", maxToolRounds)
}

func callChatGPTCodexProvider(ctx context.Context, session *base.AgentSession, inputText string, credential agentCredential, onDelta agentProviderStreamFunc, onLog agentProviderLogFunc) (*agentProviderResult, error) {
	result, _, err := callResponsesProviderWithAuthReloadRetry(ctx, session, inputText, credential, agentToolPolicy{}, "", nil, onDelta, onLog, nil)
	return result, err
}

func callPlatformResponsesProvider(ctx context.Context, session *base.AgentSession, inputText string, credential agentCredential, onDelta agentProviderStreamFunc, onLog agentProviderLogFunc) (*agentProviderResult, error) {
	result, _, err := callResponsesProviderWithAuthReloadRetry(ctx, session, inputText, credential, agentToolPolicy{}, "", nil, onDelta, onLog, nil)
	return result, err
}

func sessionTitle(input string) string {
	text := strings.TrimSpace(input)
	if text == "" {
		return "新会话"
	}
	if idx := strings.Index(text, "\n\n图片附件："); idx >= 0 {
		text = strings.TrimSpace(text[:idx])
	}
	text = strings.NewReplacer("\r", "\n", "\t", " ").Replace(text)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		text = line
		break
	}
	for _, sep := range []string{"。", "？", "?", "！", "!", "；", ";"} {
		if idx := strings.Index(text, sep); idx > 0 {
			text = strings.TrimSpace(text[:idx])
			break
		}
	}
	text = strings.Trim(text, " ，,。.!！?？:：；;、")
	for _, prefix := range []string{"请帮我", "帮我", "帮忙", "麻烦", "请", "我希望", "我想", "现状", "现在"} {
		text = strings.TrimSpace(strings.TrimPrefix(text, prefix))
	}
	if strings.Contains(text, "会话") && strings.Contains(text, "标题") {
		return "会话标题自动生成"
	}
	if strings.Contains(text, "工具名称") || strings.Contains(text, "工具名") {
		return "工具名称规范化"
	}
	runes := []rune(text)
	if len(runes) > 18 {
		runes = runes[:18]
	}
	title := strings.Trim(string(runes), " ，,。.!！?？:：；;、")
	if title == "" {
		return "新会话"
	}
	return title
}

func isGenericAgentSessionTitle(title string) bool {
	text := strings.TrimSpace(title)
	if text == "" {
		return true
	}
	compact := strings.NewReplacer(" ", "", "\t", "", "-", "", "_", "").Replace(text)
	if strings.Contains(compact, "新会话") {
		return true
	}
	switch compact {
	case "会话A", "会话B", "A会话", "B会话", "APane新会话", "BPane新会话", "主回归会话", "日志定位助手":
		return true
	default:
		return false
	}
}

func resolveAgentSessionTitle(current string, incoming string, inputText string) string {
	current = strings.TrimSpace(current)
	incoming = strings.TrimSpace(incoming)
	generated := sessionTitle(inputText)
	if generated != "" && generated != "新会话" && (current == "" || isGenericAgentSessionTitle(current)) && (incoming == "" || isGenericAgentSessionTitle(incoming)) {
		return generated
	}
	if incoming != "" && !isGenericAgentSessionTitle(incoming) {
		return incoming
	}
	if current != "" {
		return current
	}
	if generated != "" {
		return generated
	}
	if incoming != "" {
		return incoming
	}
	return "新会话"
}

func makeSessionKey() string {
	return "sess_" + uuid.NewString()
}

func getOrCreateAgentSession(params map[string]interface{}) (*base.AgentSession, error) {
	sessionID := strings.TrimSpace(gocast.ToString(params["agent_session_id"]))
	sessionKey := strings.TrimSpace(gocast.ToString(params["session_key"]))
	createTime := agentParamNow(params, "create_time")
	modifyTime := agentParamText(params, "modify_time", createTime)
	lastActiveTime := agentParamText(params, "last_active_time", modifyTime)

	if sessionID != "" {
		if session, ok, err := queryAgentSession(map[string]interface{}{"agent_session_id": sessionID}); err != nil {
			return nil, err
		} else if ok {
			return updateAgentSession(session, params)
		}
	}
	if sessionKey != "" {
		if session, ok, err := queryAgentSession(map[string]interface{}{"session_key": sessionKey}); err != nil {
			return nil, err
		} else if ok {
			return updateAgentSession(session, params)
		}
	}

	session := base.AgentSession{
		AgentSessionID: "agent_session_" + uuid.NewString(),
		SessionKey:     sessionKey,
		SceneCode:      normalizeScene(gocast.ToString(params["scene_code"])),
		Title:          resolveAgentSessionTitle("", gocast.ToString(params["title"]), gocast.ToString(params["input_text"])),
		Status:         agentStatusActive,
		UserID:         gocast.ToString(params["session_user_id"]),
		SystemPrompt:   gocast.ToString(params["system_prompt"]),
		Model:          normalizeModel(gocast.ToString(params["model"])),
		ToolPolicyJSON: gocast.ToString(params["tool_policy_json"]),
		McpPolicyJSON:  gocast.ToString(params["mcp_policy_json"]),
		LastActiveTime: lastActiveTime,
		CreateTime:     createTime,
		ModifyTime:     modifyTime,
		CreateUser:     gocast.ToString(params["session_user_id"]),
		ModifyUser:     gocast.ToString(params["session_user_id"]),
		IsDelete:       "0",
	}
	if session.SessionKey == "" {
		session.SessionKey = makeSessionKey()
	}
	if status := strings.TrimSpace(gocast.ToString(params["status"])); status != "" {
		session.Status = status
	}
	if _, err := callAgentLowcodeService("agent.session_save", map[string]interface{}{
		"agent_session_id": session.AgentSessionID,
		"session_key":      session.SessionKey,
		"scene_code":       session.SceneCode,
		"title":            session.Title,
		"status":           session.Status,
		"user_id":          session.UserID,
		"system_prompt":    session.SystemPrompt,
		"model":            session.Model,
		"tool_policy_json": session.ToolPolicyJSON,
		"mcp_policy_json":  session.McpPolicyJSON,
		"context_summary":  session.ContextSummary,
		"last_response_id": session.LastResponseID,
		"last_active_time": session.LastActiveTime,
		"expire_time":      session.ExpireTime,
		"create_time":      session.CreateTime,
		"modify_time":      session.ModifyTime,
		"create_user":      session.CreateUser,
		"modify_user":      session.ModifyUser,
		"is_delete":        session.IsDelete,
	}); err != nil {
		return nil, err
	}
	return &session, nil
}

func updateAgentSession(session *base.AgentSession, params map[string]interface{}) (*base.AgentSession, error) {
	if value := strings.TrimSpace(gocast.ToString(params["scene_code"])); value != "" {
		session.SceneCode = value
	}
	if value := resolveAgentSessionTitle(session.Title, gocast.ToString(params["title"]), gocast.ToString(params["input_text"])); value != "" {
		session.Title = value
	}
	if value := strings.TrimSpace(gocast.ToString(params["status"])); value != "" {
		session.Status = value
	}
	if value := strings.TrimSpace(gocast.ToString(params["system_prompt"])); value != "" {
		session.SystemPrompt = value
	}
	if value := strings.TrimSpace(gocast.ToString(params["model"])); value != "" {
		session.Model = value
	}
	if value := strings.TrimSpace(gocast.ToString(params["tool_policy_json"])); value != "" {
		session.ToolPolicyJSON = value
	}
	if value := strings.TrimSpace(gocast.ToString(params["mcp_policy_json"])); value != "" {
		session.McpPolicyJSON = value
	}
	modifyTime := agentParamNow(params, "modify_time")
	session.LastActiveTime = agentParamText(params, "last_active_time", modifyTime)
	session.ModifyTime = modifyTime
	session.ModifyUser = gocast.ToString(params["session_user_id"])
	if _, err := callAgentLowcodeService("agent.session_update", map[string]interface{}{
		"agent_session_id": session.AgentSessionID,
		"scene_code":       session.SceneCode,
		"title":            session.Title,
		"status":           session.Status,
		"system_prompt":    session.SystemPrompt,
		"model":            session.Model,
		"tool_policy_json": session.ToolPolicyJSON,
		"mcp_policy_json":  session.McpPolicyJSON,
		"last_active_time": session.LastActiveTime,
		"modify_time":      session.ModifyTime,
		"modify_user":      session.ModifyUser,
	}); err != nil {
		return nil, err
	}
	return session, nil
}

func nextMessageSeq(agentSessionID string) int64 {
	result, err := callAgentLowcodeService("agent.message_max_seq_query", map[string]interface{}{
		"agent_session_id": agentSessionID,
	})
	if err != nil {
		return 1
	}
	type seqResult struct {
		SeqNo int64 `json:"seq_no"`
	}
	var seq seqResult
	if err := decodeAgentLowcodeData(result.Data, &seq); err != nil {
		return 1
	}
	return seq.SeqNo + 1
}

func appendAgentMessage(agentSessionID string, agentRunID string, role string, messageType string, contentText string, contentJSON string, createUser string, createTime ...string) (*base.AgentMessage, error) {
	messageCreateTime := nowText()
	if len(createTime) > 0 {
		if value := strings.TrimSpace(createTime[0]); value != "" {
			messageCreateTime = value
		}
	}
	message := base.AgentMessage{
		AgentMessageID: "agent_message_" + uuid.NewString(),
		AgentSessionID: agentSessionID,
		AgentRunID:     agentRunID,
		Role:           role,
		MessageType:    messageType,
		ContentText:    contentText,
		ContentJSON:    contentJSON,
		SeqNo:          nextMessageSeq(agentSessionID),
		Source:         "agent_runtime",
		Status:         "completed",
		CreateTime:     messageCreateTime,
		CreateUser:     createUser,
		IsDelete:       "0",
	}
	if _, err := callAgentLowcodeService("agent.message_save", map[string]interface{}{
		"agent_message_id": message.AgentMessageID,
		"agent_session_id": message.AgentSessionID,
		"agent_run_id":     message.AgentRunID,
		"role":             message.Role,
		"message_type":     message.MessageType,
		"content_text":     message.ContentText,
		"content_json":     message.ContentJSON,
		"seq_no":           message.SeqNo,
		"source":           message.Source,
		"token_count":      message.TokenCount,
		"status":           message.Status,
		"create_time":      message.CreateTime,
		"create_user":      message.CreateUser,
		"is_delete":        message.IsDelete,
	}); err != nil {
		return nil, err
	}
	return &message, nil
}

func ensureStreamingAssistantMessage(agentSessionID string, agentRunID string, createUser string) (*base.AgentMessage, error) {
	if message, ok, err := queryLatestAssistantMessage(agentRunID); err != nil {
		return nil, err
	} else if ok {
		if message.Status == "" || message.Status == "completed" {
			message.Status = runStatusRunning
			message.CreateUser = createUser
			_, _ = callAgentLowcodeService("agent.message_stream_start_update", map[string]interface{}{
				"agent_message_id": message.AgentMessageID,
				"status":           message.Status,
				"create_user":      message.CreateUser,
			})
		}
		return message, nil
	}
	message := base.AgentMessage{
		AgentMessageID: "agent_message_" + uuid.NewString(),
		AgentSessionID: agentSessionID,
		AgentRunID:     agentRunID,
		Role:           "assistant",
		MessageType:    "assistant",
		ContentText:    "",
		SeqNo:          nextMessageSeq(agentSessionID),
		Source:         "agent_runtime",
		Status:         runStatusRunning,
		CreateTime:     nowText(),
		CreateUser:     createUser,
		IsDelete:       "0",
	}
	if _, err := callAgentLowcodeService("agent.message_save", map[string]interface{}{
		"agent_message_id": message.AgentMessageID,
		"agent_session_id": message.AgentSessionID,
		"agent_run_id":     message.AgentRunID,
		"role":             message.Role,
		"message_type":     message.MessageType,
		"content_text":     message.ContentText,
		"content_json":     message.ContentJSON,
		"seq_no":           message.SeqNo,
		"source":           message.Source,
		"token_count":      message.TokenCount,
		"status":           message.Status,
		"create_time":      message.CreateTime,
		"create_user":      message.CreateUser,
		"is_delete":        message.IsDelete,
	}); err != nil {
		return nil, err
	}
	return &message, nil
}

func updateAgentMessageContent(messageID string, contentText string, contentJSON string, status string) error {
	params := map[string]interface{}{
		"agent_message_id": messageID,
		"content_text":     contentText,
		"status":           status,
	}
	service := "agent.message_content_update"
	if contentJSON != "" {
		params["content_json"] = contentJSON
		service = "agent.message_content_json_update"
	}
	_, err := callAgentLowcodeService(service, params)
	return err
}

func decodeAgentRunRequest(requestJSON string) map[string]interface{} {
	requestData := map[string]interface{}{}
	if requestJSON != "" {
		_ = json.Unmarshal([]byte(requestJSON), &requestData)
	}
	return requestData
}

func isAgentStreamResponse(requestData map[string]interface{}) bool {
	return gocast.ToBool(requestData["stream_response"]) || gocast.ToBool(requestData["stream"])
}

func callAgentProvider(ctx context.Context, session *base.AgentSession, inputText string, requestJSON string, onDelta agentProviderStreamFunc, onTool agentToolEventFunc, onLog agentProviderLogFunc) (*agentProviderResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	credential := resolveAgentCredential()
	requestData := decodeAgentRunRequest(requestJSON)
	toolPolicy := resolveAgentToolPolicy(session, requestData)
	toolPolicy = adaptAgentToolPolicyForPrompt(toolPolicy, inputText)
	if mockResponse := strings.TrimSpace(gocast.ToString(requestData["mock_response"])); mockResponse != "" {
		if onDelta != nil {
			onDelta(mockResponse)
		}
		return &agentProviderResult{
			ResponseID: "mock_" + uuid.NewString(),
			OutputText: mockResponse,
			RawJSON: map[string]interface{}{
				"mode": "request_mock",
			},
			Mocked: true,
		}, nil
	}

	callWithCredential := func(current agentCredential) (*agentProviderResult, error) {
		if current.Token == "" {
			outputText := "Mock assistant: " + inputText
			if onDelta != nil {
				onDelta(outputText)
			}
			return &agentProviderResult{
				ResponseID: "mock_" + uuid.NewString(),
				OutputText: outputText,
				RawJSON: map[string]interface{}{
					"mode": "env_mock",
				},
				Mocked: true,
			}, nil
		}
		if toolPolicy.Enabled && len(toolPolicy.toolDefinitions()) > 0 {
			return callAgentProviderWithTools(ctx, session, inputText, current, toolPolicy, onDelta, onTool, onLog)
		}
		if current.Mode == "chatgpt_access_token" {
			return callChatGPTCodexProvider(ctx, session, inputText, current, onDelta, onLog)
		}
		return callPlatformResponsesProvider(ctx, session, inputText, current, onDelta, onLog)
	}

	return callWithCredential(credential)
}

func executeAgentRun(agentRunID string) error {
	return executeAgentRunWithStream(context.Background(), agentRunID, nil, nil, nil)
}

func executeAgentRunWithStream(parentCtx context.Context, agentRunID string, clientDelta agentProviderStreamFunc, clientTool agentToolEventFunc, clientLog agentProviderLogFunc) error {
	ensureAgentRuntime()
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	run, claimed, err := claimAgentRun(agentRunID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	runCtx, cancelRunCtx := context.WithCancel(parentCtx)
	registerAgentRunCancel(run.AgentRunID, cancelRunCtx)
	defer func() {
		unregisterAgentRunCancel(run.AgentRunID)
		cancelRunCtx()
	}()
	if err := runCtx.Err(); err != nil {
		return markAgentRunCancelled(run, nil, err)
	}
	stopHeartbeat := startAgentRunHeartbeat(runCtx, run.AgentRunID)
	defer stopHeartbeat()

	session, ok, err := queryAgentSession(map[string]interface{}{"agent_session_id": run.AgentSessionID})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("agent_session 不存在: %s", run.AgentSessionID)
	}
	userMessageList, err := queryAgentMessageList(map[string]interface{}{
		"agent_run_id": run.AgentRunID,
		"role":         "user",
		"pagination":   false,
	})
	if err != nil {
		return err
	}
	if len(userMessageList) == 0 {
		return fmt.Errorf("agent_run 用户消息不存在: %s", run.AgentRunID)
	}
	userMessage := userMessageList[len(userMessageList)-1]

	requestData := decodeAgentRunRequest(run.RequestJSON)
	if delaySecond := gocast.ToInt64(requestData["simulate_delay_second"]); delaySecond > 0 {
		select {
		case <-runCtx.Done():
			return markAgentRunCancelled(run, nil, runCtx.Err())
		case <-time.After(time.Duration(delaySecond) * time.Second):
		}
	}

	var assistantMessage *base.AgentMessage
	var streamBuilder strings.Builder
	lastFlush := time.Now()
	lastFlushedLen := 0
	flushStream := func(force bool) {
		if assistantMessage == nil {
			return
		}
		currentLen := streamBuilder.Len()
		if !force && time.Since(lastFlush) < 250*time.Millisecond && currentLen-lastFlushedLen < 64 {
			return
		}
		lastFlush = time.Now()
		lastFlushedLen = currentLen
		_ = updateAgentMessageContent(assistantMessage.AgentMessageID, streamBuilder.String(), "", runStatusRunning)
	}
	var onDelta agentProviderStreamFunc
	if isAgentStreamResponse(requestData) || clientDelta != nil {
		var streamErr error
		assistantMessage, streamErr = ensureStreamingAssistantMessage(session.AgentSessionID, run.AgentRunID, session.UserID)
		if streamErr != nil {
			run.Status = runStatusFailed
			run.CurrentStep = "failed"
			run.ErrorMsg = streamErr.Error()
			run.FinishedAt = nowText()
			run.ModifyTime = run.FinishedAt
			_, _ = callAgentLowcodeService("agent.run_fail_update", map[string]interface{}{
				"agent_run_id": run.AgentRunID,
				"error_msg":    run.ErrorMsg,
			})
			return streamErr
		}
		onDelta = func(delta string) {
			streamBuilder.WriteString(delta)
			flushStream(false)
			if clientDelta != nil {
				clientDelta(delta)
			}
		}
	}

	providerResult, err := callAgentProvider(runCtx, session, userMessage.ContentText, run.RequestJSON, onDelta, clientTool, clientLog)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
			return markAgentRunCancelled(run, assistantMessage, err)
		}
		contentJSON := ""
		var toolRunErr *agentToolRunError
		if errors.As(err, &toolRunErr) && len(toolRunErr.ToolResults) > 0 {
			resultBytes, _ := json.Marshal(agentProviderResult{
				OutputText:  "请求失败：" + err.Error(),
				ToolResults: toolRunErr.ToolResults,
				RawJSON: map[string]interface{}{
					"error": err.Error(),
				},
			})
			contentJSON = string(resultBytes)
			run.ResultJSON = contentJSON
		}
		run.Status = runStatusFailed
		run.CurrentStep = "failed"
		run.ErrorMsg = err.Error()
		run.FinishedAt = nowText()
		run.ModifyTime = run.FinishedAt
		failService := "agent.run_fail_update"
		failParams := map[string]interface{}{
			"agent_run_id": run.AgentRunID,
			"error_msg":    run.ErrorMsg,
		}
		if contentJSON != "" {
			failService = "agent.run_fail_result_update"
			failParams["result_json"] = contentJSON
		}
		_, _ = callAgentLowcodeService(failService, failParams)
		if assistantMessage != nil {
			_ = updateAgentMessageContent(assistantMessage.AgentMessageID, "请求失败："+err.Error(), contentJSON, runStatusFailed)
		}
		return err
	}

	providerResult.ToolResults = mergeAgentProviderDebugToolResults(providerResult.DebugRequests, providerResult.ToolResults)
	resultBytes, _ := json.Marshal(providerResult)
	if assistantMessage != nil {
		streamBuilder.Reset()
		streamBuilder.WriteString(providerResult.OutputText)
		if err := updateAgentMessageContent(assistantMessage.AgentMessageID, providerResult.OutputText, string(resultBytes), "completed"); err != nil {
			run.Status = runStatusFailed
			run.CurrentStep = "failed"
			run.ErrorMsg = err.Error()
			run.FinishedAt = nowText()
			run.ModifyTime = run.FinishedAt
			_, _ = callAgentLowcodeService("agent.run_fail_update", map[string]interface{}{
				"agent_run_id": run.AgentRunID,
				"error_msg":    run.ErrorMsg,
			})
			return err
		}
	} else if _, err := appendAgentMessage(session.AgentSessionID, run.AgentRunID, "assistant", "assistant", providerResult.OutputText, string(resultBytes), session.UserID); err != nil {
		run.Status = runStatusFailed
		run.CurrentStep = "failed"
		run.ErrorMsg = err.Error()
		run.FinishedAt = nowText()
		run.ModifyTime = run.FinishedAt
		_, _ = callAgentLowcodeService("agent.run_fail_update", map[string]interface{}{
			"agent_run_id": run.AgentRunID,
			"error_msg":    run.ErrorMsg,
		})
		return err
	}

	session.LastResponseID = providerResult.ResponseID
	session.LastActiveTime = nowText()
	session.ModifyTime = session.LastActiveTime
	if _, err := callAgentLowcodeService("agent.session_last_response_update", map[string]interface{}{
		"agent_session_id": session.AgentSessionID,
		"last_response_id": session.LastResponseID,
		"last_active_time": session.LastActiveTime,
		"modify_time":      session.ModifyTime,
	}); err != nil {
		return err
	}

	run.Status = runStatusCompleted
	run.CurrentStep = "completed"
	run.ErrorMsg = ""
	run.ResultJSON = string(resultBytes)
	run.FinishedAt = nowText()
	run.HeartbeatTime = run.FinishedAt
	run.LeaseExpireTime = run.FinishedAt
	run.ModifyTime = run.FinishedAt
	_, err = callAgentLowcodeService("agent.run_complete_update", map[string]interface{}{
		"agent_run_id": run.AgentRunID,
		"error_msg":    run.ErrorMsg,
		"result_json":  run.ResultJSON,
	})
	return err
}
