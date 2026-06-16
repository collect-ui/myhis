package autodesk

func GetTable() (map[string]interface{}, map[string][]string) {
	modelMap := make(map[string]interface{})
	primaryKeyMap := make(map[string][]string)
	priceDocTree := PriceDocTree{}
	modelMap["price_doc_tree"] = priceDocTree
	primaryKeyMap["price_doc_tree"] = priceDocTree.PrimaryKey()
	priceDocProject := PriceDocProject{}
	modelMap["price_doc_project"] = priceDocProject
	primaryKeyMap["price_doc_project"] = priceDocProject.PrimaryKey()
	priceDoc := PriceDoc{}
	modelMap["price_doc"] = priceDoc
	primaryKeyMap["price_doc"] = priceDoc.PrimaryKey()
	// 技术文档
	technicalDoc := TechnicalDoc{}
	modelMap["technical_doc"] = technicalDoc
	primaryKeyMap["technical_doc"] = technicalDoc.PrimaryKey()

	priceDocQingProject := PriceDocQingProject{}
	modelMap["price_doc_qing_project"] = priceDocQingProject
	primaryKeyMap["price_doc_qing_project"] = priceDocQingProject.PrimaryKey()

	priceDocQingTree := PriceDocQingTree{}
	modelMap["price_doc_qing_tree"] = priceDocQingTree
	primaryKeyMap["price_doc_qing_tree"] = priceDocQingTree.PrimaryKey()

	priceDocQing := PriceDocQing{}
	modelMap["price_doc_qing"] = priceDocQing
	primaryKeyMap["price_doc_qing"] = priceDocQing.PrimaryKey()

	tbConfig := TbConfig{}
	modelMap["tb_config"] = tbConfig
	primaryKeyMap["tb_config"] = tbConfig.PrimaryKey()

	tbProject := TbProject{}
	modelMap["tb_project"] = tbProject
	primaryKeyMap["tb_project"] = tbProject.PrimaryKey()

	docProject := DocProject{}
	modelMap[TableNameDocProject] = docProject
	primaryKeyMap[TableNameDocProject] = docProject.PrimaryKey()

	docProjectDetail := DocProjectDetail{}
	modelMap[TableNameDocProjectDetail] = docProjectDetail
	primaryKeyMap[TableNameDocProjectDetail] = docProjectDetail.PrimaryKey()

	docProjectContent := DocProjectContent{}
	modelMap[TableNameDocProjectContent] = docProjectContent
	primaryKeyMap[TableNameDocProjectContent] = docProjectContent.PrimaryKey()

	docProjectTask := DocProjectTask{}
	modelMap[TableNameDocProjectTask] = docProjectTask
	primaryKeyMap[TableNameDocProjectTask] = docProjectTask.PrimaryKey()

	tbProjectManagerConfig := TbProjectManagerConfig{}
	modelMap[TableNameTbProjectManagerConfig] = tbProjectManagerConfig
	primaryKeyMap[TableNameTbProjectManagerConfig] = tbProjectManagerConfig.PrimaryKey()

	return modelMap, primaryKeyMap
}
