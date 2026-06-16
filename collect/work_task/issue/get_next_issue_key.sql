SELECT COALESCE(
               (SELECT MAX(CAST(SUBSTRING(issue_key, LENGTH({{.project_code}}) + 2) AS UNSIGNED))
                FROM work_task_issue
                WHERE project_code = {{.project_code}}
     AND issue_key LIKE {{.project_code}} || '-%')
, 0) + 1 as max_number