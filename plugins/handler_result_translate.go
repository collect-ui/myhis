package plugins

import (
	"fmt"

	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	cacheHandler "github.com/collect-ui/collect/src/collect/service_imp/cache_handler"
	utils "github.com/collect-ui/collect/src/collect/utils"
)

// Translate 翻译处理器：支持预热、取两种模式
type Translate struct {
	templateService.BaseHandler
}

// RegistryEntry 注册表条目结构体
type RegistryEntry struct {
	Type          string
	Service       string
	KeyField      string
	TextField     string
	PreloadParams map[string]interface{}
	Cache         *CacheConfig
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Room       string
	Key        string
	PreloadKey string
	Seconds    int64
}

// getString 安全获取 map 中的字符串值
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getMap 安全获取 map 中的 map 值
func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]interface{}); ok {
			return sub
		}
	}
	return nil
}

// HandlerData 处理器入口方法
func (t *Translate) HandlerData(template *config.Template,
	handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {

	// 错误恢复：捕获 panic 并返回错误信息
	defer func() {
		if r := recover(); r != nil {
			panic(r) // 重新抛出以便框架处理
		}
	}()

	ch := cacheHandler.CacheHandler{}

	// 预热模式：无 type、无 fields
	if handlerParam.Type == "" && len(handlerParam.Fields) == 0 {
		return t.preload(handlerParam, ch, ts)
	}

	// 取模式：有 type + fields
	if len(handlerParam.Fields) > 0 {
		return t.translate(template, handlerParam, ch, ts)
	}

	return common.NotOk("translate handler param error")
}

// preload 预热方法：无 type、无 fields
func (t *Translate) preload(handlerParam *config.HandlerParam,
	ch cacheHandler.CacheHandler, ts *templateService.TemplateService) *common.Result {

	// 从 application.properties 读取完整服务名
	serviceName := utils.GetAppKey("translate_registry_service")

	// 调用服务获取注册表数据
	serviceParam := map[string]interface{}{
		"service": serviceName,
	}
	result := ts.ResultInner(serviceParam)
	if !result.Success {
		return common.NotOk("获取注册表失败: " + result.Msg)
	}

	// 解析注册表条目
	data := result.GetData()
	if data == nil {
		return common.NotOk("注册表数据为空")
	}

	// 安全类型转换
	var registryData []map[string]interface{}
	switch v := data.(type) {
	case []map[string]interface{}:
		registryData = v
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				registryData = append(registryData, m)
			}
		}
	default:
		return common.NotOk("注册表数据格式错误")
	}

	for _, entryMap := range registryData {
		// 安全获取字段值
		typeStr := getString(entryMap, "type")
		serviceStr := getString(entryMap, "service")
		keyFieldStr := getString(entryMap, "key_field")
		textFieldStr := getString(entryMap, "text_field")
		preloadParamsMap := getMap(entryMap, "preload_params")
		cacheMap := getMap(entryMap, "cache")

		// 跳过无效条目
		if typeStr == "" || serviceStr == "" || keyFieldStr == "" || textFieldStr == "" {
			continue
		}
		if preloadParamsMap == nil || cacheMap == nil {
			continue
		}

		entry := &RegistryEntry{
			Type:          typeStr,
			Service:       serviceStr,
			KeyField:      keyFieldStr,
			TextField:     textFieldStr,
			PreloadParams: preloadParamsMap,
			Cache:         parseCacheConfig(cacheMap),
		}

		// 检查 preload_params 是否有值
		if len(entry.PreloadParams) == 0 {
			continue
		}

		// 用 preload_params 调 service
		preloadData, ok := utils.Copy(entry.PreloadParams).(map[string]interface{})
		if !ok {
			continue
		}
		serviceResult := ts.ResultInner(preloadData)
		if !serviceResult.Success {
			continue
		}

		// 遍历返回数据，逐行写入缓存
		serviceData := serviceResult.GetData()
		if serviceData == nil {
			continue
		}

		// 安全类型转换
		var dataList []map[string]interface{}
		switch v := serviceData.(type) {
		case []map[string]interface{}:
			dataList = v
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					dataList = append(dataList, m)
				}
			}
		default:
			continue
		}

		for _, row := range dataList {
			// 构造缓存 key：room[STANDARD_CODE#ITEM_VALUE]
			// 直接拼接，不用 RenderVar（因为 RenderVar 不支持 # 分隔符）
			standardCode := ""
			if v, ok := row["STANDARD_CODE"]; ok {
				standardCode = fmt.Sprintf("%v", v)
			}
			keyFieldValue := ""
			if v, ok := row[entry.KeyField]; ok {
				keyFieldValue = fmt.Sprintf("%v", v)
			}
			cacheKey := entry.Cache.Room + "[" + standardCode + "#" + keyFieldValue + "]"

			// 取翻译文本
			textRaw := row[entry.TextField]
			var text string
			if textRaw != nil {
				if s, ok := textRaw.(string); ok {
					text = s
				} else {
					text = fmt.Sprintf("%v", textRaw)
				}
			}

			// 写入缓存
			cacheResult := common.Ok(text, "翻译缓存")
			ch.Set(cacheKey, *cacheResult, entry.Cache.Seconds)
		}
		ch.Wait()
	}

	return common.Ok(nil, "预热完成")
}

