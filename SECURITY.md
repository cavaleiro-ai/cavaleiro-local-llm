# Security policy

Cavaleiro Local LLM is a server that loads model files, talks to model registries, and exposes an HTTP API on your machine. The maintainers take security seriously and will work to resolve security issues.

## Reporting a vulnerability

**Please do not open a public issue.** Report privately through GitHub:

1. Go to the repository's **Security** tab.
2. Select **Report a vulnerability** (private vulnerability reporting), or open the form directly at <https://github.com/cavaleiro-ai/cavaleiro-local-llm/security/advisories/new>.

Please include:

- A description of the vulnerability and the affected component (for example `server/`, `api/`, `cmd/`, or the desktop app)
- Steps to reproduce the issue, ideally a request, Modelfile, or file that triggers it
- Your assessment of the potential impact
- Any possible mitigations

Redact secrets, keys, and private data. We ask that you give us sufficient time to investigate and address the vulnerability before disclosing it publicly.

## Vulnerabilities in other projects

If a vulnerability also affects the original project, say so in your report. Vulnerabilities in llama.cpp, MLX, or other third-party components belong to those projects.

## What is in scope

- Input handling in the HTTP API and the OpenAI/Anthropic-compatible endpoints (request validation, crashes, resource exhaustion).
- Parsing of untrusted model files, manifests, registry responses, and Modelfiles.
- File-system handling: path traversal, symlinks, and blob storage under `OLLAMA_MODELS`.
- Network defaults: listening address, CORS (`OLLAMA_ORIGINS`), host checks, and proxy behavior.
- Secrets and logs: keys under `~/.ollama`, tokens, and request logging (`OLLAMA_DEBUG_LOG_REQUESTS`).
- The CI workflows in `.github/workflows/` (for example excessive permissions or command injection).

## What to expect

This is a community-maintained project, so timelines are best effort:

- Acknowledgement of your report within **7 days**.
- An initial assessment, and a plan or a request for more information, within **14 days**.
- A fix in `main` and a published advisory crediting you (unless you prefer to stay anonymous) once the issue is resolved.

## Supported versions

Only the latest release and the `main` branch receive security fixes.

## Security best practices

Users are encouraged to implement their own security best practices, such as:

- Keeping the default `OLLAMA_HOST=127.0.0.1:11434` unless you need remote access, and putting an authenticating reverse proxy in front of any exposed instance (the API has no authentication of its own).
- Listing only the origins you trust in `OLLAMA_ORIGINS`.
- Pulling models only from registries you trust.
- Regularly updating to the latest version.
- Monitoring systems for unusual activity.
