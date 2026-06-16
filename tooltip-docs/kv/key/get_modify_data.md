# key=get_modify_data

## 作用

- 我经常遇到业务级别，需要记录某个字段是谁改的，改之前是什么，改之后是什么
- 但是利用用数据库的binlog日志展示不做不到，比如要记录版本号是谁改的，改之前是什么，改之后是什么，然后还原此条
- 一个两个情况到没有什么，主要很多地方有这样的需求，比如保存一个全量列表，之前的搞法就是直接全部删除，然后全部添加，后面发现效率不行，毕竟数量多了，删除和添加总会要占用一定时间，...
- 本身只改了一点点数据，却触发整个表的删除与新增 有了这个对比工具，我们可以对比列表的差异部分，然后数据进行，哪些删除，哪些新增，哪些修改，理应如此

## 常见用途

- 我经常遇到业务级别，需要记录某个字段是谁改的，改之前是什么，改之后是什么
- 但是利用用数据库的binlog日志展示不做不到，比如要记录版本号是谁改的，改之前是什么，改之后是什么，然后还原此条

## 执行阶段（低代码视角）

- 可用于参数处理或结果处理阶段：具体在 `handler_params`（模块前）或 `result_handler`（模块后）生效。

## 怎么用

### 参数
| 参数 | 类型 | 必须 | 说明 |
| --- | --- | --- | --- |
| handler_params[get_modify_data] | string | 是 | 处理器中key |
| modify_config | string | 是 | 规则路径，在主体中 |
| modify_config.op_field_transfer | json |  | 操作的转换字典，change_list 有些固定字段，比如name表示名称，如何有冲突可以在此处修改 |
| modify_config.fields[] | array | 是 | 数据规则 |
| modify_config.fields[rule] | string | 是 | 规则名称。compare_field_value简单字段，simple_array_value简单数组，array_obj_value数组对象 |
| modify_config.fields[field] | string | 是 | 对比的字段名称 |
| modify_config.fields[name] | string | 是 | 对比字段的中文名称 |
| modify_config.fields[left] | string |  | 左边取对象字段，如果没有就从参数中取 |
| modify_config.fields[right] | string | 是 | 右边取对象字段 |
| modify_config.fields[operation] | string |  | 仅仅对简单字段修改有效，对操作名称进行重新调整，一般是add，modify，remove |
| modify_config.fields[append_right_fields] | array |  | 拼接右边的字段，*表示所有字段 |
| modify_config.fields[append_left_fields] | array |  | 拼接左边的字段，一般数组对象修改，左右2边都有的情况，优先左边的字段，配置op_field_transfer,右边已经存在字段护理 |
| modify_config.fields[left_field] | string |  | 当左右2边数据对比字段不对等到时候，左边定位数据需要字段a，右边要取字段b。主要用于定位数据，左边的字段 |
| modify_config.fields[right_field] | string |  | 当左右2边数据对比字段不对等到时候，左边定位数据需要字段a，右边要取字段b。主要用于定位数据，右边的字段 |
| modify_config.fields[left_value_field] | string |  | 主要用于数组对象，对比行记录里面其他字段值，左边的取值 |
| modify_config.fields[right_value_field] | string |  | 主要用于数组对象，对比行记录里面其他字段值，右边的取值 |
| modify_config.fields[with_add_remove] | string |  | 主要用于数组对象，是生成添加修改记录，一个字段数组对象中只要有一个 |
| modify_config.fields[save_original] | boolean |  | 是否保留原始值，取值为value，主要用户转换，看下面transfer 和service |
| modify_config.fields[value_list_field] | string |  | 将左右2边的值取出来，从另外一个目标服务查询转换一下，为下面service提供取值列表字段，一般current_value_list |
| modify_config.fields[target_transfer_key] | string |  | 目标服务的取值关键字段，定位行数据，根据编码定位行 |
| modify_config.fields[target_transfer_value] | string |  | 转换后的取值字段，根据编码换值 |
| modify_config.fields[service] | json |  | 转换调用的服务，如果current_value_list 有冲突，可以改value_list_field |

## 示例

```yml
modify_config: doc_modify.json
handler_params:
  - key: get_modify_data
    save_field: change_list
```

## 注意事项

- 简单字段比对修改
- 简单数组字段比对新增与删除
- 数据对象比对新增修改删除
- 支持结果数据转换
- 支持操作名称修改

## 元信息

- 来源：`服务文档 -> 参数处理 -> 1.get_modify_data / 比对数据`
- 页面标题：`get_modify_data(比对数据)`
