# Contributing to Cavaleiro Local LLM

Thank you for your interest in contributing to Cavaleiro Local LLM! It stays compatible with the `ollama` CLI and the REST API of the original project, so most contributions are focused fixes and tests in the Go code. Here are a few guidelines to help get you started.

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md). Report vulnerabilities privately, as described in the [security policy](SECURITY.md).

## Set up

See the [development documentation](./docs/development.md) for instructions on how to build and run the project locally, and the [engineering notes](./docs/cavaleiro/ENGINEERING.md) for the package map and the exact test commands used for each release.

### Running the tests

Most packages are pure Go:

```shell
go test ./envconfig/... ./parser/... ./api/... ./types/...
```

Several packages need cgo even for their unit tests: the root package, `cmd`, `server`, `create`, `mlx/...`, `mlxrunner/...`, and the desktop `app/...` (`app/store` uses SQLite through cgo). They need a GCC-compatible C compiler on `PATH`, as for a full build.

- **Linux and macOS:** the system `gcc` or `clang` is enough.
- **Windows:** the Visual Studio compiler (`cl.exe`) does not work with cgo. Use a MinGW toolchain such as [llvm-mingw](https://github.com/mstorsjo/llvm-mingw) or MSYS2, and point Go at it for the test run only:

  ```powershell
  $env:CGO_ENABLED = "1"
  $env:CC = "C:\path\to\llvm-mingw\bin\x86_64-w64-mingw32-clang.exe"
  go test ./server/... ./cmd/...
  ```

`app/ui` and `app/cmd/app` also embed the desktop UI bundle, so build it first with `npm ci && npm run build` in `app/ui/app`. Some tests create symbolic links and are reported as failures on Windows unless Developer Mode is enabled or the shell runs as administrator.

### Ideal issues

* [Bugs](https://github.com/cavaleiro-ai/cavaleiro-local-llm/issues?q=is%3Aissue+is%3Aopen+label%3Abug): issues where the server or CLI stops working or where it results in an unexpected error.
* [Performance](https://github.com/cavaleiro-ai/cavaleiro-local-llm/issues?q=is%3Aissue+is%3Aopen+label%3Aperformance): issues to make model inference, downloading or uploading faster.
* [Security](SECURITY.md): issues that could lead to a security vulnerability. As mentioned in [SECURITY.md](SECURITY.md), please do not disclose security vulnerabilities publicly.

Bugs that also affect the original project are welcome here.

### Issues that are harder to review

* New features: new features (e.g. API fields, environment variables) add surface area and make the project harder to maintain in the long run as they cannot be removed without potentially breaking users in the future.
* Refactoring: large code improvements are important, but can be harder or take longer to review and merge, and they make it harder to keep up with the original project.
* Documentation: small updates to fill in or correct missing documentation are helpful, however large documentation additions can be hard to maintain over time.

### Issues that may not be accepted

* Changes that break backwards compatibility with the REST API (including the OpenAI- and Anthropic-compatible APIs), the `ollama` command, the `OLLAMA_*` environment variables, or the default port
* Changes that add significant friction to the user experience
* Changes that create a large future maintenance burden for maintainers and contributors

## Proposing a (non-trivial) change

> By "non-trivial", we mean a change that is not a bug fix or small
> documentation update. If you are unsure, please ask in a
> [GitHub issue](https://github.com/cavaleiro-ai/cavaleiro-local-llm/issues).

Before opening a non-trivial Pull Request, please open an issue to discuss the change and
get feedback from the maintainers. This helps us understand the context of the
change and how it fits into the project and prevents us from duplicating
work or you from spending time on a change that we may not be able to accept.

Tips for proposals:

* Explain the problem you are trying to solve, not what you are trying to do.
* Explain why the change is important.
* Explain how the change will be used.
* Explain how the change will be tested.

Additionally, for bonus points: Provide draft documentation you would expect to
see if the changes were accepted.

## Pull requests

Fill in the [pull request template](.github/PULL_REQUEST_TEMPLATE.md), ticking only the checks you actually ran, and add a line under **Unreleased** in the [changelog](CHANGELOG.md).

**Commit messages**

The title should look like:

    <package>: <short description>

The package is the most affected Go package. If the change does not affect Go
code, then use the directory name instead. Changes to a single well-known
file in the root directory may use the file name.

The short description should start with a lowercase letter and be a
continuation of the sentence:

      "This changes Cavaleiro Local LLM to..."

Examples:

      llm/backend/mlx: support the llama architecture
      CONTRIBUTING: provide clarity on good commit messages, and bad

Bad Examples:

      feat: add more emoji
      fix: was not using famous web framework
      chore: generify code

**Tests**

Please include tests. Strive to test behavior, not implementation. A bug fix
should come with a regression test that fails without the fix.

**New dependencies**

Dependencies should be added sparingly. If you are adding a new dependency,
please explain why it is necessary and what other ways you attempted that
did not work without it.

## Need help?

If you need help with anything, feel free to open an [issue](https://github.com/cavaleiro-ai/cavaleiro-local-llm/issues).

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE) of this repository.
