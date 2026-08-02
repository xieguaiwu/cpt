# cpt — 算法竞赛命令行工具

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

[**English**](README.md) | [**中文版**](#)

**cpt** 是 [competitive-companion](https://github.com/jmerle/competitive-companion) 浏览器扩展的轻量级终端伴侣。它打通了浏览器到终端的最后一公里：在浏览器里点一下按钮，cpt 自动保存样例并在终端运行你的代码——全流程无缝衔接。

通过 competitive-companion 久经考验的解析器，支持 **100+ 在线评测平台**（洛谷、USACO、Codeforces、AtCoder 等）。

---

## 特性

- **一键同步样例** — competitive-companion 解析题目页面，cpt 通过 HTTP 接收并保存样例
- **serve + test 一条命令** — `cpt test` 在无样例时自动启动临时服务器等待题目推送，收好后直接测试
- **自动编译 + 测试** — 自动识别 C++ / C / Python / Java / Rust / Go，编译并对拍所有样例
- **彩色判定结果** — ✅ AC / ❌ WA / ⏱ TLE / 💥 RE，WA 时逐行展示期望 vs 实际输出
- **零运行时依赖** — 单二进制文件 ~6 MB，不需要 Python / Node / Java
- **秒级编译** — Go 增量编译 <1 秒
- **保留编译产物** — 编译后的二进制保存在源码目录（`main.cpp` → `./main`），可直接复用或配合 `cpt run` 自定义样例

---

## 工作原理

```
浏览器                                  终端
┌──────────────────────┐    POST JSON    ┌───────────────┐
│ competitive-companion │ ──────────────→ │  cpt serve    │
│ （浏览器扩展）         │   :27121       │  ↓ 保存样例    │
│ 100+ OJ 解析器        │                │  ↓ 编译代码    │
└──────────────────────┘                │  ↓ 运行对拍    │
                                         └───────────────┘
```

`cpt test` 内嵌了这条流水线：当 `samples/` 还没有样例时，它会自己启动服务器、
等待题目推送，然后自动编译并测试——无需再单独运行 `cpt serve`。

---

## 快速开始

### 1. 安装 competitive-companion

安装浏览器扩展：
- [Chrome 应用商店](https://chromewebstore.google.com/detail/competitive-companion/cjnmckjndlpiamhfimnnjmnckgghkjbl)
- [Firefox 附加组件](https://addons.mozilla.org/en-US/firefox/addon/competitive-companion/)

### 2. 安装 cpt

```bash
go install github.com/yourname/cpt@latest
```

或从源码编译：

```bash
git clone https://github.com/yourname/cpt.git
cd cpt
go build -ldflags="-s -w" -o cpt .
sudo mv cpt /usr/local/bin/
```

### 3. 开始刷题

**一个终端，无需单独开服务**：

```bash
cpt test main.cpp
```

如果还没有样例，cpt 会打印等待提示，自动启动临时服务器并阻塞等待：

```
🔍 Detected language: cpp
⏳ No samples in samples/ — waiting for competitive-companion…
   Open a problem page and click the extension button (server on http://127.0.0.1:27121)
   Waiting indefinitely — Ctrl+C to abort
```

**浏览器** — 打开任意题目页面（洛谷、USACO、Codeforces、AtCoder...），点击 competitive-companion 扩展图标。cpt 立即保存样例并运行你的代码。

如果偏好常驻服务器（例如一个会话内收多道题，或用 `-r` 自动运行），`cpt serve` 仍然可用。

---

## 命令参考

### `cpt serve`

启动 HTTP 服务器，监听 competitive-companion 的推送。

```bash
cpt serve                        # 默认端口 27121，样例目录 ./samples
cpt serve -p 12345               # 自定义端口
cpt serve -d ./testcases         # 自定义样例目录
cpt serve -r ./a.out             # 收到题目后自动运行指定二进制
```

### `cpt test <源文件>`

自动识别语言、编译（如需要）、对拍所有样例。

```bash
cpt test main.cpp                # C++ → g++ -std=c++17 -O2 -Wall -Wextra -Wshadow
cpt test solution.py             # Python → python3（解释执行）
cpt test Main.java               # Java → javac → java
cpt test main.rs                 # Rust → rustc -O
cpt test main.go                 # Go → go build
cpt test main.cpp -t 5 -s 1      # 超时 5 秒，仅测第 1 个样例

# 等待模式（无样例时自动启用）
cpt test main.cpp --wait               # 强制等待新题目，丢弃旧样例
cpt test main.cpp --wait-timeout 60    # 60 秒后放弃（0 = 无限等待）
cpt test main.cpp -p 12345 --secret abc  # 自定义端口 / 共享密钥

> 编译参数与 `~/.config/nvim/lua/utils.lua`（CompileRun）保持一致。测试结束后
> 编译产物保留在源码目录（`main.cpp` → `./main`），可自定义样例直接复用：
> `./main < in.txt` 或 `cpt run ./main`。
```

### `cpt run <二进制>`

运行已编译好的程序对拍样例。

```bash
cpt run ./a.out
cpt run "python3 solve.py"
cpt run ./a.out -s 2             # 仅测第 2 个样例
```

---

## 支持的语言

| 扩展名 | 语言 | 编译命令 |
|-----------|------|------------------|
| `.cpp` `.cc` `.cxx` | C++ | `g++ -std=c++17 -O2 -Wall -Wextra -Wshadow` |
| `.c` | C | `gcc -std=c11 -O2 -Wall` |
| `.py` `.py3` | Python | `python3`（解释执行）|
| `.java` | Java | `javac` → `java` |
| `.rs` | Rust | `rustc -O` |
| `.go` | Go | `go build` |

---

## 输出示例

```
🔍 Detected language: cpp

═══════════════════════════════════
  Sample 1  ✅ AC   (2ms)
  Sample 2  ✅ AC   (2ms)
═══════════════════════════════════
  Passed: 2/2
```

出现 WA 时：

```
🔍 Detected language: cpp

═══════════════════════════════════
  Sample 1  ❌ WA   (2ms)
───────────────────────────────────
  Line 1:
    Expected: 50
    Got:      600
  Sample 2  ✅ AC   (2ms)
═══════════════════════════════════
  Passed: 1/2
```

---

## 为什么选择 cpt？

| 工具 | 浏览器同步 | 多平台 | 编译 + 运行 | 二进制大小 |
|------|:---:|:---:|:---:|:---:|
| **cpt** | ✅ 通过 competitive-companion | ✅ 100+ | ✅ | ~6 MB |
| cf-tool | ❌ 手动复制 URL | ❌ 仅 CF | ✅ | ~7 MB |
| online-judge-tools | ❌ 手动复制 URL | ✅ | ✅ | 依赖 Python |
| vscode-cph | ✅ | ✅ | ❌ | 依赖 VS Code |

cpt 是覆盖浏览器→编译→测试全流程的**最轻、最快**方案。

---

## 项目结构

```
cpt/
├── main.go
├── cmd/
│   ├── root.go          # cobra 根命令
│   ├── serve.go         # cpt serve
│   ├── run.go           # cpt run
│   └── test.go          # cpt test
└── internal/
    ├── server.go        # HTTP 服务器 (:27121)
    ├── parser.go        # competitive-companion JSON → 样例文件
    ├── compiler.go      # 语言检测与编译
    └── runner.go        # 执行与输出对比
```

---

## License

MIT
