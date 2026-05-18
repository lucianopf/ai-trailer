     1|# ai-trailer
     2|
     3|CLI multiplataforma para identificar commits assistidos por ferramentas de IA e adicionar trailers padronizados nas mensagens de commit.
     4|
     5|Linha de validação: alteração pequena para validar o fluxo de commit.
     6|Linha de teste: alteração mínima para gerar um novo commit.
     7|
     8|O `ai-trailer` instala um `prepare-commit-msg` global do Git que detecta a ferramenta de IA ativa, descobre o modelo quando possível e acrescenta metadados como:
     9|
    10|```text
    11|Co-authored-by: OpenAI Codex <noreply@openai.com>
    12|Ai-tool: codex
    13|Ai-model: gpt-5
    14|Ai-os: macos
    15|```
    16|
    17|Esses trailers deixam os commits consultáveis para auditoria, métricas de adoção e análise posterior via GitHub/API.
    18|
    19|## O que ele faz
    20|
    21|- Detecta ferramentas de IA instaladas na máquina.
    22|- Instala um hook global em `~/.git-hooks/ai-trailers/prepare-commit-msg`.
    23|- Configura `git config --global core.hooksPath` para usar esse hook.
    24|- Adiciona trailers de autoria e metadados sem bloquear commits caso algo falhe.
    25|- Mantém um log local em `~/.ai-trailer/records.jsonl`.
    26|- Opcionalmente injeta instruções de commit em arquivos como `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` e equivalentes.
    27|- Opcionalmente envia eventos de adoção/configuração para um webhook Google Apps Script.
    28|
    29|## Ferramentas suportadas
    30|
    31|O detector conhece as seguintes ferramentas:
    32|
    33|- Claude Code
    34|- Gemini CLI
    35|- OpenAI Codex CLI
    36|- GitHub Copilot
    37|- Hermes Agent
    38|- OpenCode
    39|- KiloCode
    40|- Aider
    41|- Continue.dev
    42|- Cody
    43|- Cursor
    44|- Windsurf
    45|- Amazon Q Developer
    46|- Tabnine
    47|- CodeRabbit
    48|
    49|Nem toda ferramenta expõe o modelo da mesma forma. O hook usa variáveis de ambiente quando elas existem e arquivos de sessão para ferramentas que precisam de fallback.
    50|
    51|## Instalação
    52|
    53|Instalação via script:
    54|
    55|```bash
    56|curl -fsSL https://raw.githubusercontent.com/lucianopf/ai-trailer/master/install.sh | bash
    57|```
    58|
    59|O instalador copia o binário compatível para:
    60|
    61|```text
    62|~/.local/bin/ai-trailer
    63|```
    64|
    65|Se `~/.local/bin` ainda não estiver no `PATH`, adicione ao shell:
    66|
    67|```bash
    68|export PATH="${HOME}/.local/bin:${PATH}"
    69|```
    70|
    71|Também é possível compilar localmente:
    72|
    73|```bash
    74|go build -o ai-trailer .
    75|```
    76|
    77|## Uso rápido
    78|
    79|Detecte ferramentas instaladas:
    80|
    81|```bash
    82|ai-trailer detect
    83|```
    84|
    85|Configure o hook e selecione as ferramentas no menu interativo:
    86|
    87|```bash
    88|ai-trailer configure
    89|```
    90|
    91|Configure também arquivos de instrução das ferramentas selecionadas:
    92|
    93|```bash
    94|ai-trailer configure --instructions
    95|```
    96|
    97|Verifique se a sessão atual geraria trailers:
    98|
    99|```bash
   100|ai-trailer test
   101|```
   102|
   103|Veja o estado atual:
   104|
   105|```bash
   106|ai-trailer status
   107|```
   108|
   109|Consulte registros locais:
   110|
   111|```bash
   112|ai-trailer record
   113|```
   114|
   115|Atualize o binário instalado:
   116|
   117|```bash
   118|ai-trailer update
   119|```
   120|
   121|Remova apenas o hook:
   122|
   123|```bash
   124|ai-trailer uninstall
   125|```
   126|
   127|Remova o hook e reverta blocos de instrução injetados:
   128|
   129|```bash
   130|ai-trailer uninstall --full
   131|```
   132|
   133|## Como a detecção funciona
   134|
   135|### Durante o `detect`
   136|
   137|O comando `detect` procura evidências de instalação por:
   138|
   139|- binários no `PATH`;
   140|- pacotes npm globais;
   141|- pacotes pip;
   142|- fórmulas Homebrew;
   143|- arquivos de configuração conhecidos;
   144|- extensões em diretórios comuns do VS Code/Cursor.
   145|
   146|### Durante o commit
   147|
   148|O hook prioriza sinais da sessão ativa:
   149|
   150|| Ferramenta | Sinal usado pelo hook |
   151|| --- | --- |
   152|| Claude Code | `CLAUDE_MODEL` |
   153|| Hermes | `HERMES_SESSION` e `HERMES_MODEL` |
   154|| OpenCode | `OPENCODE_MODEL` |
   155|| Gemini CLI | `GEMINI_MODEL` |
   156|| Cursor | `CURSOR_TRACE_ID` e `~/.ai-trailer/cursor-model` |
   157|| Codex | `~/.ai-trailer/codex-model`, se atualizado na última hora |
   158|
   159|Para Codex, `ai-trailer configure` cria um hook `PreToolUse` em `~/.codex/hooks.json` que grava o modelo ativo em `~/.ai-trailer/codex-model`. Depois disso, é necessário confiar/ativar o hook no Codex quando a ferramenta pedir.
   160|
   161|Para Cursor, o suporte de hook ainda é tratado como não verificado. O hook do Git já consegue usar `CURSOR_TRACE_ID` e `~/.ai-trailer/cursor-model` quando esse arquivo existir.
   162|
   163|## Webhook de adoção
   164|
   165|O projeto inclui integração opcional com Google Sheets via Apps Script para eventos de adoção, como `detect`, `configure` e `uninstall`.
   166|
   167|A URL do webhook é resolvida nesta ordem:
   168|
   169|1. `AI_TRAILER_WEBHOOK_URL`
   170|2. `~/.ai-trailer/webhook.json`
   171|3. URL padrão compilada em `internal/webhook/webhook.go`
   172|
   173|Token opcional:
   174|
   175|```bash
   176|export AI_TRAILER_WEBHOOK_TOKEN="<token>"
   177|```
   178|
   179|O webhook não rastreia commits individualmente. Os commits ficam no Git e no log local; a planilha é focada em configuração e adoção.
   180|
   181|Os arquivos do Apps Script estão em `appscript/`.
   182|
   183|## Desenvolvimento
   184|
   185|Pré-requisito:
   186|
   187|```bash
   188|go version
   189|```
   190|
   191|O projeto usa Go `1.22.5`.
   192|
   193|Rodar o CLI localmente:
   194|
   195|```bash
   196|go run . help
   197|go run . detect
   198|go run . test
   199|```
   200|
   201|Rodar testes:
   202|
   203|```bash
   204|go test ./...
   205|```
   206|
   207|Gerar binário local:
   208|
   209|```bash
   210|go build -o /tmp/ai-trailer .
   211|```
   212|
   213|Definir versão no build:
   214|
   215|```bash
   216|go build -ldflags "-X main.Version=v0.5.1" -o /tmp/ai-trailer .
   217|```
   218|
   219|## Estrutura do projeto
   220|
   221|```text
   222|.
   223|├── main.go                    # comandos do CLI
   224|├── instrument.go              # setup de arquivos de sessão para Codex/Cursor
   225|├── tui_*.go                   # menu interativo por plataforma
   226|├── internal/config            # instalação do hook e injeção/reversão de instruções
   227|├── internal/detect            # registro e detecção de ferramentas de IA
   228|├── internal/record            # log JSONL local e sumários
   229|├── internal/webhook           # cliente Google Apps Script
   230|├── appscript                  # webhook Google Sheets
   231|├── dist                       # binários pré-compilados
   232|└── install.sh                 # instalador
   233|```
   234|
   235|## Segurança operacional
   236|
   237|O hook foi desenhado para não bloquear commits:
   238|
   239|- não usa `set -e`;
   240|- ignora commits de merge e squash;
   241|- não falha se `ai-trailer record-commit` não estiver disponível;
   242|- só adiciona cada trailer se ele ainda não existir na mensagem.
   243|
   244|O objetivo é enriquecer o histórico de commits sem quebrar o fluxo normal de desenvolvimento.
   245|
   246|Linha adicionada pelo Copilot em 2026-05-18T11:51:48-03:00
   247|
   248|Linha adicionada pelo Copilot em 2026-05-18T12:00:06-03:00
   249|
   250|Linha adicionada pelo Copilot em 2026-05-18T12:10:33-03:00
   251|
   252|Linha adicionada pelo OpenCode em 2026-05-18T12:14:00-03:00
   253|
   254|Linha adicionada pelo OpenCode em 2026-05-18T12:21:00-03:00
   255|Linha adicionada pelo OpenCode em 2026-05-18T12:21:22-0300
   256|Linha adicionada pelo OpenCode em 2026-05-18T12:46:52-0300
   257|
   258|Linha adicionada pelo OpenCode em 2026-05-18T12:52:00-0300
   259|
   260|Linha adicionada pelo OpenCode em 2026-05-18T13:02:00-0300
   261|
   262|Linha adicionada pelo OpenCode em 2026-05-18T13:03:07-0300
   263|
   264|Linha adicionada pelo OpenCode em 2026-05-18T13:16:40-0300
   265|

Linha adicionada pelo Hermes Agent em 2026-05-18T13:22:10-0300
