

### 功能

- 计算每个servername的request/s
- 计算每个upstream的request/s

### 使用

```bash
# zabbix
# 自动发现 s对应serverzone  u对应upstream
go run  main.go -f "nginx-request.json" -s
go run  main.go -f "nginx-request.json" -u

go run  main.go -f "nginx-request.json" -s -o screen.abc.com
go run  main.go -f "nginx-request.json" -u -o test-10.11.100.79:9093

# 夜莺
go run  main.go -f "nginx-request.json" -n
# 指定endpoint上报地址
go run  main.go -f "nginx-request.json" -c "http://10.51.1.31:5810/api/transfer/push" -p 10.10.10.11 -n
```



```bash
# zabbix配置
UserParameter=serverzones.discovery,/bin/ngx-vts-zbx -c "http://127.0.0.1/status/format/json" -s
UserParameter=serverzone.reqs[*],/bin/ngxx-vts-zbx -s -o $1

UserParameter=upstreamzones.discovery,/bin/ngx-vts-zbx -u
UserParameter=upstreamzone.reqs[*],/bin/ngx-vts-zbx -u -o $1
```