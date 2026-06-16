select *
from (select a.*, rownum rn from (require('./base.sql')) a where rownum <= {{.end}})
where rn > {{.start}}