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
  -v /path/to/database:/app/database \
  # 可选：启用 CloudConvert（替换为你的 API key）
  -e CLOUDCONVERT_API_KEY=your_api_key \
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
| `MAX_RECOVERY_ATTEMPTS` | 单次录制的最大重连尝试次数 | `5` |
| `OUTPUT_DIR` | 录制文件保存目录 | `records` |
| `SECRET_DIR` | Cookie 和 Token 保存目录 | `secrets` |
| `CONVERT_FLV_TO_MP4` | 在下载时是否将 FLV 转为 MP4 | `false` |
| `DELETE_FLV_AFTER_CONVERT` | 转换后是否删除原始 FLV 文件 | `false` |
| `BACKEND_HOST` | 后端主机（用于生成Cookie域名） | `localhost:8080` |
| `FRONTEND_URL` | 前端 URL（用于 CORS 与 cookie 域） | `http://localhost:8080` |
| `USERNAME` | 可选：启用用户名/密码认证时的用户名 | (未设置) |
| `PASSWORD` | 可选：启用用户名/密码认证时的密码 | (未设置) |
| `JWT_SECRET` | JWT 签名密钥 | `bilirec_secret` |
| `DEBUG` | 启用调试模式（会开启 pprof 和临时 hex token） | `false` |
| `PRODUCTION_MODE` | 启用生产模式（影响 cookie 与 CORS） | `false` |
| `DATABASE_DIR` | 本地数据库目录（bbolt，用于持久化转换任务等） | `database` |
| `CLOUDCONVERT_THRESHOLD` | 使用 CloudConvert 的文件大小阈值（字节） | `1073741824` (1 GB) |
| `CLOUDCONVERT_API_KEY` | 可选：CloudConvert API Key（为空则禁用 CloudConvert） | (未设置) |

### 示例配置

```bash
export ANONYMOUS_LOGIN=false
export PORT=8080
export MAX_CONCURRENT_RECORDINGS=5
export MAX_RECORDING_HOURS=10
export MAX_RECOVERY_ATTEMPTS=5
export OUTPUT_DIR=/path/to/records
export SECRET_DIR=/path/to/secrets
export DATABASE_DIR=/path/to/database
export CONVERT_FLV_TO_MP4=false
export DELETE_FLV_AFTER_CONVERT=false
# 可选：CloudConvert（如果启用会对大文件使用云端转换）
export CLOUDCONVERT_THRESHOLD=1073741824
export CLOUDCONVERT_API_KEY=
export BACKEND_HOST=localhost:8080
export FRONTEND_URL=http://localhost:8080
export JWT_SECRET=bilirec_secret
export DEBUG=false
# 可选：启用 REST API 认证
export USERNAME=admin
export PASSWORD=changeme
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

#### 认证

如果设置了 `USERNAME` 与 `PASSWORD`，REST API 会启用基于 JWT 的认证（登录会在 cookie 中设置 `jwtToken`）。使用：

```http
POST /login
Content-Type: application/json

{ "user": "<username>", "pass": "<password>" }
```

登录成功后会在响应中设置 JWT cookie（键名 `jwtToken`），随后对需要认证的接口请携带该 cookie。若未设置用户名/密码，API 默认为公开访问。

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
  GET /files/browse/*
  ```

- **下载文件**
  ```
  GET /files/download/*
  ```
  下载接口直接返回存储的文件，**不再支持**通过查询参数进行即时格式转换（此前的 `?format=...` 参数已移除）。
  若要将录制的 FLV 转为 MP4，请启用 `CONVERT_FLV_TO_MP4`：在录制完成时，recorder 会将 FLV 文件加入转换队列，由后台任务异步转换为 MP4（转换行为受 `DELETE_FLV_AFTER_CONVERT` 控制）。
  当同时设置了 `CLOUDCONVERT_API_KEY` 且文件大小 >= `CLOUDCONVERT_THRESHOLD`（默认 1 GB）时，系统会优先使用 CloudConvert（异步任务，可通过 `/convert/tasks` 查询转换状态）；否则由本地 ffmpeg 后台任务处理。

- **删除多个文件**
  ```
  DELETE /files/batch
  ```
  请求体：JSON 数组，包含要删除的相对文件路径，示例：

  ```json
  ["room123/20250101.flv", "room456/20250102.flv"]
  ```

- **删除目录**
  ```
  DELETE /files/{path}
  ```

#### 转换任务

- **列出进行中的转换任务**（需要认证，返回任务信息）
  ```
  GET /convert/tasks
  ```

