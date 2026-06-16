package plugins

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"io"
	"moon/model/devops"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HttpProxyService struct {
	templateService.BaseHandler
}

var (
	proxyCookieJarLock sync.RWMutex
	proxyCookieJarMap  = map[string]http.CookieJar{}
	proxyLogTableOnce  sync.Once
	proxyLogTableErr   error
)

const proxyCookieStoreDir = "database/http_proxy_cookies"

type proxyCookieStoreFile struct {
	Scope     string                      `json:"scope"`
	UpdatedAt string                      `json:"updated_at"`
	Items     []proxyCookieStoreHostEntry `json:"items"`
}

type proxyCookieStoreHostEntry struct {
	URL     string                   `json:"url"`
	Cookies []proxyCookiePersistItem `json:"cookies"`
}

type proxyCookiePersistItem struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	Path       string `json:"path,omitempty"`
	Domain     string `json:"domain,omitempty"`
	Expires    string `json:"expires,omitempty"`
	MaxAge     int    `json:"max_age,omitempty"`
	Secure     bool   `json:"secure"`
	HttpOnly   bool   `json:"http_only"`
	SameSite   string `json:"same_site,omitempty"`
	RawExpires string `json:"raw_expires,omitempty"`
}

func (s *HttpProxyService) Result(template *config.Template, ts *templateService.TemplateService) *common.Result {
	startAt := time.Now()
	params := template.GetParams()
	method := strings.ToUpper(stringValue(params["request_method"]))
	if method == "" {
		method = http.MethodPost
	}
	requestURL := stringValue(params["request_url"])
	if requestURL == "" {
		return common.NotOk("request_url 不能为空")
	}
	projectCode := stringValue(params["project_code"])
	cookieScopeInput := stringValue(params["cookie_scope"])
	clearCookie := boolValue(params["clear_cookie"])
	cookieScope, domainPrefix, hostName, err := getProxyCookieScope(projectCode, requestURL, cookieScopeInput)
	if err != nil {
		return common.NotOk(err.Error())
	}

	headers, err := normalizeProxyHeaders(params["request_header"])
	if err != nil {
		return common.NotOk(err.Error())
	}

	data := params["request_data"]
	if data == nil {
		data = map[string]interface{}{}
	}
	requestHeaderText := marshalLogJSON(headers)
	requestDataText := marshalLogJSON(data)

	reqURL := requestURL
	var bodyReader io.Reader
	if method == http.MethodGet {
		nextURL, err := appendQueryParams(requestURL, data)
		if err != nil {
			return common.NotOk(err.Error())
		}
		reqURL = nextURL
	} else {
		if !hasHeaderIgnoreCase(headers, "Content-Type") {
			headers["Content-Type"] = "application/json"
		}
		bodyBytes, err := json.Marshal(data)
		if err != nil {
			return common.NotOk("request_data 不是合法 JSON 对象")
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}
	logEntry := &devops.HttpProxyRequestLog{
		HttpProxyRequestLogID: "http_proxy_log_" + uuid.NewString(),
		CreateTime:            time.Now().Format(time.RFC3339Nano),
		ProjectCode:           projectCode,
		CookieScope:           cookieScope,
		RequestMethod:         method,
		RequestURL:            reqURL,
		RequestHeaderText:     truncateLogText(requestHeaderText, 200000),
		RequestDataText:       truncateLogText(requestDataText, 500000),
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		logEntry.ErrorText = truncateLogText(err.Error(), 4000)
		logEntry.DurationMs = time.Since(startAt).Milliseconds()
		logID, logErr := s.saveProxyRequestLog(logEntry)
		if logErr != nil {
			return common.NotOk(fmt.Sprintf("%s; 请求日志写入失败: %s", err.Error(), logErr.Error()))
		}
		return common.NotOk(fmt.Sprintf("%s (log_id=%s)", err.Error(), logID))
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	jar, err := getProxyCookieJar(cookieScope, clearCookie)
	if err != nil {
		return common.NotOk(err.Error())
	}
	sentCookies := jar.Cookies(req.URL)
	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}
	resp, err := client.Do(req)
	if err != nil {
		logEntry.ErrorText = truncateLogText(err.Error(), 4000)
		logEntry.DurationMs = time.Since(startAt).Milliseconds()
		logID, logErr := s.saveProxyRequestLog(logEntry)
		if logErr != nil {
			return common.NotOk(fmt.Sprintf("%s; 请求日志写入失败: %s", err.Error(), logErr.Error()))
		}
		return common.NotOk(fmt.Sprintf("%s (log_id=%s)", err.Error(), logID))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logEntry.ResponseStatusCode = resp.StatusCode
		logEntry.ResponseStatusText = resp.Status
		logEntry.ErrorText = truncateLogText(err.Error(), 4000)
		logEntry.DurationMs = time.Since(startAt).Milliseconds()
		logID, logErr := s.saveProxyRequestLog(logEntry)
		if logErr != nil {
			return common.NotOk(fmt.Sprintf("%s; 请求日志写入失败: %s", err.Error(), logErr.Error()))
		}
		return common.NotOk(fmt.Sprintf("%s (log_id=%s)", err.Error(), logID))
	}
	setCookies := resp.Cookies()
	storedCookies := jar.Cookies(req.URL)
	persistError := persistProxyCookies(cookieScope, req.URL, setCookies, storedCookies)
	rawText := string(respBody)
	text := rawText
	var responseJSON interface{}
	if json.Valid(respBody) {
		if err := json.Unmarshal(respBody, &responseJSON); err == nil {
			if pretty, err := json.MarshalIndent(responseJSON, "", "  "); err == nil {
				text = string(pretty)
			}
		}
	}

	result := map[string]interface{}{
		"status_code":   resp.StatusCode,
		"status_text":   resp.Status,
		"response_text": text,
		"response_json": responseJSON,
		"request": map[string]interface{}{
			"method":  method,
			"url":     reqURL,
			"headers": headers,
			"data":    data,
		},
		"cookie": map[string]interface{}{
			"scope":              cookieScope,
			"project_code":       projectCode,
			"domain_prefix":      domainPrefix,
			"host":               hostName,
			"sent_count":         len(sentCookies),
			"set_cookie_count":   len(setCookies),
			"stored_count":       len(storedCookies),
			"sent_cookie_header": cookiesToHeader(sentCookies),
			"sent_cookies":       cookiesToObjects(sentCookies),
			"set_cookies":        cookiesToObjects(setCookies),
			"stored_cookies":     cookiesToObjects(storedCookies),
		},
	}
	if persistError != nil {
		result["cookie_persist_error"] = persistError.Error()
	}
	logEntry.ResponseStatusCode = resp.StatusCode
	logEntry.ResponseStatusText = resp.Status
	logEntry.ResponseText = truncateLogText(rawText, 500000)
	logEntry.DurationMs = time.Since(startAt).Milliseconds()
	logID, logErr := s.saveProxyRequestLog(logEntry)
	if logID != "" {
		result["proxy_log_id"] = logID
	}
	if logErr != nil {
		result["proxy_log_error"] = logErr.Error()
	}
	return common.Ok(result, "请求发送成功")
}

func (s *HttpProxyService) saveProxyRequestLog(entry *devops.HttpProxyRequestLog) (string, error) {
	if entry == nil {
		return "", nil
	}
	gormDB := s.GetGormDb()
	if gormDB == nil {
		return "", fmt.Errorf("数据库未初始化")
	}
	if err := ensureHttpProxyRequestLogTable(gormDB); err != nil {
		return entry.HttpProxyRequestLogID, err
	}
	if strings.TrimSpace(entry.HttpProxyRequestLogID) == "" {
		entry.HttpProxyRequestLogID = "http_proxy_log_" + uuid.NewString()
	}
	if strings.TrimSpace(entry.CreateTime) == "" {
		entry.CreateTime = time.Now().Format(time.RFC3339Nano)
	}
	return entry.HttpProxyRequestLogID, gormDB.Create(entry).Error
}

func ensureHttpProxyRequestLogTable(gormDB *gorm.DB) error {
	proxyLogTableOnce.Do(func() {
		proxyLogTableErr = gormDB.AutoMigrate(&devops.HttpProxyRequestLog{})
	})
	return proxyLogTableErr
}

func marshalLogJSON(raw interface{}) string {
	if raw == nil {
		return ""
	}
	body, err := json.Marshal(raw)
	if err == nil {
		return string(body)
	}
	return fmt.Sprintf("%v", raw)
}

func truncateLogText(text string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen]
}

