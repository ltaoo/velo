# macOS Sleep Guide

本文档整理 macOS 睡眠相关的概念、能力、状态、命令和验证方法。重点面向开发者和需要让后台服务持续运行的场景。

最后更新：2026-06-06。

## 1. 核心结论

macOS 里“睡眠”不是一个单一状态，至少要区分：

- 显示器睡眠：屏幕黑了，但系统可能仍然醒着，CPU 和进程仍在运行。
- 系统睡眠：整机进入低功耗状态，普通进程不继续执行，网络服务通常不可持续提供。
- 休眠/standby/autopoweroff：系统已经睡眠后，进一步进入更省电、更深层的状态，唤醒更慢。
- 断言 assertion：进程临时告诉系统“我现在在做事，请不要因为空闲而睡眠”。

对后端服务来说，关键不是屏幕是否黑，而是系统有没有进入 system sleep。只要系统没有 system sleep，普通后端进程就可以继续运行。

## 2. 常见状态

### Awake

系统醒着，CPU 正常调度，用户态进程正常运行。显示器可以亮着，也可以关闭。

### Display Sleep

显示器进入低功耗状态。表现是屏幕黑了、外接显示器可能无信号、重新按键/移动鼠标后很快亮起。

这不等于系统睡眠。显示器睡眠期间，后端服务、下载、编译任务可以继续运行，前提是系统本身仍然醒着。

相关 `pmset` 参数：

```sh
pmset -c displaysleep 5
pmset -b displaysleep 5
```

`0` 表示不因空闲关闭显示器。

### Idle System Sleep

系统因为用户长时间无操作并且没有阻止睡眠的断言而进入睡眠。进入后，普通进程会暂停执行。后端服务不会继续处理请求，除非系统被唤醒。

相关 `pmset` 参数：

```sh
pmset -c sleep 0
pmset -b sleep 20
```

`0` 表示不因空闲进入系统睡眠。

### Explicit Sleep

用户或系统显式触发的睡眠，例如：

- Apple 菜单中的 Sleep。
- 合上 MacBook 屏幕。
- 电量过低。
- 某些硬件、策略或管理配置触发的睡眠。

注意：`pmset sleep 0` 和 `PreventUserIdleSystemSleep` 主要阻止“空闲导致的系统睡眠”，不保证阻止所有显式睡眠原因。Apple 的 IOKit 文档也说明，阻止用户空闲睡眠的断言不阻止合盖、Apple 菜单、低电量等睡眠原因。

### Safe Sleep / Hibernate

macOS 可以在睡眠时把内存镜像写入磁盘，以便断电后恢复会话。`pmset` 里用 `hibernatemode` 控制常见模式：

- `hibernatemode 0`：普通睡眠，历史上常用于台式机；不把内存备份到持久存储。
- `hibernatemode 3`：便携式 Mac 常见默认值；写入内存镜像，同时睡眠时继续给内存供电，通常从内存快速恢复。
- `hibernatemode 25`：更接近深度休眠；写入内存镜像并移除内存供电，唤醒更慢但更省电。

这些值还会受到 `standby`、`autopoweroff` 等设置影响。

### Standby

standby 是睡眠后的更深层节能状态。支持的机器在普通睡眠一段时间后，可能写入休眠镜像并降低更多硬件供电。

常见相关参数：

```text
standby
standbydelayhigh
standbydelaylow
highstandbythreshold
```

不同机型、芯片、macOS 版本支持情况不同，以 `pmset -g cap`、`pmset -g custom` 实际输出为准。

### Auto Power Off

`autopoweroff` 是支持平台上的更低功耗睡眠阶段。`pmset` 手册描述它会在睡眠达到延迟时间后写入休眠镜像，并进入更低功耗状态；从这个状态唤醒通常比普通睡眠更慢。

常见相关参数：

```text
autopoweroff
autopoweroffdelay
```

### Dark Wake / Maintenance Wake

