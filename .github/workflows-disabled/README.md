# Disabled workflows

GitHub only runs workflows stored in `.github/workflows/`. The files in this folder came from the original project and were moved here on purpose, because running them from Cavaleiro Local LLM would publish under the original project's identity, need credentials only its organization has, or depend on runners and services this fork does not control.

| Workflow | Why it is disabled |
|---|---|
| `release.yaml` | Builds, signs, and notarizes release artifacts (Apple and Google signing secrets), pushes Docker images to Docker Hub as `ollama/ollama`, and creates GitHub releases. Uses the `release` environment, non-`GITHUB_TOKEN` secrets, and self-hosted runners (`linux`, `windows`). |
| `latest.yaml` | Logs in to Docker Hub (`DOCKER_ACCESS_TOKEN`) and retags the original project's images as `latest`. |
| `test-install.yaml` | Runs `scripts/install.sh`, which downloads and installs the official binaries from `ollama.com`, not this fork. |
| `test-llamacpp-update.yaml` | Builds full release payloads on self-hosted (`linux`, `linux-arm64`, `windows`) and large paid (`macos-26-xlarge`) runners. |
| `test-native-selfhosted.yaml` | The `linux` and `windows` native superbuild jobs split out of `test.yaml`; they run on the self-hosted `linux` and `windows` runners. |
| `test-tokenizer.yaml` | Downloads tokenizer data from the ollama.com model registry (`registry.ollama.ai`) and fails if any case is skipped. |

The remaining workflows in `.github/workflows/` (`test.yaml`, `test-xgrammar.yaml`) run on GitHub-hosted runners with a read-only `GITHUB_TOKEN`. On macOS, `test.yaml` and `test-xgrammar.yaml` call `.github/scripts/prepare_mlx_darwin.sh`, which reads a matching MLX payload from the original project's public GitHub releases (read-only, configurable with `OLLAMA_MLX_RELEASE_REPO`) and builds it from source when none matches.

## Re-enabling a workflow

1. Decide where the artifacts go: your own Docker Hub or GHCR namespace, your own GitHub releases, your own signing identities. Never reuse the `ollama/ollama` image name, the `ollama.com` download URLs, or the original project's signing identities.
2. Replace the image names, release URLs, environment names, and runner labels in the file (`runs-on: linux` and `runs-on: windows` are self-hosted labels; use `ubuntu-latest`/`windows-latest` or register your own runners).
3. Create the secrets the file reads (`DOCKER_ACCESS_TOKEN`, `APPLE_IDENTITY`, `APPLE_PASSWORD`, `MACOS_SIGNING_KEY`, `MACOS_SIGNING_KEY_PASSWORD`, `GOOGLE_SIGNING_CREDENTIALS`) and variables (`DOCKER_USER`, `DOCKER_REPO`, `APPLE_ID`, `APPLE_TEAM_ID`, `KEY_CONTAINER`, `OLLAMA_CERT`) in your repository settings, and a protected `release` environment. The variable names are kept for compatibility; their values must be your own.
4. Give each job the narrowest `permissions:` it needs, then move the file back to `.github/workflows/` and test it on a fork or a pre-release tag first.
