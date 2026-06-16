package plugins

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"io"
	"log"
	"math"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Sftp struct {
	templateService.BaseHandler
}

type SftpFile struct {
	templateService.TsFile
	File *sftp.File
}

func (s *SftpFile) ReadAt(b []byte, off int64) (int, error) {
	return s.File.ReadAt(b, off)
}
func (s *SftpFile) Size() int64 {
	stat, _ := s.File.Stat()
	return stat.Size()

}

func getTargetPath(field string, params map[string]interface{}) (string, error) {
	tpl, error := config.CastTemplate(field)
	if error != nil {
		return "", error
	}
	targetPath := utils.RenderTplData(tpl, params).(string)
	if targetPath == "" || targetPath == "~" {
		targetPath = "./"
	}
	return targetPath, error
}
func (si *Sftp) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	conn := ts.GetThirdData(SshName).(*SSHConnection)
	//field := handlerParam.Field
	params := template.GetParams()
	result := make(map[string]interface{})
	sftpClient, err := sftp.NewClient(conn.Client)
	if err != nil {
		return common.NotOk(err.Error())
	}
	defer sftpClient.Close()
	fields := handlerParam.Fields
	// 依次编译模板
	for _, subField := range fields {
		saveField := subField.SaveField
		if subField.Type == "pwd" {
			cwd, err := sftpClient.Getwd()
			if err != nil {
				return common.NotOk(err.Error())
			}
			targetPath, err := getTargetPath(subField.Field, params)
			if targetPath == "." || targetPath == "./" || targetPath == "" || targetPath == "~" {
				result[saveField] = cwd
			} else {
				result[saveField] = targetPath
			}
		} else if subField.Type == "dir" {
			targetPath, err := getTargetPath(subField.Field, params)
			if err != nil {
				return common.NotOk(err.Error())
			}
			fList, err := sftpClient.ReadDir(targetPath)
			if err != nil {
				return common.NotOk(err.Error())
			}
			infoList := make([]map[string]interface{}, 0)
			for _, file := range fList {
				item := make(map[string]interface{})
				item["mode"] = file.Mode()
				item["name"] = file.Name()
				item["is_dir"] = file.IsDir()
				item["modify_time"] = utils.DateFormat(file.ModTime(), "")
				item["size"] = file.Size()
				item["size_info"] = getSize(file.Size())
				item["sys"] = file.Sys()
				infoList = append(infoList, item)
			}
			result[saveField] = infoList
		} else if subField.Type == "dir_tree" {
			targetPath, err := getTargetPath(subField.Field, params)
			if err != nil {
				return common.NotOk(err.Error())
			}
			excludeSet := parseExcludeDirSet(params["exclude_dirs"])
			infoList, err := getDirTree(sftpClient, targetPath, excludeSet)
			if err != nil {
				return common.NotOk(err.Error())
			}
			result[saveField] = infoList
		} else if subField.Type == "dir_tree_shell" {
			targetPath, err := getTargetPath(subField.Field, params)
			if err != nil {
				return common.NotOk(err.Error())
			}
			excludeSet := parseExcludeDirSet(params["exclude_dirs"])
			start := time.Now()
			out, err := handlerCmd(buildFindTreeCmd(targetPath, excludeSet), conn.Client, params)
			if err != nil {
				return common.NotOk(err.Error())
			}
			dirInfo, err := parseFindOutputToTree(targetPath, out)
			if err != nil {
				return common.NotOk(err.Error())
			}
			dirInfo["elapsed_ms"] = time.Since(start).Milliseconds()
			result[saveField] = dirInfo["dir"]
			result["dir_list"] = dirInfo["dir_list"]
			result["current_dir"] = dirInfo["current_dir"]
			result["elapsed_ms"] = dirInfo["elapsed_ms"]
		} else if subField.Type == "remove" {
			targetPath, err := getTargetPath(subField.Field, params)
			if err != nil {
				return common.NotOk(err.Error())
			}
			err = sftpClient.Remove(targetPath)
			if err != nil {
				return common.NotOk(err.Error())
			}
		} else if subField.Type == "upload" {
			file := ts.File

			if file == nil {
				return common.NotOk("上传文件不能为空")
			}
			targetDir, err := getTargetPath(subField.Field, params)
			if err != nil {
				return common.NotOk(err.Error())
			}

			sftpClient.MkdirAll(targetDir)
			targetPath := targetDir + "/" + ts.FileHeader.Filename
			remoteFile, err := sftpClient.Create(targetPath)
			defer remoteFile.Close()
			var offset int64 = 0
			var bufsize int64 = 1024 * 1024
			buf := make([]byte, bufsize)
			for {
				n, err := file.ReadAt(buf, offset)
				if err != nil && err != io.EOF {
					log.Panicln("read file error", err)
					break
				}
				if n == 0 {
					break
				}
				_, err = remoteFile.Write(buf[:n])
				if err != nil {
					log.Panicln("write file error", err)
					break
				}
				offset += int64(n)
			}

		} else if subField.Type == "download" {
			targetPath, err := getTargetPath(subField.Field, params)
			if err != nil {
				return common.NotOk(err.Error())
			}
			srcFile, err := sftpClient.Open(targetPath)
			if err != nil {
				common.NotOk(err.Error())
			}
			var f templateService.TsFile
			f = &SftpFile{File: srcFile}
			ts.IsFileResponse = true
			ts.SetTsFile(f)
			ts.ResponseFileName = utils.FileName(srcFile.Name())
			tsFile := ts.GetTsFile()
			size := tsFile.Size()
			c := ts.GetContext()
			c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", ts.ResponseFileName)) //fmt.Sprintf("attachment; filename=%s", filename)对下载的文件重命名
			c.Writer.Header().Add("Content-Type", "application/octet-stream")
			c.Writer.Header().Add("Content-Length", strconv.FormatInt(size, 10))
			var offset int64 = 0
			var bufsize int64 = 1024 * 1024
			buf := make([]byte, bufsize)
			for {
				n, err := tsFile.ReadAt(buf, offset)
				if err != nil && err != io.EOF {
					log.Panicln("read file error", err)
					break
				}
				if n == 0 {
					break
				}
				_, err = c.Writer.Write(buf[:n])
				if err != nil {
					log.Panicln("write file error", err)
					break
				}
				offset += int64(n)
			}
			c.Writer.Flush()
		} else if subField.Type == "read_text" {
			targetPath, err := getTargetPath(subField.Field, params)
			if err != nil {
				return common.NotOk(err.Error())
			}
			srcFile, err := sftpClient.Open(targetPath)
			if err != nil {
				return common.NotOk(err.Error())
			}
			defer srcFile.Close()

			fileName := path.Base(targetPath)
			ext := strings.TrimPrefix(strings.ToLower(path.Ext(fileName)), ".")
			mimeType := detectMimeByExt(ext)
			if isPdfExtension(ext) || strings.Contains(mimeType, "application/pdf") {
				size := int64(0)
				modifyTime := ""
				createTime := ""
				if stat, statErr := srcFile.Stat(); statErr == nil && stat != nil {
					size = stat.Size()
					modifyTime = utils.DateFormat(stat.ModTime(), "")
					createTime = ""
				}
				if mimeType == "" {
					mimeType = "application/pdf"
				}
				result[saveField] = map[string]interface{}{
					"name":           fileName,
					"path":           targetPath,
					"ext":            ext,
					"size":           size,
					"mime":           mimeType,
					"kind":           "pdf",
					"modify_time":    modifyTime,
					"create_time":    createTime,
					"truncated":      false,
					"content_text":   "",
					"content_base64": "",
					"preview_mode":   "stream",
				}
				continue
			}

			maxBytes := gocast.ToInt64(params["max_bytes"])
			if maxBytes <= 0 {
				maxBytes = 2 * 1024 * 1024
			}

			limitReader := io.LimitReader(srcFile, maxBytes+1)
			contentBytes, err := io.ReadAll(limitReader)
			if err != nil {
				return common.NotOk(err.Error())
			}

			truncated := false
			if int64(len(contentBytes)) > maxBytes {
				truncated = true
				contentBytes = contentBytes[:maxBytes]
			}

			if mimeType == "" {
				mimeType = "application/octet-stream"
				if len(contentBytes) > 0 {
					mimeType = http.DetectContentType(contentBytes)
				}
			}
			isImage := isImageExtension(ext) || strings.HasPrefix(mimeType, "image/")
			isDocx := isDocxExtension(ext) || strings.Contains(mimeType, "wordprocessingml.document")
			isPDF := isPdfExtension(ext) || strings.Contains(mimeType, "application/pdf")
			isText := !isImage && !isDocx && !isPDF && isLikelyTextContent(contentBytes)
			kind := "binary"
			if isImage {
				kind = "image"
			} else if isDocx {
				kind = "docx"
			} else if isPDF {
				kind = "pdf"
			} else if isText {
				kind = "text"
			}

			size := int64(len(contentBytes))
			modifyTime := ""
			createTime := ""
			if stat, statErr := srcFile.Stat(); statErr == nil && stat != nil {
				size = stat.Size()
				modifyTime = utils.DateFormat(stat.ModTime(), "")
				// 跨平台 SFTP 不保证可获取创建时间，默认返回空字符串。
				createTime = ""
			}

			contentText := ""
			contentBase64 := ""
			if kind == "text" {
				contentText = string(contentBytes)
			} else if kind == "image" || kind == "docx" {
				contentBase64 = base64.StdEncoding.EncodeToString(contentBytes)
			}
			if kind == "pdf" {
				truncated = false
			}

			result[saveField] = map[string]interface{}{
				"name":           fileName,
				"path":           targetPath,
				"ext":            ext,
				"size":           size,
				"mime":           mimeType,
				"kind":           kind,
				"modify_time":    modifyTime,
				"create_time":    createTime,
				"truncated":      truncated,
				"content_text":   contentText,
				"content_base64": contentBase64,
			}
			if kind == "pdf" {
				result[saveField].(map[string]interface{})["preview_mode"] = "stream"
			}
		} else if subField.Type == "read_text_batch" {
			pathList := parsePathList(params["paths"])
			if len(pathList) == 0 {
				renderField, _ := getTargetPath(subField.Field, params)
				pathList = parsePathList(renderField)
			}
			if len(pathList) == 0 {
				return common.NotOk("paths 不能为空")
			}

			maxBytes := gocast.ToInt64(params["max_bytes"])
			if maxBytes <= 0 {
				maxBytes = 2 * 1024 * 1024
			}

			items := make([]map[string]interface{}, 0, len(pathList))
			for _, p := range pathList {
				item := map[string]interface{}{"path": p, "ok": false, "content": "", "size": 0, "truncated": false}
				srcFile, openErr := sftpClient.Open(p)
				if openErr != nil {
					item["error"] = openErr.Error()
					items = append(items, item)
					continue
				}
				limitReader := io.LimitReader(srcFile, maxBytes+1)
				contentBytes, readErr := io.ReadAll(limitReader)
				srcFile.Close()
				if readErr != nil {
					item["error"] = readErr.Error()
					items = append(items, item)
					continue
				}
				truncated := false
				if int64(len(contentBytes)) > maxBytes {
					truncated = true
					contentBytes = contentBytes[:maxBytes]
				}
				item["ok"] = true
				item["content"] = string(contentBytes)
				item["size"] = len(contentBytes)
				item["truncated"] = truncated
				items = append(items, item)
			}
			result[saveField] = map[string]interface{}{"items": items, "count": len(items)}
		} else if subField.Type == "write_text" {
			targetPath, err := getTargetPath(subField.Field, params)
			if err != nil {
				return common.NotOk(err.Error())
			}
			content := gocast.ToString(params["content"])
			maxWriteBytes := gocast.ToInt64(params["max_write_bytes"])
			if maxWriteBytes <= 0 {
				maxWriteBytes = 5 * 1024 * 1024
			}
			if int64(len(content)) > maxWriteBytes {
				return common.NotOk(fmt.Sprintf("内容超过限制，最大 %d 字节", maxWriteBytes))
			}

			targetDir := path.Dir(targetPath)
			if targetDir != "" && targetDir != "." {
				if err := sftpClient.MkdirAll(targetDir); err != nil {
					return common.NotOk(err.Error())
				}
			}

			dstFile, err := sftpClient.Create(targetPath)
			if err != nil {
				return common.NotOk(err.Error())
			}
			defer dstFile.Close()

			if _, err := dstFile.Write([]byte(content)); err != nil {
				return common.NotOk(err.Error())
			}

			result[saveField] = map[string]interface{}{
				"path":    targetPath,
				"size":    len(content),
				"success": true,
			}

		}

	}

	r := common.Ok(result, "处理参数成功")
	return r
}