`pmset -g log` 里可能看到 `DarkWake`、`MaintenanceWake`、`SleepService` 等事件。这通常表示机器没有完整亮屏，但系统短暂或部分唤醒以执行维护、网络、推送或系统任务。

对服务验证来说，不要只看这些事件名。要重点看是否出现完整的 `Entering Sleep` 以及后续 `Wake from ...`，并结合你的服务日志是否有长时间断档。

## 3. 电源来源与配置作用域

`pmset` 的作用域：

- `-c`：接入电源适配器时，charger / AC power。
- `-b`：使用电池时，battery power。
- `-a`：所有电源来源。
- `-u`：UPS，主要面向台式机/服务器场景。

示例：

```sh
# 接电源时：显示器 5 分钟后关闭，但系统不因空闲睡眠。
sudo pmset -c displaysleep 5 sleep 0 disksleep 0

# 使用电池时：显示器 5 分钟后关闭，系统 20 分钟后睡眠。
sudo pmset -b displaysleep 5 sleep 20 disksleep 10
```

如果目标是“后端服务继续跑，但屏幕可以黑”，推荐组合是：

```sh
sudo pmset -c displaysleep 5 sleep 0 disksleep 0
```

如果目标是“接电源时屏幕也不要黑”，使用：

```sh
sudo pmset -c displaysleep 0 sleep 0 disksleep 0
```

## 4. 重要 pmset 参数

### 空闲计时器

| 参数 | 含义 | `0` 的含义 |
| --- | --- | --- |
| `displaysleep` | 显示器空闲睡眠时间，分钟 | 显示器不因空闲关闭 |
| `sleep` | 系统空闲睡眠时间，分钟 | 系统不因空闲睡眠 |
| `disksleep` | 磁盘空闲休眠时间，分钟 | 磁盘不因空闲休眠 |

### 睡眠深度与恢复

| 参数 | 含义 |
| --- | --- |
| `hibernatemode` | 控制 Safe Sleep / hibernate 行为，常见值为 `0`、`3`、`25` |
| `standby` | 是否允许睡眠后进入 standby |
| `standbydelayhigh` | 电量高于阈值时，进入 standby 前的延迟，通常以秒计 |
| `standbydelaylow` | 电量低于阈值时，进入 standby 前的延迟，通常以秒计 |
| `highstandbythreshold` | 区分 high/low standby delay 的电量阈值 |
| `autopoweroff` | 是否允许进入更低功耗的 autopoweroff 状态 |
| `autopoweroffdelay` | 进入 autopoweroff 前的延迟 |
| `destroyfvkeyonstandby` | standby 时是否销毁 FileVault key；安全性更高但唤醒体验更重 |

### 唤醒与网络

| 参数 | 含义 |
| --- | --- |
| `womp` | Wake on Magic Packet；对应系统设置里的 Wake for network access 的一部分语义 |
| `tcpkeepalive` | 睡眠期间维持某些 TCP / 网络可达能力，具体行为依机型和系统策略变化 |
| `networkoversleep` | 网络睡眠相关行为；并非所有平台使用，手册里也提示不建议随意修改 |
| `powernap` | 允许睡眠时执行部分系统维护、邮件、iCloud 等任务 |
| `ttyskeepawake` | 有活跃 tty 时阻止 idle system sleep |
| `acwake` | 电源来源变化时唤醒 |
| `lidwake` | 打开 MacBook 屏幕时唤醒 |
| `autorestart` | 断电恢复后自动重启，台式机场景常见 |

不是所有参数都在所有 Mac 上可用。Apple 支持文档也明确说明，系统设置中的部分睡眠/唤醒选项取决于 Mac 型号。

## 5. 断言 Assertions

断言是进程级的临时睡眠控制。它和 `pmset` 的持久配置不同：

- `pmset`：修改系统级空闲计时器，通常需要管理员权限，会持久保存。
- assertion：某个进程运行期间临时生效，进程结束或释放断言后失效。

查看当前断言：

