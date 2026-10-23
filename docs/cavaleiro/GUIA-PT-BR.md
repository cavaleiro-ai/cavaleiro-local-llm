# Guia do Cavaleiro Local LLM (português)

Este guia explica o que é o Cavaleiro Local LLM, como ele funciona por dentro, como instalar e usar no Windows, no macOS e no Linux, como configurar, como resolver problemas comuns e como contribuir. Para a visão geral rápida, veja o [README em português](../../README.pt-BR.md).

## Sumário

1. [O que é e para quem](#1-o-que-é-e-para-quem)
2. [Arquitetura](#2-arquitetura)
3. [Instalação passo a passo](#3-instalação-passo-a-passo)
4. [Primeiros passos](#4-primeiros-passos)
5. [Exemplos práticos](#5-exemplos-práticos)
6. [Configuração](#6-configuração)
7. [Solução de problemas](#7-solução-de-problemas)
8. [Como contribuir](#8-como-contribuir)

## 1. O que é e para quem

O **Cavaleiro Local LLM** é um servidor e uma linha de comando para rodar modelos de linguagem abertos (Gemma, Llama, Qwen, DeepSeek, Mistral e muitos outros) **no seu próprio computador**. Ele usa o mesmo comando `ollama`, a mesma API REST na porta `11434` e o mesmo formato de modelos do projeto original, e acrescenta correções de robustez, de Windows e de segurança, cada uma com teste de regressão (veja o [CHANGELOG](../../CHANGELOG.md)).

Ele é útil para:

- **Quem quer privacidade:** os prompts e as respostas ficam na sua máquina quando você usa modelos locais.
- **Desenvolvedores:** uma API HTTP local, compatível com OpenAI e Anthropic, para integrar modelos em aplicações, scripts e agentes de código (Claude Code, Codex, OpenCode e outros).
- **Quem estuda IA:** dá para experimentar modelos diferentes, ajustar parâmetros e criar variações com um `Modelfile`.
- **Quem já usa o projeto original:** os mesmos clientes e integrações funcionam sem mudanças.

O projeto não tem afiliação com o projeto original. A biblioteca de modelos continua sendo a de ollama.com (`ollama.com/library`).

## 2. Arquitetura

O projeto é escrito em Go. O binário `ollama` faz dois papéis: **cliente** (comandos como `run`, `pull`, `list`) e **servidor** (`ollama serve`). A inferência de fato roda em processos nativos: o `llama-server` do [llama.cpp](https://github.com/ggml-org/llama.cpp) para modelos GGUF, e o executor [MLX](https://github.com/ml-explore/mlx) para modelos safetensors.

```
  Você / apps / agentes
          │  ollama run, curl, SDKs (Python, JS), OpenAI/Anthropic SDKs
          ▼
 ┌──────────────────────────────── CLI (cmd/) ─────────────────────────────┐
 │ cobra: serve, run, pull, push, create, list, ps, show, launch, ...      │
 │ api/ (cliente Go) ── HTTP + NDJSON em streaming ─────────────┐          │
 └──────────────────────────────────────────────────────────────┼──────────┘
                                                                ▼
 ┌──────────────────────────── servidor (server/) ──────────────────────────┐
 │ gin: CORS (OLLAMA_ORIGINS) + checagem de host                            │
 │ /api/*  ─┬─ chat, generate, embed  ──► scheduler (sched.go)              │
 │ /v1/*  ──┘  (middleware/ traduz OpenAI e Anthropic para a API nativa)    │
 │          ├─ pull/push (images.go, download.go, upload.go, transfer/) ◄──► registry
 │          └─ create (create.go, create/, parser/ para Modelfile)          │
 │ template/, model/renderers: monta o prompt   model/parsers, thinking/,   │
 │                                              tools/: lê tool calls etc. │
 └───────────────┬──────────────────────────────────────────────────────────┘
                 │ carrega/descarrega modelos conforme memória e keep_alive
                 ▼
 ┌─────────── runners (processos filhos) ───────────┐   discover/: GPUs
 │ llm/ ──► llama-server (llama.cpp, GGUF)          │   (CUDA, ROCm,
 │ mlxrunner/ ──► MLX (safetensors)                 │    Vulkan, Metal)
 └──────────────────────────────────────────────────┘
                 ▲
   ~/.ollama/models: manifests/, manifests-v2/, blobs/  (manifest/, types/model/)
```

**Componentes principais**

| Pasta | Responsabilidade |
|---|---|
| `main.go`, `cmd/` | Ponto de entrada e comandos da CLI (cobra). `cmd/launch/` integra ferramentas externas (`ollama launch claude`, `codex`, `opencode`…). |
| `api/` | Tipos da API e cliente Go usado pela CLI e por outros programas. |
| `server/` | Servidor HTTP (gin), rotas, scheduler de modelos, pull/push, criação de modelos, proxy de nuvem. |
| `middleware/`, `openai/`, `anthropic/` | Compatibilidade com as APIs da OpenAI e da Anthropic. |
| `llm/`, `llama/` | Gerência do processo `llama-server` e cola de build do llama.cpp. |
| `mlx/`, `mlxrunner/` | Bindings cgo do MLX, arquiteturas de modelos, tokenizer e xgrammar. |
| `discover/` | Detecção de GPUs e bibliotecas de aceleração. |
| `envconfig/` | Leitura das variáveis `OLLAMA_*`. |
| `manifest/`, `types/model/` | Nomes de modelos, manifests e caminhos dos blobs. |
| `parser/` | Leitura do `Modelfile`. |
| `template/`, `model/renderers/`, `model/parsers/`, `thinking/`, `harmony/`, `tools/` | Montagem de prompts e leitura das respostas (raciocínio, chamadas de ferramenta). |
| `fs/gguf`, `fs/safetensors`, `create/`, `transfer/` | Formatos de arquivo, importação/quantização e transferência paralela de blobs. |
| `progress/`, `readline/`, `format/` | Barras de progresso, edição de linha e formatação na CLI. |
| `app/` | App de desktop (bandeja do Windows, macOS), interface React, atualizador e banco SQLite. |
| `integration/` | Testes de integração (tag `integration`, precisam de modelos reais). |
| `CMakeLists.txt`, `cmake/`, `scripts/`, `Dockerfile` | Build nativo, scripts de empacotamento e imagem Docker. |

**Fluxo de uma conversa:** `ollama run gemma4` chama `POST /api/chat` → o servidor valida a requisição e resolve o nome do modelo → o scheduler reaproveita ou inicia um runner com a memória disponível → o prompt é montado pelo template/renderer do modelo → o runner gera tokens → os parsers separam texto, raciocínio e chamadas de ferramenta → a resposta volta em NDJSON, linha a linha, e a CLI imprime em tempo real.

**Pontos de extensão:** `Modelfile` (FROM, PARAMETER, TEMPLATE, SYSTEM, ADAPTER, MESSAGE…), integrações em `cmd/launch/`, renderers e parsers por família de modelo, arquiteturas MLX em `mlxrunner/model/`, e backends de GPU escolhidos no CMake.

## 3. Instalação passo a passo

O projeto é distribuído como código-fonte. **Atenção:** os instaladores de ollama.com e a imagem `ollama/ollama` instalam o projeto original, não este.

Em todos os sistemas você vai precisar de:

- **Git**
- **Go** na versão indicada em `go.mod` (`go version` para conferir)
- **CMake 3.24+**
- **Um compilador C/C++**
- **Ninja** (recomendado)

### Windows 10/11

1. Instale o [Git](https://git-scm.com/download/win), o [Go](https://go.dev/dl/), o [CMake](https://cmake.org/download/) e o [Ninja](https://github.com/ninja-build/ninja/releases) (ou `winget install Git.Git GoLang.Go Kitware.CMake Ninja-build.Ninja`).
2. Instale o [Visual Studio 2022](https://visualstudio.microsoft.com/downloads/) (ou as Build Tools) com a carga de trabalho **Desenvolvimento para desktop com C++**.
3. Opcional, para GPU: [CUDA](https://developer.nvidia.com/cuda-downloads) (NVIDIA), [ROCm](https://rocm.docs.amd.com/) (AMD) ou o [Vulkan SDK](https://vulkan.lunarg.com/sdk/home) (AMD/Intel; defina `$env:VULKAN_SDK`).
4. Abra o **Developer PowerShell for VS 2022** e compile:

   ```powershell
   git clone https://github.com/cavaleiro-ai/cavaleiro-local-llm.git
   cd cavaleiro-local-llm
   $env:CMAKE_GENERATOR = "Ninja"
   cmake -B build .
   cmake --build build --parallel 8
   .\ollama.exe serve
   ```

5. Para GPU NVIDIA, por exemplo: `cmake -B build . -DOLLAMA_LLAMA_BACKENDS=cuda_v13`.

> Se você também tem o projeto original instalado, ele pode já estar usando a porta 11434. Feche o app da bandeja ou use outra porta: `$env:OLLAMA_HOST = "127.0.0.1:11500"`.

### macOS

1. Instale as ferramentas de linha de comando do Xcode: `xcode-select --install`. Para MLX com Metal no Apple Silicon, instale o [Xcode](https://developer.apple.com/xcode/) e rode `xcodebuild -downloadComponent MetalToolchain`.
2. Instale Go, CMake e Ninja (por exemplo com [Homebrew](https://brew.sh): `brew install go cmake ninja`).
3. Compile:

   ```shell
   git clone https://github.com/cavaleiro-ai/cavaleiro-local-llm.git
   cd cavaleiro-local-llm
   cmake -B build .
   cmake --build build --parallel 8
   ./ollama serve
   ```

No Apple Silicon o build já inclui Metal e MLX.

### Linux

1. Instale as dependências (Ubuntu/Debian):

   ```shell
   sudo apt update
   sudo apt install -y git build-essential cmake ninja-build
   ```

   e instale o Go a partir de [go.dev/dl](https://go.dev/dl/) (a versão dos repositórios costuma ser antiga).
2. Opcional, para GPU: CUDA, ROCm ou Vulkan (`sudo apt install vulkan-sdk` quando o repositório da LunarG estiver configurado).
3. Compile:

   ```shell
   git clone https://github.com/cavaleiro-ai/cavaleiro-local-llm.git
   cd cavaleiro-local-llm
   cmake -B build . -DOLLAMA_LLAMA_BACKENDS=cuda_v13   # ou omita para só CPU
   cmake --build build --parallel 8
   ./ollama serve
   ```

4. Para instalar em `/usr/local`: `sudo cmake --install build --prefix /usr/local`.

### Docker (qualquer sistema)

```shell
docker build -t cavaleiro-local-llm .
docker run -d -v ollama:/root/.ollama -p 11434:11434 --name cavaleiro-local-llm cavaleiro-local-llm
docker exec -it cavaleiro-local-llm ollama run gemma4
```

Para ROCm: `docker build --build-arg FLAVOR=rocm -t cavaleiro-local-llm .`.

## 4. Primeiros passos

Com o servidor rodando (`ollama serve`), abra outro terminal:

```shell
ollama --version          # confere a instalação
ollama pull gemma4        # baixa um modelo
ollama run gemma4         # conversa no terminal (digite /bye para sair)
ollama list               # modelos baixados
ollama ps                 # modelos carregados na memória
ollama stop gemma4        # descarrega da memória
ollama rm gemma4          # apaga do disco
```

Dentro do `ollama run`, `/?` mostra os comandos disponíveis, como `/set parameter temperature 0.2` e `/show info`.

## 5. Exemplos práticos

### Chamar a API com curl

```shell
curl http://localhost:11434/api/chat -d '{
  "model": "gemma4",
  "messages": [{"role": "user", "content": "Resuma em uma frase: o que é um LLM?"}],
  "stream": false
}'
```

No PowerShell, use `Invoke-RestMethod`:

```powershell
$body = @{ model = "gemma4"; prompt = "Diga olá em três idiomas"; stream = $false } | ConvertTo-Json
Invoke-RestMethod -Uri http://localhost:11434/api/generate -Method Post -Body $body -ContentType "application/json"
```

### Usar o SDK da OpenAI apontando para o servidor local

```python
from openai import OpenAI

client = OpenAI(base_url="http://localhost:11434/v1", api_key="ollama")  # a chave é ignorada
resposta = client.chat.completions.create(
    model="gemma4",
    messages=[{"role": "user", "content": "Explique recursão para uma criança."}],
)
print(resposta.choices[0].message.content)
```

### Criar um assistente com Modelfile

Crie um arquivo `Modelfile`:

```
FROM gemma4
PARAMETER temperature 0.3
SYSTEM """Você é um revisor de textos em português do Brasil. Responda com o texto corrigido e uma lista curta das mudanças."""
```

E rode:

```shell
ollama create revisor -f Modelfile
ollama run revisor "eu vou fazer a prova amanha, se deus quiser"
```

Para importar um arquivo GGUF baixado, use `FROM ./modelo.gguf`. No Windows, caminhos como `FROM ~\Modelos\modelo.gguf` funcionam (correção deste projeto).

### Embeddings

```shell
curl http://localhost:11434/api/embed -d '{"model": "embeddinggemma", "input": ["primeiro texto", "segundo texto"]}'
```

### Integrar com agentes de código

```shell
ollama launch claude      # Claude Code usando um modelo local ou de nuvem
ollama launch codex
ollama launch opencode
```

## 6. Configuração

A configuração é feita por variáveis de ambiente lidas pelo **servidor** (`ollama serve`). Os nomes continuam `OLLAMA_*` por compatibilidade.

| Variável | Para que serve | Padrão |
|---|---|---|
| `OLLAMA_HOST` | Endereço e porta do servidor (e do cliente) | `127.0.0.1:11434` |
| `OLLAMA_MODELS` | Pasta dos modelos | `~/.ollama/models` |
| `OLLAMA_ORIGINS` | Origens extras permitidas pelo CORS, separadas por vírgula (espaços são ignorados) | localhost e apps locais |
| `OLLAMA_KEEP_ALIVE` | Quanto tempo o modelo fica na memória (`5m`, `1h`, `-1` = sempre) | `5m` |
| `OLLAMA_CONTEXT_LENGTH` | Tamanho de contexto padrão | conforme a VRAM |
| `OLLAMA_NUM_PARALLEL` | Requisições paralelas por modelo | `1` |
| `OLLAMA_MAX_LOADED_MODELS` | Modelos carregados ao mesmo tempo | automático |
| `OLLAMA_FLASH_ATTENTION` | Ativa flash attention | desligado |
| `OLLAMA_KV_CACHE_TYPE` | Quantização do cache K/V (`f16`, `q8_0`, `q4_0`) | `f16` |
| `OLLAMA_DEBUG` | Logs detalhados (`1`; `2` para trace) | desligado |
| `OLLAMA_NO_CLOUD` | Desliga recursos de nuvem (modelos remotos, busca na web, recomendações) | desligado |
| `HTTPS_PROXY` | Proxy para baixar modelos | — |

Como definir:

- **Windows (PowerShell, só na sessão):** `$env:OLLAMA_HOST = "0.0.0.0:11434"; ollama serve`. Para fixar: `setx OLLAMA_MODELS "D:\modelos"` e abra um novo terminal.
- **macOS/Linux (sessão):** `OLLAMA_HOST=0.0.0.0:11434 ollama serve`.
- **Linux com systemd:** `sudo systemctl edit ollama` e acrescente `Environment="OLLAMA_HOST=0.0.0.0:11434"` na seção `[Service]`.

`ollama serve --help` lista as variáveis aceitas pelo servidor. Também existe `~/.ollama/server.json`, onde `{"disable_ollama_cloud": true}` tem o mesmo efeito de `OLLAMA_NO_CLOUD=1`.

**Segurança:** a API não tem autenticação. Só exponha o servidor na rede (`OLLAMA_HOST=0.0.0.0`) atrás de um proxy reverso com autenticação ou numa rede confiável, e liste em `OLLAMA_ORIGINS` apenas os sites em que você confia.

## 7. Solução de problemas

**Onde ficam os logs?** No terminal onde você rodou `ollama serve`. No Linux com systemd: `journalctl -u ollama`. No app de desktop do Windows: `%LOCALAPPDATA%\Ollama\server.log`; no macOS: `~/.ollama/logs/server.log`. Rode com `OLLAMA_DEBUG=1` para mais detalhes.

| Sintoma | O que fazer |
|---|---|
| `Error: could not connect to ollama server` | Inicie o servidor com `ollama serve` e confira se `OLLAMA_HOST` é o mesmo no cliente e no servidor. |
| `bind: address already in use` / porta 11434 ocupada | Outra instância (do projeto original, por exemplo) está rodando. Feche-o ou use outra porta com `OLLAMA_HOST`. |
| O modelo roda só na CPU | Confira se o build incluiu o backend da sua GPU (`-DOLLAMA_LLAMA_BACKENDS=...`) e procure no log as linhas de detecção de GPU. Veja [docs/gpu.mdx](../gpu.mdx). |
| Falta de memória ao carregar | Use um modelo menor ou mais quantizado, reduza `OLLAMA_CONTEXT_LENGTH` ou `num_ctx`, ou defina `OLLAMA_KV_CACHE_TYPE=q8_0` com flash attention. |
| Download lento ou falhando atrás de proxy | Defina `HTTPS_PROXY` no ambiente do **servidor**, não só do cliente. |
| Erro de CORS no navegador | Acrescente a origem do seu site em `OLLAMA_ORIGINS`, por exemplo `https://meuapp.exemplo, http://localhost:3000`. |
| `ollama create` não encontra o arquivo | Caminhos relativos no `Modelfile` são relativos à pasta do `Modelfile`; no Windows, `~\` aponta para a sua pasta de usuário. |
| Build falha no Windows com erro do compilador | Rode o CMake de dentro do Developer PowerShell do Visual Studio e use o Ninja. |

Mais respostas em [docs/faq.mdx](../faq.mdx) e [docs/troubleshooting.mdx](../troubleshooting.mdx) (em inglês).

## 8. Como contribuir

1. Leia o [CONTRIBUTING.md](../../CONTRIBUTING.md) e o [Código de Conduta](../../CODE_OF_CONDUCT.md).
2. Abra uma issue antes de mudanças grandes, descrevendo o problema real.
3. Para corrigir um bug, escreva primeiro um teste que falha e depois a correção que faz o teste passar.
4. Rode os testes dos pacotes que você mudou. Pacotes Go puros rodam com `go test ./pacote/...`; `server`, `cmd`, `create`, `mlx` e `app` precisam de cgo e de um compilador C compatível com GCC (no Windows, MinGW/llvm-mingw — veja o CONTRIBUTING).
5. Use mensagens de commit no formato `pacote: descrição curta` e preencha o modelo de pull request.
6. Vulnerabilidades devem ser relatadas em particular, como explica o [SECURITY.md](../../SECURITY.md).

As notas técnicas completas (mapa de arquitetura, baseline de testes, cada correção e como foi verificada) estão em [ENGINEERING.md](ENGINEERING.md), em inglês.
