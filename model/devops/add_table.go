package devops

func GetTable() (map[string]interface{}, map[string][]string) {
	modelMap := make(map[string]interface{})
	primaryKeyMap := make(map[string][]string)

	webshellLog := WebshellLog{}
	modelMap["webshell_log"] = webshellLog
	primaryKeyMap["webshell_log"] = webshellLog.PrimaryKey()

	httpProxyRequestLog := HttpProxyRequestLog{}
	modelMap["http_proxy_request_log"] = httpProxyRequestLog
	primaryKeyMap["http_proxy_request_log"] = httpProxyRequestLog.PrimaryKey()

	webshellToken := WebshellToken{}
	modelMap["webshell_token"] = webshellToken
	primaryKeyMap["webshell_token"] = webshellToken.PrimaryKey()

	tableServerEnv := ServerEnv{}
	modelMap["server_env"] = tableServerEnv
	primaryKeyMap["server_env"] = tableServerEnv.PrimaryKey()
	//
	tableServerInstallSoft := ServerInstallSoft{}
	modelMap["server_install_soft"] = tableServerInstallSoft
	primaryKeyMap["server_install_soft"] = tableServerInstallSoft.PrimaryKey()

	serverInstance := ServerInstance{}
	modelMap["server_instance"] = serverInstance
	primaryKeyMap["server_instance"] = serverInstance.PrimaryKey()

	serverOsUsers := ServerOsUsers{}
	modelMap["server_os_users"] = serverOsUsers
	primaryKeyMap["server_os_users"] = serverOsUsers.PrimaryKey()

	workspaceProject := WebshellWorkspaceProject{}
	modelMap["webshell_workspace_project"] = workspaceProject
	primaryKeyMap["webshell_workspace_project"] = workspaceProject.PrimaryKey()

	workspaceFile := WebshellWorkspaceFile{}
	modelMap["webshell_workspace_file"] = workspaceFile
	primaryKeyMap["webshell_workspace_file"] = workspaceFile.PrimaryKey()

	workspaceFileRecent := WebshellWorkspaceFileRecent{}
	modelMap["webshell_workspace_file_recent"] = workspaceFileRecent
	primaryKeyMap["webshell_workspace_file_recent"] = workspaceFileRecent.PrimaryKey()

	quickText := WebshellQuickText{}
	modelMap["webshell_quick_text"] = quickText
	primaryKeyMap["webshell_quick_text"] = quickText.PrimaryKey()

	webSQLCommitEvent := WebSQLCommitEvent{}
	modelMap["websql_commit_event"] = webSQLCommitEvent
	primaryKeyMap["websql_commit_event"] = webSQLCommitEvent.PrimaryKey()

	webSQLRecentSQL := WebSQLRecentSQL{}
	modelMap["websql_recent_sql"] = webSQLRecentSQL
	primaryKeyMap["websql_recent_sql"] = webSQLRecentSQL.PrimaryKey()

	websqlFavorite := WebsqlFavorite{}
	modelMap["websql_favorite"] = websqlFavorite
	primaryKeyMap["websql_favorite"] = websqlFavorite.PrimaryKey()

	return modelMap, primaryKeyMap
}
