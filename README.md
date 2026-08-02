# cpt — Competitive Programming Tool

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

[**中文版**](README_zh.md) | [**English**](#)

**cpt** is a lightweight terminal companion for the [competitive-companion](https://github.com/jmerle/competitive-companion) browser extension. It bridges the browser-to-terminal gap for competitive programming: click a button in your browser, and cpt saves sample test cases and runs your solution — all in one seamless flow.

Supports **100+ online judges** (Luogu, USACO, Codeforces, AtCoder, and more) through competitive-companion's battle-tested parsers.

---

## Features

- **One-click sample sync** — competitive-companion parses the problem page; cpt receives the data via HTTP and saves samples locally
- **Auto-compile & test** — detects C++, C, Python, Java, Rust, Go; compiles and runs against all samples
- **Colored verdicts** — ✅ AC / ❌ WA / ⏱ TLE / 💥 RE, with line-by-line diff for wrong answers
- **Zero runtime deps** — single ~6 MB static binary, no Python/Node/Java required
- **Instant compilation** — Go incremental builds in <1 second
- **Reusable binary** — compiled output stays in the source dir (`main.cpp` → `./main`), ready to re-run against custom samples or via `cpt run ./main`

---

## How It Works

```
Browser                                  Terminal
┌──────────────────────┐    POST JSON    ┌───────────────┐
│ competitive-companion │ ──────────────→ │  cpt serve    │
│ (browser extension)   │   :27121       │  ↓ save samples│
│ 100+ OJ parsers       │                │  ↓ compile     │
└──────────────────────┘                │  ↓ run & diff │
                                         └───────────────┘
```

---

## Quick Start

### 1. Install competitive-companion

Install the browser extension:
- [Chrome Web Store](https://chromewebstore.google.com/detail/competitive-companion/cjnmckjndlpiamhfimnnjmnckgghkjbl)
- [Firefox Add-on](https://addons.mozilla.org/en-US/firefox/addon/competitive-companion/)

### 2. Install cpt

```bash
go install github.com/yourname/cpt@latest
```

Or build from source:

```bash
git clone https://github.com/yourname/cpt.git
cd cpt
go build -ldflags="-s -w" -o cpt .
sudo mv cpt /usr/local/bin/
```

### 3. Start coding

**Terminal 1** — start the server:
```bash
cpt serve
```

**Browser** — open any problem page (Luogu, USACO, Codeforces, AtCoder...), click the competitive-companion extension icon. cpt automatically saves samples.

**Terminal 2** — write your solution, then:
```bash
cpt test main.cpp
```

---

## Commands

### `cpt serve`

Start the HTTP server that listens for competitive-companion.

```bash
cpt serve                        # default port 27121, samples dir ./samples
cpt serve -p 12345               # custom port
cpt serve -d ./testcases         # custom samples directory
cpt serve -r ./a.out             # auto-run binary after receiving a problem
```

### `cpt test <source>`

Auto-detect language, compile (if needed), and run against all samples.

```bash
cpt test main.cpp                # C++ → g++ -std=c++17 -O2 -Wall -Wextra -Wshadow
cpt test solution.py             # Python → python3 (interpreted)
cpt test Main.java               # Java → javac → java
cpt test main.rs                 # Rust → rustc -O
cpt test main.go                 # Go → go build
cpt test main.cpp -t 5 -s 1      # timeout 5s, only sample 1

> Compiler flags mirror `~/.config/nvim/lua/utils.lua` (CompileRun). The compiled
> binary is kept in the source directory (`main.cpp` → `./main`) after testing, so
> you can re-run it against custom samples: `./main < in.txt` or `cpt run ./main`.
```

### `cpt run <binary>`

Run an already-compiled binary against samples.

```bash
cpt run ./a.out
cpt run "python3 solve.py"
cpt run ./a.out -s 2             # only sample 2
```

### `cpt list` (planned) / `cpt clean` (planned)

---

## Supported Languages

| Extension | Language  | Compile Command          |
|-----------|-----------|--------------------------|
| `.cpp` `.cc` `.cxx` | C++ | `g++ -std=c++17 -O2 -Wall -Wextra -Wshadow` |
| `.c`      | C         | `gcc -std=c11 -O2 -Wall` |
| `.py` `.py3` | Python | `python3` (interpreted) |
| `.java`   | Java      | `javac` → `java`         |
| `.rs`     | Rust      | `rustc -O`               |
| `.go`     | Go        | `go build`               |

---

## Sample Output

```
🔍 Detected language: cpp

═══════════════════════════════════
  Sample 1  ✅ AC   (2ms)
  Sample 2  ✅ AC   (2ms)
═══════════════════════════════════
  Passed: 2/2
```

When there's a wrong answer:

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

## Why cpt?

| Tool | Browser sync | Multi-OJ | Compile & Run | Binary size |
|------|:---:|:---:|:---:|:---:|
| **cpt** | ✅ via competitive-companion | ✅ 100+ | ✅ | ~6 MB |
| cf-tool | ❌ manual URL | ❌ CF only | ✅ | ~7 MB |
| online-judge-tools | ❌ manual URL | ✅ | ✅ | requires Python |
| vscode-cph | ✅ | ✅ | ❌ | requires VS Code |

cpt is the **lightest, fastest** option that covers the full browser→compile→test pipeline.

---

## Project Structure

```
cpt/
├── main.go
├── cmd/
│   ├── root.go          # cobra root
│   ├── serve.go         # cpt serve
│   ├── run.go           # cpt run
│   └── test.go          # cpt test
└── internal/
    ├── server.go        # HTTP server (:27121)
    ├── parser.go        # competitive-companion JSON → sample files
    ├── compiler.go      # language detection & compilation
    └── runner.go        # binary execution & output diff
```

---

## License

MIT
