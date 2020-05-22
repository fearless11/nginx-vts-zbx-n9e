

### 需求

- 抓取nginx-module-vts的数据
- 计算每个servername、upstream的每秒请求数
- 发送到监控平台zabbix和夜莺

### 文档

- [nginx-moudle-vts](https://github.com/vozlt/nginx-module-vts)
- [zabbix-LLD](https://www.zabbix.com/documentation/3.4/manual/discovery/low_level_discovery)
- [zabbix自动发现](https://blog.csdn.net/yin138/article/details/83183346)
- [n9e-push-api-metrics](https://n9e.didiyun.com/zh/docs/api/data/)


### 设计

- 方案一（不合理）： 利用zabbix的LLD每次执行时发送请求抓数据,出现断点,机器负载高
- 方案二: 定期每秒抓取数据计算后写文件,zabbix的LLD每次执行读文件数据

#### 方案一 

- 源码: main.go

   ```bash
   # 编译
   go build -o nginx-vts-zbx
   mv nginx-vts-zbx ./bin

   # for zabbix LLD
   # 查看所有servernames
   bin/nginx-vts-zbx -s
   # 查看servername的req/s
   bin/nginx-vts-zbx -s -o "test.abc.com"

   # 查看所有upstreams
   bin/nginx-vts-zbx -u
   # 查看单个upstream的req/s
   bin/nginx-vts-zbx -u -o "test_upstream-10.11.100.79:9000"

   # for nightinagle
   # 一分钟上报一次, 默认endpoint为10.10.10.10
   # -t 开启nightingale,关闭zabbix LLD
   # -a 指定transfer
   # -p 上报endpoint
   bin/nginx-vts-zbx -t -a "http://10.51.1.31:5810/api/transfer/push" -p 10.10.10.111
   ```

   ```bash
   # 部署
   # for zabbix
   # zabbix client
   # cat /usr/local/zabbix/conf/zabbix_agentd/userparameter_ngx_vts.conf
   UserParameter=serverzones.discovery,/bin/nginx-vts-zbx -c "http://127.0.0.1/status/format/json" -s
   UserParameter=serverzone.reqs[*],/bin/nginx-vts-zbx -s -o $1
   UserParameter=upstreamzones.discovery,/bin/nginx-vts-zbx -u
   UserParameter=upstreamzone.reqs[*],/bin/nginx-vts-zbx -u -o $1

   # zabbix server 验证
   zabbix_get -s 10.201.0.11 -k serverzones.discovery
   zabbix_get -s 10.201.0.11 -k serverzone.reqs[s.abc.com]
   zabbix_get -s 10.201.0.11 -k upstreamzones.discovery
   zabbix_get -s 10.201.0.11 -k upstreamzone.reqs["weixin-10.21.4.157:8087"]
   ```

#### 方案二

- fetch-nginx-vts: 每分钟获取nginx-vts的数据计算后写文件
- ngx-vts-zbx: 读文件提供给zabbix和夜莺

    ```bash
    ## fetch-nginx-vts
    go build -o fetch-nginx-vts
    # 指定nginx-vts源
    ./fetch-nginx-vts -u "http://screen.abc.com/ngx_status/format/json"
    # 指定写的文件
    ./fetch-nginx-vts -f "nginx-request.json"

    ## ngx-vts-zbx
    go build -o ngx-vts-zbx
    # zabbix的LLD  s: serverzone  u: upstream
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
    ## 部署
    ## for nightinagle : 用cron每分钟写入
    * * * * * /usr/local/monitor/nginx_vts/fetch-ngx-vts >/dev/null 2>&1
    * * * * * /usr/local/monitor/nginx_vts/ngx-vts-zbx -n -p 10.11.100.230 >/dev/null 2>&1

    ## for zabbix LLD
    /etc/zabbix/zabbix_agentd.d/scripts/ngx-vts-zbx
    # 不改zabbix用户自定义参数
    ln -s  /etc/zabbix/zabbix_agentd.d/scripts/ngx-vts-zbx /bin/nginx-vts-zbx
    ```

### 知识

### nginx-moudle-vts
- install && config

  ```bash
  # check nginx install module
  nginx/sbin/nginx -t
  configure arguments: ... --add-module=../nginx-module-vts

  # install
  wget http://nginx.org/download/nginx-1.15.10.tar.gz
  tar xvf nginx-1.15.10.tar.gz -C /opt
  cd /opt/nginx-1.15.10
  git clone git://github.com/vozlt/nginx-module-vts.git
  ./configure --prefix=/usr/local/nginx --add-module=nginx-module-vts

  # config
  http {
      vhost_traffic_status_zone;
      ...
      server {
          ...
          location /status {
              vhost_traffic_status_display;
              vhost_traffic_status_display_format html;
          }
      }
  }

  # access
  http://127.0.0.1/status/format/json
  ```

### zabbix LLD

- 理解：通过自动发现配置宏 ; 通过用户参数获取自动发现的宏

- 用zabbix-agent的实现逻辑

  1. 用户自定参数UserParameter配置两个脚本
     ```bash
     xxx.discovery,script1
     yyy[*],script2 $1

     脚本一: 返回JSON数据,里面的key为zabbix使用宏
	 脚本二: 利用脚本一中宏的value作为参数,获取的详细信息
     ```
  2. 界面配置, 如ngx-vts-zbx.pdf所示

### zabbix-client

- config 

    ```bash
    # zabbix agent config
    #  -c可选 默认为 http://127.0.0.1/ngx_status/format/json

    # cat /usr/local/zabbix/conf/zabbix_agentd/userparameter_ngx_vts.conf
    UserParameter=serverzones.discovery,/bin/nginx-vts-zbx -c "http://xxx/ngx_status/format/json" -s
    UserParameter=serverzone.reqs[*],/bin/nginx-vts-zbx -s -o $1

    UserParameter=upstreamzones.discovery,/bin/nginx-vts-zbx -u
    UserParameter=upstreamzone.reqs[*],/bin/nginx-vts-zbx -u -o $1

    # restart 
    ps aux |grep zabbix |grep -v grep | awk '{print $2}' | xargs kill
    /usr/local/zabbix/sbin/zabbix_agentd -c /usr/local/zabbix/conf/zabbix_agentd.conf
    ```

- confirm 

    ```yaml
    # /usr/local/zabbix/sbin/zabbix_agentd -c /usr/local/zabbix/conf/zabbix_agentd.conf -t serverzones.discovery
    {
        "data": [
                {
                    "{#SERVERZONE}": "test.abc.com"
                },
                {
                    "{#SERVERZONE}": "all"
                }
           ]
    }
    ```

   ```yaml
   # /usr/local/zabbix/sbin/zabbix_agentd -c /usr/local/zabbix/conf/zabbix_agentd.conf -t serverzone.reqs["test.abc.com"]
   10
   ```
   ```yaml
   # /usr/local/zabbix/sbin/zabbix_agentd -c /usr/local/zabbix/conf/zabbix_agentd.conf -t upstreamzones.discovery
   {
        "data": [
            {
                "{#UPSTREAMNAME}": "test_upstream-10.11.100.17:9000"
            },
            {
                "{#UPSTREAMNAME}": "test_upstream-10.11.100.79:9000"
            }
        ]
    }
   ```
   ```yaml
   # /usr/local/zabbix/sbin/zabbix_agentd -c /usr/local/zabbix/conf/zabbix_agentd.conf -t upstreamzone.reqs["127.0.0.1:9000"]
   9
   ```
  
### trobule

- ```Value should be a JSON object.```
   ```bash
   # zabbix server执行, 返回为空数据 { "data": [] }, 重启agent正常 :)
   zabbix_get -s 10.11.9.23 -k "serverzones.discovery"
   ```
-  ```Cannot create item: item with the same key “upstreamzone.reqs[{#UPSTRAMNAME}]” already exists ```
   ```bash
   返回的key{#xxx}必须为大写{#XXX}
   {#UPSTREAMNAME} 拼写错误 {#UPSTRAMNAME} 
   ```
- ```ZBX_NOTSUPPORTED: Special characters "\, ', ", `, *, ?, [, ], {, }, ~, $, !, &, ;, (, ), <, >, |, #, @, 0x0a" are not allowed```
  ```bash
  # 替换*为all或者其他
  strings.Replace(o, "*", "all", -1)
  ```

