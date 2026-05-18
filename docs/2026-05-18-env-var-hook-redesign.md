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

## Insight

Quando uma ferramenta de IA executa `git commit` como subprocesso, o processo filho **herda o ambiente do pai**. As ferramentas já expõem vars de ambiente que identificam a sessão. Não é necessário processo daemon, session file, ou temp files.

Quando o commit é feito fora do contexto da ferramenta (terminal separado), o hook sai silenciosamente — comportamento correto.

---

## Design

### Hook `prepare-commit-msg`

Leitura de env vars em ordem de prioridade. Primeira que bater, vence.

| Ferramenta | Var de detecção | Var de modelo |
|---|---|---|
| Claude Code | `CLAUDE_MODEL` | `CLAUDE_MODEL` |
| Codex | `CODEX_SANDBOX_ENV` ou `OPENAI_CODEX_*` | `OPENAI_MODEL` |
| Cursor | `CURSOR_TRACE_ID` | — (não exposto) |
| Hermes | `HERMES_SESSION` | `HERMES_MODEL` |
| OpenCode | `OPENCODE_MODEL` | `OPENCODE_MODEL` |
| Gemini CLI | `GEMINI_MODEL` | `GEMINI_MODEL` |
| Copilot | `Co-authored-by` já no commit msg | — |

**Trailers escritos:**
```
Ai-tool: claude-code
Ai-model: claude-sonnet-4-6
Ai-os: macos
Co-authored-by: Claude <noreply@anthropic.com>
```

Regras:
- Skip em commits de merge e squash (`COMMIT_SOURCE`)
- `Ai-model` omitido se ferramenta não expõe modelo
- `Co-authored-by` omitido se já presente no commit message
- Se nenhuma var de detecção encontrada → exit 0, sem modificação

### Hook script (Go embed)

`hook_script.go` fica com ~50 linhas. Sem `detect_parent_process_tool()`, sem `detect_model()` em camadas, sem referências a `/tmp/*-current-model`.

### CLI — o que muda

**`configure`:** só instala o git hook globalmente via `core.hooksPath`. Remove:
- `installClaudeStatusLine()`
- `installCodexHooks()`
- `installOpenCodePlugin()`
- Campos `InstrumentTempFile` / `InstrumentSetup` nos Tool structs

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
| `CURSOR_TRACE_ID=abc` | `Ai-tool: cursor`, sem `Ai-model` |
| `HERMES_SESSION=x` + `HERMES_MODEL=llama-3` | `Ai-tool: hermes`, `Ai-model: llama-3` |
| Duas vars presentes (`CLAUDE_MODEL` + `HERMES_SESSION`) | Prioridade da ordem da tabela (Claude vence) |

---

## Critérios de sucesso

1. Commit dentro de sessão Claude Code → trailers corretos em `git log --format="%B"`
2. Commit fora de sessão de IA → sem trailers
3. `ai-trailer configure` em máquina nova → hook funcionando em < 30s
4. `ai-trailer test` imprime output correto no env da sessão atual
5. Codex: verificar empiricamente quais vars estão disponíveis no subprocesso git

---

## O que este spec não cobre

- Codex: vars exatas a serem validadas empiricamente com `ai-trailer test`
- Cursor: `CURSOR_TRACE_ID` a confirmar (env herdado pelo subprocesso git?)
- `record` command: mantido sem alteração para histórico local de commits
- Webhook: mantido sem alteração
