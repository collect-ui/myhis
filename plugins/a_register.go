package plugins

import (
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
)

func GetRegisterList() []templateService.ModuleResult {
	l := make([]templateService.ModuleResult, 0)
	l = append(l, &AnalysisIp{})
	l = append(l, &Shell{})
	l = append(l, &ShellTerm{})
	l = append(l, &ParamKey2Arr{})
	l = append(l, &RenameField{})
	l = append(l, &MultiArr{})        // 数组成数组
	l = append(l, &HandlerPassword{}) // 数组成数组
	l = append(l, &ValueTransfer{})   // 值转换
	l = append(l, &ReadFile{})        // 值转换
	l = append(l, &MailAccountAuthJSONImport{})
	l = append(l, &Sftp{})               // 值转换
	l = append(l, &AnalysisAttendance{}) // 值转换
	l = append(l, &ToLocalFile{})
	l = append(l, &LocalFileWrite{})
	l = append(l, &GenSportLevel{})
	l = append(l, &Xml2Json{})
	l = append(l, &SchemaTransfer{})
	l = append(l, &GenDocProject{})
	l = append(l, &GenSign{})
	l = append(l, &GenDoc{})
	l = append(l, &RenderDoc{})
	l = append(l, &ExtractBid{})
	l = append(l, &FixJson{})
	l = append(l, &HandlerTreeLevelOrder{})
	l = append(l, &ClientIp{})
	l = append(l, &AgentSessionService{})
	l = append(l, &AgentRunService{})
	l = append(l, &HttpProxyService{})
	l = append(l, &WebSQLService{})
	l = append(l, &WorkspaceContentSearchService{})
	l = append(l, &WorkspaceFileAccessService{})
	l = append(l, &WorkspaceFileMutationService{})
	l = append(l, &SyncLock{})
	// 执行shell
	l = append(l, &Ssh{})
	// 翻译处理器
	l = append(l, &Translate{})
	return l
}
