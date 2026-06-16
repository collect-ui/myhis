将linux-service/wiki.service 拷贝进去 到/etc/systemd/system/wiki.service
调整startup.sh
#目标目录在/data/wiki
将startup脚本替换（由于服务不能用nohup启动）
```shell
/data/wiki/bin > /data/wiki/output.log 2>&1
```
```shell
# 授权
chmod +x /data/wiki/startup.sh
chmod +x /data/wiki/shutdown.sh
sudo systemctl daemon-reload
sudo systemctl enable wiki.service
sudo systemctl start wiki.service
```

