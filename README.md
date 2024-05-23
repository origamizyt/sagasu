<h1 align="center">💎Sagasu - 轻量级远程文件系统</h1>

<p align="center">
    <b>
        ❗注意：仅支持 Windows❗
    </b>
</p>

## 🌀 编译

编译需要 go，npm 以及：
```
go install github.com/akavel/rsrc@latest
go install github.com/go-bindata/go-bindata/...
go install github.com/elazarl/go-bindata-assetfs/...
```

```
git clone https://github.com/origamizyt/sagasu.git
cd sagasu

npm i
npm run build   # 生成静态资源
npm run gobuild # 编译
```

以上命令在 `src-go` 目录下生成 `sagasu.exe` 可执行文件。

## ✈️ 使用

切换至可执行文件所在目录，并以管理员身份运行以下命令：
```
.\sagasu init
```

此命令将创建注册表项，并将默认配置写入 `~\sagasu-config.toml`。

对要共享的文件夹点击右键，选择 “使用 Sagasu 共享”，即可启动 Sagasu。

## ⚙️ 配置

### 全局配置

配置文件位于 `~\sagasu-config.toml` 中，以下是配置结构：

**Assoc.IconCache**

- 类型：string
- 描述：图标缓存位置

**Assoc.Custom**

- 类型：object
- 描述：Key 代表匹配的模板，如 `*.go`, `file.*`。Value 为有 Name 和 Icon 两个属性的 object，分别为关联名称，关联的图标位置（必须为 ico）。

示例：
```toml
[Assoc.Custom]
"*.txt" = { Name = "文本文件", Icon = "...\\text.ico" }
```

如果不在 `Custom` 中指定，Sagasu 将尝试在注册表中寻找文件关联。

**Tree.DefaultFlag**

- 类型：string
- 有效值：invisible, visible, readonly, readwrite
- 描述：没有规则匹配的情况下默认的访问控制级别。详见规则配置一节。

**Tree.RulesFile**

- 类型：string
- 描述：默认的规则文件名称。

**Tree.ShowHidden**

- 类型：boolean
- 描述：是否将访问控制级别为 invisible 的文件视为隐藏文件。如果开启，则在 UI 中开启 +H 开关可以看到这些文件（仍然不可读写）。

**Tree.CachePolicy**

- 类型：string
- 有效值：never, upload, always
- 描述：指定路径规则的更新时机。never 代表每次访问都将重新读取规则，耗费资源但可以实时更新；upload 代表每次上传文件都将重新读取规则；always 将一直使用第一次访问时加载的规则。

**Http.Host**

- 类型：string
- 描述：绑定的主机名。如果为 0.0.0.0 将在屏幕上显示二维码。

**Http.Port**

- 类型：int
- 描述：绑定的端口号。

**Http.Debug**

- 类型：boolean
- 描述：是否将 Gin 设为 Debug 模式。设置为 true 将输出额外的日志。

### 规则配置

规则决定 Sagasu 对于文件的访问控制级别。共有四个级别，分别为 invisible, visible, readonly 和 readwrite。

- **invisible**：文件对于客户端不可见。调用 API 不会返回文件信息，无法辨别一个文件是 invisible 还是不存在。
- **visible**：文件对于客户端可见，但不可读写（不可下载、复制、剪切、被粘贴至）。
- **readonly**：文件可读不可写（可以下载、复制，不可剪切、被粘贴至）。
- **readwrite**：文件可读可写（可以下载、复制、剪切、被粘贴至）。

> 注意：在 Tree.ShowHidden 配置为 true 时 invisible 文件将成为 visible 文件，但 UI 会默认隐藏它们。

特殊的，对于上传至与粘贴至不存在的文件位置，如果目标文件被创建后的访问级别低于 readwrite，则操作同样会失败，文件不会被创建。

默认的规则配置位于共享目录中的 `.rules.yml` 中。一个有效的配置文件具有如下结构：
```yaml
invisible:
    - pattern1
    - pattern2
visible:
    - pattern3
readonly:
    - pattern4
readwrite:
    - pattern5
```

如果某几个级别无需匹配则可以忽略。注意，模式只能匹配当前及子目录中的内容。

假设有文件 `\foo\bar\baz\a.txt`，它的访问级别将通过如下顺序搜索（越靠上优先级越高）：
```
\foo\bar\baz\.rules.yml 中匹配 a.txt 的项
\foo\bar\.rules.yml 中匹配 baz\a.txt 的项
\foo\.rules.yml 中匹配 bar\baz\a.txt 的项
\.rules.yml 中匹配 foo\bar\baz\a.txt 的项

\foo\bar\.rules.yml 中匹配 baz 的项
\foo\.rules.yml 中匹配 bar\baz 的项
\.rules.yml 中匹配 foo\bar\baz 的项

\foo\.rules.yml 中匹配 bar 的项
\.rules.yml 中匹配 foo\bar 的项

\.rules.yml 中匹配 foo 的项
```

如果都不存在则使用默认值。

