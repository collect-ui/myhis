package doc

func GetTable() (map[string]interface{}, map[string][]string) {
	modelMap := make(map[string]interface{})
	primaryKeyMap := make(map[string][]string)
	// 文件夹
	frontendDocGroup := FrontendDocGroup{}
	modelMap[FrontendDocGroupName] = frontendDocGroup
	primaryKeyMap[FrontendDocGroupName] = frontendDocGroup.PrimaryKey()

	// 文件夹
	imgDir := ImgDir{}
	modelMap[ImgDirName] = imgDir
	primaryKeyMap[ImgDirName] = imgDir.PrimaryKey()
	// 文件夹
	img := Img{}
	modelMap[ImgName] = img
	primaryKeyMap[ImgName] = img.PrimaryKey()

	return modelMap, primaryKeyMap
}
