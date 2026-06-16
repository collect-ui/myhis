package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	collect "github.com/collect-ui/collect/src/collect/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/sijms/go-ora/v2"
	"moon/model"
	"moon/plugins"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// HmacWithShaTobase64 生成 HMAC-SHA256 签名并编码为 Base64
func HmacWithShaTobase64(message string, sign string, secret string) string {
	// 创建一个新的 HMAC 实例，使用 SHA256 算法
	h := hmac.New(sha256.New, []byte(secret))

	// 将消息和签名拼接在一起
	combinedMessage := message + sign

	// 将拼接后的消息写入 HMAC 实例
	h.Write([]byte(combinedMessage))

	// 计算 HMAC 签名
	signature := h.Sum(nil)

	// 将签名编码为 Base64
	base64Signature := base64.StdEncoding.EncodeToString(signature)

	return base64Signature
}

// @hosturl :  like  wss://tts-api.xfyun.cn/v2/tts
// @apikey : apiKey
// @apiSecret : apiSecret
func assembleAuthUrl(hosturl string, apiKey, apiSecret string) string {
	ul, err := url.Parse(hosturl)
	if err != nil {
		fmt.Println(err)
	}
	//签名时间
	date := time.Now().UTC().Format(time.RFC1123)
	//参与签名的字段 host ,date, request-line
	signString := []string{"host: " + ul.Host, "date: " + date, "GET " + ul.Path + " HTTP/1.1"}
	//拼接签名字符串
	sgin := strings.Join(signString, "\n")
	//签名结果
	sha := HmacWithShaTobase64("hmac-sha256", sgin, apiSecret)
	//构建请求参数 此时不需要urlencoding
	authUrl := fmt.Sprintf("api_key=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", apiKey,
		"hmac-sha256", "host date request-line", sha)
	//将请求参数使用base64编码
	authorization := base64.StdEncoding.EncodeToString([]byte(authUrl))
	v := url.Values{}
	v.Add("host", ul.Host)
	v.Add("date", date)
	v.Add("authorization", authorization)
	//将编码后的字符串url encode后添加到url后面
	callurl := hosturl + "?" + v.Encode()
	return callurl
}
func main2() {
	fmt.Println(assembleAuthUrl("wss://tts-api.xfyun.cn/v2/tts", "128a773f8ae545652713e4c0e3b3dba9", "MDU5ODExZDJjNzI4ODRkOGU2ZDgyM2Ji"))
}
func getContentType(filePath string) string {
	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))

	// 根据扩展名返回对应的 MIME 类型
	switch ext {
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js", ".mjs":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".txt":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".gz":
		return "application/gzip"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	default:
		// 默认返回二进制流类型
		return "application/octet-stream"
	}
}
func serveStatic(urlPrefix, root string, cache bool) gin.HandlerFunc {
	fs := http.Dir(root)
	//fileServer := http.FileServer(fs)

	return func(c *gin.Context) {

		p := c.Request.URL.Path

		if !strings.HasPrefix(p, urlPrefix) {
			c.Next()
			return
		}

		// 去掉 URL 前缀
		p = strings.TrimPrefix(p, urlPrefix)
		servedPath := p

		// 检查客户端是否支持 gzip
		if strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			gzFilePath := p + ".gz"
			if f, err := fs.Open(gzFilePath); err == nil {
				// 返回 .gz 文件
				setStaticCacheHeaders(c, cache, servedPath)
				c.Header("Content-Encoding", "gzip")
				c.Header("Content-Type", getContentType(p))
				info, _ := f.Stat()
				http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), f)
				return
			}
		}

		// 尝试查找文件
		f, err := fs.Open(p)
		if err != nil {
			// 如果文件不存在，尝试查找 index.html
			servedPath = "index.html"
			f, err = fs.Open("index.html")
			if err != nil {
				c.Next()
				return
			}
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil || info.IsDir() {
			// 如果文件是目录或无法获取文件信息，尝试查找 index.html
			servedPath = "index.html"
			f, err = fs.Open("index.html")
			if err != nil {
				c.Next()
				return
			}
			defer f.Close()
			info, _ = f.Stat()
		}

		// 提供文件服务
		setStaticCacheHeaders(c, cache, servedPath)
		http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), f)
		c.Abort()
	}
}

func setStaticCacheHeaders(c *gin.Context, cache bool, servedPath string) {
	// SPA 入口 HTML 和 fallback 的 index.html 不做强缓存，避免浏览器长期引用旧 bundle。
	shouldCache := cache && !strings.HasSuffix(strings.ToLower(strings.TrimSpace(servedPath)), ".html")
	if shouldCache {
		c.Header("Cache-Control", "public, max-age=5184000") // 60天 = 60 * 24 * 60 * 60 秒
		expires := time.Now().Add(60 * 24 * time.Hour)
		c.Header("Expires", expires.Format(http.TimeFormat))
		return
	}

	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache") // 兼容 HTTP/1.0
	c.Header("Expires", "0")       // 立即过期
}

