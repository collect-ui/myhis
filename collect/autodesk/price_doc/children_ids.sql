WITH RECURSIVE
    descendants AS (

        SELECT
            price_doc_tree_id
        FROM
            price_doc_tree
        WHERE
            price_doc_tree_id = {{.price_doc_tree_id}}

        UNION ALL


        SELECT
            t.price_doc_tree_id
        FROM
            price_doc_tree t
        INNER JOIN
            descendants d
        ON
            t.parent_id = d.price_doc_tree_id
    )
SELECT price_doc_tree_id FROM descendants;