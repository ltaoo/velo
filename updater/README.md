# Auto-Update System (自动更新系统)

## 概述

`updater` 包提供了一个完整的应用程序自动更新解决方案，支持从多种更新源（GitHub Releases、自定义 HTTP 服务器）检测、下载和应用更新。该系统具有跨平台支持、安全验证、断点续传、自动回滚等企业级特性。

## 核心功能

### ✨ 主要特性

- **多源支持**：支持 GitHub Releases 和自定义 HTTP 服务器作为更新源
- **智能故障转移**：按优先级自动切换更新源，确保高可用性
- **安全验证**：HTTPS 强制、SHA256 校验和验证、可执行文件完整性检查
- **断点续传**：网络中断后自动恢复下载，节省带宽
- **自动回滚**：更新失败时自动恢复到原始版本
- **跨平台**：支持 Windows、Linux、macOS（包括 Apple Silicon）
- **进度报告**：实时下载和安装进度回调
- **灵活配置**：YAML 配置文件，支持多种更新策略
- **结构化日志**：JSON 格式日志，便于分析和监控

## 快速开始

### 1. 安装依赖

```bash
go get github.com/blang/semver/v4
go get gopkg.in/yaml.v3
```

### 2. 基本用法

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "your-app/pkg/updater"
    "your-app/config"
)

func main() {
    // 加载配置
    cfg, err := config.LoadConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建更新器
    upd, err := updater.NewUpdater(&cfg.Update, "1.0.0")
    if err != nil {
        log.Fatal(err)
    }
    
    // 检查更新
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    release, err := upd.CheckForUpdates(ctx)
    if err != nil {
        log.Printf("检查更新失败: %v", err)
        return
    }
    
    if release == nil {
        fmt.Println("已是最新版本")
        return
    }
    
    fmt.Printf("发现新版本: %s\n", release.Version)
    fmt.Printf("更新说明:\n%s\n", release.ReleaseNotes)
    
    // 下载并应用更新
    err = upd.PerformUpdate(ctx, func(progress updater.DownloadProgress) {
        fmt.Printf("下载进度: %.1f%% (%d/%d bytes)\n",
            progress.Percentage,
            progress.BytesDownloaded,
            progress.TotalBytes)
    })
    
    if err != nil {
        log.Printf("更新失败: %v", err)
        return
    }
    
    fmt.Println("更新成功！应用将重启...")
}
```

### 3. 启动时自动检查

```go
func main() {
    // 加载配置
    cfg, err := config.LoadConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建更新器
    upd, err := updater.NewUpdater(&cfg.Update, version.Version)
    if err != nil {
        log.Printf("创建更新器失败: %v", err)
    } else {
        // 在后台检查更新
        go func() {
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()
            
            release, err := upd.CheckForUpdates(ctx)
            if err != nil {
                log.Printf("检查更新失败: %v", err)
                return
            }
            
            if release != nil {
                // 通知用户有新版本
                notifyUser(release)
            }
        }()
    }
    
    // 启动应用主逻辑
    runApp()
}
```

## 配置选项

### 配置文件格式

创建 `config.yaml` 或使用 `config/update_config.template.yaml` 作为模板：

```yaml
update:
  # 是否启用自动更新
  enabled: true
  
  # 检查更新频率: "startup" (启动时), "daily" (每日), "weekly" (每周), "manual" (手动)
  check_frequency: startup
  
  # 更新渠道: "stable" (稳定版), "beta" (测试版)
  channel: stable
  
  # 是否自动下载更新（false 则仅通知）
  auto_download: false
  
  # 超时设置（秒）
  timeout: 300
  
  # 更新源列表（按优先级排序）
  sources:
    # GitHub Releases 源
    - type: github
      priority: 1
      enabled: true
      github_repo: "owner/repo"  # 格式: "owner/repo"
      github_token: ""           # 可选，避免 API 限流
    
    # 自定义 HTTP 源
    - type: http
      priority: 2
      enabled: true
      manifest_url: "https://updates.example.com/manifest.json"
```

### 配置选项说明

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | true | 是否启用自动更新功能 |
| `check_frequency` | string | "startup" | 检查更新的频率 |
| `channel` | string | "stable" | 更新渠道（stable/beta） |
| `auto_download` | bool | false | 是否自动下载更新 |
| `timeout` | int | 300 | 操作超时时间（秒） |
| `sources` | array | [] | 更新源列表 |

### 更新源配置

#### GitHub Releases 源

```yaml
- type: github
  priority: 1
  enabled: true
  github_repo: "owner/repo"
  github_token: ""  # 可选，用于提高 API 限流阈值
```

#### 自定义 HTTP 源

```yaml
- type: http
  priority: 2
  enabled: true
  manifest_url: "https://updates.example.com/manifest.json"