func isImageExtension(ext string) bool {
	if ext == "" {
		return false
	}
	switch ext {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp", "svg", "ico", "avif":
		return true
	default:
		return false
	}
}

func isDocxExtension(ext string) bool {
	return ext == "docx"
}

func isPdfExtension(ext string) bool {
	return ext == "pdf"
}

func detectMimeByExt(ext string) string {
	switch ext {
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "pdf":
		return "application/pdf"
	case "svg":
		return "image/svg+xml"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "bmp":
		return "image/bmp"
	case "ico":
		return "image/x-icon"
	case "avif":
		return "image/avif"
	default:
		return ""
	}
}

func isLikelyTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	nonText := 0
	for _, b := range data {
		if b == 0 {
			return false
		}
		if b == 9 || b == 10 || b == 13 {
			continue
		}
		if b >= 32 && b <= 126 {
			continue
		}
		if b >= 128 {
			continue
		}
		nonText++
	}
	return float64(nonText)/float64(len(data)) < 0.1
}

func parsePathList(raw interface{}) []string {
	pathList := make([]string, 0)
	appendPath := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" {
			pathList = append(pathList, v)
		}
	}
	switch v := raw.(type) {
	case []string:
		for _, item := range v {
			appendPath(item)
		}
	case []interface{}:
		for _, item := range v {
			appendPath(gocast.ToString(item))
		}
	case string:
		vv := strings.TrimSpace(v)
		if vv == "" {
			return pathList
		}
		if strings.HasPrefix(vv, "[") && strings.HasSuffix(vv, "]") {
			var arr []interface{}
			if err := json.Unmarshal([]byte(vv), &arr); err == nil {
				for _, item := range arr {
					appendPath(gocast.ToString(item))
				}
				return pathList
			}
		}
		for _, item := range strings.Split(vv, ",") {
			appendPath(item)
		}
	default:
		appendPath(gocast.ToString(v))
	}
	return pathList
}

