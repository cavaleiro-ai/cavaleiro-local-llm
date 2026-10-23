[English](README.md) · **Português (Brasil)**

<a id="top"></a>

<p align="center">
  <img src="docs/assets/cavaleiro-banner.svg" alt="Cavaleiro Local LLM — rode modelos de linguagem abertos na sua própria máquina." width="800" />
</p>

# Cavaleiro Local LLM

**Rode modelos abertos na sua própria máquina, com uma API REST local que funciona com as ferramentas que você já usa.**

O Cavaleiro Local LLM é mantido pela [Cavaleiro AI](https://github.com/cavaleiro-ai). Ele mantém inalterados o comando `ollama`, a API REST e o formato de modelos, para que clientes e integrações existentes continuem funcionando, e acrescenta correções verificadas para travamentos, comportamento no Windows e tratamento de entradas — cada uma com um teste de regressão. Veja [O que o Cavaleiro Local LLM acrescenta](#o-que-o-cavaleiro-local-llm-acrescenta).

## Conteúdo

- [Instalação](#instalação)
- [Primeiros passos](#primeiros-passos)
- [API REST](#api-rest)
- [Backends suportados](#backends-suportados)
- [Documentação](#documentação)
- [O que o Cavaleiro Local LLM acrescenta](#o-que-o-cavaleiro-local-llm-acrescenta)
- [Compatibilidade](#compatibilidade)
- [Licença](#licença)

## Instalação

O Cavaleiro Local LLM é distribuído como código-fonte. Os instaladores em ollama.com (`install.sh`, `install.ps1`, `OllamaSetup.exe`, `Ollama.dmg`) e a imagem Docker `ollama/ollama` instalam o projeto original, não o Cavaleiro Local LLM.

### Pré-requisitos

- [Go](https://go.dev/doc/install) (a versão indicada em `go.mod`)
- [CMake](https://cmake.org/download/) 3.24 ou mais recente
- Um compilador C/C++: Clang no macOS, ferramentas C++ do Visual Studio 2022 no Windows, ou GCC/Clang no Linux
- [Ninja](https://github.com/ninja-build/ninja/releases) no `PATH` é recomendado, principalmente no Windows

### Compilar a partir do código-fonte

```shell
git clone https://github.com/cavaleiro-ai/cavaleiro-local-llm.git
cd cavaleiro-local-llm
cmake -B build .
cmake --build build --parallel 8
./ollama serve
```

Isso gera o binário `ollama` na raiz do repositório e o runtime nativo (o `llama-server` do llama.cpp e, no Apple Silicon, o MLX) em `build/lib/ollama`. No macOS arm64 a compilação inclui inferência com Metal; nas outras plataformas, inferência em CPU, a menos que você escolha backends de GPU:

```shell
cmake -B build . -DOLLAMA_LLAMA_BACKENDS="cuda_v13;vulkan"
```

Os valores aceitos são `cuda_v12`, `cuda_v13`, `rocm_v7_1`, `rocm_v7_2`, `vulkan`, `cuda_jetpack5` e `cuda_jetpack6`. As observações por plataforma (Developer PowerShell no Windows, Vulkan SDK, ROCm, MLX) estão em [docs/development.md](docs/development.md). Um passo a passo completo em português para Windows, macOS e Linux está em [docs/cavaleiro/GUIA-PT-BR.md](docs/cavaleiro/GUIA-PT-BR.md).

Para instalar num prefixo padrão:

```shell
cmake --install build --prefix /caminho/da/instalacao
```

### Docker

```shell
docker build -t cavaleiro-local-llm .
docker run -d -v ollama:/root/.ollama -p 11434:11434 --name cavaleiro-local-llm cavaleiro-local-llm
```

Para ROCm, acrescente `--build-arg FLAVOR=rocm` ao build.

### Bibliotecas

As bibliotecas cliente `ollama` para [Python](https://pypi.org/project/ollama/) e [JavaScript](https://www.npmjs.com/package/ollama) funcionam com este servidor, assim como outros clientes da mesma API REST.

## Primeiros passos

Inicie o servidor e, em outro terminal, rode um modelo:

```shell
ollama serve
ollama run gemma4
```

`ollama` sem argumentos abre um menu interativo para rodar um modelo ou conectar uma integração. Para abrir uma integração específica, como [Claude Code](docs/integrations/claude-code.mdx), [Codex](docs/integrations/codex.mdx), [OpenCode](docs/integrations/opencode.mdx) ou [OpenClaw](docs/integrations/openclaw.mdx):

```shell
ollama launch claude
```

Os modelos são baixados da biblioteca pública em [ollama.com/library](https://ollama.com/library). Veja o [quickstart](docs/quickstart.mdx) e a [referência da CLI](docs/cli.mdx) para mais detalhes.

## API REST

O servidor expõe uma API REST em `http://localhost:11434`:

```shell
curl http://localhost:11434/api/chat -d '{
  "model": "gemma4",
  "messages": [{
    "role": "user",
    "content": "Por que o céu é azul?"
  }],
  "stream": false
}'
```

### Python

```shell
pip install ollama
```

```python
from ollama import chat

response = chat(model='gemma4', messages=[
  {
    'role': 'user',
    'content': 'Por que o céu é azul?',
  },
])
print(response.message.content)
```

### JavaScript

```shell
npm i ollama
```

```javascript
import ollama from "ollama";

const response = await ollama.chat({
  model: "gemma4",
  messages: [{ role: "user", content: "Por que o céu é azul?" }],
});
console.log(response.message.content);
```

Também há endpoints compatíveis com OpenAI e Anthropic: veja [compatibilidade com OpenAI](docs/api/openai-compatibility.mdx) e [compatibilidade com Anthropic](docs/api/anthropic-compatibility.mdx). A referência completa está em [docs/api.md](docs/api.md).

## Backends suportados

- [llama.cpp](https://github.com/ggml-org/llama.cpp) para modelos GGUF.
- [MLX](https://github.com/ml-explore/mlx) para modelos safetensors (padrão no Apple Silicon; builds com CUDA são opcionais).

## Documentação

- [Referência da CLI](docs/cli.mdx)
- [Referência da API REST](docs/api.md)
- [Importar modelos](docs/import.mdx)
- [Referência do Modelfile](docs/modelfile.mdx)
- [Perguntas frequentes](docs/faq.mdx) e [solução de problemas](docs/troubleshooting.mdx)
- [Compilar a partir do código-fonte](docs/development.md)
- [Integrações da comunidade](docs/community-integrations.md) — apps de chat, editores, SDKs e frameworks que usam esta API REST
- [Notas de engenharia](docs/cavaleiro/ENGINEERING.md) (em inglês) — mapa da arquitetura, baseline de testes e todas as mudanças deste fork
- [Guia completo em português](docs/cavaleiro/GUIA-PT-BR.md)

A pasta `docs/` contém os manuais de uso, em inglês. Instaladores, modelos na nuvem, busca na web e contas descritos ali são serviços de ollama.com, não deste projeto; todo o resto corresponde a este projeto, exceto pelas mudanças listadas no [changelog](CHANGELOG.md).

## O que o Cavaleiro Local LLM acrescenta

Tudo o que existe no projeto original no commit `8be580a`, mais ([changelog completo](CHANGELOG.md)):

- **Fim do travamento na inicialização com `OLLAMA_ORIGINS="a, b"`.** Espaços e entradas vazias são ignorados em vez de derrubar o servidor (issue original #18011).
- **Leitura robusta da autenticação do registry.** Um cabeçalho `WWW-Authenticate` malformado vindo de um registry não derruba mais o servidor (issue original #17034).
- **A CLI não entra em pânico com digests curtos.** `ollama list`, `ollama ps`, `ollama create` e `ollama push` lidam com digests de qualquer tamanho e com progresso sem mensagem (issue original #14250).
- **Erros confiáveis no cliente Go da API.** Respostas de erro sem corpo, como um `502` de um proxy, passam a ser reportadas em vez de tratadas como sucesso; linhas em branco de keep-alive são ignoradas.
- **Melhor no Windows.** `FROM ~\modelos\modelo.gguf` funciona em Modelfiles, e a detecção de processos não entra mais em pânico em máquinas com mais de 512 processos, além de ficar mais de 100 vezes mais rápida.
- **Um fork seguro.** Os workflows de release, assinatura e publicação no Docker ficam [desativados](.github/workflows-disabled/README.md) até você configurar a sua própria identidade; arquivos de comunidade, formulários de issue e documentação em português estão incluídos.

## Compatibilidade

O Cavaleiro Local LLM muda o nome do projeto, não os identificadores, para que scripts, clientes e integrações existentes continuem funcionando:

| O quê | Valor (inalterado) |
|---|---|
| Comando | `ollama` (`ollama serve`, `ollama run`, …) |
| Caminho do módulo Go | `github.com/ollama/ollama` |
| Variáveis de ambiente | `OLLAMA_HOST`, `OLLAMA_MODELS`, `OLLAMA_ORIGINS`, `OLLAMA_KEEP_ALIVE` e as demais `OLLAMA_*` |
| API REST | `/api/*`, mais os endpoints compatíveis com OpenAI (`/v1/*`) e Anthropic (`/v1/messages`) |
| Endereço padrão | `127.0.0.1:11434` |
| Pasta de dados | `~/.ollama` (modelos, chaves, `server.json`) |
| Biblioteca e registry de modelos | `ollama.com` / `registry.ollama.ai` |
| App de desktop e marca dentro do produto | inalterados (o nome do projeto original) |

**Serviços de rede do projeto original.** Alguns recursos acessam ollama.com: `ollama pull`/`push` usam o registry de lá; modelos na nuvem, busca na web e login usam ollama.com; o servidor atualiza as recomendações de modelos a partir de ollama.com, a menos que os recursos de nuvem estejam desligados. Defina `OLLAMA_NO_CLOUD=1` (ou `"disable_ollama_cloud": true` em `~/.ollama/server.json`) para desligar os recursos de nuvem. O servidor não inclui telemetria de uso, e este projeto não acrescenta nenhuma. O atualizador automático do app de desktop consulta ollama.com e instala versões do **projeto original**; desative as atualizações automáticas nas configurações do app se você usa um build deste projeto.

## Licença

Baseado no Ollama · licença MIT · veja [LICENSE](LICENSE). Sem afiliação com o projeto original.

<p align="center">
  <a href="#top">Voltar ao topo ↑</a>
</p>