```sh
pmset -g assertions
```

常见字段：

| 字段 | 含义 |
| --- | --- |
| `PreventUserIdleDisplaySleep` | 阻止显示器因用户空闲而睡眠 |
| `PreventUserIdleSystemSleep` | 阻止系统因用户空闲而睡眠；显示器仍可能睡眠 |
| `PreventSystemSleep` | 更强的系统睡眠阻止断言，常见于特定 AC 场景 |
| `UserIsActive` | 系统认为最近有用户活动 |
| `BackgroundTask` | 后台任务断言 |
| `ApplePushServiceTask` | Apple 推送/系统服务相关任务 |
| `NetworkClientActive` | 网络客户端活动 |
| `ExternalMedia` | 外部介质相关断言 |

例如你看到：

```text
PreventUserIdleSystemSleep     1
pid 102(powerd): ... PreventUserIdleSystemSleep named: "Powerd - Prevent sleep while display is on"
```

这表示当前有断言阻止系统因空闲睡眠，不表示机器已经睡眠。

### caffeinate

`caffeinate` 是 macOS 自带命令，用来创建断言。

常见用法：

```sh
# 阻止 idle system sleep，直到 Ctrl-C。
caffeinate -i

# 阻止显示器睡眠。
caffeinate -d

# 阻止 idle system sleep，并运行一个命令；命令退出后断言释放。
caffeinate -i ./long-running-task.sh

# 在 AC power 下创建 PreventSystemSleep 风格的断言。
caffeinate -s ./long-running-task.sh

# 保持指定秒数。
caffeinate -i -t 3600

# 等待某个 PID 结束前保持断言。
caffeinate -i -w 12345
```

注意：如果你正在验证 `pmset -c sleep 0` 是否能让服务过夜运行，不要同时使用 `caffeinate`，否则无法区分是系统设置生效，还是 `caffeinate` 断言生效。

### App API

App 可以通过两类 API 表达活动：

- Foundation `ProcessInfo.beginActivity(options:reason:)`
- IOKit `IOPMAssertionCreateWithName` / `IOPMAssertionCreateWithDescription`

常见语义：

- `idleSystemSleepDisabled` / `kIOPMAssertionTypePreventUserIdleSystemSleep`：阻止系统因空闲睡眠，但显示器仍可能关闭。
- `idleDisplaySleepDisabled` / `kIOPMAssertionTypePreventUserIdleDisplaySleep`：要求屏幕保持亮起。

Apple 文档提醒，禁用系统或显示器睡眠会明显影响用户体验和耗电，应在任务完成后及时释放。

## 6. 系统设置与 pmset 的关系

系统设置里的这些选项通常对应 `pmset` 里的配置：

| 系统设置概念 | 相关 pmset 参数 |
| --- | --- |
| Turn display off when inactive | `displaysleep` |
| Prevent automatic sleeping when display is off | `sleep 0` 或相关策略 |
| Put hard disks to sleep when possible | `disksleep` |
| Enable Power Nap | `powernap` |
| Wake for network access | `womp`、网络唤醒相关策略 |

`pmset` 手册说明，`pmset` 修改的是系统电源管理设置，保存到系统级偏好设置文件。实际 UI 名称会随 macOS 版本变化，但底层概念基本一致。

## 7. 如何判断是否真的睡眠

不要只看屏幕是否黑。建议同时看三类证据。

### 1. pmset 日志

```sh
pmset -g log | rg "Entering Sleep|Wake from|DarkWake|MaintenanceWake"
```

重点：

- 出现 `Entering Sleep`：系统进入过睡眠。
- 出现 `Wake from ...`：系统从睡眠唤醒。
- 只有 assertion summary，不表示已经睡眠。
- `DarkWake` / `MaintenanceWake` 说明有维护唤醒或暗唤醒，需要结合上下文判断。

### 2. 断言状态

```sh
pmset -g assertions
```

重点：

