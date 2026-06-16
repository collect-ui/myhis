# 低代码文档缺口待办（module + handler key）

## 作用

- 跟踪低代码模块与处理器文档缺口，作为补文档待办清单与验收依据。

## 元信息

- 生成时间：2026-05-07 17:04:28 +0800
- 扫描范围：`/data/project/sport/collect/**/*.yml` + `/data/project/moongod-backend/backend_data_service/**/*.yml`
- 文档目录：`/data/project/sport/tooltip-docs/kv/module` 与 `/data/project/sport/tooltip-docs/kv/key`

## 统计

- module 已注册：30
- module 实际使用：29
- module 待补文档：20
- handler key 已注册：139
- handler key 实际使用：133
- handler key 待补文档：115

## 异常校验（需要人工确认）

### module: 使用了但未在 module_handler 注册
- [ ] config（引用2处）: /data/project/moongod-backend/backend_data_service/config/config/index.yml:64;/data/project/moongod-backend/backend_data_service/config/config/index.yml:89
- [ ] monitor（引用34处）: /data/project/moongod-backend/backend_data_service/monitor/host/index.yml:32;/data/project/moongod-backend/backend_data_service/monitor/host/index.yml:55;/data/project/moongod-backend/backend_data_service/monitor/host/index.yml:71

### handler key: 使用了但未在 data/request/result handler 注册
- [ ] jmespath（引用1处）: /data/project/sport/collect/jmespath/test/index.yml:9
- [ ] mail_account_auth_json_import（引用1处）: /data/project/sport/collect/system/mail_account/index.yml:748
- [ ] read_file（引用2处）: /data/project/sport/collect/webshell/http_console_store/index.yml:9;/data/project/sport/collect/doc/ui/index.yml:13
- [ ] to_obj（引用1处）: /data/project/moongod-backend/backend_data_service/message/monitor_msg_channel_group/index.yml:24

## Module 待补文档

| 状态 | key | 注册来源 | 使用次数 | 文档目标 |
| --- | --- | --- | ---: | --- |
| [ ] | agent_run | go:module_handler:AgentRunService::outer | 1 | tooltip-docs/kv/module/agent_run.md |
| [ ] | agent_session | go:module_handler:AgentSessionService::outer | 1 | tooltip-docs/kv/module/agent_session.md |
| [ ] | bulk_delete | py:module_handler:backend.config.service_imp.module_handler.bulk_delete:BulkDeleteService: | 1 | tooltip-docs/kv/module/bulk_delete.md |
| [ ] | bulk_update | py:module_handler:collect.service_imp.model.bulk_update:BulkUpdateService: | 40 | tooltip-docs/kv/module/bulk_update.md |
| [ ] | config | 未注册 | 2 | tooltip-docs/kv/module/config.md |
| [ ] | ftp | py:module_handler:collect.service_imp.ftp.ftp_service:FtpService: | 1 | tooltip-docs/kv/module/ftp.md |
| [ ] | http_proxy | go:module_handler:HttpProxyService::outer | 1 | tooltip-docs/kv/module/http_proxy.md |
| [ ] | jenkins | py:module_handler:collect.service_imp.jenkins.jenkins_service:JenkinsService: | 8 | tooltip-docs/kv/module/jenkins.md |
| [ ] | mail_account_auth_json_import | go:module_handler:MailAccountAuthJSONImport::outer | 0 | tooltip-docs/kv/module/mail_account_auth_json_import.md |
| [ ] | monitor | 未注册 | 34 | tooltip-docs/kv/module/monitor.md |
| [ ] | mysql_update | py:module_handler:collect.service_imp.sql.sql_update:SqlUpdateService: | 0 | tooltip-docs/kv/module/mysql_update.md |
| [ ] | read_file | go:module_handler:ReadFile::outer | 0 | tooltip-docs/kv/module/read_file.md |
| [ ] | redis_sentinel | py:module_handler:collect.service_imp.redis.redis_sentinel_service:RedisSentinelService: | 1 | tooltip-docs/kv/module/redis_sentinel.md |
| [ ] | session_op | py:module_handler:collect.service_imp.session.session_service:SessionService: | 1 | tooltip-docs/kv/module/session_op.md |
| [ ] | ssh | go:module_handler:Ssh::outer<br>py:module_handler:collect.service_imp.flow.collect_ssh:CollectSSHService: | 37 | tooltip-docs/kv/module/ssh.md |
| [ ] | workflow | py:module_handler:collect.service_imp.flow.work_flow:WorkFlowService: | 3 | tooltip-docs/kv/module/workflow.md |
| [ ] | workspace_content_search | go:module_handler:WorkspaceContentSearchService::outer | 1 | tooltip-docs/kv/module/workspace_content_search.md |

