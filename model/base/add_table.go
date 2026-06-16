package base

func GetTable() (map[string]interface{}, map[string][]string) {
	modelMap := make(map[string]interface{})
	primaryKeyMap := make(map[string][]string)
	tableCollectDoc := CollectDoc{}
	modelMap["collect_doc"] = tableCollectDoc
	primaryKeyMap["collect_doc"] = tableCollectDoc.PrimaryKey()
	//32 collect_doc_demo
	tableCollectDocDemo := CollectDocDemo{}
	modelMap["collect_doc_demo"] = tableCollectDocDemo
	primaryKeyMap["collect_doc_demo"] = tableCollectDocDemo.PrimaryKey()
	//33 collect_doc_important
	tableCollectDocImportant := CollectDocImportant{}
	modelMap["collect_doc_important"] = tableCollectDocImportant
	primaryKeyMap["collect_doc_important"] = tableCollectDocImportant.PrimaryKey()
	//34 collect_doc_params
	tableCollectDocParams := CollectDocParams{}
	modelMap["collect_doc_params"] = tableCollectDocParams
	primaryKeyMap["collect_doc_params"] = tableCollectDocParams.PrimaryKey()
	//35 collect_doc_result
	tableCollectDocResult := CollectDocResult{}
	modelMap["collect_doc_result"] = tableCollectDocResult
	primaryKeyMap["collect_doc_result"] = tableCollectDocResult.PrimaryKey()
	//36 config_detail
	tableConfigDetail := ConfigDetail{}
	modelMap["config_detail"] = tableConfigDetail
	primaryKeyMap["config_detail"] = tableConfigDetail.PrimaryKey()
	//37 config_group
	tableConfigGroup := ConfigGroup{}
	modelMap["config_group"] = tableConfigGroup
	primaryKeyMap["config_group"] = tableConfigGroup.PrimaryKey()
	//89 doc_group
	tableDocGroup := DocGroup{}
	modelMap["doc_group"] = tableDocGroup
	primaryKeyMap["doc_group"] = tableDocGroup.PrimaryKey()
	//242 role
	tableRole := Role{}
	modelMap["role"] = tableRole
	primaryKeyMap["role"] = tableRole.PrimaryKey()

	//242 role
	userRoleIdList := UserRoleIDList{}
	modelMap["user_role_id_list"] = userRoleIdList
	primaryKeyMap["user_role_id_list"] = userRoleIdList.PrimaryKey()
	//353 user_role_id_list
	userAccount := UserAccount{}
	modelMap["user_account"] = userAccount
	primaryKeyMap["user_account"] = userAccount.PrimaryKey()

	attachment := Attachment{}
	modelMap["attachment"] = attachment
	primaryKeyMap["attachment"] = attachment.PrimaryKey()

	// 菜单
	sysMenu := SysMenu{}
	modelMap["sys_menu"] = sysMenu
	primaryKeyMap["sys_menu"] = sysMenu.PrimaryKey()

	roleMenu := RoleMenu{}
	modelMap["role_menu"] = roleMenu
	primaryKeyMap["role_menu"] = roleMenu.PrimaryKey()

	schemaPageData := SchemaPageData{}
	modelMap[schemaPageData.TableName()] = schemaPageData
	primaryKeyMap[schemaPageData.TableName()] = schemaPageData.PrimaryKey()

	schemaPageField := SchemaPageField{}
	modelMap[schemaPageField.TableName()] = schemaPageField
	primaryKeyMap[schemaPageField.TableName()] = schemaPageField.PrimaryKey()

	userChangeHistory := UserChangeHistory{}
	modelMap[userChangeHistory.TableName()] = userChangeHistory
	primaryKeyMap[userChangeHistory.TableName()] = userChangeHistory.PrimaryKey()

	registration := Registration{}
	modelMap[registration.TableName()] = registration
	primaryKeyMap[registration.TableName()] = registration.PrimaryKey()

	sysBtn := SysBtn{}
	modelMap[sysBtn.TableName()] = sysBtn
	primaryKeyMap[sysBtn.TableName()] = sysBtn.PrimaryKey()

	btnRoleIDList := BtnRoleIDList{}
	modelMap[btnRoleIDList.TableName()] = btnRoleIDList
	primaryKeyMap[btnRoleIDList.TableName()] = btnRoleIDList.PrimaryKey()

	locationTracker := LocationTracker{}
	modelMap[locationTracker.TableName()] = locationTracker
	primaryKeyMap[locationTracker.TableName()] = locationTracker.PrimaryKey()

	schemaPage := SchemaPage{}
	modelMap[schemaPage.TableName()] = schemaPage
	primaryKeyMap[schemaPage.TableName()] = schemaPage.PrimaryKey()

	feedback := Feedback{}
	modelMap[feedback.TableName()] = feedback
	primaryKeyMap[feedback.TableName()] = feedback.PrimaryKey()

	feedbackAttachment := FeedbackAttachment{}
	modelMap[feedbackAttachment.TableName()] = feedbackAttachment
	primaryKeyMap[feedbackAttachment.TableName()] = feedbackAttachment.PrimaryKey()

	mailAccount := MailAccount{}
	modelMap[mailAccount.TableName()] = mailAccount
	primaryKeyMap[mailAccount.TableName()] = mailAccount.PrimaryKey()

	agentSession := AgentSession{}
	modelMap[agentSession.TableName()] = agentSession
	primaryKeyMap[agentSession.TableName()] = agentSession.PrimaryKey()

	agentMessage := AgentMessage{}
	modelMap[agentMessage.TableName()] = agentMessage
	primaryKeyMap[agentMessage.TableName()] = agentMessage.PrimaryKey()

	agentRun := AgentRun{}
	modelMap[agentRun.TableName()] = agentRun
	primaryKeyMap[agentRun.TableName()] = agentRun.PrimaryKey()

	return modelMap, primaryKeyMap
}