func stringValue(raw interface{}) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", raw))
}

func hasHeaderIgnoreCase(headers map[string]string, target string) bool {
	for key := range headers {
		if strings.EqualFold(strings.TrimSpace(key), target) {
			return true
		}
	}
	return false
}

func boolValue(raw interface{}) bool {
	if raw == nil {
		return false
	}
	switch value := raw.(type) {
	case bool:
		return value
	case int:
		return value != 0
	case int32:
		return value != 0
	case int64:
		return value != 0
	case float64:
		return value != 0
	case string:
		text := strings.TrimSpace(strings.ToLower(value))
		return text == "1" || text == "true" || text == "yes" || text == "y"
	default:
		text := strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", raw)))
		return text == "1" || text == "true" || text == "yes" || text == "y"
	}
}

func getProxyCookieScope(projectCode string, rawURL string, customScope string) (scope string, domainPrefix string, hostName string, err error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", err
	}
	hostName = normalizeHostName(parsedURL.Hostname())
	domainPrefix = normalizeDomainPrefix(hostName)
	scopePart := strings.TrimSpace(customScope)
	if scopePart == "" {
		scopePart = domainPrefix
	}
	projectPart := strings.TrimSpace(projectCode)
	if projectPart == "" {
		projectPart = "_global"
	}
	scope = strings.ToLower(projectPart + "::" + scopePart)
	return scope, domainPrefix, hostName, nil
}