func main4() {
	text := `项目总投资为
462.8 万元
fdfdsfdsfdsf元招标金额为850万元，`

	// 定义正则表达式（启用单行模式）
	//reg := `(?s)质量(?:标准|要求)[：:]\s*((?:.|\n)+?[；;。])`
	reg := "(?s)项目总投资为(\\s*([\\d.,]+)\\s*万?元)"

	// 编译正则表达式
	nameRegex := regexp.MustCompile(reg)

	// 匹配并提取质量要求
	match := nameRegex.FindStringSubmatch(text)
	if len(match) > 1 {
		requirement := match[1]  // 提取捕获组中的质量要求
		fmt.Println(requirement) // 输出: 必须符合现行国家有关工程施工质量验收规范和标准的要求，达到合格及以上标准；
	} else {
		fmt.Println("未找到质量要求")
	}
}
func isTrustedOrigin(origin string) bool {
	if origin == "" || strings.HasPrefix(origin, "http://localhost") {
		return true
	}
	// 添加你的域名白名单
	allowedDomains := []string{"https://yourdomain.com", "https://www.yourdomain.com"}
	for _, domain := range allowedDomains {
		if origin == domain {
			return true
		}
	}
	return false
}

func DynamicSessionOptions() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 动态安全配置
		isSecure := c.Request.TLS != nil || // 自动检测HTTPS
			c.GetHeader("X-Forwarded-Proto") == "https" // 支持代理场景

		// 2. SameSite策略 (所有HTTPS域名用None，否则Lax)
		sameSite := http.SameSiteNoneMode
		if !isSecure || isLocalhostOrIP(c.Request.Host) {
			sameSite = http.SameSiteLaxMode
		}

		// 3. 动态设置Domain (仅对非IP和非localhost的域名生效)
		domain := ""
		if !isLocalhostOrIP(c.Request.Host) {
			domain = extractRootDomain(c.Request.Host)
		}

		// 4. 应用配置
		session := sessions.Default(c)
		session.Options(sessions.Options{
			Path:     "/",
			Domain:   domain,
			MaxAge:   86400 * 30,
			Secure:   isSecure,
			HttpOnly: true,
			SameSite: sameSite,
		})

		// 5. 显式保存配置
		if err := session.Save(); err != nil {
			fmt.Printf("Session save error: %v", err)
		}

		c.Next()
	}
}

// 辅助函数：判断是否本地或IP访问
func isLocalhostOrIP(host string) bool {
	host = strings.Split(host, ":")[0] // 去除端口
	return host == "localhost" || net.ParseIP(host) != nil
}

// 辅助函数：提取根域名
func extractRootDomain(host string) string {
	parts := strings.Split(host, ":")
	domainParts := strings.Split(parts[0], ".")
	if len(domainParts) >= 2 {
		return "." + strings.Join(domainParts[len(domainParts)-2:], ".") // 如 ".iqiaoqi.com"
	}
	return parts[0]
}
func main() {
	// todo go profile 使用
	//gin.SetMode(gin.ReleaseMode)
	//go func() {
	//	log.Println(http.ListenAndServe("localhost:6060", nil))
	//}()
	r := gin.Default()
	gin.SetMode(gin.DebugMode)
	r.Use(gin.Logger())
	// 全局设置跨域头
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Range")
		c.Header("Access-Control-Expose-Headers", "Accept-Ranges, Content-Length, Content-Range")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})
	// 生成cookies
	store := cookie.NewStore([]byte("secret"))

	r.Use(sessions.Sessions("session_id", store))
	r.Use(DynamicSessionOptions()) // 添加动态选项中间件
	r.Static("/static", "./static")

	dirStr := collect.GetAppKey("dirList")
	dirList := strings.Split(dirStr, ";")
	for _, file := range dirList {
		if collect.IsValueEmpty(file) {
			continue
		}
		fileInfo := strings.Split(file, ",")
		cache := false
		if fileInfo[2] == "true" {
			cache = true
		}
		r.Use(serveStatic(fileInfo[0], fileInfo[1], cache))
	}

	fileStr := collect.GetAppKey("fileList")
	fileList := strings.Split(fileStr, ";")
	for _, file := range fileList {
		if collect.IsValueEmpty(file) {
			continue
		}
		fileInfo := strings.Split(file, ",")
		r.StaticFile(fileInfo[0], fileInfo[1])
	}
	// 设置数据库
	templateService.SetDatabaseModel(&model.TableData{})
	// 设置外部处理器
	templateService.SetOuterModuleRegister(plugins.GetRegisterList())
	// 添加定时任务
	templateService.RunScheduleService()
	// 添加启动服务
	templateService.RunStartupService()
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/collect-ui")
	})
	r.POST("/template_data/data", func(c *gin.Context) {
		templateService.HandlerRequest(c)
	})
	plugins.RegisterAgentStreamRoutes(r)
	plugins.RegisterWorkspaceFilePreviewRoutes(r)

	r.GET("/template_data/ws/:token", func(context *gin.Context) {
		templateService.HandlerWsRequest(context)
	})

	serverPort := collect.GetAppKey("server_port")
	isHttps := collect.GetAppKey("is_https")
	domains := strings.Split(collect.GetAppKey("domain"), ",")

	if isHttps == "true" {
		tlsConfig := &tls.Config{}
		for _, domain := range domains {
			certFile := "/etc/letsencrypt/live/" + domain + "/fullchain.pem"
			keyFile := "/etc/letsencrypt/live/" + domain + "/privkey.pem"
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				panic(err)
			}
			tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		}
		server := &http.Server{
			Addr:      ":" + serverPort,
			Handler:   r,
			TLSConfig: tlsConfig,
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			panic(err)
		}
	} else {
		r.Run(":" + serverPort) // listen and serve on 0.0.0.0:8080
	}

}
