package sport

func GetTable() (map[string]interface{}, map[string][]string) {

	modelMap := make(map[string]interface{})
	primaryKeyMap := make(map[string][]string)
	trainingProject := TrainingProject{}
	modelMap["training_project"] = trainingProject
	primaryKeyMap["training_project"] = trainingProject.PrimaryKey()

	course := Course{}
	modelMap["course"] = course
	primaryKeyMap["course"] = course.PrimaryKey()
	courseVideo := CourseVideo{}
	modelMap["course_video"] = courseVideo
	primaryKeyMap["course_video"] = courseVideo.PrimaryKey()
	// 成绩
	courseGrade := CourseGrade{}
	modelMap["course_grade"] = courseGrade
	primaryKeyMap["course_grade"] = courseGrade.PrimaryKey()
	// 签到
	courseAttendance := CourseAttendance{}
	modelMap["course_attendance"] = courseAttendance
	primaryKeyMap["course_attendance"] = courseAttendance.PrimaryKey() // 签到

	return modelMap, primaryKeyMap
}
