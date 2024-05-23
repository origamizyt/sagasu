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
