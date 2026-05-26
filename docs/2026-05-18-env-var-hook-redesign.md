# ai-trailer — Hook Redesign: Env Var Detection

**Data:** 2026-05-18  
**Status:** Approved  
**Escopo:** Simplificação do hook `prepare-commit-msg` e do comando `configure`. Remove toda a detecção por processo pai, temp files por ferramenta e camadas de fallback. Substitui por leitura direta de env vars no momento do commit.

---

## Problema

A implementação atual acumulou múltiplas estratégias de detecção:
- Detecção por processo pai (`ps` / `/proc` walk) — complexa e frágil em multi-ferramenta
- Temp files por ferramenta (`/tmp/claude-current-model`, `/tmp/codex-current-model`) — requer instrumentação separada por ferramenta
- 3 camadas de fallback no `detect_model()` — diffícil de manter, difícil de testar
- `installClaudeStatusLine()`, `installCodexHooks()`, `installOpenCodePlugin()` — mecanismos distintos, cada um com sua forma de quebrar

## Princípios

**Modelo é obrigatório.** Se não conseguimos saber o modelo, preferimos não escrever o trailer a escrever dados incompletos. `Ai-tool` sem `Ai-model` só é aceitável para ferramentas que comprovadamente não expõem o modelo (ex: Cursor).

**Env vars quando garantido, PreToolUse quando não.** Para ferramentas que confirmadamente expõem modelo via env var no subprocesso (`CLAUDE_MODEL`, `HERMES_MODEL`, etc.) — env var puro, sem setup extra. Para ferramentas onde o env var não é garantido (Codex, Cursor) — `ai-trailer configure` instala um PreToolUse hook mínimo que escreve o modelo num arquivo em `~/.ai-trailer/`.

Quando o commit é feito fora do contexto da ferramenta (terminal separado), o hook sai silenciosamente — comportamento correto.

---

## Design

### Hook `prepare-commit-msg`

Duas fontes de dados, consultadas nessa ordem por ferramenta:

**Fonte 1 — env vars** (ferramentas com garantia de modelo no subprocesso):

| Ferramenta | Var de detecção | Var de modelo | Garantia |
|---|---|---|---|
| Claude Code | `CLAUDE_MODEL` | `CLAUDE_MODEL` | ✅ confirmado |
| Hermes | `HERMES_SESSION_ID` | `HERMES_MODEL` | ✅ confirmado |
| OpenCode | `OPENCODE_MODEL` | `OPENCODE_MODEL` | ⚠️ não implementado — hook usa pgrep+db |
| Gemini CLI | `GEMINI_MODEL` | `GEMINI_MODEL` | ✅ confirmado |
| Copilot | `Co-authored-by` já no msg | — | nativo, sem model |

**Fonte 2 — session files** (ferramentas onde env var não é garantido):

Arquivo escrito pelo PreToolUse hook instalado pelo `configure`. Formato: texto simples, uma linha com o model ID.

| Ferramenta | Session file | Escrito por |
|---|---|---|
| Codex | `~/.ai-trailer/codex-model` | PreToolUse hook em `~/.codex/hooks/` |
| Cursor | `~/.ai-trailer/cursor-model` | PreToolUse hook (se Cursor suportar) |

O hook lê o session file só quando a ferramenta é detectada (ex: `CURSOR_TRACE_ID` presente) mas a var de modelo não está disponível.

**Trailers escritos:**
```
Ai-tool: claude-code
Ai-model: claude-sonnet-4-6
Ai-os: macos
Co-authored-by: Claude <noreply@anthropic.com>
```

Regras:
- Skip em commits de merge e squash (`COMMIT_SOURCE`)
- `Ai-model` omitido apenas se ferramenta comprovadamente não expõe modelo (Cursor) e session file não existe
- `Co-authored-by` omitido se já presente no commit message
- Se nenhuma detecção → exit 0, sem modificação

### Hook script (Go embed)

`hook_script.go` fica com ~70 linhas. Sem `detect_parent_process_tool()`, sem `detect_model()` em 3 camadas, sem referências a `/tmp/*-current-model`. Session files ficam em `~/.ai-trailer/` (não em `/tmp/`).

### CLI — o que muda

**`configure`:** instala o git hook globalmente + PreToolUse hooks para ferramentas que precisam de session file (Codex, Cursor). Remove:
- `installClaudeStatusLine()` — substituído por env var direta
- `installOpenCodePlugin()` — substituído por env var direta
- Campos `InstrumentTempFile` / `InstrumentSetup` nos Tool structs — substituídos pela tabela acima

**`test` (novo):** roda o hook em dry-run no env atual, imprime o que seria escrito sem criar commit. Útil para debug por ferramenta.

```
$ ai-trailer test
  Env detected: CLAUDE_MODEL=claude-sonnet-4-6
  Would append:
    Ai-tool: claude-code
    Ai-model: claude-sonnet-4-6
    Ai-os: macos
    Co-authored-by: Claude <noreply@anthropic.com>
```

**`status`:** mantém, mostra hook instalado + vars detectadas no env atual.

**`update`:** sem mudança.

**`uninstall`:** simplifica — não precisa reverter arquivos de instrumentação.

---

## Testes (`hook_script_test.go`)

| Cenário | Esperado |
|---|---|
| `CLAUDE_MODEL=claude-sonnet-4-6` | `Ai-tool: claude-code`, `Ai-model: claude-sonnet-4-6` |
| Nenhuma var de IA | Hook não modifica o arquivo |
| `CLAUDE_MODEL` + source=merge | Hook não modifica o arquivo |
| `CLAUDE_MODEL` + `Co-authored-by` já presente | Não duplica trailer |
| `CURSOR_TRACE_ID=abc` + `~/.ai-trailer/cursor-model` existe com `claude-3.7-sonnet` | `Ai-tool: cursor`, `Ai-model: claude-3.7-sonnet` |
| `CURSOR_TRACE_ID=abc` + sem session file | `Ai-tool: cursor`, sem `Ai-model` |
| `HERMES_SESSION_ID=x` + `HERMES_MODEL=llama-3` | `Ai-tool: hermes`, `Ai-model: llama-3` |
| Duas vars presentes (`CLAUDE_MODEL` + `HERMES_SESSION`) | Prioridade da ordem da tabela (Claude vence) |
| `CURSOR_TRACE_ID` + session file com conteúdo vazio | sem `Ai-model` |

---

## Critérios de sucesso

1. Commit dentro de sessão Claude Code → trailers corretos em `git log --format="%B"`
2. Commit fora de sessão de IA → sem trailers
3. `ai-trailer configure` em máquina nova → hook funcionando em < 30s
4. `ai-trailer test` imprime output correto no env da sessão atual
5. Codex: verificar empiricamente quais vars estão disponíveis no subprocesso git

---

## O que este spec não cobre

- Codex: o PreToolUse hook de model dump precisa ser validado empiricamente com `ai-trailer test` após install
- Cursor: verificar se Cursor suporta PreToolUse hooks e qual o formato do payload
- `record` command: mantido sem alteração para histórico local de commits
- Webhook: mantido sem alteração
