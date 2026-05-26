# AI Tracker — Config & Adoption Tracking

Google Sheets + Apps Script para rastrear **adoção de ferramentas de IA** na empresa.

## O que é rastreado

| Evento | Quando dispara | Dados enviados |
|--------|---------------|----------------|
| `detect` | `ai-trailer detect` | Quais ferramentas foram encontradas na máquina |
| `configure` | `ai-trailer configure` | Ferramentas detectadas + configuradas + hook + CLAUDE.md |
| `setup` | `AI_TRAILER_WEBHOOK_URL=<URL>` (env var) | Webhook configurado |
| `uninstall` | `ai-trailer uninstall` | Remoção da configuração |

**Commits NÃO são rastreados aqui** — serão extraídos via GitHub API posteriormente.

## Colunas da planilha

| Col | Campo | Descrição |
|-----|-------|-----------|
| A | Timestamp | ISO 8601 |
| B | User Email | `git config user.email` |
| C | User Name | `git config user.name` |
| D | System User | `$USER` / `whoami` |
| E | Hostname | Nome da máquina |
| F | Platform | linux / darwin / win32 / wsl |
| G | Event | detect / configure / setup / uninstall |
| H | Tools Detected | Ferramentas auto-detectadas (lista) |
| I | Tools Configured | Ferramentas selecionadas (lista) |
| J | Hook Installed | true/false |
| K | CLAUDE.md Updated | true/false |
| L | Webhook Configured | true/false |
| M | Client Version | Versão do CLI |
| N | Extra | JSON adicional |

## Deploy

1. Google Sheets → Extensions → Apps Script
2. Colar `appscript/Code.gs` no editor
3. Deploy → New Deployment → Web App (Anyone)
4. Copiar URL
5. Exporte as variáveis de ambiente no shell:
   ```bash
   export AI_TRAILER_WEBHOOK_URL="<URL>"
   export AI_TRAILER_WEBHOOK_TOKEN="<TOKEN>"  # opcional
   ```