// translate 取方法：有 type + fields
func (t *Translate) translate(template *config.Template,
	handlerParam *config.HandlerParam, ch cacheHandler.CacheHandler,
	ts *templateService.TemplateService) *common.Result {

	// 获取注册表配置
	entry := t.getRegistry(handlerParam.Type, ts)
	if entry == nil {
		return common.NotOk("entry nil for type=" + handlerParam.Type)
	}

	// 获取要翻译的数据列表
	// 优先从 handlerParam.Foreach 指定的变量取数据
	// 如果 Foreach 为空，则从模板结果（SQL 查询结果）取数据
	var dataList []map[string]interface{}
	foreach := handlerParam.Foreach
	if foreach != "" {
		// 从指定变量取数据
		params := template.GetParams()
		data := utils.RenderVar(foreach, params)
		if data != nil {
			switch v := data.(type) {
			case []map[string]interface{}:
				dataList = v
			case []interface{}:
				for _, item := range v {
					if m, ok := item.(map[string]interface{}); ok {
						dataList = append(dataList, m)
					}
				}
			}
		}
	} else {
		// 从模板结果取数据（SQL 查询结果）
		result := template.GetResult()
		if result != nil && result.GetData() != nil {
			switch v := result.GetData().(type) {
			case []map[string]interface{}:
				dataList = v
			case []interface{}:
				for _, item := range v {
					if m, ok := item.(map[string]interface{}); ok {
						dataList = append(dataList, m)
					}
				}
			}
		}
	}

	// 无数据可翻译
	if len(dataList) == 0 {
		return common.Ok(nil, "ok")
	}

	// 遍历 fields，逐个翻译
	for _, field := range handlerParam.Fields {
		from := field.From
		to := field.To

		for _, item := range dataList {
			// 取编码值：从 item 中取 field.Field 指定的字段值
			// 例如 field.Field="[item.outp_type_code]"，取 item["outp_type_code"]
			fieldExprResult := utils.RenderVar(field.Field, map[string]interface{}{"item": item})
			if fieldExprResult == nil {
				continue
			}
			itemValue, _ := fieldExprResult.(string)
			if itemValue == "" {
				continue
			}

			// 构造缓存 key：room[from#itemValue]
			// 例如 code[PIX0021#1]
			cacheKey := entry.Cache.Room + "[" + from + "#" + itemValue + "]"

			// 查缓存
			cacheResult, ok := ch.Get(cacheKey)
			if ok {
				// 缓存命中，根据实际存储类型提取文本
				// ch.Set 存储的是 common.Result 值（非指针）
				if result, ok := cacheResult.(common.Result); ok {
					if text, ok := result.Data.(string); ok {
						item[to] = text
					}
				} else if result, ok := cacheResult.(*common.Result); ok {
					if text, ok := result.Data.(string); ok {
						item[to] = text
					}
				} else if cacheMap, ok := cacheResult.(map[string]interface{}); ok {
					if data, ok := cacheMap["data"]; ok {
						if text, ok := data.(string); ok {
							item[to] = text
						}
					}
				}
				continue
			}

			// 未命中 → 调 service 查一个
			serviceParam := map[string]interface{}{
				"service":       entry.Service,
				"standard_code": from,
			}
			serviceResult := ts.ResultInner(serviceParam)
			if !serviceResult.Success {
				continue
			}

			// 从结果中找到匹配行
			serviceData := serviceResult.GetData()
			if serviceData == nil {
				continue
			}

			// 安全类型转换
			var rows []map[string]interface{}
			switch v := serviceData.(type) {
			case []map[string]interface{}:
				rows = v
			case []interface{}:
				for _, item := range v {
					if m, ok := item.(map[string]interface{}); ok {
						rows = append(rows, m)
					}
				}
			default:
				continue
			}

			for _, row := range rows {
				if row[entry.KeyField] == itemValue {
					if textRaw, ok := row[entry.TextField]; ok {
						if text, ok := textRaw.(string); ok {
							item[to] = text
						} else {
							item[to] = fmt.Sprintf("%v", textRaw)
						}
					}

					// 写入缓存
					cacheResult := common.Ok(item[to], "翻译缓存")
					ch.Set(cacheKey, *cacheResult, entry.Cache.Seconds)
					ch.Wait()
					break
				}
			}
		}
	}

	return common.Ok(nil, "翻译完成")
}