func normalizeHostName(host string) string {
	text := strings.TrimSpace(strings.ToLower(host))
	text = strings.Trim(text, "[]")
	if text == "" {
		return "_unknown_host"
	}
	if strings.Contains(text, ":") {
		if hostOnly, _, err := net.SplitHostPort(text); err == nil && strings.TrimSpace(hostOnly) != "" {
			return strings.TrimSpace(strings.ToLower(hostOnly))
		}
	}
	return text
}

func normalizeDomainPrefix(host string) string {
	if host == "" || host == "_unknown_host" {
		return "_unknown_prefix"
	}
	if net.ParseIP(host) != nil {
		return host
	}
	if host == "localhost" {
		return "localhost"
	}
	parts := strings.Split(host, ".")
	if len(parts) == 0 {
		return host
	}
	return strings.TrimSpace(strings.ToLower(parts[0]))
}

func getProxyCookieJar(scope string, clearCookie bool) (http.CookieJar, error) {
	scopeKey := normalizeScopeKey(scope)
	if clearCookie {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		proxyCookieJarLock.Lock()
		proxyCookieJarMap[scopeKey] = jar
		proxyCookieJarLock.Unlock()
		_ = clearProxyCookiesOnDisk(scopeKey)
		return jar, nil
	}
	proxyCookieJarLock.RLock()
	jar := proxyCookieJarMap[scopeKey]
	proxyCookieJarLock.RUnlock()
	if jar != nil {
		return jar, nil
	}
	newJar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	proxyCookieJarLock.Lock()
	defer proxyCookieJarLock.Unlock()
	if existing := proxyCookieJarMap[scopeKey]; existing != nil {
		return existing, nil
	}
	if err := loadProxyCookiesFromDisk(scopeKey, newJar); err != nil {
		return nil, err
	}
	proxyCookieJarMap[scopeKey] = newJar
	return newJar, nil
}

func normalizeScopeKey(scope string) string {
	scopeKey := strings.TrimSpace(scope)
	if scopeKey == "" {
		scopeKey = "_global::_unknown_prefix"
	}
	return strings.ToLower(scopeKey)
}

func persistProxyCookies(scope string, reqURL *url.URL, setCookies []*http.Cookie, storedCookies []*http.Cookie) error {
	if reqURL == nil {
		return nil
	}
	hostURL := normalizeCookieHostURL(reqURL)
	if hostURL == nil {
		return nil
	}
	scopeKey := normalizeScopeKey(scope)
	storePath, err := getProxyCookieStorePath(scopeKey)
	if err != nil {
		return err
	}
	proxyCookieJarLock.Lock()
	defer proxyCookieJarLock.Unlock()
	storeFile, err := readProxyCookieStoreFile(storePath, scopeKey)
	if err != nil {
		return err
	}
	mergedCookies := mergeProxyCookiesForPersist(hostURL, setCookies, storedCookies)
	storeFile = upsertProxyCookieStoreEntry(storeFile, hostURL.String(), mergedCookies)
	storeFile.Scope = scopeKey
	storeFile.UpdatedAt = time.Now().Format(time.RFC3339)
	return writeProxyCookieStoreFile(storePath, storeFile)
}

func loadProxyCookiesFromDisk(scopeKey string, jar http.CookieJar) error {
	storePath, err := getProxyCookieStorePath(scopeKey)
	if err != nil {
		return err
	}
	storeFile, err := readProxyCookieStoreFile(storePath, scopeKey)
	if err != nil {
		return err
	}
	for _, item := range storeFile.Items {
		u, err := url.Parse(strings.TrimSpace(item.URL))
		if err != nil || u == nil {
			continue
		}
		cookies := restoreCookiesFromPersist(item.Cookies)
		if len(cookies) == 0 {
			continue
		}
		jar.SetCookies(u, cookies)
	}
	return nil
}

