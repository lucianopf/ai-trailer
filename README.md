     1|     1|     1|     1|     1|     1|     1|# ai-trailer
     2|     2|     2|     2|     2|     2|     2|
     3|     3|     3|     3|     3|     3|     3|CLI multiplataforma para identificar commits assistidos por ferramentas de IA e adicionar trailers padronizados nas mensagens de commit.
     4|     4|     4|     4|     4|     4|     4|
     5|     5|     5|     5|     5|     5|     5|Linha de validação: alteração pequena para validar o fluxo de commit.
     6|     6|     6|     6|     6|     6|     6|Linha de teste: alteração mínima para gerar um novo commit.
     7|     7|     7|     7|     7|     7|     7|Linha de teste adicional: novo commit pequeno no README.
     7|     7|     7|     7|     7|     7|     7|
     8|     8|     8|     8|     8|     8|     8|O `ai-trailer` instala um `prepare-commit-msg` global do Git que detecta a ferramenta de IA ativa, descobre o modelo quando possível e acrescenta metadados como:
     9|     9|     9|     9|     9|     9|     9|
    10|    10|    10|    10|    10|    10|    10|```text
    11|    11|    11|    11|    11|    11|    11|Co-authored-by: OpenAI Codex <noreply@openai.com>
    12|    12|    12|    12|    12|    12|    12|Ai-tool: codex
    13|    13|    13|    13|    13|    13|    13|Ai-model: gpt-5
    14|    14|    14|    14|    14|    14|    14|Ai-os: macos
    15|    15|    15|    15|    15|    15|    15|```
    16|    16|    16|    16|    16|    16|    16|
    17|    17|    17|    17|    17|    17|    17|Esses trailers deixam os commits consultáveis para auditoria, métricas de adoção e análise posterior via GitHub/API.
    18|    18|    18|    18|    18|    18|    18|
    19|    19|    19|    19|    19|    19|    19|## O que ele faz
    20|    20|    20|    20|    20|    20|    20|
    21|    21|    21|    21|    21|    21|    21|- Detecta ferramentas de IA instaladas na máquina.
    22|    22|    22|    22|    22|    22|    22|- Instala um hook global em `~/.git-hooks/ai-trailers/prepare-commit-msg`.
    23|    23|    23|    23|    23|    23|    23|- Configura `git config --global core.hooksPath` para usar esse hook.
    24|    24|    24|    24|    24|    24|    24|- Adiciona trailers de autoria e metadados sem bloquear commits caso algo falhe.
    25|    25|    25|    25|    25|    25|    25|- Mantém um log local em `~/.ai-trailer/records.jsonl`.
    26|    26|    26|    26|    26|    26|    26|- Opcionalmente injeta instruções de commit em arquivos como `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` e equivalentes.
    27|    27|    27|    27|    27|    27|    27|- Opcionalmente envia eventos de adoção/configuração para um webhook Google Apps Script.
    28|    28|    28|    28|    28|    28|    28|
    29|    29|    29|    29|    29|    29|    29|## Ferramentas suportadas
    30|    30|    30|    30|    30|    30|    30|
    31|    31|    31|    31|    31|    31|    31|O detector conhece as seguintes ferramentas:
    32|    32|    32|    32|    32|    32|    32|
    33|    33|    33|    33|    33|    33|    33|- Claude Code
    34|    34|    34|    34|    34|    34|    34|- Gemini CLI
    35|    35|    35|    35|    35|    35|    35|- OpenAI Codex CLI
    36|    36|    36|    36|    36|    36|    36|- GitHub Copilot
    37|    37|    37|    37|    37|    37|    37|- Hermes Agent
    38|    38|    38|    38|    38|    38|    38|- OpenCode
    39|    39|    39|    39|    39|    39|    39|- KiloCode
    40|    40|    40|    40|    40|    40|    40|- Aider
    41|    41|    41|    41|    41|    41|    41|- Continue.dev
    42|    42|    42|    42|    42|    42|    42|- Cody
    43|    43|    43|    43|    43|    43|    43|- Cursor
    44|    44|    44|    44|    44|    44|    44|- Windsurf
    45|    45|    45|    45|    45|    45|    45|- Amazon Q Developer
    46|    46|    46|    46|    46|    46|    46|- Tabnine
    47|    47|    47|    47|    47|    47|    47|- CodeRabbit
    48|    48|    48|    48|    48|    48|    48|
    49|    49|    49|    49|    49|    49|    49|Nem toda ferramenta expõe o modelo da mesma forma. O hook usa variáveis de ambiente quando elas existem e arquivos de sessão para ferramentas que precisam de fallback.
    50|    50|    50|    50|    50|    50|    50|
    51|    51|    51|    51|    51|    51|    51|## Instalação
    52|    52|    52|    52|    52|    52|    52|
    53|    53|    53|    53|    53|    53|    53|Instalação via script:
    54|    54|    54|    54|    54|    54|    54|
    55|    55|    55|    55|    55|    55|    55|```bash
    56|    56|    56|    56|    56|    56|    56|curl -fsSL https://raw.githubusercontent.com/lucianopf/ai-trailer/master/install.sh | bash
    57|    57|    57|    57|    57|    57|    57|```
    58|    58|    58|    58|    58|    58|    58|
    59|    59|    59|    59|    59|    59|    59|O instalador copia o binário compatível para:
    60|    60|    60|    60|    60|    60|    60|
    61|    61|    61|    61|    61|    61|    61|```text
    62|    62|    62|    62|    62|    62|    62|~/.local/bin/ai-trailer
    63|    63|    63|    63|    63|    63|    63|```
    64|    64|    64|    64|    64|    64|    64|
    65|    65|    65|    65|    65|    65|    65|Se `~/.local/bin` ainda não estiver no `PATH`, adicione ao shell:
    66|    66|    66|    66|    66|    66|    66|
    67|    67|    67|    67|    67|    67|    67|```bash
    68|    68|    68|    68|    68|    68|    68|export PATH="${HOME}/.local/bin:${PATH}"
    69|    69|    69|    69|    69|    69|    69|```
    70|    70|    70|    70|    70|    70|    70|
    71|    71|    71|    71|    71|    71|    71|Também é possível compilar localmente:
    72|    72|    72|    72|    72|    72|    72|
    73|    73|    73|    73|    73|    73|    73|```bash
    74|    74|    74|    74|    74|    74|    74|go build -o ai-trailer .
    75|    75|    75|    75|    75|    75|    75|```
    76|    76|    76|    76|    76|    76|    76|
    77|    77|    77|    77|    77|    77|    77|## Uso rápido
    78|    78|    78|    78|    78|    78|    78|
    79|    79|    79|    79|    79|    79|    79|Detecte ferramentas instaladas:
    80|    80|    80|    80|    80|    80|    80|
    81|    81|    81|    81|    81|    81|    81|```bash
    82|    82|    82|    82|    82|    82|    82|ai-trailer detect
    83|    83|    83|    83|    83|    83|    83|```
    84|    84|    84|    84|    84|    84|    84|
    85|    85|    85|    85|    85|    85|    85|Configure o hook e selecione as ferramentas no menu interativo:
    86|    86|    86|    86|    86|    86|    86|
    87|    87|    87|    87|    87|    87|    87|```bash
    88|    88|    88|    88|    88|    88|    88|ai-trailer configure
    89|    89|    89|    89|    89|    89|    89|```
    90|    90|    90|    90|    90|    90|    90|
    91|    91|    91|    91|    91|    91|    91|Configure também arquivos de instrução das ferramentas selecionadas:
    92|    92|    92|    92|    92|    92|    92|
    93|    93|    93|    93|    93|    93|    93|```bash
    94|    94|    94|    94|    94|    94|    94|ai-trailer configure --instructions
    95|    95|    95|    95|    95|    95|    95|```
    96|    96|    96|    96|    96|    96|    96|
    97|    97|    97|    97|    97|    97|    97|Verifique se a sessão atual geraria trailers:
    98|    98|    98|    98|    98|    98|    98|
    99|    99|    99|    99|    99|    99|    99|```bash
   100|   100|   100|   100|   100|   100|   100|ai-trailer test
   101|   101|   101|   101|   101|   101|   101|```
   102|   102|   102|   102|   102|   102|   102|
   103|   103|   103|   103|   103|   103|   103|Veja o estado atual:
   104|   104|   104|   104|   104|   104|   104|
   105|   105|   105|   105|   105|   105|   105|```bash
   106|   106|   106|   106|   106|   106|   106|ai-trailer status
   107|   107|   107|   107|   107|   107|   107|```
   108|   108|   108|   108|   108|   108|   108|
   109|   109|   109|   109|   109|   109|   109|Consulte registros locais:
   110|   110|   110|   110|   110|   110|   110|
   111|   111|   111|   111|   111|   111|   111|```bash
   112|   112|   112|   112|   112|   112|   112|ai-trailer record
   113|   113|   113|   113|   113|   113|   113|```
   114|   114|   114|   114|   114|   114|   114|
   115|   115|   115|   115|   115|   115|   115|Atualize o binário instalado:
   116|   116|   116|   116|   116|   116|   116|
   117|   117|   117|   117|   117|   117|   117|```bash
   118|   118|   118|   118|   118|   118|   118|ai-trailer update
   119|   119|   119|   119|   119|   119|   119|```
   120|   120|   120|   120|   120|   120|   120|
   121|   121|   121|   121|   121|   121|   121|Remova apenas o hook:
   122|   122|   122|   122|   122|   122|   122|
   123|   123|   123|   123|   123|   123|   123|```bash
   124|   124|   124|   124|   124|   124|   124|ai-trailer uninstall
   125|   125|   125|   125|   125|   125|   125|```
   126|   126|   126|   126|   126|   126|   126|
   127|   127|   127|   127|   127|   127|   127|Remova o hook e reverta blocos de instrução injetados:
   128|   128|   128|   128|   128|   128|   128|
   129|   129|   129|   129|   129|   129|   129|```bash
   130|   130|   130|   130|   130|   130|   130|ai-trailer uninstall --full
   131|   131|   131|   131|   131|   131|   131|```
   132|   132|   132|   132|   132|   132|   132|
   133|   133|   133|   133|   133|   133|   133|## Como a detecção funciona
   134|   134|   134|   134|   134|   134|   134|
   135|   135|   135|   135|   135|   135|   135|### Durante o `detect`
   136|   136|   136|   136|   136|   136|   136|
   137|   137|   137|   137|   137|   137|   137|O comando `detect` procura evidências de instalação por:
   138|   138|   138|   138|   138|   138|   138|
   139|   139|   139|   139|   139|   139|   139|- binários no `PATH`;
   140|   140|   140|   140|   140|   140|   140|- pacotes npm globais;
   141|   141|   141|   141|   141|   141|   141|- pacotes pip;
   142|   142|   142|   142|   142|   142|   142|- fórmulas Homebrew;
   143|   143|   143|   143|   143|   143|   143|- arquivos de configuração conhecidos;
   144|   144|   144|   144|   144|   144|   144|- extensões em diretórios comuns do VS Code/Cursor.
   145|   145|   145|   145|   145|   145|   145|
   146|   146|   146|   146|   146|   146|   146|### Durante o commit
   147|   147|   147|   147|   147|   147|   147|
   148|   148|   148|   148|   148|   148|   148|O hook prioriza sinais da sessão ativa:
   149|   149|   149|   149|   149|   149|   149|
   150|   150|   150|   150|   150|   150|   150|| Ferramenta | Sinal usado pelo hook |
   151|   151|   151|   151|   151|   151|   151|| --- | --- |
   152|   152|   152|   152|   152|   152|   152|| Claude Code | `CLAUDE_MODEL` |
   153|   153|   153|   153|   153|   153|   153|| Hermes | `HERMES_SESSION` e `HERMES_MODEL` |
   154|   154|   154|   154|   154|   154|   154|| OpenCode | `OPENCODE_MODEL` |
   155|   155|   155|   155|   155|   155|   155|| Gemini CLI | `GEMINI_MODEL` |
   156|   156|   156|   156|   156|   156|   156|| Cursor | `CURSOR_TRACE_ID` e `~/.ai-trailer/cursor-model` |
   157|   157|   157|   157|   157|   157|   157|| Codex | `~/.ai-trailer/codex-model`, se atualizado na última hora |
   158|   158|   158|   158|   158|   158|   158|
   159|   159|   159|   159|   159|   159|   159|Para Codex, `ai-trailer configure` cria um hook `PreToolUse` em `~/.codex/hooks.json` que grava o modelo ativo em `~/.ai-trailer/codex-model`. Depois disso, é necessário confiar/ativar o hook no Codex quando a ferramenta pedir.
   160|   160|   160|   160|   160|   160|   160|
   161|   161|   161|   161|   161|   161|   161|Para Cursor, o suporte de hook ainda é tratado como não verificado. O hook do Git já consegue usar `CURSOR_TRACE_ID` e `~/.ai-trailer/cursor-model` quando esse arquivo existir.
   162|   162|   162|   162|   162|   162|   162|
   163|   163|   163|   163|   163|   163|   163|## Webhook de adoção
   164|   164|   164|   164|   164|   164|   164|
   165|   165|   165|   165|   165|   165|   165|O projeto inclui integração opcional com Google Sheets via Apps Script para eventos de adoção, como `detect`, `configure` e `uninstall`.
   166|   166|   166|   166|   166|   166|   166|
   167|   167|   167|   167|   167|   167|   167|A URL do webhook é resolvida nesta ordem:
   168|   168|   168|   168|   168|   168|   168|
   169|   169|   169|   169|   169|   169|   169|1. `AI_TRAILER_WEBHOOK_URL`
   170|   170|   170|   170|   170|   170|   170|2. `~/.ai-trailer/webhook.json`
   171|   171|   171|   171|   171|   171|   171|3. URL padrão compilada em `internal/webhook/webhook.go`
   172|   172|   172|   172|   172|   172|   172|
   173|   173|   173|   173|   173|   173|   173|Token opcional:
   174|   174|   174|   174|   174|   174|   174|
   175|   175|   175|   175|   175|   175|   175|```bash
   176|   176|   176|   176|   176|   176|   176|export AI_TRAILER_WEBHOOK_TOKEN="<token>"
   177|   177|   177|   177|   177|   177|   177|```
   178|   178|   178|   178|   178|   178|   178|
   179|   179|   179|   179|   179|   179|   179|O webhook não rastreia commits individualmente. Os commits ficam no Git e no log local; a planilha é focada em configuração e adoção.
   180|   180|   180|   180|   180|   180|   180|
   181|   181|   181|   181|   181|   181|   181|Os arquivos do Apps Script estão em `appscript/`.
   182|   182|   182|   182|   182|   182|   182|
   183|   183|   183|   183|   183|   183|   183|## Desenvolvimento
   184|   184|   184|   184|   184|   184|   184|
   185|   185|   185|   185|   185|   185|   185|Pré-requisito:
   186|   186|   186|   186|   186|   186|   186|
   187|   187|   187|   187|   187|   187|   187|```bash
   188|   188|   188|   188|   188|   188|   188|go version
   189|   189|   189|   189|   189|   189|   189|```
   190|   190|   190|   190|   190|   190|   190|
   191|   191|   191|   191|   191|   191|   191|O projeto usa Go `1.22.5`.
   192|   192|   192|   192|   192|   192|   192|
   193|   193|   193|   193|   193|   193|   193|Rodar o CLI localmente:
   194|   194|   194|   194|   194|   194|   194|
   195|   195|   195|   195|   195|   195|   195|```bash
   196|   196|   196|   196|   196|   196|   196|go run . help
   197|   197|   197|   197|   197|   197|   197|go run . detect
   198|   198|   198|   198|   198|   198|   198|go run . test
   199|   199|   199|   199|   199|   199|   199|```
   200|   200|   200|   200|   200|   200|   200|
   201|   201|   201|   201|   201|   201|   201|Rodar testes:
   202|   202|   202|   202|   202|   202|   202|
   203|   203|   203|   203|   203|   203|   203|```bash
   204|   204|   204|   204|   204|   204|   204|go test ./...
   205|   205|   205|   205|   205|   205|   205|```
   206|   206|   206|   206|   206|   206|   206|
   207|   207|   207|   207|   207|   207|   207|Gerar binário local:
   208|   208|   208|   208|   208|   208|   208|
   209|   209|   209|   209|   209|   209|   209|```bash
   210|   210|   210|   210|   210|   210|   210|go build -o /tmp/ai-trailer .
   211|   211|   211|   211|   211|   211|   211|```
   212|   212|   212|   212|   212|   212|   212|
   213|   213|   213|   213|   213|   213|   213|Definir versão no build:
   214|   214|   214|   214|   214|   214|   214|
   215|   215|   215|   215|   215|   215|   215|```bash
   216|   216|   216|   216|   216|   216|   216|go build -ldflags "-X main.Version=v0.5.1" -o /tmp/ai-trailer .
   217|   217|   217|   217|   217|   217|   217|```
   218|   218|   218|   218|   218|   218|   218|
   219|   219|   219|   219|   219|   219|   219|## Estrutura do projeto
   220|   220|   220|   220|   220|   220|   220|
   221|   221|   221|   221|   221|   221|   221|```text
   222|   222|   222|   222|   222|   222|   222|.
   223|   223|   223|   223|   223|   223|   223|├── main.go                    # comandos do CLI
   224|   224|   224|   224|   224|   224|   224|├── instrument.go              # setup de arquivos de sessão para Codex/Cursor
   225|   225|   225|   225|   225|   225|   225|├── tui_*.go                   # menu interativo por plataforma
   226|   226|   226|   226|   226|   226|   226|├── internal/config            # instalação do hook e injeção/reversão de instruções
   227|   227|   227|   227|   227|   227|   227|├── internal/detect            # registro e detecção de ferramentas de IA
   228|   228|   228|   228|   228|   228|   228|├── internal/record            # log JSONL local e sumários
   229|   229|   229|   229|   229|   229|   229|├── internal/webhook           # cliente Google Apps Script
   230|   230|   230|   230|   230|   230|   230|├── appscript                  # webhook Google Sheets
   231|   231|   231|   231|   231|   231|   231|├── dist                       # binários pré-compilados
   232|   232|   232|   232|   232|   232|   232|└── install.sh                 # instalador
   233|   233|   233|   233|   233|   233|   233|```
   234|   234|   234|   234|   234|   234|   234|
   235|   235|   235|   235|   235|   235|   235|## Segurança operacional
   236|   236|   236|   236|   236|   236|   236|
   237|   237|   237|   237|   237|   237|   237|O hook foi desenhado para não bloquear commits:
   238|   238|   238|   238|   238|   238|   238|
   239|   239|   239|   239|   239|   239|   239|- não usa `set -e`;
   240|   240|   240|   240|   240|   240|   240|- ignora commits de merge e squash;
   241|   241|   241|   241|   241|   241|   241|- não falha se `ai-trailer record-commit` não estiver disponível;
   242|   242|   242|   242|   242|   242|   242|- só adiciona cada trailer se ele ainda não existir na mensagem.
   243|   243|   243|   243|   243|   243|   243|
   244|   244|   244|   244|   244|   244|   244|O objetivo é enriquecer o histórico de commits sem quebrar o fluxo normal de desenvolvimento.
   245|   245|   245|   245|   245|   245|   245|
   246|   246|   246|   246|   246|   246|   246|Linha adicionada pelo Copilot em 2026-05-18T11:51:48-03:00
   247|   247|   247|   247|   247|   247|   247|
   248|   248|   248|   248|   248|   248|   248|Linha adicionada pelo Copilot em 2026-05-18T12:00:06-03:00
   249|   249|   249|   249|   249|   249|   249|
   250|   250|   250|   250|   250|   250|   250|Linha adicionada pelo Copilot em 2026-05-18T12:10:33-03:00
   251|   251|   251|   251|   251|   251|   251|
   252|   252|   252|   252|   252|   252|   252|Linha adicionada pelo OpenCode em 2026-05-18T12:14:00-03:00
   253|   253|   253|   253|   253|   253|   253|
   254|   254|   254|   254|   254|   254|   254|Linha adicionada pelo OpenCode em 2026-05-18T12:21:00-03:00
   255|   255|   255|   255|   255|   255|   255|Linha adicionada pelo OpenCode em 2026-05-18T12:21:22-0300
   256|   256|   256|   256|   256|   256|   256|Linha adicionada pelo OpenCode em 2026-05-18T12:46:52-0300
   257|   257|   257|   257|   257|   257|   257|
   258|   258|   258|   258|   258|   258|   258|Linha adicionada pelo OpenCode em 2026-05-18T12:52:00-0300
   259|   259|   259|   259|   259|   259|   259|
   260|   260|   260|   260|   260|   260|   260|Linha adicionada pelo OpenCode em 2026-05-18T13:02:00-0300
   261|   261|   261|   261|   261|   261|   261|
   262|   262|   262|   262|   262|   262|   262|Linha adicionada pelo OpenCode em 2026-05-18T13:03:07-0300
   263|   263|   263|   263|   263|   263|   263|
   264|   264|   264|   264|   264|   264|   264|Linha adicionada pelo OpenCode em 2026-05-18T13:16:40-0300
   265|   265|   265|   265|   265|   265|   265|
   266|   266|   266|   266|   266|   266|
   267|   267|   267|   267|   267|   267|Linha adicionada pelo Hermes Agent em 2026-05-18T13:22:10-0300
   268|   268|   268|   268|   268|   268|
   269|   269|   269|   269|   269|
   270|   270|   270|   270|   270|Linha adicionada pelo Hermes Agent em 2026-05-18T13:28:57-0300
   271|   271|   271|   271|   271|
   272|   272|   272|   272|
   273|   273|   273|   273|Linha adicionada pelo Hermes Agent em 2026-05-18T13:32:23-0300
   274|   274|   274|   274|
   275|   275|   275|
   276|   276|   276|Linha adicionada pelo Hermes Agent em 2026-05-18T13:35:50-0300
   277|   277|   277|
   278|   278|
   279|   279|Linha adicionada pelo Hermes Agent em 2026-05-18T13:38:34-0300
   280|   280|
   281|
   282|Linha adicionada pelo Hermes Agent em 2026-05-18T13:43:48-0300
   283|

Linha adicionada pelo Hermes Agent em 2026-05-18T13:44:36-0300
