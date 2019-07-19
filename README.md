### deploy-kit
基于 Golang 的容器应用部署命令行工具，支持前端、后端部署。该命令行应用基于 golang 第三方 cli 包构建。

### Usage
部署两类服务：app（微服务），web（前端服务），部署微服务指定 app 命令，部署前端服务指定 web 命令。
```
NAME:
   deploy - deploy application

USAGE:
   deploy [global options] command [command options] [arguments...]

VERSION:
   v1.0

COMMANDS:
     list, ls           list all of services
     app                deploy microservice application
     web                deploy web application
     lsbranch, lsb      list the code branches of service
     upload-cdn, upcdn  upload file or directory to gcs bucket
     help, h            Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

#### 使用示例
列出当前有哪些服务可以部署：
```bash
deploy ls
```
列出 user 服务代码有哪些分支：
```bash
deploy lsb -s user
```
部署 user 微服务 master 分支到 stage 环境：
```bash
deploy app -s user -b master -e stage
```

部署前端 admin 服务 master 分支到 Stage 环境
```bash
deploy web -s admin -b master -e stage
```

### 相关配置
需要在 `$HOME/.dpcfg/` 目录下放置相关配置文件：
- conf.yaml：部署配置文件；
- service.yaml：服务列表信息；
- MicroServiceDockerfile: 微服务 Dockerfile 文件；
- kube-confg: Kubernetes 集群连接配置文件；
- _json_key: gcr 镜像仓库认证文件;
### 工具依赖包
```
go get github.com/olekukonko/tablewriter
go get github.com/urfave/cli
go get gopkg.in/yaml.v3
go get github.com/buger/goterm
go get github.com/logrusorgru/aurora
```