注意：invisible 的搜索顺序不同，其顺序为：
```
\.rules.yml 中匹配 foo 的项

\foo\.rules.yml 中匹配 bar 的项
\.rules.yml 中匹配 foo\bar 的项

\foo\bar\.rules.yml 中匹配 baz 的项
\foo\.rules.yml 中匹配 bar\baz 的项
\.rules.yml 中匹配 foo\bar\baz 的项

\foo\bar\baz\.rules.yml 中匹配 a.txt 的项
\foo\bar\.rules.yml 中匹配 baz\a.txt 的项
\foo\.rules.yml 中匹配 bar\baz\a.txt 的项
\.rules.yml 中匹配 foo\bar\baz\a.txt 的项
```

这是由于如果父目录为 invisible，在遍历树时该节点根本不会被加载，即如果 bar 为 invisible，则 bar 下所有项，包括规则配置文件将被忽略。此时子项单独设置访问级别也没有用。

## 🎩 API

以下为 HTTP API。

**/tree/:path**

获取 `path` 目录下的文件与文件夹列表。参数无需转义，按照 catch-all 传递。

如果成功，状态为 200，返回值见 `src/api.ts#Backend.tree`。

如果目录不存在或为 invisible，状态为 404，返回值为：
```json
{
    "ok": false,
    "error": "第一个不存在的路径部分"
}
```

如果发生内部错误，状态为 500，返回值为：
```json
{
    "ok": false
}
```

**/fileicon/:path**

获取 `path` 代表的文件图标，以图标形式返回。

如果成功，状态为 200。

如果文件不存在或为 invisible，状态为 404，返回值为：
```json
{
    "ok": false,
    "error": "第一个不存在的路径部分"
}
```

如果发生内部错误，状态为 500，返回值为：
```json
{
    "ok": false
}
```

**/foldericon**

获取文件夹图标。

如果成功，状态为 200。

如果失败，状态为 500，返回值为：
```json
{
    "ok": false
}
```

**/file/:path?download=:bool**

获取文件内容，指定 download=true 将强制浏览器下载而非预览。

如果成功，状态为 200。

如果文件不存在或为 invisible，状态为 404，返回值为：
```json
{
    "ok": false,
    "error": "第一个不存在的路径部分"
}
```

如果发生内部错误，状态为 500，返回值为：
```json
{
    "ok": false
}
```

**/upload/:path** (WebSocket)

第一帧为头，格式为 JSON：
```json
{
    "count": 100, // 分块数量，UI 中为 5MB 一块
    "key": "KEY的HEX编码" // blake2b Key
}
```

服务器回复帧为 JSON true。

之后 N 帧为数据帧，为 256bits keyed-blake2b 哈希，后跟文件内容。

服务器回复帧为 JSON true/false，代表哈希是否匹配。若为 false 请重发该帧。

如果成功，关闭代码为 1000。

如果客户端协议错误，关闭代码为 1008。

如果服务器内部错误，关闭代码为 1011。

如果请求不是 WebSocket，返回 HTTP 400，返回值为：
```json
{
    "ok": false
}
```

如果有不存在或 invisible 的父目录，返回 HTTP 404，返回值为：
```json
{
    "ok": false,
    "error": "第一个不存在的路径部分"
}
```

如果目标文件级别低于 readwrite，返回 HTTP 403，返回值为：
```json
{
    "ok": false
}
```

**/move** (POST)

Body 为 JSON：
```json
{
    "from": ["path", "to", "src"],
    "to": ["path", "to", "dst"]
}
```

如果成功，状态为 200，返回值为：
```json
{
    "ok": true
}
```

如果文件不存在或为 invisible，状态为 404，返回值为：
```json
{
    "ok": false,
    "error": "第一个不存在的路径部分"
}
```

如果源文件或目标文件级别低于 readwrite，状态为 403，返回值为：
```json
{
    "ok": false
}
```

如果发生内部错误，状态为 500，返回值为：
```json
{
    "ok": false
}
```

**/copy** (POST)

Body 为 JSON：
```json
{
    "from": ["path", "to", "src"],
    "to": ["path", "to", "dst"]
}
```

如果成功，状态为 200，返回值为：
```json
{
    "ok": true
}
```

如果有不存在或 invisible 的父目录，状态为 404，返回值为：
```json
{
    "ok": false,
    "error": "第一个不存在的路径部分"
}
```

如果源文件级别低于 readonly 或目标文件级别低于 readwrite，状态为 403，返回值为：
```json
{
    "ok": false
}
```

如果发生内部错误，状态为 500，返回值为：
```json
{
    "ok": false
}
```

**/delete/:path** (POST)

删除指定位置的文件。

如果成功，状态为 200，返回值为：
```json
{
    "ok": true
}
```

如果文件不存在或为 invisible，状态为 404，返回值为：
```json
{
    "ok": false,
    "error": "第一个不存在的路径部分"
}
```

如果目标文件级别低于 readwrite，状态为 403，返回值为：
```json
{
    "ok": false
}
```

如果发生内部错误，状态为 500，返回值为：
```json
{
    "ok": false
}
```