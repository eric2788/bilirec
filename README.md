# Bilirec - Bilibili 直播录制工具

一个用 Go 语言编写的 Bilibili 直播录制工具，支持自动录制直播流并保存为 FLV 格式。

## 功能特性

- ✅ 手动触发录制任务，实时录制直播流
- ✅ 支持多个直播间同时录制
- ✅ 自动处理流中断和恢复
- ✅ RESTful API 管理录制任务
- ✅ 文件管理和下载功能
- ✅ 支持匿名登录或账号登录
- ✅ 自动刷新 Cookie 保持登录状态
- ✅ 低内存与低 CPU 占用，适合在资源受限设备（如树莓派）上运行

## 安装

### 使用 Docker

可以通过构建镜像或直接运行容器来启动 Bilirec。

从源码构建镜像并运行（示例）：

```bash
# 在仓库根目录构建镜像
docker build -t bilirec:latest .

# 运行容器（示例）
docker run -d \
  --name bilirec \
  -p 8080:8080 \
  -e PORT=8080 \
  -e FRONTEND_URL=http://localhost:8080 \
  -v /path/to/records:/app/records \
  -v /path/to/secrets:/app/secrets \
  bilirec:latest
```

如果你有可用的镜像仓库（例如 GHCR 或 Docker Hub），也可以直接拉取并运行镜像（示例）：

```bash
docker pull ghcr.io/eric2788/bilirec:latest
docker run -d --name bilirec -p 8080:8080 ghcr.io/eric2788/bilirec:latest
```

### 从源码构建

```bash
git clone <repository-url>
cd bilirec
go build -o bilirec main.go
```

## 配置

所有配置通过环境变量设置：

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `ANONYMOUS_LOGIN` | 是否使用匿名登录 | `false` |
| `PORT` | API 服务端口 | `8080` |
| `MAX_CONCURRENT_RECORDINGS` | 最大同时录制数 | `3` |
| `MAX_RECORDING_HOURS` | 单次录制最长时间（小时） | `5` |
| `OUTPUT_DIR` | 录制文件保存目录 | `records` |
| `SECRET_DIR` | Cookie 和 Token 保存目录 | `secrets` |
| `FRONTEND_URL` | 前端 URL（用于 CORS 与 cookie 域） | `http://localhost:8080` |
| `JWT_SECRET` | JWT 签名密钥 | `bilirec_secret` |
| `PRODUCTION_MODE` | 启用生产模式（影响 cookie 与 CORS） | `false` |

### 示例配置

```bash
export ANONYMOUS_LOGIN=false
export PORT=8080
export MAX_CONCURRENT_RECORDINGS=5
export MAX_RECORDING_HOURS=10
export OUTPUT_DIR=/path/to/records
export SECRET_DIR=/path/to/secrets
export FRONTEND_URL=http://localhost:8080
export JWT_SECRET=bilirec_secret
export PRODUCTION_MODE=false
```

## 使用方法

### 启动服务

```bash
./bilirec
```

首次启动如果未使用匿名登录，会显示二维码，使用 Bilibili 手机 APP 扫码登录。

### API 接口

> Swagger UI 会在服务器运行时于根路径 `/` 提供 — 在浏览器中打开该地址即可查看与测试 API。

#### 录制管理

- **开始录制**
  ```
  POST /record/:roomID/start
  ```

- **停止录制**
  ```
  POST /record/:roomID/stop
  ```

- **获取录制状态**
  ```
  GET /record/:roomID/status
  ```

- **获取录制统计**
  ```
  GET /record/:roomID/stats
  ```
  返回：
  ```json
  {
    "bytes_written": 1048576,
    "status": "recording",
    "start_time": 1234567890,
    "recovered_count": 0,
    "elapsed_seconds": 120
  }
  ```

- **列出所有录制任务**
  ```
  GET /record/list
  ```

#### 文件管理

- **列出文件**
  ```
  GET /files/*
  ```

- **下载文件**
  ```
  POST /files/*
  ```
  可选查询参数：`?format=mp4`（或其他支持的格式）用于在流式传输时转换文件格式。

#### 房间信息

参考 [`internal/controllers/room/room.go`](internal/controllers/room/room.go) 获取更多房间相关接口。

## 开发与调试

- **启用调试**：设置环境变量 `DEBUG=true` 启用调试模式，服务器启动时会在日志中打印一个临时十六进制令牌（hex token）。
- **pprof 性能分析**：调试模式下会在 `/debug/pprof` 挂载 pprof 以便性能分析。该路由受保护：可以在请求头 `Authorization` 中填入启动日志中显示的 hex 令牌来访问，或使用已配置的 `USERNAME` / `PASSWORD` 进行 Basic Auth 登录（若已设置）。
- **实现参考**：该逻辑位于 `internal/modules/rest/rest.go` 中（`DEBUG` 控制是否启用，令牌或用户名/密码用于授权访问）。

## 项目结构

```
.
├── main.go                          # 程序入口
├── internal/
│   ├── controllers/                 # HTTP 控制器
│   │   ├── file/                    # 文件管理
│   │   ├── record/                  # 录制管理
│   │   └── room/                    # 房间信息
│   ├── modules/                     # 核心模块
│   │   ├── bilibili/                # Bilibili API 封装
│   │   ├── config/                  # 配置管理
│   │   └── rest/                    # REST 服务
│   └── services/                    # 业务逻辑
│       ├── file/                    # 文件服务
│       ├── recorder/                # 录制服务
│       └── stream/                  # 流处理服务
├── pkg/
│   └── pool/                        # 缓冲池
└── utils/                           # 工具函数
```

## 核心实现

### 录制流程

1. 通过 [`bilibili.Client`](internal/modules/bilibili/bilibili.go) 获取直播流地址
2. 使用 [`stream.Service`](internal/services/stream/stream.go) 读取流数据
3. [`recorder.Service`](internal/services/recorder/recorder.go) 管理录制任务
4. 数据写入到 FLV 文件，保存在配置的输出目录

### 关键特性

- **自动恢复**: 当流中断时自动重连，详见 [`recorder.Service`](internal/services/recorder/recorder.go)
- **缓冲池**: 使用 [`pool.BufferPool`](pkg/pool/pool.go) 减少内存分配
- **定期刷盘**: 每 5 秒自动刷新写入缓冲，防止数据丢失
- **低资源占用**: 设计注重低内存和低 CPU 使用，适合树莓派等资源受限设备。
- **Cookie 管理**: 自动刷新 Bilibili Cookie 保持登录状态

## 依赖项

主要依赖库：

- [github.com/gofiber/fiber/v3](https://github.com/gofiber/fiber) - Web 框架
- [github.com/CuteReimu/bilibili/v2](https://github.com/CuteReimu/bilibili) - Bilibili API 客户端
- [go.uber.org/fx](https://github.com/uber-go/fx) - 依赖注入框架
- [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus) - 日志库

## 许可证

请参阅项目许可证文件。

## 贡献

欢迎提交 Issue 和 Pull Request！