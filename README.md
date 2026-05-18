# ai-trailer

CLI multiplataforma para identificar commits assistidos por ferramentas de IA e adicionar trailers padronizados nas mensagens de commit.

Linha de validação: alteração pequena para validar o fluxo de commit.
Linha de teste: alteração mínima para gerar um novo commit.

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
- KiloCode
- Aider
- Continue.dev
- Cody
- Cursor
- Windsurf
- Amazon Q Developer
- Tabnine
- CodeRabbit

Nem toda ferramenta expõe o modelo da mesma forma. O hook usa variáveis de ambiente quando elas existem e arquivos de sessão para ferramentas que precisam de fallback.

## Instalação

Instalação via script:

```bash
curl -fsSL https://raw.githubusercontent.com/lucianopf/ai-trailer/master/install.sh | bash
```

O instalador copia o binário compatível para:

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
| Claude Code | `CLAUDE_MODEL` |
| Hermes | `HERMES_SESSION` e `HERMES_MODEL` |
| OpenCode | `OPENCODE_MODEL` |
| Gemini CLI | `GEMINI_MODEL` |
| Cursor | `CURSOR_TRACE_ID` e `~/.ai-trailer/cursor-model` |
| Codex | `~/.ai-trailer/codex-model`, se atualizado na última hora |

Para Codex, `ai-trailer configure` cria um hook `PreToolUse` em `~/.codex/hooks.json` que grava o modelo ativo em `~/.ai-trailer/codex-model`. Depois disso, é necessário confiar/ativar o hook no Codex quando a ferramenta pedir.

Para Cursor, o suporte de hook ainda é tratado como não verificado. O hook do Git já consegue usar `CURSOR_TRACE_ID` e `~/.ai-trailer/cursor-model` quando esse arquivo existir.

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

Definir versão no build:

```bash
go build -ldflags "-X main.Version=v0.5.1" -o /tmp/ai-trailer .
```

## Estrutura do projeto

```text
.
├── main.go                    # comandos do CLI
├── instrument.go              # setup de arquivos de sessão para Codex/Cursor
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

Linha adicionada pelo Copilot em 2026-05-18T11:51:48-03:00

Linha adicionada pelo Copilot em 2026-05-18T12:00:06-03:00

Linha adicionada pelo Copilot em 2026-05-18T12:10:33-03:00

Linha adicionada pelo OpenCode em 2026-05-18T12:14:00-03:00

Linha adicionada pelo OpenCode em 2026-05-18T12:21:00-03:00
Linha adicionada pelo OpenCode em 2026-05-18T12:21:22-0300
Linha adicionada pelo OpenCode em 2026-05-18T12:46:52-0300
