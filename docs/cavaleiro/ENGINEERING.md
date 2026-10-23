# Cavaleiro Local LLM — engineering notes

This document records how Cavaleiro Local LLM 1.0.0 was derived from the original project: the architecture map used to find problems, the test baseline, every change with its evidence, what was not done, and what to do next. The Portuguese user guide is [GUIA-PT-BR.md](GUIA-PT-BR.md); the user-facing summary is the [CHANGELOG](../../CHANGELOG.md).

- **Original project:** branch `main`, commit `8be580ad0939c1ea528ba5ece9d585f095190e06` (2026-10-17, "mlx: add PPLX Decider v1.2 support"), MIT License.
- **Test machine:** Windows 10 Pro 19045, 16 CPUs, Go 1.26.5, about 680–740 running processes, shared with other heavy jobs (timings are noisy).

## 1. Architecture map

Cavaleiro Local LLM is a single Go binary, `ollama`, that is both the CLI and the HTTP server. Inference runs in native child processes.

```
 clients: ollama CLI, curl, ollama-python/js, OpenAI & Anthropic SDKs, `ollama launch` integrations
     │ HTTP, NDJSON streaming
     ▼
 server/ (gin) ── CORS (envconfig.AllowedOrigins) ── allowedHostsMiddleware (DNS-rebinding guard)
     ├─ /api/generate, /api/chat, /api/embed ──► Scheduler (sched.go) ──► llm/ ──► llama-server (llama.cpp)
     │      template/, model/renderers (prompt)            └──────────► mlxrunner/ (MLX, cgo)
     │      model/parsers, thinking/, harmony/, tools/ (output)        discover/ (GPU detection)
     ├─ /v1/* (middleware/: OpenAI, Anthropic → native handlers; cloud passthrough)
     ├─ /api/pull, /api/push ──► images.go (registry auth), download.go, upload.go, transfer/
     ├─ /api/create, /api/blobs ──► create.go, create/ (safetensors import, quantize), fs/gguf, fs/safetensors
     └─ /api/tags, /api/show, /api/ps, /api/delete, /api/copy ──► manifest/, types/model/
 storage: OLLAMA_MODELS (~/.ollama/models): manifests/, manifests-v2/, blobs/sha256-<hex>
```

| Component | Responsibility | Entry points |
|---|---|---|
| `main.go`, `cmd/` | Cobra CLI: `serve`, `run`, `pull`, `push`, `create`, `list`, `ps`, `show`, `stop`, `cp`, `rm`, `signin`, `launch`… | `cmd.NewCLI` |
| `cmd/launch/` | Integrations that configure third-party agents (Claude Code, Codex, OpenCode, OpenClaw, Copilot CLI, Droid, VS Code…) | `ollama launch <name>` |
| `api/` | Request/response types and the Go client (`Client.stream` for NDJSON, `Client.do` for JSON) | `api.ClientFromEnvironment` |
| `server/` | Routes (`routes.go`), scheduler (`sched.go`), registry pull/push (`images.go`), create (`create.go`), cloud proxy | `server.Serve`, `(*Server).GenerateRoutes` |
| `middleware/`, `openai/`, `anthropic/` | OpenAI/Anthropic compatibility layers | gin middlewares on `/v1/*` |
| `llm/`, `llama/` | `llama-server` process management; llama.cpp build glue and patches | `llm.NewLlamaServer` |
| `mlx/`, `mlxrunner/` | MLX bindings (cgo, dlopen of mlx-c), model architectures, tokenizer, xgrammar | `ollama runner` |
| `discover/` | CPU/GPU discovery (CUDA, ROCm, Vulkan, Metal) | `discover.GPUDevices` |
| `envconfig/` | `OLLAMA_*` environment variables and `~/.ollama/server.json` | package functions |
| `manifest/`, `types/model/` | Model names, manifests, blob paths, digest validation | `model.ParseName`, `manifest.BlobsPath` |
| `parser/` | Modelfile parser and path expansion | `parser.ParseFile`, `Modelfile.CreateRequest` |
| `template/`, `model/renderers/`, `model/parsers/`, `thinking/`, `harmony/`, `tools/` | Prompt rendering and output parsing per model family | used by chat/generate handlers |
| `progress/`, `readline/`, `format/` | CLI rendering | — |
| `app/` | Desktop app: Windows tray, macOS app, React UI (`app/ui/app`), updater, SQLite store | `app/cmd/app` |
| `auth/` | ed25519 key signing for ollama.com | — |
| `integration/` | End-to-end tests (`//go:build integration`, need real models) | — |

