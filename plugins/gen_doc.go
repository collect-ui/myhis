package plugins

import (
	"bytes"
	"fmt"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/fumiama/go-docx"
	"os"
	"strings"
)

type GenDoc struct {
	templateService.BaseHandler
}

// 在第一个 ### 标题前插入序号
func addSequenceToHeader(markdown string, seq string) string {
	lines := strings.Split(markdown, "\n")

	for i, line := range lines {
		if strings.HasPrefix(line, "### ") {
			// 插入序号并保留原内容
			lines[i] = fmt.Sprintf("### %s%s", seq, line[4:])
			break // 只处理第一个匹配的行
		}
		if strings.HasPrefix(line, "## ") {
			// 插入序号并保留原内容
			lines[i] = fmt.Sprintf("## %s%s", seq, line[3:])
			break // 只处理第一个匹配的行
		}
	}

	return strings.Join(lines, "\n")
}
func (si *GenDoc) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	path := utils.RenderVar(handlerParam.Path, params).(string)
	arr, _ := utils.RenderVarToArrMap(handlerParam.Foreach, params)
	w := docx.New().WithDefaultTheme()
	title := w.AddParagraph()
	title.Justification("center")
	title.AddText(handlerParam.Name).Size("26").Bold()

	// 遍历 Markdown 字符串列表
	for _, mdContent := range arr {
		// 解析 Markdown 内容并添加到 DOCX 文档
		content := mdContent["doc_content"].(string)
		seq := mdContent["seq"].(string)
		content = addSequenceToHeader(content, seq)
		if !utils.IsValueEmpty(content) {
			si.addMarkdownToDocx(w, content)
		}

	}

	file := utils.RenderVar(handlerParam.File, params)
	// 拼接模板
	if !utils.IsValueEmpty(file) {
		// 换一页
		para := w.AddParagraph()
		para.AddPageBreaks()
		readFile, err := os.Open(file.(string))
		if err == nil {
			fileinfo, err := readFile.Stat()
			if err != nil {
				return common.NotOk(err.Error())
			}
			size := fileinfo.Size()
			doc, err := docx.Parse(readFile, size)
			w.AppendFile(doc)
		}

	}

	// 如果目录不存在则创建目录
	dir := utils.ParentDirName(path)
	if !utils.IsPathExist(dir) {
		if err := utils.CreateDirs(dir); err != nil {
			return common.NotOk(err.Error())
		}
	}

	// 创建并保存 DOCX 文件
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// 将文档写入文件
	_, err = w.WriteTo(f)
	if err != nil {
		panic(err)
	}
	// 返回处理结果
	r := common.Ok(nil, "处理参数成功")
	return r
}

func getTitlePrefix(line string) string {
	if !strings.Contains(line, "#") {
		return ""
	}
	return strings.Split(line, " ")[0]
}

func countLeadingSpaces(s string) string {
	var buffer bytes.Buffer
	for _, ch := range s {
		if ch == ' ' {
			buffer.WriteString("\u2009")
		} else {
			break // 遇到非空格字符就停止
		}
	}
	return buffer.String()
}

// 解析 Markdown 内容并添加到 DOCX 文档
func (si *GenDoc) addMarkdownToDocx(w *docx.Docx, mdContent string) {
	lines := strings.Split(mdContent, "\n")

	// 判断上次换行没有
	lastHasLine := false
	beforeDict := map[string]string{
		"#":    "",
		"##":   "",
		"###":  "",
		"####": "\u3000",
	}
	before := ""

	for _, oldLine := range lines {
		line := strings.TrimSpace(oldLine)
		leftSpaces := countLeadingSpaces(oldLine)
		if line == "" {
			if !lastHasLine { // 处理换行
				w.AddParagraph()
			}
			lastHasLine = true
			//before = ""
			continue // 忽略空行
		}

		para := w.AddParagraph()
		prefix := getTitlePrefix(line)
		if !utils.IsValueEmpty(prefix) { // 处理当前标题
			before = beforeDict[prefix]
			para.AddText(before)
		} else { // 处理上一个标题
			para.AddText(before + "\u3000")
		}

		lastHasLine = false

		// 解析标题
		if strings.HasPrefix(line, "# ") {
			para.AddText(strings.TrimPrefix(line, "# ")).Size("28").Bold()
			continue
		}
		if strings.HasPrefix(line, "## ") {
			para.AddText(strings.TrimPrefix(line, "## ")).Size("26").Bold()
			continue
		}
		if strings.HasPrefix(line, "### ") {
			para.AddText(strings.TrimPrefix(line, "### ")).Size("22").Bold()
			continue
		}

		if strings.HasPrefix(line, "#### ") {
			para.AddText(strings.TrimPrefix(line, "#### ")).Size("20").Bold()
			continue
		}
		if strings.Index(line, "-") == 0 {
			line = strings.Replace(line, "-", "", 1)
			para.AddText(leftSpaces).Size("18").Bold()
			para.AddText("•").Size("18").Bold()
			para.AddText("\u2009").Size("18").Bold()

		}
		// 解析加粗文本
		if strings.Contains(line, "**") {
			parts := strings.Split(line, "**")
			for i, part := range parts {
				if i%2 == 1 {
					para.AddText("\u2009" + part).Size("18").Bold() // 加粗文本
				} else {
					para.AddText(part) // 普通文本
				}
			}
			continue
		}

		// 解析斜体文本
		if strings.Contains(line, "*") {
			parts := strings.Split(line, "*")
			//para := w.AddParagraph()
			for i, part := range parts {
				if i%2 == 1 {
					para.AddText(part).Italic() // 斜体文本
				} else {
					para.AddText(part) // 普通文本
				}
			}
			continue
		}

		// 解析列表
		if strings.HasPrefix(line, "- ") {
			//para := w.AddParagraph()
			para.AddText("\u3000\u3000•\u3000" + strings.TrimPrefix(line, "- ")) // 添加列表符号
			continue
		}

		// 默认处理：普通文本
		//para := w.AddParagraph()

		para.AddText(line)
	}
}