// getRegistry 获取注册表配置
func (t *Translate) getRegistry(typeName string, ts *templateService.TemplateService) *RegistryEntry {

	// 从 application.properties 读取完整服务名
	serviceName := utils.GetAppKey("translate_registry_service")

	// 调用服务
	serviceParam := map[string]interface{}{
		"service": serviceName,
	}
	result := ts.ResultInner(serviceParam)
	if !result.Success {
		return nil
	}

	// 解析注册表条目，找到匹配的 type
	data := result.GetData()
	if data == nil {
		return nil
	}

	// 安全类型转换
	var registryData []map[string]interface{}
	switch v := data.(type) {
	case []map[string]interface{}:
		registryData = v
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				registryData = append(registryData, m)
			}
		}
	default:
		return nil
	}

	for _, entryMap := range registryData {
		typeStr := getString(entryMap, "type")
		if typeStr == typeName {
			serviceStr := getString(entryMap, "service")
			keyFieldStr := getString(entryMap, "key_field")
			textFieldStr := getString(entryMap, "text_field")
			preloadParamsMap := getMap(entryMap, "preload_params")
			cacheMap := getMap(entryMap, "cache")

			if serviceStr == "" || keyFieldStr == "" || textFieldStr == "" {
				return nil
			}
			if preloadParamsMap == nil || cacheMap == nil {
				return nil
			}

			return &RegistryEntry{
				Type:          typeStr,
				Service:       serviceStr,
				KeyField:      keyFieldStr,
				TextField:     textFieldStr,
				PreloadParams: preloadParamsMap,
				Cache:         parseCacheConfig(cacheMap),
			}
		}
	}
	return nil
}

// parseCacheConfig 解析缓存配置
func parseCacheConfig(cacheMap map[string]interface{}) *CacheConfig {
	if cacheMap == nil {
		return &CacheConfig{}
	}
	seconds := int64(0)
	if v, ok := cacheMap["seconds"]; ok {
		switch val := v.(type) {
		case float64:
			seconds = int64(val)
		case int64:
			seconds = val
		case int:
			seconds = int64(val)
		}
	}
	return &CacheConfig{
		Room:       getString(cacheMap, "room"),
		Key:        getString(cacheMap, "key"),
		PreloadKey: getString(cacheMap, "preload_key"),
		Seconds:    seconds,
	}
}
