select
a.*
from (require(base.sql)) as a
order by
  CASE a.status
    WHEN 'pending' THEN 1
    WHEN 'success' THEN 2
    WHEN 'fail' THEN 3
    ELSE 4
  END,

a.create_time desc