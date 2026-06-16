package his

func GetTable() (map[string]interface{}, map[string][]string) {
	modelMap := make(map[string]interface{})
	primaryKeyMap := make(map[string][]string)

	pixOutpRegMaster := PixOutpRegMaster{}
	modelMap[pixOutpRegMaster.TableName()] = pixOutpRegMaster
	primaryKeyMap[pixOutpRegMaster.TableName()] = pixOutpRegMaster.PrimaryKey()

	pixOutpRegPool := PixOutpRegPool{}
	modelMap[pixOutpRegPool.TableName()] = pixOutpRegPool
	primaryKeyMap[pixOutpRegPool.TableName()] = pixOutpRegPool.PrimaryKey()

	pixOutpRegMasterAtime := PixOutpRegMasterAtime{}
	modelMap[pixOutpRegMasterAtime.TableName()] = pixOutpRegMasterAtime
	primaryKeyMap[pixOutpRegMasterAtime.TableName()] = pixOutpRegMasterAtime.PrimaryKey()

	pixOutpRegMasterDetail := PixOutpRegMasterDetail{}
	modelMap[pixOutpRegMasterDetail.TableName()] = pixOutpRegMasterDetail
	primaryKeyMap[pixOutpRegMasterDetail.TableName()] = pixOutpRegMasterDetail.PrimaryKey()

	pixOutpRegSchedule := PixOutpRegSchedule{}
	modelMap[pixOutpRegSchedule.TableName()] = pixOutpRegSchedule
	primaryKeyMap[pixOutpRegSchedule.TableName()] = pixOutpRegSchedule.PrimaryKey()

	pixOutpRegScheduleAtime := PixOutpRegScheduleAtime{}
	modelMap[pixOutpRegScheduleAtime.TableName()] = pixOutpRegScheduleAtime
	primaryKeyMap[pixOutpRegScheduleAtime.TableName()] = pixOutpRegScheduleAtime.PrimaryKey()

	pixOutpRegMasterLog := PixOutpRegMasterLog{}
	modelMap[pixOutpRegMasterLog.TableName()] = pixOutpRegMasterLog
	primaryKeyMap[pixOutpRegMasterLog.TableName()] = pixOutpRegMasterLog.PrimaryKey()

	return modelMap, primaryKeyMap
}
