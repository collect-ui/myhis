package model

import (
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	"moon/model/autodesk"
	"moon/model/base"
	"moon/model/devops"
	"moon/model/doc"
	"moon/model/his"
	"moon/model/sport"
	"moon/model/work_task"
)

var modelMap map[string]interface{}
var primaryKeyMap map[string][]string

type TableData struct {
	templateService.DatabaseModel
}

// 生成一个脚本自动填充这个
func init() {
	//todo 如果用hashmap 效率慢，可以换二叉树，目前1、200个表很快
	modelMap = make(map[string]interface{})
	primaryKeyMap = make(map[string][]string)
	//基础的

	baseTableMap, basePkMap := base.GetTable()
	for k, v := range baseTableMap {
		modelMap[k] = v

	}
	for k, v := range basePkMap {
		primaryKeyMap[k] = v
	}
	// 自动动化
	baseTableMap, basePkMap = autodesk.GetTable()
	for k, v := range baseTableMap {
		modelMap[k] = v

	}
	for k, v := range basePkMap {
		primaryKeyMap[k] = v
	}

	//运维的
	baseTableMap, basePkMap = devops.GetTable()
	for k, v := range baseTableMap {
		modelMap[k] = v

	}
	for k, v := range basePkMap {
		primaryKeyMap[k] = v
	}

	//运动的
	baseTableMap, basePkMap = sport.GetTable()
	for k, v := range baseTableMap {
		modelMap[k] = v

	}
	for k, v := range basePkMap {
		primaryKeyMap[k] = v
	}

	//文档的
	baseTableMap, basePkMap = doc.GetTable()
	for k, v := range baseTableMap {
		modelMap[k] = v
	}
	for k, v := range basePkMap {
		primaryKeyMap[k] = v
	}

	//文档的
	baseTableMap, basePkMap = doc.GetTable()
	for k, v := range baseTableMap {
		modelMap[k] = v
	}
	for k, v := range basePkMap {
		primaryKeyMap[k] = v
	}
	//任务
	baseTableMap, basePkMap = work_task.GetTable()
	for k, v := range baseTableMap {
		modelMap[k] = v
	}
	for k, v := range basePkMap {
		primaryKeyMap[k] = v
	}
	//his
	baseTableMap, basePkMap = his.GetTable()
	for k, v := range baseTableMap {
		modelMap[k] = v
	}
	for k, v := range basePkMap {
		primaryKeyMap[k] = v
	}

}
func (*TableData) GetModel(tableName string) interface{} {
	return modelMap[tableName]
}
func (*TableData) CloneModel(tableName string) interface{} {
	data := modelMap[tableName]
	return utils.Copy(data)
}
func (*TableData) GetPrimaryKey(tableName string) []string {
	return primaryKeyMap[tableName]
}
