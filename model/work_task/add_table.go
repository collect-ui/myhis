package work_task

func GetTable() (map[string]interface{}, map[string][]string) {
	modelMap := make(map[string]interface{})
	primaryKeyMap := make(map[string][]string)
	workTaskIssue := WorkTaskIssue{}
	modelMap[TableNameWorkTaskIssue] = workTaskIssue
	primaryKeyMap[TableNameWorkTaskIssue] = workTaskIssue.PrimaryKey()

	workTaskVersion := WorkTaskVersion{}
	modelMap[workTaskVersion.TableName()] = workTaskVersion
	primaryKeyMap[workTaskVersion.TableName()] = workTaskVersion.PrimaryKey()

	return modelMap, primaryKeyMap
}
