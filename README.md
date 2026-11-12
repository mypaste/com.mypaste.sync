# 同步

MyPaste extension `com.mypaste.sync` (version `1.0.2`).

把历史与资产加密后上传到自己的 S3 存储，也可以从远端恢复。归档、加密、S3 签名、凭据与自动上传计划都在扩展自己的进程里完成；数据只经宿主的只读视图读取，连接与恢复仍由宿主确认。凭据只存放在扩展的私有存储中，不进入页面、日志或宿主设置。

This repository is an independent copy extracted from the former `mypaste-code-extensions` monorepo at commit `2af6675ab1473c9e38146ea0b6da68256a293440`. It contains only this extension's tracked source files. The published `.mypaste-plugin` archives and marketplace index live in [`mypaste-extensions-catalog`](https://github.com/mypaste/mypaste-extensions-catalog).

The manifest speaks MyPaste plugin protocol v3. Installation, capability mediation, and content-digest verification are performed by the MyPaste desktop host.

## Files

- `mypaste.plugin.json`: package identity and capability declaration.
- `run.sh`, `src/`, `panel.html`, or `run`: package implementation.

## License

MIT. See [LICENSE](LICENSE).
