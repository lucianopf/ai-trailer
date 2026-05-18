# ai-trailer

CLI multiplataforma para identificar commits assistidos por ferramentas de IA e adicionar trailers padronizados nas mensagens de commit.

O `ai-trailer` instala um `prepare-commit-msg` global do Git que detecta a ferramenta de IA ativa, descobre o modelo quando possível e acrescenta metadados como:

```text
Co-authored-by: OpenAI Codex <noreply@openai.com>
Ai-tool: codex
Ai-model: gpt-5
Ai-os: macos
```

Esses trailers deixam os commits consultáveis para auditoria, métricas de adoção e análise posterior via GitHub/API.

## O que ele faz

- Detecta ferramentas de IA instaladas na máquina.
- Instala um hook global em `~/.git-hooks/ai-trailers/prepare-commit-msg`.
- Configura `git config --global core.hooksPath` para usar esse hook.
- Adiciona trailers de autoria e metadados sem bloquear commits caso algo falhe.
- Mantém um log local em `~/.ai-trailer/records.jsonl`.
- Opcionalmente injeta instruções de commit em arquivos como `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` e equivalentes.
- Opcionalmente envia eventos de adoção/configuração para um webhook Google Apps Script.

## Ferramentas suportadas

O detector conhece as seguintes ferramentas:

- Claude Code
- Gemini CLI
- OpenAI Codex CLI
- GitHub Copilot
- Hermes Agent
- OpenCode
- Windsurf
- KiloCode
- Aider
- Continue.dev
- Cody
- Cursor
- Amazon Q Developer
- Tabnine
- CodeRabbit

Nem toda ferramenta expõe o modelo da mesma forma. O hook usa variáveis de ambiente quando elas existem e arquivos de sessão para ferramentas que precisam de fallback.

## Instalação

```bash
curl -fsSL https://raw.githubusercontent.com/lucianopf/ai-trailer/master/install.sh | bash
```

O instalador detecta a plataforma automaticamente (macOS/Linux/WSL, amd64/arm64), baixa o binário certo das GitHub Releases e instala em:

```text
~/.local/bin/ai-trailer
```

Se `~/.local/bin` ainda não estiver no `PATH`, adicione ao shell:

```bash
export PATH="${HOME}/.local/bin:${PATH}"
```

Também é possível compilar localmente:

```bash
go build -o ai-trailer .
```

## Uso rápido

Detecte ferramentas instaladas:

```bash
ai-trailer detect
```

Configure o hook e selecione as ferramentas no menu interativo:

```bash
ai-trailer configure
```

Configure também arquivos de instrução das ferramentas selecionadas:

```bash
ai-trailer configure --instructions
```

Verifique se a sessão atual geraria trailers:

```bash
ai-trailer test
```

Veja o estado atual:

```bash
ai-trailer status
```

Consulte registros locais:

```bash
ai-trailer record
```

Atualize o binário instalado:

```bash
ai-trailer update
```

Remova apenas o hook:

```bash
ai-trailer uninstall
```

Remova o hook e reverta blocos de instrução injetados:

```bash
ai-trailer uninstall --full
```

## Como a detecção funciona

### Durante o `detect`

O comando `detect` procura evidências de instalação por:

- binários no `PATH`;
- pacotes npm globais;
- pacotes pip;
- fórmulas Homebrew;
- arquivos de configuração conhecidos;
- extensões em diretórios comuns do VS Code/Cursor.

### Durante o commit

O hook prioriza sinais da sessão ativa:

| Ferramenta | Sinal usado pelo hook |
| --- | --- |
| Claude Code | `CLAUDECODE=1` ou `CLAUDE_CODE_ENTRYPOINT` ou `CLAUDE_MODEL` |
| Hermes | `HERMES_SESSION_ID`, `HERMES_HOME` ou `_HERMES_GATEWAY` |
| Gemini CLI | `GEMINI_MODEL` |
| Windsurf | `WINDSURF_EXTENSION_VERSION` |
| Cursor | `CURSOR_TRACE_ID` + `~/.ai-trailer/cursor-model` |
| Codex | trailer nativo `Co-authored-by: Codex <...>` + `~/.codex/state_5.sqlite` |
| GitHub Copilot | trailer nativo `Co-authored-by: Copilot <...>` + VS Code state DB |
| OpenCode | processo `opencode` ativo + `opencode.db` modificado nos últimos 10 min |

O modelo do Codex é lido diretamente de `~/.codex/state_5.sqlite` — sem necessidade de configurar hooks adicionais.

## Webhook de adoção

O projeto inclui integração opcional com Google Sheets via Apps Script para eventos de adoção, como `detect`, `configure` e `uninstall`.

A URL do webhook é resolvida nesta ordem:

1. `AI_TRAILER_WEBHOOK_URL`
2. `~/.ai-trailer/webhook.json`
3. URL padrão compilada em `internal/webhook/webhook.go`

Token opcional:

```bash
export AI_TRAILER_WEBHOOK_TOKEN="<token>"
```

O webhook não rastreia commits individualmente. Os commits ficam no Git e no log local; a planilha é focada em configuração e adoção.

Os arquivos do Apps Script estão em `appscript/`.

## Desenvolvimento

Pré-requisito:

```bash
go version
```

O projeto usa Go `1.22.5`.

Rodar o CLI localmente:

```bash
go run . help
go run . detect
go run . test
```

Rodar testes:

```bash
go test ./...
```

Gerar binário local:

```bash
go build -o /tmp/ai-trailer .
```

Gerar binários de release e publicar:

```bash
make release
```

## Estrutura do projeto

```text
.
├── main.go                    # comandos do CLI
├── instrument.go              # setup de arquivos de sessão para Cursor
├── tui_*.go                   # menu interativo por plataforma
├── internal/config            # instalação do hook e injeção/reversão de instruções
├── internal/detect            # registro e detecção de ferramentas de IA
├── internal/record            # log JSONL local e sumários
├── internal/webhook           # cliente Google Apps Script
├── appscript                  # webhook Google Sheets
├── dist                       # binários pré-compilados
└── install.sh                 # instalador
```

## Segurança operacional

O hook foi desenhado para não bloquear commits:

- não usa `set -e`;
- ignora commits de merge e squash;
- não falha se `ai-trailer record-commit` não estiver disponível;
- só adiciona cada trailer se ele ainda não existir na mensagem.

O objetivo é enriquecer o histórico de commits sem quebrar o fluxo normal de desenvolvimento.

<!-- test: claude-code model detection via native trailer -->
