## Summary

<!-- What changes, in one or two sentences. Title format: "<package>: <short description>" (see CONTRIBUTING.md). -->

## Why

<!-- The problem this solves. Link the issue: "Closes #123". -->

## What changed

<!-- Packages and behavior touched. Call out anything that affects the REST API, the `ollama` CLI, `OLLAMA_*` environment variables, or on-disk formats. -->

-

## How it was verified

<!-- Paste the relevant output. Tick only what you actually ran. -->

- [ ] `go test ./<package>/...` for every package touched (state whether cgo was enabled)
- [ ] A regression test that fails before the change and passes after it (for bug fixes)
- [ ] `go vet` on the packages touched
- [ ] `gofmt -l` reports nothing for the files touched
- [ ] `golangci-lint run` (if available)
- [ ] Manually exercised with `ollama serve` / the CLI (describe below)

## Risks and follow-ups

<!-- What could break, what was not tested (for example GPU backends or other operating systems), what remains pending. -->

## Checklist

- [ ] The `ollama` command, `OLLAMA_*` variables, REST API, and default port stay compatible.
- [ ] A line was added under **Unreleased** in [CHANGELOG.md](../CHANGELOG.md).
- [ ] No secrets, private endpoints, or personal settings were added.