func clearProxyCookiesOnDisk(scopeKey string) error {
	storePath, err := getProxyCookieStorePath(scopeKey)
	if err != nil {
		return err
	}
	if err := os.Remove(storePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func getProxyCookieStorePath(scopeKey string) (string, error) {
	if err := os.MkdirAll(proxyCookieStoreDir, 0o755); err != nil {
		return "", err
	}
	hash := sha1.Sum([]byte(scopeKey))
	fileName := hex.EncodeToString(hash[:]) + ".json"
	return filepath.Join(proxyCookieStoreDir, fileName), nil
}

func readProxyCookieStoreFile(storePath string, scopeKey string) (proxyCookieStoreFile, error) {
	content, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return proxyCookieStoreFile{
				Scope: scopeKey,
				Items: []proxyCookieStoreHostEntry{},
			}, nil
		}
		return proxyCookieStoreFile{}, err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return proxyCookieStoreFile{
			Scope: scopeKey,
			Items: []proxyCookieStoreHostEntry{},
		}, nil
	}
	var storeFile proxyCookieStoreFile
	if err := json.Unmarshal(content, &storeFile); err != nil {
		return proxyCookieStoreFile{}, err
	}
	if strings.TrimSpace(storeFile.Scope) == "" {
		storeFile.Scope = scopeKey
	}
	if storeFile.Items == nil {
		storeFile.Items = []proxyCookieStoreHostEntry{}
	}
	return storeFile, nil
}

func writeProxyCookieStoreFile(storePath string, storeFile proxyCookieStoreFile) error {
	if storeFile.Items == nil {
		storeFile.Items = []proxyCookieStoreHostEntry{}
	}
	body, err := json.MarshalIndent(storeFile, "", "  ")
	if err != nil {
		return err
	}
	tmpFile := storePath + ".tmp"
	if err := os.WriteFile(tmpFile, body, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpFile, storePath); err != nil {
		_ = os.Remove(tmpFile)
		return err
	}
	return nil
}

func normalizeCookieHostURL(reqURL *url.URL) *url.URL {
	if reqURL == nil {
		return nil
	}
	scheme := strings.TrimSpace(strings.ToLower(reqURL.Scheme))
	if scheme != "http" && scheme != "https" {
		scheme = "http"
	}
	host := strings.TrimSpace(reqURL.Hostname())
	if host == "" {
		return nil
	}
	return &url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   "/",
	}
}

func mergeProxyCookiesForPersist(hostURL *url.URL, setCookies []*http.Cookie, storedCookies []*http.Cookie) []proxyCookiePersistItem {
	merged := map[string]proxyCookiePersistItem{}
	add := func(c *http.Cookie, preferMeta bool) {
		if c == nil {
			return
		}
		name := strings.TrimSpace(c.Name)
		if name == "" {
			return
		}
		domain := strings.TrimSpace(c.Domain)
		path := strings.TrimSpace(c.Path)
		if path == "" {
			path = "/"
		}
		key := strings.ToLower(name + "|" + domain + "|" + path)
		item := proxyCookiePersistItem{
			Name:       name,
			Value:      c.Value,
			Path:       path,
			Domain:     domain,
			RawExpires: strings.TrimSpace(c.RawExpires),
			MaxAge:     c.MaxAge,
			Secure:     c.Secure,
			HttpOnly:   c.HttpOnly,
			SameSite:   encodeSameSite(c.SameSite),
		}
		if !c.Expires.IsZero() {
			item.Expires = c.Expires.Format(time.RFC3339)
		}
		if prev, ok := merged[key]; ok {
			if preferMeta {
				if item.Domain == "" {
					item.Domain = prev.Domain
				}
				if item.Path == "" {
					item.Path = prev.Path
				}
				if item.Expires == "" {
					item.Expires = prev.Expires
				}
				if item.RawExpires == "" {
					item.RawExpires = prev.RawExpires
				}
				if item.MaxAge == 0 {
					item.MaxAge = prev.MaxAge
				}
			} else {
				item.Domain = prev.Domain
				item.Path = prev.Path
				item.Expires = prev.Expires
				item.RawExpires = prev.RawExpires
				item.MaxAge = prev.MaxAge
				item.Secure = prev.Secure
				item.HttpOnly = prev.HttpOnly
				item.SameSite = prev.SameSite
			}
		}
		if item.Domain == "" && hostURL != nil {
			item.Domain = strings.TrimSpace(hostURL.Hostname())
		}
		merged[key] = item
	}
	for _, c := range setCookies {
		add(c, true)
	}
	for _, c := range storedCookies {
		add(c, false)
	}
	result := make([]proxyCookiePersistItem, 0, len(merged))
	for _, item := range merged {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func upsertProxyCookieStoreEntry(storeFile proxyCookieStoreFile, urlKey string, cookies []proxyCookiePersistItem) proxyCookieStoreFile {
	next := make([]proxyCookieStoreHostEntry, 0, len(storeFile.Items)+1)
	replaced := false
	for _, item := range storeFile.Items {
		if strings.TrimSpace(item.URL) == strings.TrimSpace(urlKey) {
			if len(cookies) > 0 {
				item.Cookies = cookies
				next = append(next, item)
			}
			replaced = true
			continue
		}
		next = append(next, item)
	}
	if !replaced && len(cookies) > 0 {
		next = append(next, proxyCookieStoreHostEntry{
			URL:     urlKey,
			Cookies: cookies,
		})
	}
	storeFile.Items = next
	return storeFile
}

func restoreCookiesFromPersist(items []proxyCookiePersistItem) []*http.Cookie {
	result := make([]*http.Cookie, 0, len(items))
	now := time.Now()
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		c := &http.Cookie{
			Name:       name,
			Value:      item.Value,
			Path:       defaultCookiePath(item.Path),
			Domain:     strings.TrimSpace(item.Domain),
			MaxAge:     item.MaxAge,
			Secure:     item.Secure,
			HttpOnly:   item.HttpOnly,
			SameSite:   decodeSameSite(item.SameSite),
			RawExpires: strings.TrimSpace(item.RawExpires),
		}
		if strings.TrimSpace(item.Expires) != "" {
			if expireAt, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Expires)); err == nil {
				if expireAt.Before(now) {
					continue
				}
				c.Expires = expireAt
			}
		}
		result = append(result, c)
	}
	return result
}