```

## Release Manifest 格式

自定义 HTTP 源需要提供符合以下格式的 `manifest.json`：

```json
{
  "version": "1.2.3",
  "published_at": "2026-01-14T10:00:00Z",
  "release_notes": "## What's New\n- Feature A\n- Bug fix B",
  "assets": {
    "windows_amd64": {
      "url": "https://example.com/releases/v1.2.3/app_windows_amd64.zip",
      "size": 10485760,
      "checksum": "abc123...",
      "name": "app_windows_amd64.zip"
    },
    "linux_amd64": {
      "url": "https://example.com/releases/v1.2.3/app_linux_amd64.tar.gz",
      "size": 8388608,
      "checksum": "def456...",
      "name": "app_linux_amd64.tar.gz"
    },
    "darwin_amd64": {
      "url": "https://example.com/releases/v1.2.3/app_darwin_amd64.zip",
      "size": 9437184,
      "checksum": "ghi789...",
      "name": "app_darwin_amd64.zip"
    },
    "darwin_arm64": {
      "url": "https://example.com/releases/v1.2.3/app_darwin_arm64.zip",
      "size": 9437184,
      "checksum": "jkl012...",
      "name": "app_darwin_arm64.zip"
    }
  }
}
```

### 平台键格式

平台键使用 `{os}_{arch}` 格式：

- `windows_amd64` - Windows 64位
- `windows_386` - Windows 32位
- `linux_amd64` - Linux 64位
- `linux_arm64` - Linux ARM64
- `darwin_amd64` - macOS Intel
- `darwin_arm64` - macOS Apple Silicon

### 生成 Manifest

使用提供的脚本从 GitHub Release 自动生成 manifest：

```bash
# Python 脚本
python scripts/generate_manifest.py owner/repo v1.2.3

# Shell 脚本
./scripts/generate_manifest.sh owner/repo v1.2.3
```

## API 文档

### 核心类型

#### UpdateOrchestrator

主要的更新协调器，管理整个更新流程。

```go
type UpdateOrchestrator struct {
    // 内部字段...
}
```

#### ReleaseInfo

发布信息结构。

```go
type ReleaseInfo struct {
    Version      string    // 版本号
    PublishedAt  time.Time // 发布时间
    ReleaseNotes string    // 更新说明
    AssetURL     string    // 下载链接
    AssetSize    int64     // 文件大小
    Checksum     string    // SHA256 校验和
    IsNewer      bool      // 是否为新版本
}
```

#### DownloadProgress

下载进度信息。

```go
type DownloadProgress struct {
    BytesDownloaded int64   // 已下载字节数
    TotalBytes      int64   // 总字节数
    Percentage      float64 // 下载百分比
    Speed           int64   // 下载速度（字节/秒）
}
```

### 主要方法

#### NewUpdater

创建新的更新器实例。

```go
func NewUpdater(config *config.UpdateConfig, currentVersion string) (*UpdateOrchestrator, error)
```

**参数：**
- `config`: 更新配置
- `currentVersion`: 当前应用版本

**返回：**
- `*UpdateOrchestrator`: 更新器实例
- `error`: 创建失败时返回错误

#### CheckForUpdates

检查是否有新版本可用。

```go
func (uo *UpdateOrchestrator) CheckForUpdates(ctx context.Context) (*ReleaseInfo, error)
```

**参数：**
- `ctx`: 上下文，用于超时控制

**返回：**
- `*ReleaseInfo`: 新版本信息，如果已是最新版本则返回 nil
- `error`: 检查失败时返回错误

#### DownloadUpdate

下载更新包。

```go
func (uo *UpdateOrchestrator) DownloadUpdate(
    ctx context.Context,
    release *ReleaseInfo,
    progressCallback DownloadCallback,
) (string, error)
```

**参数：**
- `ctx`: 上下文
- `release`: 要下载的版本信息
- `progressCallback`: 进度回调函数

**返回：**
- `string`: 下载文件的路径
- `error`: 下载失败时返回错误

#### ApplyUpdate

应用更新（替换可执行文件并重启）。

```go
func (uo *UpdateOrchestrator) ApplyUpdate(ctx context.Context, updatePath string) error
```

**参数：**
- `ctx`: 上下文
- `updatePath`: 更新包文件路径

**返回：**
- `error`: 应用失败时返回错误

#### PerformUpdate

执行完整的更新流程（检查 → 下载 → 应用）。

```go
func (uo *UpdateOrchestrator) PerformUpdate(
    ctx context.Context,
    progressCallback DownloadCallback,
) error
```

**参数：**
- `ctx`: 上下文
- `progressCallback`: 下载进度回调函数

**返回：**
- `error`: 更新失败时返回错误

#### ShouldCheckForUpdates

根据配置判断是否应该检查更新。

```go
func (uo *UpdateOrchestrator) ShouldCheckForUpdates() bool
```

**返回：**
- `bool`: 是否应该检查更新

### 回调函数类型

```go
type DownloadCallback func(progress DownloadProgress)
```

下载进度回调函数，在下载过程中定期调用以报告进度。

## 高级用法

### 自定义更新源

实现 `VersionChecker` 接口以支持自定义更新源：

```go
type VersionChecker interface {
    CheckLatest(ctx context.Context, currentVersion string) (*ReleaseInfo, error)
    GetSourceName() string
}