## Handler Key 待补文档

| 状态 | key | 注册来源 | 使用次数 | 文档目标 |
| --- | --- | --- | ---: | --- |
| [ ] | add_all_param | py:result_handler:collect.service_imp.result_handlers.handlers.add_all_param:AddAllParam: | 2 | tooltip-docs/kv/key/add_all_param.md |
| [ ] | agg | go:data_handler:Agg::inner | 1 | tooltip-docs/kv/key/agg.md |
| [ ] | analysis_attendance | go:data_handler:AnalysisAttendance::outer | 1 | tooltip-docs/kv/key/analysis_attendance.md |
| [ ] | analysis_ip | go:data_handler:AnalysisIp::outer | 1 | tooltip-docs/kv/key/analysis_ip.md |
| [ ] | arr2arrObj | py:request_handler:collect.service_imp.request_handlers.handlers.arr2arrobj:Arr2ArrObj: | 33 | tooltip-docs/kv/key/arr2arrObj.md |
| [ ] | arr2arrayObj | go:data_handler:Arr2arrayObj::inner | 4 | tooltip-docs/kv/key/arr2arrayObj.md |
| [ ] | arr2fieldobj | py:request_handler:collect.service_imp.request_handlers.handlers.arr2fieldobj:Arr2FieldObj: | 3 | tooltip-docs/kv/key/arr2fieldobj.md |
| [ ] | arrayValue | py:request_handler:collect.service_imp.request_handlers.handlers.arrayValue:arrayValue: | 3 | tooltip-docs/kv/key/arrayValue.md |
| [ ] | array_zip | go:data_handler:ArrayZip::inner | 0 | tooltip-docs/kv/key/array_zip.md |
| [ ] | arrsort | py:result_handler:collect.service_imp.result_handlers.handlers.array_sort:ArraySort: | 2 | tooltip-docs/kv/key/arrsort.md |
| [ ] | before_10am | py:request_handler:backend.config.service_imp.request_handlers.handlers.before_10am:Before10am: | 1 | tooltip-docs/kv/key/before_10am.md |
| [ ] | cal_score | py:request_handler:backend.config.service_imp.request_handlers.handlers.cal_score:CalScore: | 1 | tooltip-docs/kv/key/cal_score.md |
| [ ] | clear_cache | py:request_handler:collect.service_imp.request_handlers.handlers.clear_cache:ClearCache: | 8 | tooltip-docs/kv/key/clear_cache.md |
| [ ] | client_ip | go:data_handler:ClientIp::outer | 1 | tooltip-docs/kv/key/client_ip.md |
| [ ] | combine | py:result_handler:collect.service_imp.result_handlers.handlers.combine:Combine: | 1 | tooltip-docs/kv/key/combine.md |
| [ ] | data2excel | go:data_handler:Data2Excel::inner | 8 | tooltip-docs/kv/key/data2excel.md |
| [ ] | distinct | py:request_handler:collect.service_imp.request_handlers.handlers.distinct:Distinct: | 14 | tooltip-docs/kv/key/distinct.md |
| [ ] | echo_all_param | py:request_handler:backend.config.service_imp.request_handlers.handlers.echo_param:EchoParam: | 2 | tooltip-docs/kv/key/echo_all_param.md |
| [ ] | excel2data | go:data_handler:Excel2Data::inner | 6 | tooltip-docs/kv/key/excel2data.md |
| [ ] | excel_data | py:request_handler:collect.service_imp.request_handlers.handlers.excelData:excelData: | 12 | tooltip-docs/kv/key/excel_data.md |
| [ ] | extract_bid | go:data_handler:ExtractBid::outer | 1 | tooltip-docs/kv/key/extract_bid.md |
| [ ] | field2array | go:data_handler:Field2Array::inner<br>py:request_handler:collect.service_imp.request_handlers.handlers.field2array:Field2Array: | 31 | tooltip-docs/kv/key/field2array.md |
| [ ] | field2json | py:result_handler:collect.service_imp.result_handlers.handlers.field2json:Field2JSONService: | 7 | tooltip-docs/kv/key/field2json.md |
| [ ] | file2data | py:request_handler:collect.service_imp.request_handlers.handlers.file2data:File2data: | 5 | tooltip-docs/kv/key/file2data.md |
| [ ] | file2json | go:data_handler:File2Json::inner | 4 | tooltip-docs/kv/key/file2json.md |
| [ ] | file2result | go:data_handler:File2Result::inner | 9 | tooltip-docs/kv/key/file2result.md |
| [ ] | file2str | go:data_handler:File2Str::inner | 2 | tooltip-docs/kv/key/file2str.md |
| [ ] | file_move | go:data_handler:FileMove::inner | 0 | tooltip-docs/kv/key/file_move.md |
| [ ] | files2result | go:data_handler:Files2Result::inner | 1 | tooltip-docs/kv/key/files2result.md |
| [ ] | fix_json | go:data_handler:FixJson::outer | 2 | tooltip-docs/kv/key/fix_json.md |
| [ ] | friday_handler | py:request_handler:backend.config.service_imp.request_handlers.handlers.friday_handler:FridayHandler: | 1 | tooltip-docs/kv/key/friday_handler.md |
| [ ] | gen_doc | go:data_handler:GenDoc::outer | 1 | tooltip-docs/kv/key/gen_doc.md |
| [ ] | gen_doc_project | go:data_handler:GenDocProject::outer | 1 | tooltip-docs/kv/key/gen_doc_project.md |
| [ ] | gen_file | py:request_handler:collect.service_imp.request_handlers.handlers.gen_file:GenFile: | 7 | tooltip-docs/kv/key/gen_file.md |
| [ ] | gen_sign | go:data_handler:GenSign::outer | 7 | tooltip-docs/kv/key/gen_sign.md |
| [ ] | gen_sport_level | go:data_handler:GenSportLevel::outer | 1 | tooltip-docs/kv/key/gen_sport_level.md |
| [ ] | get_array_obj | py:request_handler:collect.service_imp.request_handlers.handlers.get_array_obj:GetArrayObj: | 1 | tooltip-docs/kv/key/get_array_obj.md |
| [ ] | get_event_list | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_event_list:GetEventList: | 1 | tooltip-docs/kv/key/get_event_list.md |
| [ ] | get_half_year | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_half_year:GetHalfYear: | 1 | tooltip-docs/kv/key/get_half_year.md |
| [ ] | get_new_version | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_new_version:GetNewVersion: | 1 | tooltip-docs/kv/key/get_new_version.md |
| [ ] | get_pdm_data | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_pdm_data:GetPdmData: | 1 | tooltip-docs/kv/key/get_pdm_data.md |
| [ ] | get_pw_suffix | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_pw_suffix:GetPwWeek: | 4 | tooltip-docs/kv/key/get_pw_suffix.md |
| [ ] | get_py | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_py:GetPy: | 1 | tooltip-docs/kv/key/get_py.md |
| [ ] | get_session_data | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_session_data:GetSessionData: | 9 | tooltip-docs/kv/key/get_session_data.md |
| [ ] | get_success_issue_list | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_success_issue_list:GetSuccessIssueList: | 1 | tooltip-docs/kv/key/get_success_issue_list.md |
| [ ] | get_wb | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_wb:GetWb: | 1 | tooltip-docs/kv/key/get_wb.md |
| [ ] | get_work_days | py:request_handler:backend.config.service_imp.request_handlers.handlers.get_work_days:GetWorkDays: | 1 | tooltip-docs/kv/key/get_work_days.md |
| [ ] | group_by | go:data_handler:GroupBy::inner<br>py:request_handler:collect.service_imp.request_handlers.handlers.group_by:GroupBy: | 27 | tooltip-docs/kv/key/group_by.md |
| [ ] | handler_cache | go:data_handler:HandlerCache::inner | 1 | tooltip-docs/kv/key/handler_cache.md |
| [ ] | handler_delay_param | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_delay_param:HandlerDelayParam: | 1 | tooltip-docs/kv/key/handler_delay_param.md |
| [ ] | handler_function_data | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_function_data:HandlerFunctionData: | 1 | tooltip-docs/kv/key/handler_function_data.md |
| [ ] | handler_issue_img | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_issue_img:HandlerIssueImg: | 0 | tooltip-docs/kv/key/handler_issue_img.md |
| [ ] | handler_n_count | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_n_count:HandlerNCount: | 1 | tooltip-docs/kv/key/handler_n_count.md |
| [ ] | handler_password | go:data_handler:HandlerPassword::outer | 13 | tooltip-docs/kv/key/handler_password.md |
| [ ] | handler_paste_data | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_paste_data:HandlerPasteData: | 1 | tooltip-docs/kv/key/handler_paste_data.md |
| [ ] | handler_planning_warning_type | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_planning_warning_type:HandlerPlanningWarningType: | 1 | tooltip-docs/kv/key/handler_planning_warning_type.md |
| [ ] | handler_project_sys_code | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_project_sys_code:HandlerProjectSysCode: | 1 | tooltip-docs/kv/key/handler_project_sys_code.md |
| [ ] | handler_to_attachment | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_to_attachment:HandlerToAttachment: | 1 | tooltip-docs/kv/key/handler_to_attachment.md |
| [ ] | handler_to_url_attachment | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_to_url_attachment:HandlerToURLAttachment: | 1 | tooltip-docs/kv/key/handler_to_url_attachment.md |
| [ ] | handler_todo | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_todo:HandlerTodo: | 1 | tooltip-docs/kv/key/handler_todo.md |
| [ ] | handler_tree_level_order | go:data_handler:HandlerTreeLevelOrder::outer | 2 | tooltip-docs/kv/key/handler_tree_level_order.md |
| [ ] | handler_week_info | py:request_handler:backend.config.service_imp.request_handlers.handlers.handler_week_info:HandlerWeekInfo: | 1 | tooltip-docs/kv/key/handler_week_info.md |
| [ ] | ignore_data | go:data_handler:IgnoreData::inner<br>py:request_handler:collect.service_imp.request_handlers.handlers.ignore_data:IgnoreData: | 2 | tooltip-docs/kv/key/ignore_data.md |
| [ ] | ignore_from_reg_list | py:result_handler:collect.service_imp.result_handlers.handlers.ignore_from_reg_list:IgnoreFromRegList: | 3 | tooltip-docs/kv/key/ignore_from_reg_list.md |
| [ ] | jmespath | 未注册 | 1 | tooltip-docs/kv/key/jmespath.md |
| [ ] | level_2 | py:result_handler:collect.service_imp.result_handlers.handlers.level_2:Level2: | 3 | tooltip-docs/kv/key/level_2.md |
| [ ] | local_file_write | go:data_handler:LocalFileWrite::outer | 2 | tooltip-docs/kv/key/local_file_write.md |
| [ ] | mail_account_auth_json_import | 未注册 | 1 | tooltip-docs/kv/key/mail_account_auth_json_import.md |
| [ ] | max | py:request_handler:collect.service_imp.request_handlers.handlers.max:Max: | 1 | tooltip-docs/kv/key/max.md |
| [ ] | move_up_down | py:request_handler:backend.config.service_imp.request_handlers.handlers.move_up_down:MoveUpDown: | 2 | tooltip-docs/kv/key/move_up_down.md |
| [ ] | mul_arr | py:request_handler:collect.service_imp.request_handlers.handlers.mul_arr:MulArr: | 17 | tooltip-docs/kv/key/mul_arr.md |
| [ ] | multi_arr | go:data_handler:MultiArr::outer | 1 | tooltip-docs/kv/key/multi_arr.md |
| [ ] | obj_in_arr | py:request_handler:collect.service_imp.request_handlers.handlers.obj_in_array:ObjInArray: | 2 | tooltip-docs/kv/key/obj_in_arr.md |
| [ ] | order_by | go:data_handler:OrderBy::inner | 1 | tooltip-docs/kv/key/order_by.md |
| [ ] | param_key2arr | go:data_handler:ParamKey2Arr::outer | 1 | tooltip-docs/kv/key/param_key2arr.md |
| [ ] | pending_review_requirements | py:request_handler:backend.config.service_imp.request_handlers.handlers.pending_review_requirements:PendingReviewRequirements: | 1 | tooltip-docs/kv/key/pending_review_requirements.md |
| [ ] | pending_ui_link_requirements | py:request_handler:backend.config.service_imp.request_handlers.handlers.pending_ui_link_requirements:PendingUiLinkRequirements: | 1 | tooltip-docs/kv/key/pending_ui_link_requirements.md |
| [ ] | prevent_duplication | go:data_handler:PreventDuplication::inner | 0 | tooltip-docs/kv/key/prevent_duplication.md |
| [ ] | read_file | 未注册 | 2 | tooltip-docs/kv/key/read_file.md |
| [ ] | reg2list | py:request_handler:collect.service_imp.request_handlers.handlers.reg2list:Reg2List: | 1 | tooltip-docs/kv/key/reg2list.md |
| [ ] | reg_field | py:request_handler:collect.service_imp.request_handlers.handlers.reg_field:RegField: | 0 | tooltip-docs/kv/key/reg_field.md |
| [ ] | rename_field | go:data_handler:RenameField::outer | 1 | tooltip-docs/kv/key/rename_field.md |
| [ ] | render_doc | go:data_handler:RenderDoc::outer<br>py:request_handler:backend.config.service_imp.request_handlers.handlers.render_doc:RenderDoc: | 3 | tooltip-docs/kv/key/render_doc.md |
| [ ] | render_doc_tpl | py:request_handler:backend.config.service_imp.request_handlers.handlers.render_doc_tpl:RenderDocTpl: | 2 | tooltip-docs/kv/key/render_doc_tpl.md |
| [ ] | result2excel | py:result_handler:collect.service_imp.result_handlers.handlers.result2excel:Result2Excel: | 21 | tooltip-docs/kv/key/result2excel.md |
| [ ] | result_msg | py:result_handler:collect.service_imp.result_handlers.handlers.result_msg:ResultMsg: | 25 | tooltip-docs/kv/key/result_msg.md |
| [ ] | row2col | py:result_handler:collect.service_imp.result_handlers.handlers.row2col:Row2Col: | 1 | tooltip-docs/kv/key/row2col.md |
| [ ] | runpy | py:request_handler:backend.config.service_imp.request_handlers.handlers.runpy:PyHandler: | 4 | tooltip-docs/kv/key/runpy.md |
| [ ] | runtime_codetext | py:result_handler:backend.config.service_imp.result_handlers.handlers.runtime_codetext:RuntimeCodeText: | 3 | tooltip-docs/kv/key/runtime_codetext.md |
| [ ] | runtime_python_code | py:result_handler:backend.config.service_imp.result_handlers.handlers.runtime_pythoncode:RuntimePythonCode: | 4 | tooltip-docs/kv/key/runtime_python_code.md |
| [ ] | schema_transfer | go:data_handler:SchemaTransfer::outer | 10 | tooltip-docs/kv/key/schema_transfer.md |
| [ ] | session_add | go:data_handler:SessionAdd::inner | 1 | tooltip-docs/kv/key/session_add.md |
| [ ] | session_get | go:data_handler:SessionGet::inner | 1 | tooltip-docs/kv/key/session_get.md |
| [ ] | session_remove | go:data_handler:SessionRemove::inner | 1 | tooltip-docs/kv/key/session_remove.md |
| [ ] | sftp | go:data_handler:Sftp::outer | 0 | tooltip-docs/kv/key/sftp.md |
| [ ] | shell | go:data_handler:Shell::outer | 0 | tooltip-docs/kv/key/shell.md |
| [ ] | shell_term | go:data_handler:ShellTerm::outer | 0 | tooltip-docs/kv/key/shell_term.md |
| [ ] | split | py:result_handler:collect.service_imp.result_handlers.handlers.split:Split: | 2 | tooltip-docs/kv/key/split.md |
| [ ] | str2file | go:data_handler:Str2File::inner | 1 | tooltip-docs/kv/key/str2file.md |
| [ ] | str2json | go:data_handler:Str2Json::inner | 2 | tooltip-docs/kv/key/str2json.md |
| [ ] | template_log_msg | py:request_handler:collect.service_imp.request_handlers.handlers.template_log_msg:TemplateLogMsg: | 1 | tooltip-docs/kv/key/template_log_msg.md |
| [ ] | to_code_table_by_script | py:request_handler:backend.config.service_imp.request_handlers.handlers.to_code_table_by_script:ToCodeTableByScript: | 1 | tooltip-docs/kv/key/to_code_table_by_script.md |
| [ ] | to_iconfont_data | py:request_handler:backend.config.service_imp.request_handlers.handlers.to_iconfont_data:ToIconFontData: | 1 | tooltip-docs/kv/key/to_iconfont_data.md |
| [ ] | to_list | go:data_handler:ToList::inner | 6 | tooltip-docs/kv/key/to_list.md |
| [ ] | to_local_file | go:data_handler:ToLocalFile::outer | 2 | tooltip-docs/kv/key/to_local_file.md |
| [ ] | to_nacos_addr_data | py:request_handler:backend.config.service_imp.request_handlers.handlers.to_nacos_addr_data:ToNacosAddrData: | 2 | tooltip-docs/kv/key/to_nacos_addr_data.md |
| [ ] | to_nginx_data | py:result_handler:backend.config.service_imp.result_handlers.handlers.to_nginx_data:ToNginxData: | 2 | tooltip-docs/kv/key/to_nginx_data.md |
| [ ] | to_obj | 未注册 | 1 | tooltip-docs/kv/key/to_obj.md |
| [ ] | to_sys_param_by_script | py:request_handler:backend.config.service_imp.request_handlers.handlers.to_sys_param_by_script:ToSysParamByScript: | 1 | tooltip-docs/kv/key/to_sys_param_by_script.md |
| [ ] | to_tree | go:data_handler:ToTree::inner<br>py:result_handler:collect.service_imp.result_handlers.handlers.to_tree:ToTree: | 25 | tooltip-docs/kv/key/to_tree.md |
| [ ] | tree_node_name | py:result_handler:collect.service_imp.result_handlers.handlers.tree_node_name:TreeNodeName: | 1 | tooltip-docs/kv/key/tree_node_name.md |
| [ ] | update_order | go:data_handler:UpdateOrder::inner | 1 | tooltip-docs/kv/key/update_order.md |
| [ ] | value_arr | py:result_handler:collect.service_imp.result_handlers.handlers.value_arr:ValueArr: | 27 | tooltip-docs/kv/key/value_arr.md |
| [ ] | value_transfer | go:data_handler:ValueTransfer::outer | 1 | tooltip-docs/kv/key/value_transfer.md |
| [ ] | xml2json | go:data_handler:Xml2Json::outer | 1 | tooltip-docs/kv/key/xml2json.md |