func parseExcludeDirSet(raw interface{}) map[string]struct{} {
	result := map[string]struct{}{}
	if raw == nil {
		return result
	}
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name != "" {
			result[name] = struct{}{}
		}
	}
	switch v := raw.(type) {
	case string:
		for _, item := range strings.Split(v, ",") {
			add(item)
		}
	case []string:
		for _, item := range v {
			add(item)
		}
	case []interface{}:
		for _, item := range v {
			add(gocast.ToString(item))
		}
	default:
		add(gocast.ToString(v))
	}
	return result
}

func getDirTree(sftpClient *sftp.Client, targetPath string, excludeSet map[string]struct{}) ([]map[string]interface{}, error) {
	fList, err := sftpClient.ReadDir(targetPath)
	if err != nil {
		return nil, err
	}
	infoList := make([]map[string]interface{}, 0, len(fList))
	for _, file := range fList {
		item := make(map[string]interface{})
		item["mode"] = file.Mode()
		item["name"] = file.Name()
		item["is_dir"] = file.IsDir()
		item["modify_time"] = utils.DateFormat(file.ModTime(), "")
		item["size"] = file.Size()
		item["size_info"] = getSize(file.Size())
		item["sys"] = file.Sys()
		item["path"] = path.Join(targetPath, file.Name())

		if file.IsDir() {
			if _, excluded := excludeSet[file.Name()]; excluded {
				item["children"] = []map[string]interface{}{}
				item["excluded"] = true
			} else {
				children, childErr := getDirTree(sftpClient, path.Join(targetPath, file.Name()), excludeSet)
				if childErr != nil {
					item["children_error"] = childErr.Error()
				} else {
					item["children"] = children
				}
			}
		}
		infoList = append(infoList, item)
	}
	sort.Slice(infoList, func(i, j int) bool {
		iDir := gocast.ToBool(infoList[i]["is_dir"])
		jDir := gocast.ToBool(infoList[j]["is_dir"])
		if iDir != jDir {
			return iDir
		}
		return gocast.ToString(infoList[i]["name"]) < gocast.ToString(infoList[j]["name"])
	})
	return infoList, nil
}