**Configuration:** environment variables (`envconfig/config.go`, listed by `ollama serve --help`) and `~/.ollama/server.json` (`disable_ollama_cloud`).

**Extension points:** Modelfile commands, `cmd/launch` integrations, renderers/parsers per model family, MLX architectures (`mlxrunner/model/*`), GPU backends selected in CMake.

**Build and release:** CMake superbuild (`CMakeLists.txt`, `cmake/`, `llama/server`, `mlx/`) builds the Go binary and the native payload; `Dockerfile` builds images; `scripts/` package installers. The inherited CI (`.github/workflows/test.yaml`) runs `go test` with cgo on Ubuntu, macOS, and Windows, plus golangci-lint; releases are built, signed, and published by `release.yaml` (disabled here, see §6).

**Tests:** standard `*_test.go` per package; benchmarks are smoke-run with `-bench=. -benchtime=1x` in CI; `integration/` is opt-in.

## 2. Baseline

Two runs were needed because no C compiler is installed on the test machine and the Visual Studio compiler cannot be used with cgo. A portable [llvm-mingw](https://github.com/mstorsjo/llvm-mingw) 20261006 toolchain was unpacked in a scratch folder and passed with `CC` for run B only; nothing was installed globally. Both runs used an isolated `HOME`/`USERPROFILE`, `-count=1 -p 4 -timeout 30m -json`, against an untouched clone of the original project.

| Run | Command | Packages | Tests pass / fail / skip | Duration |
|---|---|---|---|---|
| A — pure Go | `CGO_ENABLED=0 go test -count=1 -p 4 <99 packages>` (all of `go list ./...` except `app/ui`, `app/cmd/app`, `app/webview`, `mlxrunner/xgrammar`) | 41 pass, 40 fail, 18 without tests | 6708 / 85 / 165 | 1231 s |
| B — cgo | `CGO_ENABLED=1 CC=x86_64-w64-mingw32-clang go test -count=1 -p 4 <39 packages that failed to build or need cgo in run A, plus app/cmd/app, app/webview, mlxrunner/xgrammar>` | 25 pass, 3 fail, 11 without tests | 3953 / 3 / 417 | 672 s |

`go vet` (run A packages, `CGO_ENABLED=0`): only pre-existing warnings (`possible misuse of reflect.SliceHeader/unsafe.Pointer` in `app/dialog`, `app/wintray`, `discover`) plus the cgo-only packages that do not compile without cgo.

**Packages that need cgo** (fail to build with `CGO_ENABLED=0`): the root package, `cmd`, `create`, `server`, `mlx`, `mlx/generator`, `mlx/mlxtest`, `mlxrunner` and all `mlxrunner/...` packages (they import `mlx`), `mlxrunner/xgrammar`, `app/webview`, `app/cmd/app`; `app/store`, `app/server`, `app/updater`, and `cmd/launch` compile but their SQLite tests need cgo.

**Pre-existing failures with cgo enabled, and their causes:**

| Test | Cause |
|---|---|
| `app/updater` `TestIsInstallerRunning` | Real bug, fixed here (§4.6): panics with more than 512 processes. |
| `cmd/launch` `TestCodexAppManagedAuthLifecycle` | Asserts POSIX file mode `0600`; Windows reports `0666`. Test portability, not changed. |
| `app/cmd/app` (build) | Embeds the React UI bundle (`app/dist`), which needs `npm ci && npm run build`. Not built. |

**Pre-existing failures in pure-Go packages (run A), unrelated to cgo:**

| Test | Cause |
|---|---|
| `compatmigrate` `TestEnsureLocalCompatibilityMigrationRepairsDanglingV2Entry`, `manifest` `TestResolvePathForNameFallsBackWhenV2EntryDangling`, `TestResolvePathForNameDanglingWithoutLegacyIsNotExists` | Create symbolic links; Windows refuses without Developer Mode or administrator rights ("A required privilege is not held by the client"). |
| `parser` `TestCreateRequestFileGlobNoMatchUsesModelName` | Creates a file named `model-*.gguf`; `*` is not allowed in Windows file names. |
| `transfer` `TestDownloadParallelism` | Wall-clock budget (1.5 s) exceeded on a loaded machine (6.6 s); timing-sensitive. |

## 3. Findings

| # | Finding | Severity | Status | Evidence / test |
|---|---|---|---|---|
| 1 | Malformed registry `WWW-Authenticate` header (`Bearer realm=`) panics `getValue` during pull/push; unquoted values lose their first character (original issue #17034) | High (remote DoS of `ollama serve` by a malicious registry or on-path attacker on `--insecure`) | Fixed | `server/auth_test.go` `TestParseRegistryChallengeMalformed` |
| 2 | `OLLAMA_ORIGINS` entries are not trimmed: `"a, b"` or a trailing comma makes gin-contrib/cors panic while building routes, so the server never starts (original issue #18011 reports it as a silent CORS failure; with gin-contrib/cors v1.7.2 it is a startup panic) | High (availability, easy misconfiguration) | Fixed | `envconfig/config_test.go` `TestOriginsTrimsEntries`, `server/cors_origins_test.go` `TestGenerateRoutesAllowedOriginsWithSpaces` |
| 3 | `IsProcRunning` (desktop updater) slices its 2048-entry PID buffer with the **byte** count from `EnumProcesses` → panic with more than 512 processes; both copies open every process and enumerate modules, which takes seconds to minutes | Medium (desktop app crash on Windows; CLI start-up delay) | Fixed (Toolhelp snapshot) | `app/updater` `TestIsInstallerRunning` (panicked before), `cmd/start_windows_test.go`; measurement in §4.6 |
| 4 | `ollama list`/`ps` slice `Digest[:12]` (original issue #14250); `create`/`push` slice `Digest[7:19]` when a progress line has no status; `push` calls `Stop` on a nil spinner when the server sends only layer progress | Medium (CLI panics against non-conforming or third-party servers) | Fixed | `cmd/progress_digest_test.go` |
| 5 | `api.Client.stream` returns `nil` for an error status with an empty body (e.g. `502` from a proxy), so `pull`/`push`/`create` look successful; blank NDJSON lines become `errors.New("")`; `Client.do` reports "something went wrong" for an empty error body | Medium (silent failures) | Fixed | `api/client_test.go` `TestClientStreamErrorStatusWithEmptyBody`, `TestClientStreamSkipsBlankLines`, `TestClientDoErrorStatusWithEmptyBody` |
| 6 | Modelfile `FROM ~\models\x.gguf` (and `~user\...`) fails on Windows with "failed to find user" | Low–Medium (Windows usability) | Fixed | `parser/expandpath_test.go` (new Windows cases) |
| 7 | `envconfig.Bool` treats any unparsable value (`off`, `no`) as **true** | Low (surprising, but pinned by `TestBool` as intended behavior) | Not changed — documented contract | `envconfig/config_test.go` `TestBool` |
| 8 | `OLLAMA_KEEP_ALIVE`/`OLLAMA_LOAD_TIMEOUT` integer seconds overflow `int64` nanoseconds for values above ~292 years | Low (contrived input) | Not changed | code reading |
| 9 | Tests that assume POSIX file modes, symlink privileges, or `*` in file names fail on Windows | Low (test portability) | Not changed (documented in §2) | baseline logs |
| 10 | Self-hosted, signing, Docker Hub, and registry-dependent workflows would run under the fork | High (fork safety) | Disabled | §6 |
| 11 | `staticcheck` SA5011: `regOpts.Token` dereferenced before a `regOpts != nil` check in `server/images.go` transfer helpers | Low (no production caller passes nil) | Not changed — follow-up | golangci-lint, §5 |

## 4. Improvements and verification

Each bug fix has a regression test that fails on the original code and passes after the fix. "Before" runs used a scratch copy of the original tree with only the new tests added; "after" runs used this repository.

### 4.1 Registry challenge parsing — `server/images.go` (`getValue`)

The index skipped `len(key)+2` characters assuming `key="`, then sliced `header[startIdx:endIdx]` even when `startIdx` was past the end. The fix moves past `key=` only, returns `""` when no value follows, reads an unquoted token up to the next comma (RFC 7235), and keeps the existing quoted-string loop.

- Before: `parseRegistryChallenge("Bearer realm=") panicked: runtime error: slice bounds out of range [:7] with length 6`; unquoted `realm=https://…` parsed as `ttps://…,service=registry,…`.
- After: `TestParseRegistryChallenge` and `TestParseRegistryChallengeMalformed` pass.

### 4.2 `OLLAMA_ORIGINS` trimming — `envconfig/config.go` (`AllowedOrigins`)

Entries are trimmed with `strings.TrimSpace` and empty entries are skipped. Defaults and order are unchanged; the existing `TestOrigins` cases still pass.

- Before: `GenerateRoutes panicked: bad origin: origins must contain '*' or include http://,https://,chrome-extension://,…` for `OLLAMA_ORIGINS="https://a.example, https://b.example,"`; `AllowedOrigins` returned `" https://b.example"` and `""`.
- After: both origins receive `Access-Control-Allow-Origin`; no empty or padded entries.

### 4.3 Windows `~\` paths in Modelfiles — `parser/parser.go` (`expandPathImpl`)

`~` followed by any path separator (`os.IsPathSeparator`, which accepts `\` and `/` on Windows and only `/` elsewhere) expands to the current user's home; `~user` ends at the first separator. Linux/macOS behavior is identical.

- Before: `expandPathImpl("~\\myfolder\\myfile.txt") returned error` (lookup of a user named `\myfolder\myfile.txt`).
- After: `TestExpandPath` passes with the new cases `~\myfolder\myfile.txt`, `~anotheruser\docs\file.txt`, and `~nonexistentuser\file.txt` (error).

### 4.4 Go API client error handling — `api/client.go`

- `stream`: an error status that reaches the end of the body without any record (empty body) now returns `AuthorizationError` (401) or `StatusError` with `Status`; whitespace-only lines are skipped.
- `checkError` (used by `do`): when the body yields no message, `Status` is set so the error reads `502 Bad Gateway` instead of the generic text. Responses with a message are unchanged (existing `TestClientDo` and `TestHeadBlob` cases pass).
- Before: `expected an error for an empty error response, got nil` (502/404/401); `Pull() error = ` (empty error from a blank line); `error = "something went wrong, please see the ollama server logs for details", want "502 Bad Gateway"`.
- After: the three new tests pass.

### 4.5 CLI digest and spinner panics — `cmd/cmd.go`

`ListHandler`/`ListRunningHandler` use `m.Digest[:min(12, len(m.Digest))]` (the shape of the original project's pull request for issue #14250); `CreateHandler`/`PushHandler` use a new `shortDigest` helper (strip `sha256:`, keep at most 12 characters — identical output for real digests); `PushHandler` checks the spinner for `nil` like the code above it already does.

- Before: `list panicked on a short digest: runtime error: slice bounds out of range [:12] with length 3` (same for `ps`); `PushHandler panicked: … [:19] with length 10`; with a full digest and no status, `PushHandler panicked: runtime error: invalid memory address or nil pointer dereference`; `CreateHandler panicked: … [:19] with length 10`.
- After: `TestListHandlersShortDigest`, `TestPushHandlerProgressWithoutStatus` (both cases), `TestCreateHandlerProgressShortDigest` pass, as do the existing `TestListHandler`, `TestListRunningHandlerShowsRunner`, `TestPushHandler`, `TestCreateHandler`.

### 4.6 Windows process detection — `app/updater/updater_windows.go`, `cmd/start_windows.go`

`windows.EnumProcesses` reports bytes written; the updater copy used that number as a slice length (`pids[:ret]`), so 737 processes (2948 bytes) panicked against the 2048-entry buffer. The CLI copy (used by `ollama` to avoid starting the app during an upgrade) re-allocated on the same misreading and did not panic. Both copies also opened every process and enumerated its modules. They now walk a `CreateToolhelp32Snapshot` process list and compare `ProcessEntry32.ExeFile`, which needs no process handles and also sees processes the user cannot open.

- Before: `TestIsInstallerRunning` → `panic: runtime error: slice bounds out of range [:2948] with capacity 2048` (also in the baseline run on the original code: `[:2608]`).
- After: `TestIsInstallerRunning` and the new `TestIsProcRunningFindsCurrentProcess` pass in 0.02 s each. With the byte-count fix alone and the module scan kept, the same tests took 84–107 s and 0.1–149 s on the loaded machine, which is why the scan itself was replaced.
- Measurement (a standalone program comparing both strategies on the same process list, four runs): module scan 9.6 s, 39.1 s, 4.8 s, 18.4 s for about 680 processes; Toolhelp snapshot 31 ms, 37 ms, 28 ms, 20 ms.

### 4.7 Documentation and developer experience

- README rebuilt for the fork (install from source, compatibility table, network services of the original project, license line); [README.pt-BR.md](../../README.pt-BR.md); [GUIA-PT-BR.md](GUIA-PT-BR.md); this file; [CHANGELOG](../../CHANGELOG.md).
- [CONTRIBUTING.md](../../CONTRIBUTING.md) and [docs/development.md](../development.md) explain which packages need cgo and how to run them on Windows with a MinGW compiler.
- Community files: [SECURITY.md](../../SECURITY.md) (private reporting, scope), [CODE_OF_CONDUCT.md](../../CODE_OF_CONDUCT.md), issue forms, pull request template, Dependabot for Actions.
- The community integrations list moved to [docs/community-integrations.md](../community-integrations.md).

## 5. Final results

The same two commands were re-run against this repository after all changes.

| Run | Tests pass / fail / skip (baseline → final) | Packages pass / fail | Duration |
|---|---|---|---|
| A — pure Go (`CGO_ENABLED=0`, same 99 packages) | 6708 / 85 / 165 → **6716 / 84 / 165** | 41 / 40 → 41 / 40 | 1231 s → 536 s |
| B — cgo (llvm-mingw, same 39 packages) | 3953 / 3 / 417 → **3965 / 7 / 417** | 25 / 3 → 25 / 3 (`app/updater` now passes; `app/store` flaked, see below) | 672 s → 1284 s (machine under heavier load) |

- Run A: +8 passing tests (the new `envconfig` and `api` tests, and `app/updater` `TestIsInstallerRunning`, which no longer panics); the remaining 84 failures are the same cgo-stub, symlink, file-name, and timing failures listed in §2. The 40 failing packages are the same packages as in the baseline (most cannot build without cgo).
- Run B: +12 passing tests: the new `server` and `cmd` tests and `TestIsInstallerRunning` (+17), minus five `app/store` subtests (`TestAppCompletionReachesCLI/existing_database`, `TestLegacyAppCompletionReachesCLI/1` and `/16`, and their parents) that failed once under load. `app/store` is not modified by this project; re-running it alone passed twice (`go test -count=1 ./app/store`: `ok` in 17.8 s and 8.2 s). The other failures are the pre-existing `cmd/launch` file-mode test and the `app/cmd/app` UI-bundle build.
- Every new regression test fails on the original tree and passes here (§4).
- `gofmt -l` reports nothing and `go vet` (with cgo) passes for `api`, `envconfig`, `parser`, `server`, `cmd`, and `app/updater`. `golangci-lint` v2.5.0 with the repository's `.golangci.yaml` (run through `go run`, cgo enabled) on the same packages reports 6 findings, all on unchanged original lines: two `gofumpt` formatting notes (`cmd/cmd.go` naked `return` in a closure, `server/routes_generate_test.go`) and `staticcheck` SA5011 in `server/images.go` (`regOpts.Token` is read before a later `regOpts != nil` check in `downloadWithTransfer`/`pushWithTransfer`; every production caller passes a non-nil value, so it is recorded as a follow-up rather than changed). The inherited CI pins no linter version, so its results may differ.
- `git diff --check` on the changed files reports only two trailing-whitespace lines that already exist in the original `docs/development.md` (not reformatted).

## 6. Fork safety

| Workflow | Action | Reason |
|---|---|---|
| `release.yaml` | Moved to `.github/workflows-disabled/` | Apple/Google signing secrets, Docker Hub `ollama/ollama` pushes, GitHub releases, self-hosted runners, `release` environment |
| `latest.yaml` | Moved | Docker Hub login and retagging |
| `test-install.yaml` | Moved | Installs official binaries from ollama.com |
| `test-llamacpp-update.yaml` | Moved | Self-hosted and paid large runners |
| `test-tokenizer.yaml` | Moved | Downloads from the ollama.com registry (`registry.ollama.ai`) and fails on skips |
| `test.yaml` | Kept; `permissions: contents: read` added; self-hosted `linux`/`windows` jobs moved to `workflows-disabled/test-native-selfhosted.yaml` | GitHub-hosted jobs need only `GITHUB_TOKEN` |
| `test-xgrammar.yaml` | Kept | GitHub-hosted, read-only |

`.github/scripts/prepare_mlx_darwin.sh` (used by the kept macOS jobs) reads public MLX payloads from the original project's GitHub releases (`OLLAMA_MLX_RELEASE_REPO`, default `ollama/ollama`) and builds from source when none matches; it was kept because the fork publishes no MLX payloads and the access is read-only. Actions stay pinned as the original project pinned them (`@v4`/`@v5` tags). The original project ships no `CODEOWNERS`, `FUNDING.yml`, or bot configuration. The issue chooser links only to this repository.

**Identifiers kept:** module path `github.com/ollama/ollama`, binary `ollama`, `OLLAMA_*` variables, `~/.ollama`, API routes, port 11434, registry `registry.ollama.ai`/`ollama.com`, the desktop app and its in-product branding, ollama.com URLs in code (cloud, web search, updater, model recommendations). The inherited manuals in `docs/` use the Cavaleiro Local LLM name for the product; they keep the names of ollama.com services (accounts, keys, cloud, web search, library), of the official installers and the `ollama/ollama` image, of version numbers, of third-party UI labels, and the desktop app's in-product name, and their community links point to this repository's issues.

## 7. What was not done, and why

- **Full native build and GPU tests** (CMake superbuild, CUDA/ROCm/Vulkan/Metal, `llama-server`, MLX runtime): needs CMake and GPU SDKs; out of scope for this machine. MLX tests skip without the runtime library (most of the 417 skips in run B).
- **`app/ui` and `app/cmd/app`**: need the React bundle (`npm ci && npm run build`); not built, so their Go tests were not run. `npm test` for the UI was not run.
- **Repository-wide golangci-lint**: run only on the changed packages (see §5); the pre-existing findings there were not fixed to avoid churn in unrelated lines.
- **Integration tests** (`integration/`, build tag `integration`): require pulling real models, which this work avoids.
- **Findings 7–9** were left as they are: changing `envconfig.Bool` would break a pinned contract, the overflow needs absurd input, and the Windows test portability issues are environment assumptions rather than product bugs. Skipping those tests would weaken the suite.
- **`docs/docs.json`** keeps the Mintlify site configuration of the original project (logo, navbar links to ollama.com); only the site name was changed.

## 8. Known limitations

- No binary releases or Docker images are published for this fork; the desktop app's updater installs releases of the original project if left enabled.
- `ollama pull`, cloud models, web search, and model recommendations depend on ollama.com, as in the original project.
- The Windows-specific fixes were verified on Windows 10 only; the Linux/macOS behavior of the shared code is covered by the unchanged original tests.

## 9. Recommended next steps

1. Run the kept CI workflow on GitHub to confirm the Ubuntu and macOS legs, and resolve the pre-existing golangci-lint findings listed in §5 (make `regOpts` handling consistent in `server/images.go`).
2. Offer fixes 4.1, 4.2, 4.4–4.6 to the original project (original issues #17034, #18011, #14250 are still open) to reduce divergence.
3. Make the Windows-incompatible tests portable (skip symlink tests when the privilege is missing, avoid `*` in file names, compare ACL-based permissions on Windows).
4. Decide on a release channel (signing identity, image namespace) before re-enabling any workflow in `.github/workflows-disabled/`.
5. Consider rejecting invalid `OLLAMA_ORIGINS` entries with a clear error instead of relying on gin-contrib/cors to panic.