// 自定义实现
type CustomChecker struct {
    apiURL string
}

func (c *CustomChecker) CheckLatest(ctx context.Context, currentVersion string) (*ReleaseInfo, error) {
    // 实现自定义检查逻辑
    return &ReleaseInfo{...}, nil
}

func (c *CustomChecker) GetSourceName() string {
    return "custom-source"
}
```

### 状态持久化

更新器会自动保存状态到文件：

```go
type UpdateState struct {
    LastCheckTime   time.Time `json:"last_check_time"`
    LastUpdateTime  time.Time `json:"last_update_time"`
    SkippedVersions []string  `json:"skipped_versions"`
    CurrentVersion  string    `json:"current_version"`
}
```

状态文件位置：`~/.app_name/update_state.json`

### 错误处理

系统定义了详细的错误类型：

```go
type UpdateError struct {
    Category ErrorCategory
    Message  string
    Cause    error
    Context  map[string]interface{}
}

type ErrorCategory int

const (
    ErrCategoryNetwork ErrorCategory = iota
    ErrCategoryValidation
    ErrCategoryFileSystem
    ErrCategoryPermission
    ErrCategorySecurity
    ErrCategoryConfiguration
)
```

## 平台特定说明

### Windows

- 支持 `.exe` 文件替换
- 处理文件锁定问题
- 可能需要管理员权限

### macOS

- 支持代码签名验证
- 处理 Gatekeeper 限制
- 支持 Intel 和 Apple Silicon

### Linux

- 保持可执行权限
- 支持多种发行版
- 处理 SELinux/AppArmor 权限

## 安全性

### 传输安全

- 强制使用 HTTPS 连接
- 验证 SSL/TLS 证书

### 完整性验证

- SHA256 校验和验证
- 可执行文件格式验证
- macOS 代码签名验证

### 权限控制

- 最小权限原则
- 安全的临时文件处理
- 自动清理敏感数据

## 故障排查

### 常见问题

#### 1. 检查更新失败

```
错误: failed to check for updates: context deadline exceeded
```

**解决方案**：
- 检查网络连接
- 增加 `timeout` 配置值
- 验证更新源 URL 是否可访问

#### 2. 校验和验证失败

```
错误: checksum mismatch
```

**解决方案**：
- 重新下载更新包
- 验证 manifest 中的校验和是否正确
- 检查网络是否稳定

#### 3. 权限错误

```
错误: permission denied
```

**解决方案**：
- 以管理员权限运行应用
- 检查文件系统权限
- 确保应用有写入权限

### 日志分析

系统使用结构化 JSON 日志，便于分析：

```json
{
  "level": "info",
  "time": "2026-01-14T10:00:00Z",
  "message": "checking for updates",
  "current_version": "1.0.0",
  "source": "github"
}
```

查看日志文件：`logs/app.log`

## 测试

### 运行测试

```bash
# 运行所有测试
go test ./pkg/updater/...

# 运行单元测试
go test ./pkg/updater/ -run TestUnit

# 运行属性测试
go test ./pkg/updater/ -run TestProperty

# 查看覆盖率
go test ./pkg/updater/... -cover
```

### 测试覆盖

- 单元测试：验证具体功能和边界情况
- 属性测试：验证通用正确性属性
- 集成测试：验证端到端流程

## 性能优化

### 下载优化

- 支持断点续传，节省带宽
- 并发下载多个文件块
- 自动重试失败的请求

### 缓存策略

- 缓存版本检查结果
- 避免频繁 API 调用
- 智能更新频率控制

## 相关文档

- [设计文档](design.md) - 详细的架构和设计说明
- [需求文档](requirements.md) - 完整的功能需求
- [实现计划](tasks.md) - 开发任务列表
- [启动检查实现](STARTUP_CHECK_IMPLEMENTATION.md) - 启动时检查更新的实现细节
- [编排器说明](ORCHESTRATOR_README.md) - 更新编排器的详细说明

## 示例项目

查看 `examples/` 目录获取完整示例：

- `examples/basic/` - 基本用法示例
- `examples/custom-ui/` - UI 集成示例

## 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证。

## 支持

如有问题或建议，请：

- 提交 Issue
- 查看文档
- 联系维护者

---

**版本**: 1.0.0  
**最后更新**: 2026-01-14
