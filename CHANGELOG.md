# Changelog

All notable changes to Cavaleiro Local LLM are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses [Semantic Versioning](https://semver.org/) for its own releases. The `ollama --version` string is set at build time and is independent of this version.

The original project does not keep a changelog file; its history is in its release notes and commit log.

## [Unreleased]

## [1.0.0] - 2026-10-17

First release of Cavaleiro Local LLM, based on commit `8be580ad0939c1ea528ba5ece9d585f095190e06` of the original project (2026-10-17, "mlx: add PPLX Decider v1.2 support"). The `ollama` command, the Go module path, the `OLLAMA_*` environment variables, the REST API, and the default port `11434` are unchanged. Details and evidence for every entry are in [docs/cavaleiro/ENGINEERING.md](docs/cavaleiro/ENGINEERING.md).

### Fixed

- **Server: `OLLAMA_ORIGINS` with spaces or an empty entry crashed the server at startup.** `OLLAMA_ORIGINS="https://a.example, https://b.example"` produced the origin `" https://b.example"`, which gin-contrib/cors rejects with a panic (`bad origin: origins must contain '*' or include http://,https://,...`) while the routes are built. Entries are now trimmed and empty ones skipped (`envconfig/config.go`). Original issue #18011.
- **Modelfile `FROM ~\path` on Windows.** `~\models\model.gguf` and `~user\models` were treated as a user named `\models\model.gguf` and failed with `failed to find user`. Both separators are accepted on Windows; behavior on Linux and macOS is unchanged (`parser/parser.go`).
- **API client: error responses without a body were reported as success.** A streaming call (`Pull`, `Push`, `Create`, `Chat`, `Generate`) that received, for example, `502 Bad Gateway` with an empty body from a reverse proxy returned `nil`. It now returns a `StatusError` (or `AuthorizationError` for 401) carrying the HTTP status. Blank keep-alive lines in an NDJSON stream are skipped instead of surfacing as an empty error, and non-streaming calls with an empty error body report the HTTP status instead of "something went wrong" (`api/client.go`).
- **CLI panics on short digests.** `ollama list` and `ollama ps` panicked with `slice bounds out of range [:12]` when a server reported a digest shorter than 12 characters (original issue #14250); `ollama create` and `ollama push` panicked with `[:19]` on layer progress without a status message and a short digest; `ollama push` dereferenced a nil spinner when the server sent only layer progress (`cmd/cmd.go`).
- **Windows: process detection panicked on machines with more than 512 processes.** `IsProcRunning` in the desktop app's updater treated the byte count returned by `EnumProcesses` as a number of process IDs and sliced past its 2048-entry buffer (`slice bounds out of range [:2948] with capacity 2048` on a machine with 737 processes). Both copies (`app/updater/updater_windows.go`, `cmd/start_windows.go`) now use a Toolhelp process snapshot.

### Security

- **A malformed registry `WWW-Authenticate` header could crash the server.** `getValue` assumed a quoted value and sliced past the end of `Bearer realm=` (`slice bounds out of range [:7] with length 6`), so a malicious registry, or an on-path attacker during an `--insecure` pull, could terminate `ollama serve`. Missing values now return an empty string and unquoted RFC 7235 token values are read whole (`server/images.go`). Original issue #17034.

### Changed

- **Windows process detection is more than 100x faster.** Finding a process by name (used by `ollama` to avoid starting the app during an upgrade, and by the desktop updater) opened every process and enumerated its modules; on a machine with about 680 processes that took 4.8–39 s per call in four runs, against 20–37 ms with a Toolhelp snapshot, which also sees processes the current user cannot open.
- Rebranded the project as **Cavaleiro Local LLM**: README, community files, issue forms, and repository links point to `cavaleiro-ai/cavaleiro-local-llm`. Code identifiers are unchanged (see the README's *Compatibility* section).
- `LICENSE` keeps the original MIT text and copyright line, and adds the derivative notice and `Copyright (c) 2026 Cavaleiro AI (modifications in Cavaleiro Local LLM)`.
- The community integrations list moved from the README to [docs/community-integrations.md](docs/community-integrations.md).
- The inherited manuals in `docs/` use the Cavaleiro Local LLM name for the product. Pages about the official installers, ollama.com accounts, cloud models, and web search keep the names of those services, and third-party UI labels are unchanged. The README ends with a one-line license notice.
- `.github/workflows/test.yaml` declares `permissions: contents: read`; its self-hosted native superbuild jobs moved to `.github/workflows-disabled/test-native-selfhosted.yaml`.

### Added

- Regression tests for every fix above: `server/auth_test.go`, `server/cors_origins_test.go`, `envconfig/config_test.go`, `parser/expandpath_test.go`, `api/client_test.go`, `cmd/progress_digest_test.go`, `cmd/start_windows_test.go`.
- [README.pt-BR.md](README.pt-BR.md), a Portuguese guide ([docs/cavaleiro/GUIA-PT-BR.md](docs/cavaleiro/GUIA-PT-BR.md)), and engineering notes ([docs/cavaleiro/ENGINEERING.md](docs/cavaleiro/ENGINEERING.md)).
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) (Contributor Covenant 2.1), a pull request template, issue forms (bug, feature, model support), and Dependabot for GitHub Actions.
- Windows testing notes in [CONTRIBUTING.md](CONTRIBUTING.md): which packages need cgo and how to use a MinGW compiler for a test run.

### Removed (fork safety)

- Workflows that publish, sign, or depend on the original project's infrastructure were moved to [.github/workflows-disabled/](.github/workflows-disabled/README.md): `release.yaml` (signing, Docker Hub, GitHub releases), `latest.yaml` (Docker Hub), `test-install.yaml` (installs official binaries from ollama.com), `test-llamacpp-update.yaml` (self-hosted and large paid runners), `test-tokenizer.yaml` (downloads from the ollama.com model registry), and the self-hosted jobs of `test.yaml`.
- The issue chooser links only to this repository's security form, contributing guide, and troubleshooting guide; community and issue-tracker links of the original project were removed from the public files and manuals.

[Unreleased]: https://github.com/cavaleiro-ai/cavaleiro-local-llm/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/cavaleiro-ai/cavaleiro-local-llm/releases/tag/v1.0.0
