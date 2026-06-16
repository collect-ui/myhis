# service

## 作用

- module：empty
- 服务就是对外提供一个http接口，或者对内提供基础方法，满足别人需求
- 我们常常编写一个接口，一般是要求服务器做一件事情，比如查询一个用户列表，更新一个用户信息，删除一条记录
- 我们管服务器每做一个事情叫做服务

## 常见用途

- module：empty
- 服务就是对外提供一个http接口，或者对内提供基础方法，满足别人需求

## 执行阶段（低代码视角）

- 服务定义阶段：决定服务如何被路由、执行与对外暴露。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| module | string | 是 | module 表示模块，表示你将要运行的主体模块。module是服务核心关键字 |
| key | string |  | 服务名 |

## 示例

```yml
service:
  - key: user_list_import
    module: empty
    http: true
    excel_config: "./user_list2excel.json"
    handler_params:
      - key: excel2data
        save_field: user_list
      - key: ignore_data
        foreach: "[user_list]"
        params: "params"
        fields:
          - name: "user_id 为空的数据"
            template: "{{ if .user_id }}false{{else}}true{{end}}"
      - key: service2field
        enable: "{{gt (len .user_list) 0 }}"
        service:
          service: hrm.bulk_update_user
          user_list: "[user_list]"
      - key: params2result
        fields:
          - from: "[user_list]"
            to: user_list
  - key: user_list_download
    module: empty
    http: true
    excel_config: "./user_list2excel.json"
    params:
      excel_path:
        template: './template/{{current_date_format "20220202"}}/user_{{  replace (sub_str current_date_time -8 0) ":" ""}}_{{sub_str uuid -8 0}}.xlsx'
      response_name:
        default: "用户列表.xlsx"
    handler_params:
      - key: service2field
        service:
          service: hrm.user_list
        append_param: true
        save_field: user_list
      - key: data2excel
        path: "[excel_path]"
      - key: file2result
        path: "[excel_path]"
        result_name: "[response_name]"

  - key: empty_test
    module: empty
    http: true
    handler_params:
      - key: service2field
        service:
          service: hrm.user_list
          username: "[username]"
        save_field: user_info
        template: "{{gt (len .user_info) 0}}"
        err_msg: "用户名【{{.username}}】已经存在"
    result_handler:
      - key: service2field
        service:
          service: hrm.user_list
          username: "[username]"
        save_field: user_info
        template: "{{eq (len .user_info) 0}}"
        err_msg: "用户名【{{.username}}】已经存在"
  - key: bulk_update_user
    module: bulk_upsert
    log: true
    http: true
    params:
      user_list:
        check:
          template: "{{must .user_list}}"
          err_msg: 用户列表不能为空
      fields:
        default: ["*"]
    handler_params:
      - key: update_array
        foreach: "[user_list]"
        item: item
        fields:
          - field: password
# ...(示例过长，已截断)
```

## 注意事项

- 二级服务分类，service 一般 xxx.xx。第一级表示项目,第二级表示具体服务
- 服务名定义，由入口文件services下的key+叶子目录下key2个拼接组成。比如hrm.user_list,是最上层collect/service.yml的key=hrm,user_list 是hrm/user/index.yml下面的key=user_list服务 。定义的时候就得保证唯一
- collect/service.yml 定义了所有模块、处理器。就是为了写配置的时候方便看一眼，有印象

## 元信息

- 来源：`服务文档 -> 如何编写服务 -> 1.什么是服务service / 服务描述`
- 页面标题：`什么是服务service(服务描述)`
