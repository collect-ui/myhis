package plugins

import (
	"fmt"
	"github.com/antchfx/xmlquery"
	"strings"

	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
)

type Xml2Json struct {
	templateService.BaseHandler
}

const TAG = "tag"
const CHILDREN = "children"
const AttrName = "attributes"
const Name = "name"
const Value = "value"
const Level = "level"
const ID = "level#position"

func (si *Xml2Json) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	valTpl := handlerParam.ValueTpl
	value := utils.RenderTplData(valTpl, params).(string)
	// 将XML字符串解析为map
	// 使用XPath查询
	doc, err := xmlquery.Parse(strings.NewReader(value))
	if err != nil {
		panic(err)
	}
	// 遍历所有节点
	data := traverseNodes(doc)
	r := common.Ok(data, "处理参数成功")
	return r
}
func getChildrenNode(node *xmlquery.Node, x int) []map[string]interface{} {
	children := make([]map[string]interface{}, 0)
	childData := node.FirstChild
	if childData == nil {
		return children
	}
	y := 0
	for {
		y++
		childTag := strings.TrimSpace(childData.Data)
		if !utils.IsValueEmpty(childTag) {
			childMap := getTagAttr(childData, x+1, y)
			if childData.FirstChild != nil {
				childMap[CHILDREN] = getChildrenNode(childData, x+1)
			}
			children = append(children, childMap)
		}
		childData = childData.NextSibling
		if childData == nil {
			break
		}
	}
	return children
}
func getTagAttr(node *xmlquery.Node, x int, y int) map[string]interface{} {
	tag := node.Data
	nodeMap := make(map[string]interface{})
	nodeAttrList := make([]map[string]interface{}, 0)
	for _, item := range node.Attr {
		attrItem := make(map[string]interface{})
		attrItem[Name] = item.Name.Local
		attrItem[Value] = strings.TrimSpace(item.Value)
		nodeAttrList = append(nodeAttrList, attrItem)
	}
	nodeMap[AttrName] = nodeAttrList
	nodeMap[TAG] = tag
	nodeMap[Level] = x
	nodeMap[ID] = fmt.Sprint(x, "#", y)
	return nodeMap
}
func getRootNode(node *xmlquery.Node) map[string]interface{} {
	nodeMap := getTagAttr(node, 0, 0)
	nodeMap[CHILDREN] = getChildrenNode(node, 0)
	return nodeMap
}

// 遍历所有节点并将其转换为map
func traverseNodes(node *xmlquery.Node) []map[string]interface{} {
	xmlMap := make([]map[string]interface{}, 0)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xmlquery.ElementNode && child.FirstChild != nil && child.FirstChild.Type == xmlquery.TextNode {
			xmlMap = append(xmlMap, getRootNode(child))
			break
		}
	}
	return xmlMap
}
