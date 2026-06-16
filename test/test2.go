package main

//
//import (
//	"fmt"
//	"reflect"
//	"strconv"
//	"strings"
//)
//
//// processExpression 处理各种表达式并返回结果
//func processExpression(input string, data map[string]interface{}) interface{} {
//	input = strings.ReplaceAll(input, "?.", ".")
//	// 先检查是否是表达式
//	if !strings.HasPrefix(input, "${") || !strings.HasSuffix(input, "}") {
//		return input
//	}
//
//	expr := strings.TrimPrefix(input, "${")
//	expr = strings.TrimSuffix(expr, "}")
//
//	// 处理三元运算符
//	if strings.Contains(expr, "?") {
//		parts := strings.Split(expr, "?")
//		if len(parts) != 2 {
//			return nil
//		}
//
//		condition := parts[0]
//		trueFalseParts := strings.Split(parts[1], ":")
//		if len(trueFalseParts) < 2 {
//			return nil
//		}
//
//		// 评估条件
//		condValue := processExpression("${"+condition+"}", data)
//		condBool := false
//
//		// 更完善的条件值判断
//		switch v := condValue.(type) {
//		case bool:
//			condBool = v
//		case string:
//			condBool = v != ""
//		case int, float64:
//			condBool = v != 0
//		default:
//			condBool = condValue != nil
//		}
//
//		if condBool {
//			return processExpression("${"+trueFalseParts[0]+"}", data)
//		}
//		return processExpression("${"+trueFalseParts[1]+"}", data)
//	}
//
//	// 处理取反表达式
//	if strings.HasPrefix(expr, "!!") {
//		value := processExpression("${"+expr[2:]+"}", data)
//		return value != nil && fmt.Sprintf("%v", value) != ""
//	}
//
//	// 处理条件判断表达式
//	if strings.Contains(expr, ">") {
//		condParts := strings.Split(expr, ">")
//		if len(condParts) == 2 {
//			leftStr := condParts[0]
//			rightStr := condParts[1]
//
//			leftValue := processExpression("${"+leftStr+"}", data)
//			if leftValue == nil {
//				return false
//			}
//			if strings.Contains(leftStr, ".length") {
//				length := reflect.ValueOf(leftValue).Len()
//				rightNum, err := strconv.Atoi(rightStr)
//				if err == nil {
//					return length > rightNum
//				}
//			}
//		}
//		return false
//	}
//	//如果没有包含点就直接返回
//	if !strings.Contains(expr, ".") {
//		return expr
//	}
//	// 正常获取值
//	keys := strings.Split(expr, ".")
//
//	current := data
//	for _, key := range keys {
//		if strings.HasSuffix(key, "?") {
//			key = strings.TrimSuffix(key, "?")
//		}
//		if val, ok := current[key]; ok {
//			if nextMap, ok := val.(map[string]interface{}); ok {
//				current = nextMap
//			} else {
//				return val
//			}
//		} else {
//			return nil
//		}
//	}
//	return nil
//}
//
//func main() {
//	data := map[string]interface{}{
//		"doc": map[string]interface{}{
//			"title":             "test",
//			"step_name":         "test",
//			"primary_list_name": "test",
//			"step_span":         12,
//			"api_list":          []string{"item1", "item2"},
//		},
//	}
//
//	testCases := []struct {
//		name  string
//		input string
//	}{
//		{"场景7 - 可选链判断", "${doc?.api_list?.length>0}"},
//		{"场景2 - 三元运算符", "${doc.step_name?true:false}"},
//		{"场景1 - 基本取值", "${doc.step_name}"},
//		{"场景3 - 取反表达式", "${!!doc.primary_list_name}"},
//		{"场景4 - 长度判断", "${doc.api_list.length>0}"},
//		{"场景5 - 带默认值的三元", "${doc.step_span?doc.step_span:24}"},
//		{"场景6 - 标题存在判断", "${doc.title?true:false}"},
//	}
//
//	for _, tc := range testCases {
//		result := processExpression(tc.input, data)
//		fmt.Printf("%s:\n  输入: %s\n  输出: %v (类型: %T)\n\n",
//			tc.name, tc.input, result, result)
//	}
//}