- `PreventUserIdleSystemSleep 1`：有断言阻止空闲系统睡眠。
- `PreventUserIdleDisplaySleep 0`：没有阻止显示器睡眠，屏幕可以黑。
- `PreventSystemSleep 0`：没有强阻止系统睡眠的断言。

### 3. 业务心跳

后端服务最可靠的证据是业务日志连续：

- 每分钟写一次 heartbeat。
- 早上检查日志是否跨夜连续。
- 同时检查 `pmset -g log` 里这段时间是否有 `Entering Sleep`。

如果服务日志连续、接口可访问、没有 `Entering Sleep`，说明系统没有真正睡眠。屏幕黑不影响结论。

## 8. 后端服务过夜验证流程

目标：验证接电源时，屏幕可以关闭，但后端服务继续运行。

### 睡前配置

```sh
sudo pmset -c displaysleep 5 sleep 0 disksleep 0
pmset -g ps
pmset -g custom
```

确认当前使用 AC Power，并且 AC 配置里 `sleep 0`。

### 启动服务

推荐使用 `launchd` 管理真正的长期服务。临时测试可以用 `nohup` 或终端后台进程，但要确保终端退出不会杀掉服务。

服务应写心跳日志，例如每 60 秒写一行：

```text
2026-06-06 23:59:00 heartbeat
2026-06-07 00:00:00 heartbeat
```

### 明早检查

```sh
# 1. 检查服务接口是否仍能访问。
curl http://127.0.0.1:18080

# 2. 检查服务日志是否连续。
tail -100 /tmp/ac-sleep-test/heartbeat.log

# 3. 检查睡眠日志。
pmset -g log | rg "Entering Sleep|Wake from"

# 4. 检查当前断言。
pmset -g assertions
```

判断：

- 服务接口可访问：服务进程仍在运行。
- 心跳日志跨夜连续：系统大概率没有睡眠。
- `pmset -g log` 没有对应时段的 `Entering Sleep`：系统没有进入睡眠。
- 屏幕黑了但心跳连续：只是显示器睡眠或锁屏，不是系统睡眠。

## 9. MacBook 合盖与 clamshell

MacBook 合盖是特殊场景。即使设置了 `sleep 0` 或创建了 idle sleep 断言，合盖仍可能触发睡眠。

如果要合盖使用，通常需要满足 clamshell 条件：

- 接入电源。
- 连接外部显示器。
- 连接外部键盘/鼠标或触控板。
- 某些 Apple silicon 机型还涉及外设连接许可和系统安全设置。

如果只是验证后台服务过夜运行，建议开盖、接电源、允许显示器自行关闭。

## 10. demo-macostool 中的含义

`_example/demo-macostool` 的“接电源不休眠”预设相当于：

```text
AC Power:
  displaysleep = 0
  sleep        = 0
  disksleep    = 0
```

也就是说：

- 接电源时，系统不因空闲睡眠。
- 接电源时，显示器也不因空闲关闭。
- 接电源时，磁盘也不因空闲休眠。

如果你希望“屏幕可以黑，但后端服务继续跑”，这个预设应改成类似：

```text
AC Power:
  displaysleep = 5
  sleep        = 0
  disksleep    = 0
```

UI 文案也可以改成“接电源保持系统唤醒”或“接电源服务持续运行”，避免把显示器睡眠和系统睡眠混在一起。

## 11. 排查清单

### 屏幕黑了，但服务还在跑

这通常正常。检查：

```sh
pmset -g assertions
pmset -g log | rg "Entering Sleep|Wake from"
```

如果没有 `Entering Sleep`，服务日志连续，就是显示器睡眠或锁屏。

### 服务中断了，日志有长时间空白

检查：

```sh
pmset -g log | rg "Entering Sleep|Wake from|Shutdown|Restart"
```

如果空白时段对应 `Entering Sleep`，说明系统睡了。

### `PreventUserIdleSystemSleep 1` 是不是睡眠

不是。它表示当前有断言阻止系统因空闲进入睡眠。