func buildFindTreeCmd(rootPath string, excludes map[string]struct{}) string {
	root := shellQuote(rootPath)
	if len(excludes) == 0 {
		return fmt.Sprintf("(find %s -printf '__REC__%%y|%%s|%%T@|%%p\\n' || true)", root)
	}
	names := make([]string, 0, len(excludes))
	for name := range excludes {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("-name %s", shellQuote(name)))
	}
	pruneExpr := strings.Join(parts, " -o ")
	return fmt.Sprintf("(find %s \\( %s \\) -prune -o -printf '__REC__%%y|%%s|%%T@|%%p\\n' || true)", root, pruneExpr)
}

func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "'\\''") + "'"
}

func parseFindOutputToTree(rootPath string, output string) (map[string]interface{}, error) {
	records := parseFindRecords(output)
	if len(records) == 0 {
		lower := strings.ToLower(output)
		if strings.Contains(lower, "no such file") || strings.Contains(lower, "not found") || strings.Contains(lower, "permission denied") {
			return nil, fmt.Errorf(strings.TrimSpace(output))
		}
	}
	expectedRoot := path.Clean(rootPath)
	nodeMap := map[string]map[string]interface{}{}
	resolvedRoot := ""

	for _, line := range records {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			parts = strings.SplitN(line, "|", 4)
		}
		typeCode := ""
		sizeVal := int64(0)
		mtimeRaw := ""
		fullPath := ""
		if len(parts) >= 4 {
			typeCode = parts[0]
			sizeVal, _ = strconv.ParseInt(parts[1], 10, 64)
			mtimeRaw = parts[2]
			fullPath = parts[3]
		} else {
			ok := false
			typeCode, sizeVal, mtimeRaw, fullPath, ok = parseCompactFindRecord(line, rootPath)
			if !ok {
				continue
			}
		}
		cleanPath := path.Clean(fullPath)

		isDir := typeCode == "d"
		mtime := ""
		if seconds, err := strconv.ParseFloat(mtimeRaw, 64); err == nil {
			mtime = utils.DateFormat(time.Unix(int64(seconds), 0), "")
		}

		node := map[string]interface{}{
			"name":        path.Base(cleanPath),
			"is_dir":      isDir,
			"path":        cleanPath,
			"size":        sizeVal,
			"size_info":   getSize(sizeVal),
			"modify_time": mtime,
		}
		if isDir {
			node["children"] = []map[string]interface{}{}
		}

		nodeMap[cleanPath] = node
		if cleanPath == expectedRoot {
			resolvedRoot = expectedRoot
		}
		if resolvedRoot == "" {
			if gocast.ToBool(node["is_dir"]) {
				if resolvedRoot == "" || len(cleanPath) < len(resolvedRoot) {
					resolvedRoot = cleanPath
				}
			} else if resolvedRoot == "" {
				resolvedRoot = path.Dir(cleanPath)
			}
		}
	}

	if resolvedRoot == "" {
		resolvedRoot = expectedRoot
	}
	rootNode, ok := nodeMap[resolvedRoot]
	if !ok {
		rootNode = map[string]interface{}{
			"name":        path.Base(resolvedRoot),
			"is_dir":      true,
			"path":        resolvedRoot,
			"children":    []map[string]interface{}{},
			"size":        int64(0),
			"size_info":   getSize(0),
			"modify_time": "",
		}
		if resolvedRoot == "." || resolvedRoot == "/" {
			rootNode["name"] = resolvedRoot
		}
		nodeMap[resolvedRoot] = rootNode
	}

	paths := make([]string, 0, len(nodeMap))
	for p := range nodeMap {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		if p == resolvedRoot {
			continue
		}
		n := nodeMap[p]
		parentPath := path.Dir(p)
		parent, ok := nodeMap[parentPath]
		if !ok {
			continue
		}
		children, _ := parent["children"].([]map[string]interface{})
		children = append(children, n)
		parent["children"] = children
	}

	for _, p := range paths {
		n := nodeMap[p]
		if gocast.ToBool(n["is_dir"]) {
			children, _ := n["children"].([]map[string]interface{})
			sort.Slice(children, func(i, j int) bool {
				iDir := gocast.ToBool(children[i]["is_dir"])
				jDir := gocast.ToBool(children[j]["is_dir"])
				if iDir != jDir {
					return iDir
				}
				return gocast.ToString(children[i]["name"]) < gocast.ToString(children[j]["name"])
			})
			n["children"] = children
		}
	}

	result := map[string]interface{}{
		"current_dir": resolvedRoot,
		"dir":         rootNode["children"],
	}
	assignIDsAndBuildList(rootNode, result)
	return result, nil
}

