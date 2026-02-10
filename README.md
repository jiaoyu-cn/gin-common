# gin-common
基于gin的通用组件

[![image](https://img.shields.io/github/stars/jiaoyu-cn/gin-common)](https://github.com/jiaoyu-cn/gin-common/stargazers)
[![image](https://img.shields.io/github/forks/jiaoyu-cn/gin-common)](https://github.com/jiaoyu-cn/gin-common/network/members)
[![image](https://img.shields.io/github/issues/jiaoyu-cn/gin-common)](https://github.com/jiaoyu-cn/gin-common/issues)

## 安装

```shell
go get github.com/jiaoyu-cn/gin-common
```

### 日志信息查看

此扩展中已完成了基本的日志及进程等功能，需要在`router/init.go`中添加路由来访问此控制器。

```go
import (
	gcc "github.com/jiaoyu-cn/gin-common/controllers"
	gcm "github.com/jiaoyu-cn/gin-common/middlewares"
)

// log/:act 中 log可自定义，:act不可修改
Router.GET("log/:act",
		gcm.BasicAuth("authUsername", "authPassword"),
		gcc.NewLogController(&gcc.LogConfig{
			StorageLogPath: "./storage/logs",
		}).Act)
```