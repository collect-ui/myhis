select
a.img_id, a.img_dir_id, a.name,
a.path,
{{ if .with_content }}
a.content,
{{  end }}
a.create_time,
a.type,
a.create_user


from img a
where 1=1
{{ if .img_id}}
and a.img_id = {{.img_id}}
{{ end }}
{{ if .img_dir_id }}
and a.img_dir_id = {{.img_dir_id}}
{{ end }}
order by a.create_time desc
{{ if  .pagination  }}
limit {{.start}} , {{.size}}
{{ end }}