### `sleep 0` 仍然睡了

检查睡眠原因：

```sh
pmset -g log | rg "Entering Sleep"
```

常见原因包括：

- 合盖。
- 用户手动 Sleep。
- 低电量。
- 电源断开后切到电池配置。
- MDM/企业策略。
- 系统更新或重启。
- 进程其实没用 launchd/nohup 管理，被终端退出影响。

### 本地 curl 连不上，但 lsof 显示 LISTEN

如果在受限沙箱、IDE、自动化环境中测试，可能是环境限制导致无法访问本机端口。应该在普通 Terminal 里验证：

```sh
curl http://127.0.0.1:18080
lsof -nP -iTCP:18080 -sTCP:LISTEN
```

## 12. 常用命令速查

### 查看配置

```sh
pmset -g
pmset -g live
pmset -g custom
pmset -g cap
```

### 查看电源来源

```sh
pmset -g ps
pmset -g ac
```

### 查看断言

```sh
pmset -g assertions
pmset -g assertionslog
```

### 查看睡眠/唤醒日志

```sh
pmset -g log
pmset -g log | rg "Entering Sleep|Wake from|DarkWake|MaintenanceWake"
```

### 立即触发

```sh
pmset sleepnow
pmset displaysleepnow
```

### 计划唤醒/睡眠

```sh
pmset -g sched
sudo pmset schedule wake "06/07/26 08:00:00"
sudo pmset repeat cancel
```

### 临时保持唤醒

```sh
caffeinate -i
caffeinate -d
caffeinate -i ./task.sh
caffeinate -i -t 3600
```

## 13. 推荐配置

### 后端服务/下载/构建过夜，屏幕可黑

```sh
sudo pmset -c displaysleep 5 sleep 0 disksleep 0
```

验证时不要运行 `caffeinate`。

### 演示/投屏，屏幕不要黑

```sh
caffeinate -d
```

或临时运行：

```sh
caffeinate -d -i ./presentation-task.sh
```

### 临时长任务，任务结束后恢复正常

```sh
caffeinate -i ./long-running-task.sh
```

### 长期后台服务

使用 `launchd` 管理服务进程，再配合：

```sh
sudo pmset -c displaysleep 5 sleep 0 disksleep 0
```

不要依赖一个打开的 Terminal 窗口维持进程。

## 14. 参考资料

- Apple Support: Set sleep and wake settings for your Mac  
  https://support.apple.com/guide/mac-help/set-sleep-and-wake-settings-mchle41a6ccd/mac

- Apple Support: Put your Mac to sleep or wake it  
  https://support.apple.com/guide/mac-help/put-your-mac-to-sleep-or-wake-it-mh10330/mac

- Apple Support: If your Mac screen goes black  
  https://support.apple.com/guide/mac-help/mchlp1025/mac

- Apple Support: Share your Mac resources when it is in sleep  
  https://support.apple.com/guide/mac-help/mh27905/mac

- Apple Developer Documentation: ProcessInfo.ActivityOptions  
  https://developer.apple.com/documentation/foundation/processinfo/activityoptions

- Apple Developer Documentation: NSProcessInfo  
  https://developer.apple.com/documentation/foundation/nsprocessinfo

- Apple Developer Documentation: kIOPMAssertionTypePreventUserIdleSystemSleep  
  https://developer.apple.com/documentation/iokit/kiopmassertiontypepreventuseridlesystemsleep

- Apple Developer Documentation: IOPMAssertionCreateWithDescription  
  https://developer.apple.com/documentation/iokit/1557078-iopmassertioncreatewithdescripti

- Apple Developer Archive: Energy Efficiency Guide for Mac Apps, Prioritize Work at the App Level  
  https://developer.apple.com/library/archive/documentation/Performance/Conceptual/power_efficiency_guidelines_osx/PrioritizeWorkAtTheAppLevel.html

- Local macOS manual pages: `man pmset`, `man caffeinate`
