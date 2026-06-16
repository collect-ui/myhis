select
  a.*,
  case
    when (
      select count(1)
      from mail_account m
      where ifnull(m.is_delete, '0') = '0'
        and ifnull(m.is_current_running, '0') = '1'
    ) > 0
      and a.order_index >= (
        select m.order_index
        from mail_account m
        where ifnull(m.is_delete, '0') = '0'
          and ifnull(m.is_current_running, '0') = '1'
        order by ifnull(m.current_run_mark_time, '') desc, m.order_index asc
        limit 1
      )
    then 1
    else 0
  end as current_run_back_group
from (require('./base.sql')) a
order by current_run_back_group asc, a.order_index asc, a.create_time desc
{{ if .pagination }}
limit {{.start}}, {{.size}}
{{ end }}