- **取消转换任务**（需要认证）
  ```
  DELETE /convert/tasks/:task_id
  ```
  返回 `204 No Content` 表示取消成功，若任务不存在返回 `404`。

#### 房间信息

参考 [`internal/controllers/room/room.go`](internal/controllers/room/room.go) 获取更多房间相关接口。

## 开发与调试

- **启用调试**：设置环境变量 `DEBUG=true` 启用调试模式，服务器启动时会在日志中打印一个临时十六进制令牌（hex token）。
- **pprof 性能分析**：调试模式下会在 `/debug/pprof` 挂载 pprof 以便性能分析。该路由受保护：可以在请求头 `Authorization` 中填入启动日志中显示的 hex 令牌来访问，或使用已配置的 `USERNAME` / `PASSWORD` 进行 Basic Auth 登录（若已设置）。
- **实现参考**：该逻辑位于 `internal/modules/rest/rest.go` 中（`DEBUG` 控制是否启用，令牌或用户名/密码用于授权访问）。

## 项目结构

```
.
├── .github/                          # CI / workflows (maintenance)
├── Dockerfile                        # Docker 镜像构建文件
├── internal/
│   ├── controllers/                  # HTTP 控制器
│   │   ├── convert/                  # 转换任务管理（/convert）
│   │   ├── file/                     # 文件管理
│   │   ├── record/                   # 录制管理
│   │   └── room/                     # 房间信息
│   ├── modules/                      # 核心模块
│   │   ├── bilibili/                 # Bilibili API 封装
│   │   ├── config/                   # 配置管理
│   │   └── rest/                     # REST 服务（包含 Swagger / pprof）
│   └── services/                     # 业务逻辑
│       ├── convert/                  # 转换服务（本地 ffmpeg 或 CloudConvert）
│       ├── file/                     # 文件服务
│       ├── recorder/                 # 录制服务
│       └── stream/                   # 流处理服务
├── pkg/                              # 底层库与工具
│   ├── cloudconvert/                 # CloudConvert client wrapper (optional)
│   ├── ds/                           # 数据结构（如 FLV Tag）
│   ├── flv/                          # FLV 读写与处理
│   ├── pipeline/                     # 流处理管道  
|   ├── monitor/                      # 监控与指标
│   └── pool/                         # 内存池
├── utils/                            # 工具函数
├── LICENSE
└── README.md
```

## 核心实现

### 录制流程

1. 通过 [`bilibili.Client`](internal/modules/bilibili/bilibili.go) 获取直播流地址
2. 使用 [`stream.Service`](internal/services/stream/stream.go) 读取流数据
3. [`recorder.Service`](internal/services/recorder/recorder.go) 管理录制任务（自动重连与恢复）
4. 数据写入到 FLV 文件，保存在配置的输出目录；如果启用了 `CONVERT_FLV_TO_MP4`，录制完成时会自动将 FLV 文件加入转换队列并由后台任务异步转换为 MP4（转换行为受 `DELETE_FLV_AFTER_CONVERT` 控制，转换任务可通过 `/convert/tasks` 查询）。

### 关键特性

- **自动恢复**: 当流中断时自动重连，详见 [`recorder.Service`](internal/services/recorder/recorder.go)
- **缓冲池**: 使用 [`pool.BufferPool`](pkg/pool/pool.go) 减少内存分配
- **定期刷盘**: 每 5 秒自动刷新写入缓冲，防止数据丢失
- **低资源占用**: 设计注重低内存和低 CPU 使用，适合树莓派等资源受限设备
- **文件管理**: 支持列出、预览、下载（可转换格式）、批量删除文件及删除目录，详见 `internal/controllers/file/file.go`
- **自动转换**: 如果启用 `CONVERT_FLV_TO_MP4`，录制完成时会自动将 FLV 转为 MP4；可通过 `DELETE_FLV_AFTER_CONVERT` 控制是否删除原始 FLV
- **实时修复（Realtime Fixer）**: 在流式写入场景下逐个修复 FLV Tag 的时间戳并输出，包含重复 Tag 去重（可查询去重统计），并通过内存池、去重缓存与周期清理来保持低延迟与低内存占用，适合边录制边推送或实时下载的场景。
- **REST API 文档**: Swagger UI 在根路径 `/` 提供（由 `swag` 生成，参见 `internal/modules/rest`）
- **认证与调试**: 可选用户名/密码登录（设置 `USERNAME` 和 `PASSWORD`）启用 JWT 认证；调试模式下可通过 `/debug/pprof` 访问 pprof（受临时 token 或基本 auth 保护）
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