func assignIDsAndBuildList(rootNode map[string]interface{}, result map[string]interface{}) {
	list := make([]map[string]interface{}, 0)
	rootNode["id"] = ""
	rootNode["parent_id"] = ""
	list = append(list, nodeToFlatRow(rootNode))

	children, _ := rootNode["children"].([]map[string]interface{})
	for _, child := range children {
		walkAssignNode(child, "", &list)
	}
	ensureFirstLevelID(children)
	result["dir_list"] = list
}

func ensureFirstLevelID(children []map[string]interface{}) {
	for _, child := range children {
		if strings.TrimSpace(gocast.ToString(child["id"])) == "" {
			child["id"] = stableNodeID(gocast.ToString(child["path"]))
		}
		if strings.TrimSpace(gocast.ToString(child["id"])) == "" {
			child["id"] = uuid.New().String()
		}
		child["parent_id"] = ""
	}
}

func walkAssignNode(node map[string]interface{}, parentID string, list *[]map[string]interface{}) {
	id := stableNodeID(gocast.ToString(node["path"]))
	if id == "" {
		id = uuid.New().String()
	}
	node["id"] = id
	node["parent_id"] = parentID
	*list = append(*list, nodeToFlatRow(node))

	children, _ := node["children"].([]map[string]interface{})
	for _, child := range children {
		walkAssignNode(child, id, list)
	}
}

