# cpt — Developer Documentation

> **Target audience**: AI agents and future maintainers.  
> **Last updated**: 2026-08-02 (v1.2.0)

---

## Project Overview

**cpt** (Competitive Programming Tool) bridges browser-based problem pages to terminal-based code execution. It listens for JSON payloads from the [competitive-companion](https://github.com/jmerle/competitive-companion) browser extension, saves sample test cases, compiles solutions, and runs them with colored diff reporting.

- **Language**: Go 1.25
- **Binary size**: ~6.4 MB (static, stripped)
- **Dependencies**: cobra (CLI), fatih/color (terminal color) — both vendored
- **Platform support**: 100+ OJs via competitive-companion (Luogu, USACO, Codeforces, AtCoder, etc.)

---

## Architecture

```
main.go                     # Entry point
cmd/
├── root.go                 # cobra root command
├── serve.go                # `cpt serve` — HTTP server
├── run.go                  # `cpt run <binary>` — test runner
└── test.go                 # `cpt test <source>` — compile + test, wait mode merges serve
internal/
├── server.go               # HTTP server (:27121), auth, rate limiting, onProblem hook
├── parser.go               # JSON deserialization → sample file I/O
├── compiler.go             # Language detection + compilation dispatch
└── runner.go               # Binary execution, stdout capture, diff engine
```

### Data Flow

```
Browser (competitive-companion)
  │ POST JSON to localhost:27121
  ▼
server.go (handleRequest)
  │ authenticate → rate limit → parse JSON → sanitize metadata
  ▼
parser.go (SaveSamples)
  │ cap at 100 tests → write {N}.in / {N}.out to samples/
  ▼
server.go (onProblem callback)
  │ wake waiting `cpt test` process (count of saved samples)
  ▼
runner.go (RunAll → RunTest)
  │ pipe {N}.in to stdin → capture stdout → diff with {N}.out
  ▼
Terminal: colored verdict (✅ AC / ❌ WA / ⏱ TLE / 💥 RE)
```

### Wait Mode (`cpt test`)

`cpt test <source>` checks for `*.in` files in the samples dir before compiling:

- **Samples exist** → compile + test immediately (original behavior).
- **No samples** (or `--wait`) → clear stale samples, start a temporary `internal.Server`
  on :27121 (configurable via `-p/--host/--secret`), print a waiting banner, and block
  on a buffered channel fed by `Server.SetOnProblem`. The first problem received stops
  the server and triggers compile + test. `--wait-timeout N` gives up after N seconds;
  Ctrl+C (SIGINT/SIGTERM) aborts with a clean exit via `errWaitAborted`.

Server integration is a 3-line hook: `SetOnProblem(func(count int))` is invoked
inside `handleRequest` right after `SaveSamples` (still under the save mutex), and
`cpt serve` is untouched (callback unset → no-op).

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Go over Rust** | Near-instant incremental compilation (<1s vs 30s+); stdlib covers HTTP/JSON/exec |
| **competitive-companion integration** | Avoids building 100+ OJ parsers; leverages community standard |
| **cobra CLI** | Industry standard, shell completion built-in, battle-tested |
| **Single binary** | Zero runtime deps, trivial distribution (go install / COPR / curl) |
| **Vendored dependencies** | Enables offline COPR builds; `go mod vendor` committed to repo |

---

## Security Model

### Threat Model

The HTTP server (`cpt serve`) receives untrusted JSON from a browser extension. Attack vectors considered:

1. **Remote network access** — attacker on LAN/internet reaches the port
2. **Malicious payload** — oversized JSON, excessive test cases, escape sequences
3. **Local CSRF** — malicious web page sends cross-origin POST
4. **Concurrent requests** — race conditions on shared files
5. **Auto-run abuse** — triggering binary execution with attacker-controlled input

### Defenses (v1.0.1)

| Layer | Mechanism | File:Line |
|-------|-----------|-----------|
| Bind restriction | `127.0.0.1` by default (`--host` to override) | `cmd/serve.go:57` |
| Authentication | Optional `--secret` → `X-CPT-Secret` header check | `server.go:103` |
| Body size limit | `http.MaxBytesReader` at 10 MB | `server.go:115` |
| Rate limiting | 30 req/min sliding window | `server.go:159` |
| Test count cap | Max 100 test cases per problem | `parser.go:50` |
| Output sanitization | Strip ANSI/control chars from task metadata | `server.go:169` |
| CSRF protection | Require `Content-Type: application/json` | `server.go:97` |
| Error sanitization | Generic error messages to client; details to stderr | `server.go:135` |
| Race protection | `sync.Mutex` around save + auto-run | `server.go:128` |
| Async auto-run | Non-blocking goroutine for `--run` | `server.go:142` |
| File permissions | Samples `0600`, directory `0700` | `parser.go:51,58` |

### What We Don't Protect Against

- The user running `cpt run ./malicious_binary` — running untrusted code is inherently dangerous
- Physical access to the machine
- Compromised browser extension (trust in competitive-companion)

---

## Quality Gates Applied

This project was developed following `~/prompt_boilerplates/Coding/development-quality-gates.md` v1.1.0:

| Gate | Status | Evidence |
|------|:------:|----------|
| 1. Cross-module contracts | ✅ | All exported functions grep-verified for consistent callers; `SetOnProblem` used by `cmd/test.go` only |
| 2. Precondition reachability | ✅ | `DetectLang` return values fully covered; all branches reachable; wait mode covered by 7 manual scenarios |
| 3. Boundaries & overflow | ✅ | Binary output in source dir (fixed name, no PID); re-run overwrites; file-not-found gives clear error |
| 4. State consistency | ✅ | Stateless CLI; server mutex protects shared state |
| 5. Documentation sync | ✅ | DEVELOPMENT.md + bilingual README in same commit |
| 6. Test sync | ✅ | 9 manual test scenarios covering happy/error/edge paths |
| 7. Value domain constraints | ✅ | Port 1-65535, timeout ≥1, sample count ≤100 |
| 8. Bilingual README | ✅ | `README.md` + `README_zh.md` with cross-links |
| 9. Security & secrets | ✅ | No hardcoded keys; `.gitignore` covers binaries and artifacts |

### Bug History

| # | Severity | Discovery | Root cause | Fix |
|---|----------|-----------|------------|-----|
| 1 | P0 | RE/TLE errors showed no reason | `printDiff` only called for WA verdict | Show diff for any non-empty error |
| 2 | P0 | Auto-run errors silently swallowed | `errorWriter{}` discarding writes | Direct `fmt.Printf` to stdout |
| 3 | P2 | Same-named source files → binary collision | No unique suffix on temp binary | Add `os.Getpid()` to path (reverted in v1.1.1: binary now kept at `<source_dir>/<basename>` for reuse, aligned with nvim `-o %<`) |
| 4 | P2 | Missing binary reported as RE without explanation | No pre-flight existence check | `os.Stat` check before execution |
| 5 | P1 | Bound to all interfaces (0.0.0.0) | `":%d"` format string | Explicit `127.0.0.1` with `--host` flag |
| 6 | P1 | No request authentication | Missing auth layer | `--secret` + `X-CPT-Secret` header |
| 7 | P1 | Memory exhaustion DoS | Unbounded `io.ReadAll` | `http.MaxBytesReader` at 10 MB |

---

## Build & Test

### Build

```bash
cd ~/Desktop/go-projects/cpt
go build -ldflags="-s -w" -o cpt .
# Binary: ~6.4 MB, zero runtime deps
```

### Run Tests

No unit test files (by design — this is a CLI integration tool). Manual test scenarios:

```bash
# 1. Normal flow
cpt serve --port 27140 &
curl -X POST localhost:27140/ -H 'Content-Type: application/json' -d '{...}'

# 2. cpt test
cpt test solve.cpp

# 3. cpt run
cpt run ./a.out -s 1

# 4. Wait mode — merged serve + test (v1.2.0)
cd /tmp/test
cpt test solve.cpp -p 27123          # no samples → prints waiting banner, blocks
curl -X POST localhost:27123/ -H 'Content-Type: application/json' -d '{...}'
# → saves samples, compiles, tests automatically

# 5. Wait mode edge cases
cpt test solve.cpp --wait-timeout 2  # no samples → clean timeout error, exit 1
cpt test solve.cpp --wait            # with stale samples → cleared, waits for fresh
cpt test solve.cpp --wait -p 27125 & # SIGTERM → "Waiting aborted", clean exit
cpt test solve.cpp -p 27126 --secret abc
curl -X POST localhost:27126/ -H 'Content-Type: application/json' -H 'X-CPT-Secret: abc' ...

# 6. Full security test suite (see commit 8816f3f for details)
```

### Lint

```bash
go vet ./...          # Must pass with zero output
```

---

## Deployment

### Method 1: go install

```bash
go install github.com/xieguaiwu/cpt@latest
```

### Method 2: COPR (Fedora/RHEL)

```bash
sudo dnf copr enable xieguaiwu/cpt
sudo dnf install cpt
```

COPR builds for: `fedora-43-x86_64`, `fedora-44-x86_64`, `fedora-rawhide-x86_64`

### Method 3: Source build

```bash
git clone https://github.com/xieguaiwu/cpt.git
cd cpt
go build -ldflags="-s -w" -o ~/.local/bin/cpt .
```

### COPR Packaging

Spec file: `packaging/fedora/cpt.spec`

To rebuild:
```bash
# After tagging a new version:
gh release create vX.Y.Z
curl -LO https://github.com/xieguaiwu/cpt/archive/vX.Y.Z.tar.gz
cp vX.Y.Z.tar.gz ~/rpmbuild/SOURCES/
cp packaging/fedora/cpt.spec ~/rpmbuild/SPECS/
rpmbuild -bs ~/rpmbuild/SPECS/cpt.spec
copr-cli build xieguaiwu/cpt ~/rpmbuild/SRPMS/cpt-X.Y.Z-1.fc42.src.rpm
```

---

## File Manifest

| File | Purpose | Lines |
|------|---------|-------|
| `main.go` | Entry point | 7 |
| `cmd/root.go` | Cobra root command | 25 |
| `cmd/serve.go` | `cpt serve` command | 75 |
| `cmd/run.go` | `cpt run` command | 33 |
| `cmd/test.go` | `cpt test` command + wait mode | 189 |
| `internal/server.go` | HTTP server + security + onProblem hook | 203 |
| `internal/parser.go` | JSON parsing + file I/O | 70 |
| `internal/compiler.go` | Language detection + compilation | 105 |
| `internal/runner.go` | Test execution + diff engine | 276 |
| `packaging/fedora/cpt.spec` | RPM spec for COPR | 52 |

---

## Known Limitations

1. **No Windows clipboard support** — `golang.design/x/clipboard` works on X11/Wayland only
2. **Java classpath breaks on spaces** — `compiler.go:104` uses `strings.Fields` on a space-containing string
3. **COPR not available for Fedora 42 x86_64** — builds target 43/44/rawhide; users on 42 can use `go install`
4. **competitive-companion must be installed separately** — cpt is a companion, not a replacement
5. **No persistent problem history** — samples are overwritten on each new problem receipt
6. **Single HTTP worker** — mutex serializes all save operations (acceptable for single-user tool)

---

## Future Directions

- [ ] `cpt list` — list cached samples
- [ ] `cpt clean` — remove sample files
- [ ] Shell completion generation (cobra built-in, just needs wiring)
- [ ] `cpt config` — persistent config file (~/.cpt/config.toml)
- [ ] Watchdog mode — auto-detect file changes and re-run tests
- [ ] Multi-testcase problems (batch type) from competitive-companion
- [ ] Memory limit enforcement (currently only time limit)
- [ ] TOML/YAML config for per-project settings

---

## Changelog

### v1.2.0 (2026-08-02)
- **Wait mode**: `cpt test` starts a temporary server and blocks until competitive-companion delivers a problem when no samples exist — merging `cpt serve` + `cpt test` into one command
- New `--wait` (force fresh problem, clears stale samples), `--wait-timeout` (0 = forever), `-p/--host/--secret` flags on `cpt test`
- `internal.Server` gains a `SetOnProblem` callback hook (no-op when unset; `cpt serve` behavior unchanged)

### v1.0.1 (2026-07-26)
- Security: 127.0.0.1 bind, secret auth, rate limiting, body size cap, test count cap, error sanitization, CSRF protection, file permission hardening

### v1.0.0 (2026-07-26)
- Initial release: HTTP server, 6-language compilation, colored test runner, bilingual README, COPR packaging
