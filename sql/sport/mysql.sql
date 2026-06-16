-- collect_doc definition

CREATE TABLE `collect_doc` (
  `collect_doc_id` varchar(50) NOT NULL,
  `title` varchar(200) DEFAULT NULL,
  `sub_title` varchar(200) DEFAULT NULL,
  `type` varchar(200) DEFAULT NULL,
  `parent_dir` varchar(200) DEFAULT NULL,
  `code` varchar(2550) DEFAULT NULL,
  `code_desc` varchar(2550) DEFAULT NULL,
  `order_index` int DEFAULT NULL,
  `create_time` varchar(50) DEFAULT NULL,
  `create_user` varchar(50) DEFAULT NULL,
  `is_delete` varchar(50) DEFAULT NULL,
  `code_result` text,
  PRIMARY KEY (`collect_doc_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- collect_doc_demo definition

CREATE TABLE `collect_doc_demo` (
  `doc_demo_id` varchar(50) NOT NULL,
  `collect_doc_id` varchar(50) DEFAULT NULL,
  `name` varchar(200) DEFAULT NULL,
  `code` text,
  `order_index` int DEFAULT NULL,
  `code_result` mediumtext,
  PRIMARY KEY (`doc_demo_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- collect_doc_important definition

CREATE TABLE `collect_doc_important` (
  `doc_important_id` varchar(50) NOT NULL,
  `collect_doc_id` varchar(50) DEFAULT NULL,
  `name` varchar(200) DEFAULT NULL,
  `order_index` int DEFAULT NULL,
  PRIMARY KEY (`doc_important_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- collect_doc_params definition

CREATE TABLE `collect_doc_params` (
  `doc_params_id` varchar(50) NOT NULL,
  `collect_doc_id` varchar(50) DEFAULT NULL,
  `name` varchar(200) DEFAULT NULL,
  `desc` varchar(2550) DEFAULT NULL,
  `type` varchar(200) DEFAULT NULL,
  `must` varchar(200) DEFAULT NULL,
  `order_index` int DEFAULT NULL,
  PRIMARY KEY (`doc_params_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- collect_doc_result definition

CREATE TABLE `collect_doc_result` (
  `doc_result_id` varchar(50) NOT NULL,
  `collect_doc_id` varchar(50) DEFAULT NULL,
  `name` varchar(200) DEFAULT NULL,
  `desc` varchar(2550) DEFAULT NULL,
  `type` varchar(200) DEFAULT NULL,
  `must` varchar(200) DEFAULT NULL,
  `order_index` int DEFAULT NULL,
  PRIMARY KEY (`doc_result_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- config_detail definition

CREATE TABLE `config_detail` (
  `config_detail_id` varchar(100) NOT NULL,
  `group_id` varchar(100) DEFAULT NULL,
  `name` varchar(100) DEFAULT NULL,
  `value` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`config_detail_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- config_group definition

CREATE TABLE `config_group` (
  `group_id` varchar(100) NOT NULL,
  `name` varchar(100) DEFAULT NULL,
  `description` varchar(255) DEFAULT NULL,
  `create_user` varchar(100) DEFAULT NULL,
  `create_time` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- course definition

CREATE TABLE `course` (
  `course_id` varchar(50) NOT NULL,
  `training_type` varchar(50) DEFAULT NULL,
  `teacher_id` varchar(50) DEFAULT NULL,
  `start_time` varchar(50) DEFAULT NULL,
  `create_time` varchar(50) DEFAULT NULL,
  `create_user` varchar(50) DEFAULT NULL,
  `is_delete` varchar(255) DEFAULT NULL,
  `course_stage` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`course_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- course_attendance definition

CREATE TABLE `course_attendance` (
  `course_attendance_id` varchar(50) NOT NULL,
  `course_id` varchar(50) DEFAULT NULL,
  `training_project_id` varchar(255) DEFAULT NULL,
  `student_id` varchar(50) DEFAULT NULL,
  `movies` varchar(255) DEFAULT NULL,
  `create_time` varchar(255) DEFAULT NULL,
  `spend` varchar(255) DEFAULT NULL,
  `course_type` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`course_attendance_id`),
  KEY `course_idx` (`course_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- course_grade definition

CREATE TABLE `course_grade` (
  `course_grade_id` varchar(255) NOT NULL,
  `username` varchar(50) DEFAULT NULL,
  `before_calories` varchar(255) DEFAULT NULL,
  `before_score` varchar(50) DEFAULT NULL,
  `in_calories` varchar(255) DEFAULT NULL,
  `in_score` varchar(50) DEFAULT NULL,
  `after_calories` varchar(255) DEFAULT NULL,
  `after_score` varchar(255) DEFAULT NULL,
  `course_id` varchar(255) DEFAULT NULL,
  `create_time` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`course_grade_id`),
  KEY `course_grade_course_id` (`course_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- course_video definition

CREATE TABLE `course_video` (
  `course_video_id` varchar(50) NOT NULL,
  `course_id` varchar(50) DEFAULT NULL,
  `username` varchar(50) DEFAULT NULL,
  `path` varchar(50) DEFAULT NULL,
  `create_time` varchar(50) DEFAULT NULL,
  `size` varchar(255) DEFAULT NULL,
  `course_type` varchar(255) DEFAULT NULL,
  `training_project_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`course_video_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- doc_group definition

CREATE TABLE `doc_group` (
  `doc_group_id` varchar(50) NOT NULL,
  `name` varchar(200) DEFAULT NULL,
  `type` varchar(200) DEFAULT NULL,
  `desc` varchar(200) DEFAULT NULL,
  `order_index` int DEFAULT NULL,
  `create_time` varchar(50) DEFAULT NULL,
  `create_user` varchar(50) DEFAULT NULL,
  `is_delete` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`doc_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ldap_group definition

CREATE TABLE `ldap_group` (
  `ldap_group_id` varchar(50) NOT NULL,
  `name` varchar(50) DEFAULT NULL,
  `order_index` int DEFAULT NULL,
  `has_group` varchar(100) DEFAULT NULL,
  `last_sync_time` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`ldap_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- role definition

CREATE TABLE `role` (
  `role_id` varchar(50) NOT NULL,
  `role_name` varchar(255) DEFAULT NULL,
  `order_index` int DEFAULT NULL,
  `role_code` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- role_ldap_group definition

CREATE TABLE `role_ldap_group` (
  `role_ldap_group_id` varchar(50) NOT NULL,
  `role_id` varchar(50) DEFAULT NULL,
  `ldap_group_id` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`role_ldap_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- server_env definition

CREATE TABLE `server_env` (
  `server_env_id` varchar(200) NOT NULL,
  `server_env_name` varchar(255) DEFAULT NULL,
  `notes` varchar(255) DEFAULT NULL,
  `hospital_code` varchar(255) DEFAULT NULL,
  `env_code` varchar(255) DEFAULT NULL,
  `flag_del` varchar(2) DEFAULT NULL,
  `leader` varchar(255) DEFAULT NULL,
  `sys_project_id` varchar(100) DEFAULT NULL,
  `role_auth_level` int DEFAULT NULL,
  `env_doc_url` varchar(255) DEFAULT NULL,
  `enable_ha_deploy` varchar(5) DEFAULT NULL,
  `create_time` varchar(50) DEFAULT NULL,
  `modify_time` varchar(50) DEFAULT NULL,
  `comments` varchar(225) DEFAULT NULL,
  `leader_tel` char(50) DEFAULT NULL,
  `order_id` int DEFAULT NULL,
  `server_env_code` varchar(255) DEFAULT NULL,
  `server_group_id` varchar(50) DEFAULT NULL,
  `parent_id` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`server_env_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- server_install_soft definition

CREATE TABLE `server_install_soft` (
  `service_port` varchar(8) DEFAULT NULL,
  `install_soft_id` varchar(255) NOT NULL,
  `server_id` varchar(50) NOT NULL,
  `soft_name` varchar(300) NOT NULL,
  `soft_type` varchar(50) NOT NULL,
  `soft_home` varchar(1000) NOT NULL,
  `portoffset` varchar(8) DEFAULT NULL,
  `create_time` varchar(50) DEFAULT NULL,
  `modify_time` timestamp DEFAULT NULL,
  `comments` varchar(225) DEFAULT NULL,
  `docker_container_name` varchar(100) DEFAULT NULL,
  `start_software_command` varchar(1000) DEFAULT NULL,
  `stop_software_command` varchar(1000) DEFAULT NULL,
  `restart_software_command` varchar(1000) DEFAULT NULL,
  `server_soft_logs_id` varchar(50) DEFAULT NULL,
  `server_os_user_id` int DEFAULT NULL,
  `software_live_check_command` varchar(2000) DEFAULT NULL,
  `server_soft_log_path` varchar(200) DEFAULT NULL,
  `is_main_soft` varchar(2) DEFAULT NULL,
  `is_docker` varchar(2) DEFAULT NULL,
  `pid_shell` varchar(1000) DEFAULT NULL,
  `docker_server_os_user_id` int DEFAULT NULL,
  `db_schema` varchar(50) DEFAULT NULL,
  `db_ora_sid` varchar(50) DEFAULT NULL,
  `db_url_suffix` varchar(255) DEFAULT NULL,
  `soft_version` varchar(50) DEFAULT NULL,
  `websql_cache_key` varchar(255) DEFAULT NULL,
  `ops_config_group_id` varchar(50) DEFAULT NULL,
  `install_type` varchar(50) DEFAULT NULL,
  `install_source_type` varchar(50) DEFAULT NULL,
  `install_src_path` varchar(50) DEFAULT NULL,
  `install_os_user` int DEFAULT NULL,
  `install_script` varchar(2000) DEFAULT NULL,
  `uninstall_script` varchar(2000) DEFAULT NULL,
  `soft_conf_path` varchar(200) DEFAULT NULL,
  `install_status` varchar(2) DEFAULT NULL,
  `server_soft_conf_def_id` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`install_soft_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- server_instance definition

CREATE TABLE `server_instance` (
  `server_id` varchar(255) NOT NULL,
  `server_env_id` varchar(200) NOT NULL,
  `server_ip` varchar(300) NOT NULL,
  `server_name` varchar(255) DEFAULT NULL,
  `server_port` varchar(300) NOT NULL,
  `server_user` varchar(300) DEFAULT NULL,
  `server_pwd` varchar(300) DEFAULT NULL,
  `server_statu` varchar(10) DEFAULT NULL,
  `notes` varchar(1000) DEFAULT NULL,
  `super_user` varchar(200) DEFAULT NULL,
  `super_pwd` varchar(255) DEFAULT NULL,
  `check_user` varchar(255) DEFAULT NULL,
  `check_passwd` varchar(255) DEFAULT NULL,
  `oracle_pwd` varchar(255) DEFAULT NULL,
  `server_type` varchar(255) DEFAULT NULL,
  `oracle_user` varchar(255) DEFAULT NULL,
  `server_os` varchar(200) DEFAULT NULL,
  `dblink_dump` varchar(200) DEFAULT NULL,
  `dba_pwd` varchar(100) DEFAULT NULL,
  `dba_user` varchar(100) DEFAULT NULL,
  `db_sid` varchar(200) DEFAULT NULL,
  `os_info` varchar(2000) DEFAULT NULL,
  `server_group_id` varchar(100) DEFAULT NULL,
  `dest_vm_name` varchar(50) DEFAULT NULL,
  `src_vm_name` varchar(50) DEFAULT NULL,
  `create_time` varchar(50) DEFAULT NULL,
  `modify_time` varchar(50) DEFAULT NULL,
  `comments` varchar(225) DEFAULT NULL,
  `zabbix_hostid` varchar(50) DEFAULT NULL,
  `server_hardware_id` varchar(50) DEFAULT NULL,
  `server_virtualization_type` varchar(50) DEFAULT NULL,
  `server_service_type` varchar(50) DEFAULT NULL,
  `server_os_users_id` varchar(50) DEFAULT NULL,
  `vm_hosts_id` varchar(50) DEFAULT NULL,
  `internet_address` varchar(300) DEFAULT NULL,
  `is_actived` varchar(10) DEFAULT NULL,
  `location` varchar(255) DEFAULT NULL,
  `server_os_version` varchar(200) DEFAULT NULL,
  `is_del` varchar(2) DEFAULT NULL,
  `os_log_path` varchar(255) DEFAULT NULL,
  `connect_protocol_type` varchar(50) DEFAULT NULL,
  `server_busi_name` varchar(255) DEFAULT NULL,
  `professional_name` varchar(255) DEFAULT NULL,
  `memory_size` varchar(50) DEFAULT NULL,
  `disk_total_size` varchar(50) DEFAULT NULL,
  `cpu_cores` varchar(50) DEFAULT NULL,
  `cpu_logic_count` varchar(50) DEFAULT NULL,
  `is_enable` varchar(255) DEFAULT NULL,
  `vip` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`server_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

