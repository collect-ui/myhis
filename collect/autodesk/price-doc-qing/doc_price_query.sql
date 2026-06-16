select b.code as tree_code,a.*
from (require(base.sql)) as a
left join price_doc_qing_tree b on a.doc_tree_id=b.price_doc_qing_tree_id
order by a.order_index,a.code