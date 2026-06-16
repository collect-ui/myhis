package main

import (
	"encoding/json"
	"fmt"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
	"github.com/fumiama/go-docx"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func init() {
	tagHandlers["layout-fit"] = processLayout
	tagHandlers["h3"] = processH3
	tagHandlers["p"] = processP

}
func main() {
	// 示例输入数据
	doc := map[string]interface{}{
		"title":             "示例文档",
		"summary":           "这是文档的摘要内容",
		"primary_list_name": "要点",
		"primary_list":      []map[string]interface{}{{"name": "项目1", "description": "描述1"}, {"name": "项目2", "description": "描述2"}},
		"first_img_path":    "path/to/image.png",
		"step_list": []map[string]interface{}{
			{"description": "步骤1", "code": "代码1", "language": "go"},
			{"description": "步骤2", "code": "代码2", "language": "python"},
		},
	}
	configJSON := `{
         "tag": "layout-fit",
         "visible": "${doc.title?true:false}",
         "title":"${doc.title}",
         "children": [
           {
             "tag": "p",
             "style": {
               "textIndent": "2em"
             },
             "children": "${doc.summary}"
           },
           {
             "tag": "h3",
             "visible": "${!!doc.primary_list_name}",
             "children": "${doc.primary_list_name}",
             "style": {
               "margin": 0,
               "paddingLeft": 14
             },
             "className": "border-bottom"
           },
           {
             "tag": "ol",
             "visible": "${doc.primary_list?.length>0}",
             "children": [
               {
                 "tag": "listview",
                 "itemData": "${doc.primary_list}",
                 "keyField": "index",
                 "itemAttr": {
                   "tag": "li",
                   "children": [
                     {
                       "tag": "strong",
                       "children": "${row.name}"
                     },
                     {
                       "tag": "label",
                       "visible": "${!!row.description}",
                       "children": "："
                     },
                     {
                       "tag": "label",
                       "children": "${row.description}"
                     }
                   ]
                 }
               }
             ]
           },
           {
             "tag": "h3",
             "visible": "${!!doc.first_img_name}",
             "children": "${doc.first_img_name}",
             "style": {
               "margin": 0,
               "paddingLeft": 14
             },
             "className": "border-bottom"
           },
           {
             "tag": "image",
             "src": "${doc.first_img_path}",
             "height": 300,
             "visible": "${!!doc.first_img_path}",
             "style": {
               "objectFit": "cover",
               "width": "initial",
               "paddingLeft": 100
             }
           },
           {
             "tag": "h3",
             "visible": "${!!doc.step_name}",
             "children": "${doc.step_name}",
             "style": {
               "margin": 0,
               "paddingLeft": 14
             },
             "className": "border-bottom"
           },
           {
             "tag": "ol",
             "visible": "${doc.step_list?.length>0}",
             "children": [
               {
                 "tag": "row",
                 "children": [
                   {
                     "tag": "listview",
                     "itemData": "${doc.step_list}",
                     "keyField": "index",
                     "itemAttr": {
                       "tag": "col",
                       "span": "${doc.step_span?doc.step_span:24}",
                       "children": [
                         {
                           "tag": "li",
                           "style": {
                             "padding": 10,
                             "marginRight": 10
                           },
                           "children": [
                             {
                               "tag": "div",
                               "children": [
                                 {
                                   "tag": "strong",
                                   "visible": "${!!row.description}",
                                   "children": "${row.description}"
                                 }
                               ]
                             },
                             {
                               "tag": "coder",
                               "visible": "${!!row.code && row.is_show!='1'}",
                               "language": "${row.language}",
                               "children": "${row.code}",
                               "customStyle": {
                                 "maxHeight": 360
                               }
                             },
                             {
                               "tag": "code-preview",
                               "visible": "${!!row.code && row.is_show=='1'}",
                               "style": {
                                 "height": 300
                               },
                               "hideAdd": true,
                               "className": "webshell-container",
                               "code": "${row.code}",
                               "type": "editable-card",
                               "size": "small"
                             },
                             {
                               "tag": "image",
                               "src": "${row.img_path}",
                               "width": "100%",
                               "visible": "${!!row.img_path}",
                               "style": {
                                 "objectFit": "cover",
                                 "height": "initial"
                               }
                             }
                           ]
                         }
                       ]
                     }
                   }
                 ]
               }
             ]
           },
           {
             "tag": "h3",
             "visible": "${doc?.api_list?.length>0}",
             "children": "API",
             "style": {
               "margin": 0,
               "paddingLeft": 14
             },
             "className": "border-bottom"
           },
           {
             "tag": "table",
             "domLayout": "autoHeight",
             "visible": "${doc?.api_list?.length>0}",
             "rowData": "${doc?.api_list}",
             "columnDefs": [
               {
                 "headerName": "#",
                 "width": 100,
                 "suppressSizeToFit": true,
                 "valueGetter": "node.rowIndex + 1",
                 "sortable": false,
                 "pinned": "left"
               },
               {
                 "headerName": "名称",
                 "width": 200,
                 "suppressSizeToFit": true,
                 "field": "name",
                 "editable": true
               },
               {
                 "headerName": "类型",
                 "width": 200,
                 "suppressSizeToFit": true,
                 "field": "type"
               },
               {
                 "headerName": "默认值",
                 "width": 200,
                 "suppressSizeToFit": true,
                 "field": "default_value"
               },
               {
                 "headerName": "描述",
                 "field": "description",
                 "tooltipField": "description",
                 "autoHeight": true
               }
             ]
           },
           {
             "tag": "p",
             "style": {
               "textIndent": "2em"
             },
             "visible": "${!!doc.description}",
             "children": "${doc.description}"
           }
         ]
       } ` // 这里放入你提供的完整 JSON 配置

	var config map[string]interface{}
	err := json.Unmarshal([]byte(configJSON), &config)
	if err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	w := docx.New().WithDefaultTheme()
	// 创建并保存 DOCX 文件
	f, err := os.Create("output_temp.docx")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	data := make(map[string]interface{})
	data["doc"] = doc
	processConfig(w, config, data)
	// 将文档写入文件
	_, err = w.WriteTo(f)
}

// 标签处理函数映射
var tagHandlers = map[string]func(*docx.Docx, map[string]interface{}, map[string]interface{}){}

// 并行处理子元素
func processChildren(docxFile *docx.Docx, children []map[string]interface{}, data map[string]interface{}) {
	for _, child := range children {
		processConfig(docxFile, child, data)
	}
}

// 具体layout-fit标签处理函数
func processLayout(docxFile *docx.Docx, config map[string]interface{}, data map[string]interface{}) {
	children, errMsg := utils.RenderVarToArrMap("children", config)
	titleTemp := utils.RenderVar("title", config)
	if !utils.IsValueEmpty(titleTemp) {
		para := docxFile.AddParagraph()
		title := processExpression(titleTemp.(string), data)
		para.AddText(gocast.ToString(title)).Size("18").Bold()

	}
	if utils.IsValueEmpty(errMsg) {
		processChildren(docxFile, children, data)
	}

}

// 具体layout-fit标签处理函数
func processH3(docxFile *docx.Docx, config map[string]interface{}, data map[string]interface{}) {
	para := docxFile.AddParagraph()
	text := utils.RenderVar("children", config).(string)
	value := processExpression(text, data)
	para.AddText(gocast.ToString(value)).Size("16").Bold()
	applyCachedStyle(para, "h3")

}

// 具体layout-fit标签处理函数
func processP(docxFile *docx.Docx, config map[string]interface{}, data map[string]interface{}) {
	para := docxFile.AddParagraph()
	text := utils.RenderVar("children", config).(string)
	value := processExpression(text, data)
	para.AddText(gocast.ToString(value)).Size("16")

}
func applyCachedStyle(para *docx.Paragraph, styleName string) {
	//if style, ok := styleCache[styleName]; ok {
	//	para.SetStyle(style)
	//}
}

func processConfig(docx *docx.Docx, config map[string]interface{}, data map[string]interface{}) {
	visible := utils.RenderVar("visible", config)
	if !utils.IsValueEmpty(visible) {
		v := processExpression(gocast.ToString(visible), data)
		// 如果不可见则跳过
		if !gocast.ToBool(v) {
			return
		}
	}

	tag := utils.RenderVar("tag", config).(string)
	// 使用策略模式处理不同标签
	if handler, ok := tagHandlers[tag]; ok {
		handler(docx, config, data)
	} else {
		// 默认处理子元素
		children, errMsg := utils.RenderVarToArrMap("children", config)
		if utils.IsValueEmpty(errMsg) {
			processChildren(docx, children, data)
		}

	}
}

// processExpression 处理各种表达式并返回结果
func processExpression(input string, data map[string]interface{}) interface{} {
	input = strings.ReplaceAll(input, "?.", ".")
	// 先检查是否是表达式
	if !strings.HasPrefix(input, "${") || !strings.HasSuffix(input, "}") {
		return input
	}

	expr := strings.TrimPrefix(input, "${")
	expr = strings.TrimSuffix(expr, "}")

	// 处理三元运算符
	if strings.Contains(expr, "?") {
		parts := strings.Split(expr, "?")
		if len(parts) != 2 {
			return nil
		}

		condition := parts[0]
		trueFalseParts := strings.Split(parts[1], ":")
		if len(trueFalseParts) < 2 {
			return nil
		}

		// 评估条件
		condValue := processExpression("${"+condition+"}", data)
		condBool := false

		// 更完善的条件值判断
		switch v := condValue.(type) {
		case bool:
			condBool = v
		case string:
			condBool = v != ""
		case int, float64:
			condBool = v != 0
		default:
			condBool = condValue != nil
		}

		if condBool {
			return processExpression("${"+trueFalseParts[0]+"}", data)
		}
		return processExpression("${"+trueFalseParts[1]+"}", data)
	}

	// 处理取反表达式
	if strings.HasPrefix(expr, "!!") {
		value := processExpression("${"+expr[2:]+"}", data)
		return value != nil && fmt.Sprintf("%v", value) != ""
	}

	// 处理条件判断表达式
	if strings.Contains(expr, ">") {
		condParts := strings.Split(expr, ">")
		if len(condParts) == 2 {
			leftStr := condParts[0]
			rightStr := condParts[1]

			leftValue := processExpression("${"+leftStr+"}", data)
			if leftValue == nil {
				return false
			}
			if strings.Contains(leftStr, ".length") {
				length := reflect.ValueOf(leftValue).Len()
				rightNum, err := strconv.Atoi(rightStr)
				if err == nil {
					return length > rightNum
				}
			}
		}
		return false
	}
	//如果没有包含点就直接返回
	if !strings.Contains(expr, ".") {
		return expr
	}
	// 正常获取值
	keys := strings.Split(expr, ".")

	current := data
	for _, key := range keys {
		if strings.HasSuffix(key, "?") {
			key = strings.TrimSuffix(key, "?")
		}
		if val, ok := current[key]; ok {
			if nextMap, ok := val.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return val
			}
		} else {
			return nil
		}
	}
	return nil
}