func stableNodeID(pathValue string) string {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return ""
	}
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(pathValue)).String()
}

func nodeToFlatRow(node map[string]interface{}) map[string]interface{} {
	row := map[string]interface{}{
		"id":          gocast.ToString(node["id"]),
		"parent_id":   gocast.ToString(node["parent_id"]),
		"name":        node["name"],
		"is_dir":      node["is_dir"],
		"path":        node["path"],
		"size":        node["size"],
		"size_info":   node["size_info"],
		"modify_time": node["modify_time"],
	}
	if mode, ok := node["mode"]; ok {
		row["mode"] = mode
	}
	if sys, ok := node["sys"]; ok {
		row["sys"] = sys
	}
	if excluded, ok := node["excluded"]; ok {
		row["excluded"] = excluded
	}
	if childErr, ok := node["children_error"]; ok {
		row["children_error"] = childErr
	}
	return row
}

func parseFindRecords(output string) []string {
	if strings.Contains(output, "__REC__") {
		parts := strings.Split(output, "__REC__")
		records := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				records = append(records, p)
			}
		}
		return records
	}
	return strings.Split(output, "\n")
}

func parseCompactFindRecord(line string, rootPath string) (string, int64, string, string, bool) {
	if len(line) < 2 {
		return "", 0, "", "", false
	}
	typeCode := line[:1]
	root := path.Clean(rootPath)
	idx := strings.Index(line, root)
	if idx < 1 {
		idx = strings.Index(line, "/")
		if idx < 1 {
			return "", 0, "", "", false
		}
	}
	meta := line[1:idx]
	fullPath := line[idx:]
	dot := strings.LastIndex(meta, ".")
	if dot <= 0 {
		return "", 0, "", "", false
	}
	left := meta[:dot]
	frac := meta[dot+1:]
	if len(left) < 10 {
		return "", 0, "", "", false
	}
	sec := left[len(left)-10:]
	sizeText := left[:len(left)-10]
	if sizeText == "" {
		sizeText = "0"
	}
	sizeVal, err := strconv.ParseInt(sizeText, 10, 64)
	if err != nil {
		return "", 0, "", "", false
	}
	mtimeRaw := sec + "." + frac
	return typeCode, sizeVal, mtimeRaw, fullPath, true
}

func getSize(size int64) string {

	a := gocast.ToFloat(size)
	c := gocast.ToFloat(1024)
	p := math.Pow10(2)
	if a < c {
		return gocast.ToString(math.Round(a*p)/p) + " B"
	} else if a < (c * c) {
		return gocast.ToString(math.Round(a/c*p)/p) + " KB"
	} else if a < (c * c * c) {
		return gocast.ToString(math.Round(a/(c*c)*p)/p) + " MB"
	} else if a < (c * c * c * c) {
		return gocast.ToString(math.Round(a/(c*c*c)*p)/p) + " GB"
	} else if a < (c * c * c * c * c) {
		return gocast.ToString(math.Round(a/(c*c*c*c)*p)/p) + " TB"
	} else {
		return gocast.ToString(math.Round(a/(c*c*c*c*c)*p)/p) + " PB"
	}
}
