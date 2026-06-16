select count(*) as count
from user_account  a
where
ifnull(a.status,'1')!='0' and a.is_delete = '0'
require('./base_where.common')