func defaultCookiePath(path string) string {
	text := strings.TrimSpace(path)
	if text == "" {
		return "/"
	}
	return text
}

func encodeSameSite(mode http.SameSite) string {
	switch mode {
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return "Default"
	}
}

func decodeSameSite(raw string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}

func cookiesToHeader(cookies []*http.Cookie) string {
	if len(cookies) == 0 {
		return ""
	}
	parts := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		parts = append(parts, cookie.Name+"="+cookie.Value)
	}
	return strings.Join(parts, "; ")
}

func cookiesToObjects(cookies []*http.Cookie) []map[string]interface{} {
	if len(cookies) == 0 {
		return []map[string]interface{}{}
	}
	result := make([]map[string]interface{}, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		item := map[string]interface{}{
			"name":     cookie.Name,
			"value":    cookie.Value,
			"domain":   cookie.Domain,
			"path":     cookie.Path,
			"secure":   cookie.Secure,
			"httpOnly": cookie.HttpOnly,
		}
		if !cookie.Expires.IsZero() {
			item["expires"] = cookie.Expires.Format(time.RFC3339)
			item["expires_unix"] = strconv.FormatInt(cookie.Expires.Unix(), 10)
		}
		if cookie.MaxAge != 0 {
			item["max_age"] = cookie.MaxAge
		}
		switch cookie.SameSite {
		case http.SameSiteStrictMode:
			item["same_site"] = "Strict"
		case http.SameSiteLaxMode:
			item["same_site"] = "Lax"
		case http.SameSiteNoneMode:
			item["same_site"] = "None"
		default:
			item["same_site"] = "Default"
		}
		result = append(result, item)
	}
	return result
}

func normalizeProxyHeaders(raw interface{}) (map[string]string, error) {
	if raw == nil {
		return map[string]string{}, nil
	}
	switch value := raw.(type) {
	case string:
		text := strings.TrimSpace(value)
		if text == "" {
			return map[string]string{}, nil
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(text), &parsed); err != nil {
			return nil, fmt.Errorf("request_header 解析失败: %s", err.Error())
		}
		headers := make(map[string]string, len(parsed))
		for key, item := range parsed {
			headers[key] = fmt.Sprintf("%v", item)
		}
		return headers, nil
	case map[string]interface{}:
		headers := make(map[string]string, len(value))
		for key, item := range value {
			headers[key] = fmt.Sprintf("%v", item)
		}
		return headers, nil
	case map[string]string:
		headers := make(map[string]string, len(value))
		for key, item := range value {
			headers[key] = item
		}
		return headers, nil
	default:
		return nil, fmt.Errorf("request_header 类型不支持")
	}
}

func appendQueryParams(rawURL string, data interface{}) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	query := parsedURL.Query()
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return rawURL, nil
	}
	for key, value := range dataMap {
		query.Set(key, fmt.Sprintf("%v", value))
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}
