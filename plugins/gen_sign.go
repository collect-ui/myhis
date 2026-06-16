package plugins

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"github.com/demdxx/gocast"
	uuid "github.com/satori/go.uuid"
	"io"
	"os"
)

type GenSign struct {
	templateService.BaseHandler
}

func getFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
func (si *GenSign) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	params := template.GetParams()
	chatKey := utils.GetAppKey("chat_key")
	arr, _ := utils.RenderVarToArrMap(handlerParam.Foreach, params)

	requestBody := ""
	if !utils.IsValueEmpty(handlerParam.Path) {
		filePath := gocast.ToString(utils.RenderVar(handlerParam.Path, params))
		requestBody, _ = getFileMD5(filePath)
	} else if !utils.IsValueEmpty(handlerParam.Field) { //zjb 转tbj
		field, ok := utils.RenderVar(handlerParam.Field, params).(string)
		if ok && !utils.IsValueEmpty(field) {
			requestBody = field
		} else {
			tmp, _ := utils.RenderVarToMap(handlerParam.Field, params)
			jsonStr, _ := json.Marshal(tmp)
			requestBody = string(jsonStr)
		}

	} else { // 判断项目编码，用户大模型生产文档
		projectCode := gocast.ToString(params["project_code"])
		if !utils.IsValueEmpty(arr) {
			data := make(map[string]interface{})
			groups := make([]map[string]interface{}, 0)
			for _, item := range arr {

				group := make(map[string]interface{})
				items := make([]map[string]interface{}, 0)
				children, _ := utils.RenderVarToArrMap("[children]", item)
				for _, child := range children {
					dataItem := make(map[string]interface{})
					dataItem["name"] = child["name"].(string)
					dataItem["detail"] = child["detail"].(string)
					items = append(items, dataItem)
				}
				//group["projectNo"] = projectCode
				group["docNo"] = uuid.NewV4().String()
				group["items"] = items
				group["topic"] = item["topic"]
				groups = append(groups, group)
			}
			data["groups"] = groups
			data["projectNo"] = projectCode
			jsonData, _ := json.Marshal(data)
			requestBody = string(jsonData)
		} else { // 查询大模型结果
			//dataList := make([]string, 0)
			//dataList = append(dataList, docNo)
			tmp := make(map[string]interface{})
			tmp["projectNo"] = projectCode
			jsonData, _ := json.Marshal(tmp)
			requestBody = string(jsonData)
		}
	}
	signStr := requestBody + chatKey
	hash := md5.New()
	// 将数据写入哈希对象
	hash.Write([]byte(signStr))
	// 计算 MD5 哈希值
	hashValue := hash.Sum(nil)
	sign := hex.EncodeToString(hashValue)
	params["sign"] = sign
	params["request_body"] = requestBody
	r := common.Ok(nil, "处理参数成功")
	return r
}
