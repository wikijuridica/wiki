# DECISIONS.md

## 2026-06-09 — Go como base propria do portal

Decisao: usar Go e biblioteca padrao como base do projeto.

Motivos:
- linguagem compilada, tipada e adequada a servicos com concorrencia;
- biblioteca padrao suficiente para HTTP, XML, HTML escaping, arquivos e testes;
- binario proprio e operacao sem framework frontend;
- melhor alinhamento com geracao on demand propria e cache controlado pelo projeto.

Evidencia documental consultada:
- `https://go.dev/doc/`: documenta Go como linguagem eficiente, compilada, tipada, com concorrencia e toolchain oficial.
- `https://go.dev/doc/modules/layout`: recomenda `internal/` e `cmd/` para projetos de servidor.
- `https://pkg.go.dev/net/http`: biblioteca padrao para cliente e servidor HTTP.
- `https://pkg.go.dev/encoding/xml`: biblioteca padrao para XML, usada em sitemaps.
- `https://pkg.go.dev/testing`: biblioteca padrao para testes automatizados.

Alternativas avaliadas:
- Python com biblioteca padrao: suficiente para prototipar validadores, mas menos alinhado ao objetivo de portal massivo com binario proprio e geracao HTTP concorrente;
- Next.js ou frameworks frontend: proibidos pelo contrato P0.

Consequencia: qualquer artefato Python criado no ciclo foi descartado; o contrato e a implementacao passam a ser Go-first.

## 2026-06-09 — Geracao on demand propria obrigatoria

Decisao: o portal deve gerar paginas sob demanda com codigo proprio em `internal/ondemand`, sem Next.js, ISR terceirizado ou framework equivalente.

Motivos:
- o projeto quer autonomia sobre roteamento, cache e politica de indexacao;
- rotas futuras podem chegar a milhoes de URLs e nao devem depender de build total;
- o primeiro response de pagina publica deve conter HTML textual completo.

Consequencia: build estatico e permitido como ferramenta operacional, mas nao substitui o gerador on demand.

## 2026-06-09 — Checkpoint nao encerra trabalho

Decisao: checkpoint e rastreabilidade operacional para continuar, nao entrega final nem ordem de parada.

Motivos:
- o projeto tem escopo continuo e grande;
- cada ciclo precisa deixar contexto, provas, falhas e proximo plano;
- o agente deve continuar trabalhando enquanto nao houver bloqueio P0 real.

Consequencia: `CHECKPOINT.md` deve registrar plano de continuidade explicito e nao pode ser usado como substituto da definicao de pronto.

Adendo operacional: sempre planejar o proximo passo e continuar executando. Uma resposta no thread ou um checkpoint nao encerram a missao.

Adendo de persistencia: cada ciclo deve ser commitado apos validacao, incluindo `CHECKPOINT.md`, para preservar continuidade em Git.
Antes do commit, registrar hora local e numero do ciclo no checkpoint.

Adendo de autonomia: Codex atua como engenheiro senior, arquiteto e criador de conteudo juridico. Se houver trabalho no escopo ou bug identificado, deve continuar e corrigir sem pedir aprovacao para decisao normal de engenharia.

## 2026-06-09 — Meta de 10 mil paginas com CTA subordinado a qualidade

Decisao: a meta de produto inclui no minimo 10 mil paginas juridicas informativas, com alta intencao de contratar advogado e CTA proprio de WhatsApp quando apropriado.

Motivos:
- o portal deve ter escala nacional e alta intencao comercial;
- a arquitetura precisa suportar grande volume sem URLs infinitas;
- CTA comercial nao pode sacrificar fonte, revisao, utilidade e seguranca juridica.

Consequencia: durante P0 o projeto cria plano de blueprints e politica de CTA, mas nao publica 10 mil paginas juridicas. A publicacao em escala depende dos gates P0/P1/P2.

Adendo de intencao digital: alta intencao comercial significa potencial de contratacao juridica 100% online. O primeiro lote de termos deve favorecer problemas juridicos que podem ser triados, documentados e contratados por canais digitais, especialmente WhatsApp. Demanda alta com dependencia presencial predominante nao deve ser tratada como prioridade inicial.

## 2026-06-09 — Laboratorio antes de conteudo e mudancas P0

Decisao: toda mudanca P0/P1 deve passar por laboratorio de testes, validacao, refinamento e reteste.

Motivos:
- scripts isolados podem passar e ainda deixar falha de contrato;
- conteudo juridico exige fonte correta e escrita natural;
- escala de 10 mil paginas sem laboratorio vira risco de spam, duplicidade e baixa qualidade.

Consequencia: `tools/lab-cycle` passa a ser o comando minimo de laboratorio, e conteudo juridico em escala permanece bloqueado ate pesquisa de fonte, documentacao, revisao e gates.

## 2026-06-09 — Fontes como referencia, nao scraping

Decisao: fontes oficiais e APIs publicas registradas sao referencia/proveniencia, nao alvo de scraping, clone ou espelhamento.

Motivos:
- o portal deve ser unico e editorialmente proprio;
- copiar massa de dados ou estrutura oficial cria risco de spam e baixa qualidade;
- conteudo juridico precisa ser natural, humano e contextualizado.

Consequencia: qualquer uso de fonte oficial exige pesquisa critica, registro de proveniencia e redacao propria. Ingestao automatica permanece bloqueada em P0.

## 2026-06-09 — HTML publico leve para bots valiosos

Decisao: pagina publica deve permanecer leve por contrato, com HTML textual completo, CSS minimo e sem runtime cliente.

Motivos:
- Googlebot, OAI-SearchBot e bots valiosos precisam rastrear conteudo textual com baixo custo;
- escala massiva amplifica qualquer excesso de bytes, bundle ou hidratacao;
- framework frontend, payload JavaScript e ferramenta pesada no HTML publico contrariam P0/P1.

Consequencia: `./tools/check-performance-budget` reprova HTML acima do orcamento, `<script>`, referencias a `.js/.mjs/.wasm`, assets pesados, `modulepreload`, import map, marcadores de hidratacao e sinais de runtime/framework. UI publica deve ser resolvida com HTML semantico e CSS minimo.

## 2026-06-09 — Algoritmos auditaveis e explicaveis

Decisao: gates e algoritmos do projeto devem ser inteligentes, auditaveis e explicaveis.

Motivos:
- escala massiva torna heuristica burra perigosa e cara;
- qualidade juridica depende de decisoes rastreaveis, nao de aprovacoes opacas;
- falsos positivos e falsos negativos precisam virar melhoria de algoritmo, nao excecao manual permanente.

Consequencia: quando um algoritmo estiver ingenuo, caro, opaco, permissivo demais ou agressivo demais, o agente deve escrever teste que reproduza a falha, melhorar a regra, manter mensagens especificas de reprovação e registrar a decisao. Heuristicas simples sao permitidas como etapa inicial, mas devem evoluir quando houver sinal melhor disponivel.

## 2026-06-09 — Conteudo publico em PT-BR correto

Decisao: todo conteudo visivel ao publico deve ser escrito em PT-BR com grafia correta, acentuacao correta, pontuacao clara e linguagem natural.

Motivos:
- o portal serve primeiro a humanos no Brasil;
- texto publico sem acento ou com grafia tecnica empobrece confianca editorial;
- qualidade juridica depende de clareza, naturalidade e revisao humana.

Consequencia: rascunhos tecnicos internos podem usar texto operacional sem polimento, mas paginas, metadados visiveis, CTA, navegacao e rodape publicos devem ser PT-BR correto. O algoritmo de normalizacao de qualidade deve preservar letras acentuadas para nao degradar os gates em portugues.

## 2026-06-09 — Orcamento SERP conservador baseado em pesquisa Google

Decisao: o projeto usa limites internos conservadores para `title`, metadescricao e snippet, sem afirmar que sao limites oficiais fixos do Google.

Pesquisa oficial feita no dia da sessao:
- `https://developers.google.com/search/docs/appearance/title-link?hl=pt-BR`
- `https://developers.google.com/search/docs/appearance/snippet?hl=pt-br`
- `https://developers.google.com/search/docs/essentials/technical`

Motivos:
- Google informa que nao ha limite fixo oficial para `<title>` e metadescricoes;
- os textos podem ser truncados conforme a largura do dispositivo;
- conteudo indexavel exige acesso do Googlebot, HTTP 200 e conteudo que nao viole politicas de spam.

Consequencia: `title` deve ter 20 a 65 caracteres Unicode, metadescricao deve ter 70 a 160 caracteres Unicode e paginas indexaveis recebem `max-snippet:160`. Quando faltarem dados atuais sobre Googlebot ou superficie de busca, pesquisar a Central da Pesquisa Google no dia da sessao.

## 2026-06-09 — Laboratorio contra conteudo mecanico antes do Googlebot

Decisao: conteudo raso, mecanico ou criado por permutacao de palavras-chave deve ser detectado em laboratorio antes de qualquer exposicao ao Googlebot.

Motivos:
- a meta de escala nao autoriza spam;
- humanos e Googlebot precisam receber conteudo natural, util e especifico;
- scripts isolados podem ser enganados por texto longo, mas repetitivo.

Consequencia: `./tools/lab-content-quality` cria textos temporarios em `/tmp`, aprova texto natural e reprova texto mecanico. `./tools/check-mechanical-content` bloqueia paginas indexaveis com sinais de thin content, keyword stuffing, baixa diversidade lexical, frases repetidas ou permutacao mecanica. Falsos positivos e falsos negativos devem virar melhoria de algoritmo.

## 2026-06-09 — CPU baixo no runtime publico

Decisao: CPU deve ser reservada para trafego legitimo, Googlebot, OAI-SearchBot e bots valiosos no runtime publico.

Motivos:
- producao precisa atender muitas URLs e bots sem desperdiçar CPU;
- renderizacao publica deve ser previsivel, local e barata;
- laboratorio pode ser mais pesado quando necessario para provar qualidade, mas isso nao pode virar dependencia do atendimento publico.

Consequencia: `./tools/check-cpu-budget` escaneia caminhos de runtime publico/producao e reprova execucao externa, chamadas de rede, sleeps ou loops sem limite. Testes, build, laboratorio e auditorias podem usar comandos mais pesados quando forem necessarios e registrados.

## 2026-06-09 — Modo /goal nao permite parada operacional

Decisao: quando o Codex estiver em modo `/goal`, nao deve parar enquanto houver trabalho no escopo e nao houver bloqueio P0 real comprovado.

Motivos:
- checkpoint e resposta no thread sao rastreabilidade, nao ordem de parada;
- o projeto exige continuidade autonoma e documentada;
- o proximo passo deve sobreviver a compactacao de contexto, troca de sessao e perda de chat.

Consequencia: a primeira linha de `AGENTS.md` registra a regra literal de `/goal`; `GOAL.md`, `CHECKPOINT.md` e documentos de decisao devem manter proximo passo planejado e continuidade explicita. O agente deve executar o proximo passo documentado sem aguardar nova cobranca do usuario, salvo bloqueio P0 real comprovado.

## 2026-06-09 — Sem mascarar pendencia e com rede quando necessaria

Decisao: nenhuma pendencia pode ser mascarada como entrega, e rede deve ser usada quando for necessaria para resolver o escopo.

Motivos:
- registrar bloqueio ou "pendente" como avanço desperdiça ciclo e nao resolve o projeto;
- fontes oficiais, robots.txt, termos de uso, APIs publicas e documentos atuais exigem verificacao real quando impactam decisao;
- sandbox sem rede nao e desculpa para parar quando existe mecanismo de escalonamento pela ferramenta.

Consequencia: Nada deve ser deixado para o futuro por conveniencia. Tudo que estiver no escopo e para esta sessao. Alternativa so e valida quando resolve o requisito ou produz prova executavel que desbloqueia o requisito; artefato que nao sera usado nao conta como entrega. Se a rede do sandbox falhar e a rede for necessaria, o agente deve repetir o comando com `sandbox_permissions=require_escalated` e justificativa objetiva pela ferramenta, sem perguntar no chat. Se o escalonamento for negado, a negativa vira bloqueio real com evidencia, nao pendencia mascarada.

## 2026-06-09 — Fonte especifica por matriz auditada destrava lote bloqueado

Decisao: `refresh-batch-candidate-pipeline` deve promover fonte travada por matriz quando existir URL oficial especifica auditada, em vez de depender de rascunho final preexistente por intent.

Motivos:
- o comportamento anterior travava artificialmente variantes de um mesmo subtema que ja tinham fonte especifica;
- escala massiva exige destravar por prova de fonte e paid-intent, nao por edicao manual de cada pagina;
- a Resolução CNJ 35/2007 foi verificada no portal de atos do CNJ e cobre atos notariais de inventario, partilha, divorcio consensual e uniao estavel por via administrativa.

Consequencia: `batch_source_specificity_resolutions` passou a usar `BuildSpecificSourceURLsByMatrix`; fontes amplas como Codigo Civil, CDC, CLT, raiz de orgao ou home institucional continuam bloqueadas sem URL especifica. O ciclo promoveu 102 candidatos para fonte travada e 102 rascunhos finais bloqueados, mantendo `noindex`, sem render, sem sitemap, sem publicacao e sem scraping/ingestao.

## 2026-06-09 — Ancoras oficiais especificas do Planalto como referencia bloqueada

Decisao: matrizes que ainda dependiam de CLT, Codigo Civil ou CDC amplos podem usar ancoras de artigo em `www.planalto.gov.br/ccivil_03.old/...#art...` como fonte especifica auditada, desde que o uso seja estritamente `reference_only_no_scraping_no_ingestion`.

Motivos:
- o ciclo precisava reduzir o bloqueio de fonte sem mascarar fonte ampla como especifica;
- a propria resposta web do Planalto para a CLT compilada apontou o caminho historico `ccivil_03.old`;
- artigo especifico e uma referencia juridica mais precisa do que a pagina compilada inteira para rescisao, jornada, justa causa, verbas, alimentos, guarda, revisao de pensao, negativacao e cobranca.

Consequencia: o audit URL-level registra robots, termos, hash, timeout HTTP local/escalonado e uso bloqueado; nao ha scraping, ingestao, render, sitemap ou publicacao. A fonte ampla continua na matriz como contexto, mas o destravamento de `final_source_locked_reference_only` passa a exigir a URL de artigo especifica auditada.

## 2026-06-09 — Estrategia de expansao bloqueada antes de ampliar lote

Decisao: quando fonte/readiness deixam de ser gargalo, o proximo passo deve ser registrado em `batch_expansion_strategy` antes de ampliar candidatos, para evitar checkpoint passivo.

Motivos:
- source locked e rascunho final bloqueado nao sao publicacao nem conclusao do `/goal`;
- escala massiva precisa de crescimento real, mas por passos auditaveis para nao virar spam;
- familias prontas devem crescer, enquanto familia com paid-intent reprovado deve preservar bloqueio comercial.

Consequencia: `internal/batchexpansionstrategy`, `data/editorial/batch_expansion_strategy.jsonl`, `cmd/refresh-batch-expansion-strategy`, `./tools/check-batch-expansion-strategy` e `./tools/refresh-batch-expansion-strategy` entram no laboratorio. O plano atual aumenta 5 familias de 30 para 60 candidatos no proximo gate e mantem previdenciario em 18 ate refino/bloqueio comercial, sem render, sitemap, manifest público ou publicação.

## 2026-06-09 — Banco leve separado para ingestao de termos

Decisao: ingestao de termos juridicos pode iniciar conteudos apenas como semente de rascunho, usando banco leve proprio em JSONL e camadas separadas.

Motivos:
- termos juridicos sao uma boa unidade inicial para organizar pautas e glossario sem criar spam;
- fonte auditada, termos de uso, snapshot oficial, rascunho editorial e conteudo publicado tem riscos e contratos diferentes;
- usar Go e arquivos JSONL evita dependencia externa, SDK, SaaS ou banco pesado durante P0.

Consequencia: `content/storage_contract.json` define `term_seeds`, `source_audits`, `source_snapshots`, `editorial_drafts` e `published_manifest`. `./tools/check-storage-contract` reprova camada ausente, caminho compartilhado, arquivo inexistente, record pesado, camada diretamente indexavel ou mistura entre fonte bruta e texto editorial. Termos juridicos so podem iniciar `draft_only` ate fonte, revisao, qualidade, SEO, CTA e checkpoint passarem.

## 2026-06-09 — URL oficial do Portal da Legislacao

Decisao: para o registro Planalto, a URL oficial de pesquisa do Portal da Legislacao passa a ser `https://legislacao.presidencia.gov.br/`, com atos em `https://legislacao.presidencia.gov.br/atos/?...`.

Motivos:
- pagina publica confiavel do proprio `gov.br` para o servico "Realizar pesquisa de legislacao no Portal da Legislacao" aponta o canal Web para `legislacao.presidencia.gov.br`;
- o resultado tambem descreve que a pesquisa acessa atos normativos federais de hierarquia superior e a base REFLEGIS;
- `www.planalto.gov.br` e `www4.planalto.gov.br` sao referencias historicas/auxiliares, mas falharam na auditoria HTTP local desta sessao.

Consequencia: `content/source_registry.json` registra `https://legislacao.presidencia.gov.br/` como base oficial e mantem ingestao bloqueada. A auditoria local teve timeout em `legislacao.presidencia.gov.br`, reset em `www.planalto.gov.br` e timeout em `www4.planalto.gov.br`; portanto a URL foi confirmada documentalmente, mas coleta automatica continua proibida.

## 2026-06-09 — Term seeds como pauta, nao pagina

Decisao: `data/terms/legal_terms.jsonl` pode conter sementes de termos juridicos para laboratorio editorial, desde que cada registro seja `draft_only`, PT-BR, tenha fonte, URL oficial, data de verificacao e intencao editorial.

Motivos:
- termos sao unidade leve para iniciar pauta e glossario sem criar pagina automaticamente;
- uma seed sem fonte vira risco de conteudo juridico inventado;
- um termo publicado direto seria spam ou thin content.

Consequencia: `internal/terms` e `./tools/check-term-seeds` validam as seeds. O laboratorio pode usar essas seeds para rascunhos temporarios, mas nenhuma seed vira pagina publica, CTA ou URL indexavel sem passar pelos demais gates.

## 2026-06-09 — Rascunho temporario antes de pagina publica

Decisao: seed valida pode gerar rascunho temporario em `/tmp` por `./tools/lab-term-draft`, sempre `draft/noindex` e fora do manifesto publico.

Motivos:
- permite testar escrita natural, fonte e aviso informativo antes de tocar o pipeline publico;
- impede que uma seed vire pagina mecanica ou thin content;
- preserva baixo CPU e HTML leve em producao, deixando o experimento no laboratorio.

Consequencia: `internal/draftlab` gera rascunho proprio em PT-BR e roda `quality.AnalyzeText`. O rascunho nao recebe URL publica, nao entra em sitemap, nao recebe CTA e nao altera `content/pages.json`.

## 2026-06-09 — Goal nao termina em ciclo parcial

Decisao: nenhum ciclo e final e parada. `/goal` nao pode ser marcado como completo por checkpoint, commit, laboratorio verde, P0 parcial, seed ou rascunho.

Motivos:
- o objetivo contratado inclui plataforma grande e minimo de 10 mil paginas publicas juridicas aprovadas;
- o projeto ainda esta em P0/laboratorio e nao saiu para publicacao em escala;
- marcar goal como completo em ciclo parcial distorce a missao e causa parada indevida.

Consequencia: `AGENTS.md`, `GOAL.md` e testes de contrato exigem nao marcar `/goal` como completo ate a meta publica minima estar verificada: no minimo 10 mil paginas publicas aprovadas, indexaveis, com fonte, revisao, qualidade, CTA quando cabivel, sitemap/canonical/robots corretos e validacao completa.

## 2026-06-09 — Proibido finalizar goal por ferramenta em P0

Decisao: o agente nao pode chamar `update_goal` com `status=complete` enquanto o projeto estiver em P0/laboratório ou antes de 10 mil paginas publicas juridicas aprovadas e verificadas.

Motivos:
- o goal ativo e a meta publica minima sao maiores que qualquer checkpoint de laboratorio;
- resposta final no thread, commit, check verde ou arquivo persistido nao provam o objetivo real;
- marcar completion por ferramenta encerraria a continuidade operacional contra o contrato.

Consequencia: a primeira linha de `AGENTS.md` e `GOAL.md` explicita a proibicao operacional. `internal/contract/continuity_test.go` reprova se a regra sumir dos contratos. O agente deve deixar o goal ativo, registrar o proximo ciclo e continuar trabalhando enquanto nao houver bloqueio P0 real comprovado.

## 2026-06-09 — Continuidade preserva o objetivo completo apos compactacao

Decisao: quando houver `codex_internal_context`, resumo de retomada, compactação ou subobjetivo, o Codex deve preservar o objetivo completo e nao pode reduzir o objetivo total ao recorte do ciclo atual.

Motivos:
- compactação pode mostrar apenas um subobjetivo e ocultar parte do contexto operacional;
- resposta final no thread é relatório de checkpoint, não é decisão de conclusão;
- checkpoint, commit, teste verde e retomada de ciclo sao rastreabilidade, nao aceite final.

Consequencia: e proibido chamar update_goal status=complete por engano por causa de `codex_internal_context`, compactação, subobjetivo, resposta final, checkpoint ou commit. `AGENTS.md`, `GOAL.md` e `internal/contract/continuity_test.go` devem manter essa regra literal enquanto a meta minima de 10 mil paginas publicas juridicas aprovadas nao estiver comprovada.

## 2026-06-09 — Draft editorial persistido ainda nao e publicacao

Decisao: rascunho validado pode ser persistido em `data/editorial/drafts.jsonl`, mas continua `draft/noindex`, sem rota publica, sem sitemap e sem CTA.

Motivos:
- persistir rascunho reduz perda de contexto entre ciclos sem expor conteudo incompleto;
- separar `editorial_drafts` de `published_manifest` impede que laboratorio seja confundido com pagina publica;
- validacao de qualidade deve acompanhar a persistencia, nao ficar apenas no texto temporario.

Consequencia: `internal/editorialdrafts`, `./tools/persist-term-drafts` e `./tools/check-editorial-drafts` controlam a camada editorial. O fluxo atual e seed -> draft temporario -> draft persistido; ainda nao existe publicacao indexavel desse conteudo.

## 2026-06-09 — Fila editorial antes de promocao

Decisao: draft persistido deve entrar em `data/editorial/review_queue.jsonl` como `needs_review`, com autoria, motivo e historico, ainda com `publication_allowed=false`.

Motivos:
- revisao precisa ser rastreavel antes de qualquer promocao;
- separar fila de revisao de draft e manifesto publicado evita confundir laboratorio com publicacao;
- a meta de 10 mil paginas exige processo repetivel, nao improviso por pagina.

Consequencia: `internal/reviewqueue`, `./tools/queue-editorial-review` e `./tools/check-review-queue` validam a fila. Nenhuma entrada da fila cria URL publica, sitemap, canonical ou CTA.

## 2026-06-09 — Aprovacao editorial ainda nao publica

Decisao: aprovacao editorial de laboratorio fica em `data/editorial/approved_drafts.jsonl` e continua com `publication_allowed=false`, `public_path` vazio e `noindex`.

Motivos:
- aprovacao de texto nao equivale a publicacao tecnica;
- publicacao publica exige fonte especifica, revisao juridica completa, SEO, CTA, sitemap, canonical, robots e escala segura;
- separar aprovacao editorial de manifesto publicado evita salto indevido do laboratorio para Googlebot.

Consequencia: `internal/approvals`, `./tools/approve-editorial-review` e `./tools/check-approvals` validam aprovacao sem publicacao. O proximo contrato deve preparar manifest de publicacao bloqueado antes de qualquer URL publica.

## 2026-06-09 — Manifesto de publicacao bloqueada

Decisao: aprovacao editorial deve gerar manifesto de publicacao bloqueada antes de qualquer URL publica.

Motivos:
- explicitar requisitos faltantes evita mascarar laboratorio como producao;
- publicacao em escala exige lista objetiva de pendencias por termo;
- o fluxo precisa saber quando migrar do laboratorio para host/produção controlada.

Consequencia: `internal/publicationblockers`, `./tools/block-publication` e `./tools/check-publication-blockers` registram `publication_status=blocked`, `publication_allowed=false`, `public_path=""` e requisitos como pesquisa de fonte especifica, revisor juridico real, intencao unica, canonical, SEO, CTA e capacidade de lote.

## 2026-06-09 — Termos humanos de alta intencao antes de conteudo publico

Decisao: quando os gates passarem com seguranca, o agente deve migrar o fluxo para host/produção controlada e priorizar termos juridicos mais pesquisados por humanos e com alta intencao de contratar advogado online.

Motivos:
- o objetivo e portal juridico util e comercial, nao laboratorio infinito;
- termos reais de busca humana reduzem risco de pagina artificial;
- CTA WhatsApp so faz sentido em temas com intencao de contratacao e utilidade juridica clara;
- servicos juridicos digitais sao prioridade: triagem, envio de documentos e contratacao remota devem ser viaveis sem atendimento presencial como padrao.

Consequencia: antes de criar conteudo publico, deve haver pesquisa atual de demanda/intencao, registro no banco leve, fonte confiavel, intencao unica e bloqueio contra spam. Essa regra nao libera publicacao automatica; ela define o proximo movimento apos gates seguros.

## 2026-06-09 — Caminhos seguros para demanda humana

Decisao: usar Google Trends e Google Search Central como caminhos seguros para orientar demanda humana e estrategia, sem tratar esses caminhos como fonte juridica ou como autorizacao de publicacao.

Caminhos registrados:
- Google Trends Explore Brasil: `https://trends.google.com.br/trends/explore?geo=BR`
- Ajuda do Google Trends sobre comparacao: `https://support.google.com/trends/answer/4359550?hl=pt-BR`
- FAQ de dados do Google Trends: `https://support.google.com/trends/answer/4365533?hl=pt-br`
- Google Search Central sobre Trends: `https://developers.google.com/search/docs/monitor-debug/trends-start`

Motivos:
- o Google Trends permite comparar termos e observar interesse de busca, mas seus dados sao normalizados e direcionais;
- a Central da Pesquisa Google orienta usar Trends para estrategia de conteudo sem escrever apenas porque algo esta em alta;
- o projeto precisa escolher termos com demanda humana real antes de criar pauta publica.

Consequencia: `data/terms/intent_candidates.jsonl` deve registrar URL de comparacao, fonte juridica oficial, adequacao a contratacao 100% digital e bloqueio de publicacao. Nenhum candidato vira pagina publica sem novo ciclo de fonte, revisao, qualidade, SEO e CTA.

## 2026-06-09 — Ranking refinavel de termos, sem confiar no primeiro sinal

Decisao: promover candidatos de alta intencao para `term_seeds` por ranking refinavel e testado, nao por lista fixa nem confianca cega em Google Trends.

Motivos:
- um unico sinal de demanda pode ser enganoso, sazonal ou amplo demais;
- termo presencial pode ter volume, mas baixa adequacao ao produto 100% digital;
- fonte nao oficial ou fraca aumenta risco juridico e editorial;
- concentrar todos os termos em uma area reduz aprendizado e escala.

Consequencia: `internal/termpromotion` pontua candidatos, penaliza fonte nao oficial e modo presencial, exige diversidade de areas e preserva evidencia de demanda na seed. `./tools/promote-term-candidates` e `./tools/check-promoted-term-seeds` mantem seeds como `draft_only`, sem URL publica e sem CTA publico.

## 2026-06-09 — Rascunho PT-BR natural tambem no laboratorio

Decisao: rascunhos editoriais persistidos devem usar grafia natural em PT-BR, mesmo antes de publicacao.

Motivos:
- laboratorio e onde erro mecanico deve aparecer, nao no Googlebot;
- texto sem acento ou sem conectivos naturais e sinal de algoritmo burro;
- persistir rascunho ruim aumenta risco de promover conteudo fraco depois.

Consequencia: `internal/draftlab` aplica termo de exibicao natural; `./tools/refresh-editorial-drafts` regenera drafts persistidos apos refinamento; fila de revisao atualiza registros existentes sem liberar publicacao.

## 2026-06-09 — Fonte especifica bloqueia aprovacao

Decisao: termo priorizado com fonte ampla, institucional ou generica nao pode ser aprovado nem publicado ate haver manifesto de fonte especifica resolvido.

Motivos:
- demanda humana e CTA alto nao substituem base juridica correta;
- fonte institucional pode orientar pesquisa, mas texto publico precisa regra, norma, artigo, requisito ou limite juridico aplicavel;
- bloquear aprovacao evita que rascunho de laboratorio vire conteudo publico por engano.

Consequencia: `data/editorial/source_blockers.jsonl` registra `approval_allowed=false`, `publication_allowed=false`, `public_path=""`, requisitos faltantes e proxima pesquisa por termo. `./tools/check-source-specificity-blockers` entra no laboratorio.

## 2026-06-09 — Pesquisa editorial manual antes de scripts de termo

Decisao: a estrategia principal para escolher termos de alta intencao passa a ser pesquisa editorial manual na web, nao script conservador ou gerador de termos.

Motivos:
- termos juridicos de contratacao digital dependem de leitura de intencao humana, nao apenas heuristica;
- Google Trends e util como orientacao direcional, mas nao substitui julgamento editorial;
- fontes oficiais dao autoridade, mas nao sao fonte de demanda nem autorizam clone;
- a meta de 10 mil paginas exige uma base leve, organizada e escalavel sem template mecanico.

Consequencia: `data/research/high_intent_terms.jsonl` vira o banco leve de pesquisa manual; `data/editorial/content_briefs.jsonl` inicia conteudos como briefs nao publicaveis; `./tools/check-manual-keyword-research` e `./tools/check-content-briefs` entram no laboratorio. Nenhum brief vira pagina publica sem fonte especifica, revisao, qualidade, SEO, CTA e nova validacao.

## 2026-06-09 — Rascunhos autorais antes de qualquer pagina de alta intencao

Decisao: briefs pesquisados podem virar rascunhos autorais no banco leve, mas esses rascunhos continuam bloqueados para publicacao ate passarem por fonte especifica, revisao editorial, SEO/crawl, qualidade, CTA e checkpoint.

Motivos:
- o projeto precisa comecar a construir conteudo sem gerar spam, clone ou pagina mecanica;
- Googlebot e humanos devem receber apenas paginas com valor proprio, nao texto de molde;
- CTA WhatsApp e critico, mas deve ser contextual e responsavel;
- a escala de 10 mil paginas exige banco organizado antes de renderizacao publica.

Consequencia: `data/editorial/authorial_drafts.jsonl` vira camada propria de rascunho autoral; `internal/authorialdrafts` e `./tools/check-authorial-content-drafts` reprovam abertura repetida, shape de secoes reaproveitado, heading generico, CTA raso, fonte ausente, publicacao permitida e path publico.

## 2026-06-09 — Resolução de fonte específica antes de pré-publicação

Decisao: fonte específica resolvida deve virar manifesto próprio antes de qualquer contrato de publicação, sem remover automaticamente bloqueadores nem criar URL pública.

Motivos:
- o projeto precisa diferenciar fonte ampla de fonte realmente útil para revisar o texto;
- fontes oficiais devem dar autoridade e proveniência, não texto copiado;
- Googlebot só deve ver página depois de fonte, revisão, qualidade, SEO/crawl e CTA passarem;
- a escala de 10 mil páginas exige rastreabilidade por termo.

Consequencia: `data/editorial/source_resolutions.jsonl` registra a primeira resolução para `negativa-cobertura-plano-saude`; `internal/sourceresolutions` e `./tools/check-source-specificity-resolutions` exigem lei primária, regra de cobertura, fontes oficiais específicas, score mínimo, `noindex`, `publication_allowed=false` e `public_path=""`.

## 2026-06-09 — Gate SEO/crawl de pré-publicação bloqueada

Decisao: uma rota candidata pode ter title, meta description, canonical e path planejados antes de publicar, mas esse planejamento deve ficar em gate bloqueado e nao pode renderizar HTML publico.

Motivos:
- Googlebot deve ver somente paginas realmente aprovadas;
- SEO tecnico deve ser validado antes de render publico, nao depois;
- fonte resolvida nao remove sozinha bloqueio editorial, CTA e revisao juridica;
- a escala de 10 mil paginas exige paths finitos e canonicals planejados sem criar URL prematura.

Consequencia: `data/editorial/prepublication_gates.jsonl` registra a primeira rota candidata para `negativa-cobertura-plano-saude`; `internal/prepublication` e `./tools/check-prepublication-gates` exigem fonte resolvida, blocker ativo, path limpo, canonical HTTPS, `noindex,follow`, title/meta dentro do orcamento, `render_allowed=false`, `sitemap_allowed=false`, `publication_allowed=false` e `public_path=""`.

## 2026-06-09 — Revisão jurídico-editorial bloqueada com CTA contextual

Decisao: CTA WhatsApp e parte critica do produto, mas deve passar por revisao juridico-editorial antes de ficar visivel.

Motivos:
- paginas informativas podem ter alta intencao de contratacao sem prometer resultado;
- CTA agressivo sem fonte e revisao vira risco juridico e conteudo ruim para humanos;
- o fluxo digital precisa pedir documentos relevantes sem induzir expectativa falsa;
- publicar CTA antes dos gates poderia contaminar a pagina e prejudicar confianca.

Consequencia: `data/editorial/legal_reviews.jsonl` registra a primeira revisao bloqueada; `internal/legalreviews` e `./tools/check-legal-editorial-reviews` exigem rascunho autoral, fonte resolvida, gate de pre-publicacao, notas juridicas, correcoes pendentes, CTA WhatsApp sem promessa, mensagem contextual com origem da pagina/rota candidata, `render_allowed=false`, `sitemap_allowed=false`, `publication_allowed=false` e `public_path=""`. `content/cta_policy.json` tambem exige template global com `{path}`, `{unique_intent_id}` e `{title}`.

## 2026-06-09 — Mudança para fábrica massiva de conteúdo único

Decisao: o projeto nao deve depender de revisão humana página a página nem limitar a produção a uma página ou poucas dezenas de rascunhos. A estratégia passa a ser geração massiva por lotes, com validação automática agressiva, score humano/IA-like, reescrita automática e bloqueio de lote quando houver spam, template ou baixa utilidade.

Motivos:
- a meta real é portal jurídico massivo, com potencial para milhões de páginas;
- revisão manual repetitiva não escala;
- conteúdo único e natural precisa ser propriedade do algoritmo, não exceção artesanal;
- Googlebot deve encontrar páginas informativas, úteis e leves, não templates;
- CTA WhatsApp deve ser contextual e lucrativo para advogado, sem transformar o texto em anúncio.

Consequencia: próximos ciclos devem implementar `human_content_score` e `scalable_content_batches`. Cada lote deve gerar muitas intenções únicas de contratação jurídica 100% digital, validar fonte, CTA contextual, score de naturalidade, similaridade intra-lote e bloqueio de publicação. Itens abaixo do score devem ser reescritos automaticamente e revalidados, não entregues ao usuário para correção manual.

## 2026-06-09 — Engenharia agressiva inteligente e autocrítica pré-commit

Decisao: Codex deve operar com engenharia agressiva inteligente: planejar a hipótese, executar sem passividade, validar em massa quando o escopo for massa, refinar algoritmo e registrar autocrítica antes de commitar.

Motivos:
- o contrato do projeto ja define plataforma juridica massiva, alta intencao e contratação digital;
- perguntas desnecessarias e decisões medrosas atrasam o P0;
- commit por ciclo é rastreabilidade, não conclusão do `/goal`;
- massa sem validação em massa vira spam, e validação tímida não prova milhões de páginas;
- o agente precisa apontar o que pode melhorar e o próximo ciclo antes de preservar o checkpoint.

Consequencia: antes de cada commit, `CHECKPOINT.md` deve registrar o que foi resolvido, provas, autocrítica, pendências reais, melhorias possíveis, próximo ciclo e a frase operacional de que o commit não encerra o `/goal`. O laboratório pode usar CPU agressivamente para testes, score, reescrita, auditoria e validação em massa; o baixo consumo de CPU continua obrigatório no runtime público/produção.

## 2026-06-09 — Score humano e lotes massivos bloqueados

Decisao: implementar `human_content_score` e `scalable_content_batches` como primeiras camadas executáveis da fábrica massiva de conteúdo jurídico único, mantendo tudo bloqueado para render, sitemap, indexação e publicação.

Motivos:
- o projeto precisa planejar milhões de páginas sem criar spam nem páginas públicas prematuras;
- validação massiva precisa ser artefato executável, não promessa em contrato;
- score humano/IA-like deve orientar reescrita automática e bloqueio de lote;
- CTA WhatsApp contextual precisa nascer no lote com origem e documentos esperados;
- produção continua leve, enquanto laboratório pode usar CPU para score e validação em massa.

Consequencia: `data/editorial/scalable_content_batches.jsonl` registra 1.020.000 páginas planejadas em seis famílias jurídicas digitais, todas bloqueadas. `data/editorial/human_content_scores.jsonl` registra scores humanos/naturalidade sem publicar conteúdo. `internal/humanscore`, `internal/scalablebatches`, `./tools/check-human-content-score` e `./tools/check-scalable-content-batches` entram no laboratório. O próximo ciclo deve gerar drafts em lote e testar reescrita automática de falhas, sem exposição pública.

## 2026-06-09 — Batch drafts com score e reescrita bloqueada

Decisao: implementar `batch_drafts` como camada de rascunhos de amostra por lote massivo, com texto editorial próprio, score humano calculado, prova de reescrita automática em falhas iniciais, baixa similaridade e publicação bloqueada.

Motivos:
- manifestos de milhão de páginas precisam virar amostras editoriais verificáveis antes de qualquer página pública;
- validar em massa sem amostra textual ainda não prova naturalidade, especificidade ou CTA contextual;
- reescrita automática precisa deixar evidência de falha inicial e correção;
- a escala deve evoluir por algoritmo, não por revisão manual página a página;
- Googlebot não deve ver rascunho enquanto score, fonte, revisão, SEO e publicação não estiverem completos.

Consequencia: `data/editorial/batch_drafts.jsonl` registra 18 rascunhos, três por família jurídica de lote, todos `batch_draft_scored_blocked`. `internal/batchdrafts` valida score via `internal/humanscore`, similaridade máxima, reescritas, origem de lote e bloqueio de render/sitemap/publicação. `./tools/check-batch-drafts` entra no laboratório. O próximo ciclo deve transformar amostras em geração programática ampliada e medição agregada por lote.

## 2026-06-09 — Gerador/refinador de batch drafts com métricas agregadas

Decisao: implementar `batchdraftgen` como gerador/refinador determinístico de rascunhos de lote, produzindo amostras temporárias e persistindo métricas agregadas bloqueadas.

Motivos:
- a fábrica massiva precisa gerar e validar lote por comando, não depender de amostras escritas uma a uma;
- reescrita automática deve ser comprovada por falha inicial, score final e métrica agregada;
- métricas por família permitem crescer volume sem mascarar similaridade, IA-like ou baixa especificidade;
- o laboratório pode usar CPU para gerar/refinar, mas nada deve escapar para HTML, sitemap, `public_path` ou publicação;
- CTA WhatsApp precisa nascer com origem de `unique_intent_id` para triagem digital.

Consequencia: `internal/batchdraftgen`, `cmd/generate-batch-drafts`, `./tools/generate-batch-drafts` e `./tools/check-batch-draft-generation` entram no laboratório. `data/editorial/batch_generation_metrics.jsonl` registra 6 métricas de geração bloqueada; o comando gera 30 rascunhos temporários, cinco por família, todos reescritos e com similaridade máxima 0.27 no laboratório. O próximo ciclo deve aumentar o volume por família, cruzar fonte específica por subtema e preparar gate de pré-publicação bloqueada por lote sem publicar.

## 2026-06-09 — Escala semântica com matriz de fontes por subtema

Decisao: ampliar o gerador para 10 amostras por família e criar `batch_source_matrix` como matriz leve de fontes oficiais por subtema, mantendo uso apenas referencial e sem scraping.

Motivos:
- escala maior revelou que similaridade pode subir quando o algoritmo repete vocabulário operacional de laboratório;
- o projeto exige boa semântica por tema/subtema, não mecanização de palavras;
- cada draft de lote precisa carregar fonte oficial específica, documento, risco, ação digital e CTA de origem;
- métricas de escala precisam registrar cobertura de fonte, risco estrutural e custo estimado de laboratório;
- publicar sem matriz de fonte específica criaria risco jurídico e risco de conteúdo raso.

Consequencia: `internal/batchsourcematrix`, `data/editorial/batch_source_matrix.jsonl` e `./tools/check-batch-source-matrix` entram no laboratório. `batchdraftgen` passa a gerar 60 drafts temporários com `source_matrix_id`, cobertura de matriz, risco estrutural e estimativa de CPU de laboratório; a similaridade máxima validada caiu para 0.52 após refinamento semântico. O próximo ciclo deve ampliar a matriz e o gerador para centenas de amostras por família, mantendo fonte específica e publicação bloqueada.

## 2026-06-09 — Auditoria URL-level e centenas de rascunhos por família

Decisao: criar `batch_source_url_audits` e ampliar o gerador para validar 100 amostras por família no laboratório, mantendo publicação bloqueada e sem afrouxar score, similaridade ou fonte.

Motivos:
- matriz de fonte por subtema ainda nao prova auditoria de cada URL oficial usada pelo lote;
- centenas de amostras por família exigem algoritmo semântico, não repetição de três sufixos;
- similaridade precisa diferenciar faceta e subtema, sem confundir metadado bruto com texto editorial;
- testes podem usar CPU no laboratório, mas o runtime público continua leve;
- nenhum rascunho de lote pode virar render, sitemap, `public_path` ou página indexável no P0.

Consequencia: `internal/batchsourceaudit`, `data/source-audit/batch_source_urls.jsonl` e `./tools/check-batch-source-url-audits` entram no laboratório. `batchdraftgen` passa a gerar 600 drafts temporários em teste de contrato, com 100 por família, facetas semânticas distribuídas por subtema, contexto de área, auditoria URL-level e similaridade máxima abaixo do limite de 0.64. `batchdrafts.MaximumPairSimilarity` foi otimizado para pré-computar sinais semânticos e pondera subtema/faceta sem afrouxar o limite. O próximo ciclo deve transformar essa massa temporária em gate de lote candidato, ainda bloqueado, com amostra persistida controlada e pré-publicação sem URL pública.

## 2026-06-09 — Laboratorio aprovado vira arquivo permanente bloqueado

Decisao: rascunhos massivos gerados em `/tmp` que passam nos gates e contêm informação jurídica útil devem ser preservados no repositório como `batch_draft_expansion_archive`, não descartados.

Motivos:
- checkpoint nao pode depender de diretorio temporario ou contexto compactado;
- rascunhos validados podem virar base permanente de páginas futuras depois de expansão, fonte e revisão;
- excluir dados jurídicos úteis sem prova atrasa a fabrica de conteúdo e reduz rastreabilidade;
- persistir no repo nao significa publicar, renderizar, criar sitemap ou liberar CTA público.

Consequencia: `data/editorial/batch_draft_expansion_archive.jsonl`, `internal/batchdraftarchive` e `./tools/check-batch-draft-expansion-archive` entram no laboratório. O arquivo exige 600 rascunhos, 100 por família, `source_matrix_id`, reescrita automática, baixa similaridade e bloqueio total de render/sitemap/publicação. Remoção ou rebaixamento de rascunho validado exige prova em checkpoint e gate próprio.

## 2026-06-09 — URL oficial do projeto ainda nao esta travada

Decisao: tratar `content/site.json` como fonte configuravel da base de canonical/sitemap/robots e marcar a base atual como placeholder de laboratorio, nao URL oficial do produto.

Motivos:
- o projeto ainda esta em P0 e a URL oficial publica nao foi definida;
- testes rigidos por dominio quebrariam a migracao futura sem melhorar SEO;
- canonical continua obrigatorio, mas deve ser validado por base configurada, HTTPS e path limpo;
- pre-publicacao bloqueada pode planejar canonical candidato sem criar URL publica.

Consequencia: `content/site.json` passa a declarar `base_url_mode`, `official_url_status` e `official_url_locked`. `internal/prepublication` valida canonical candidato contra a base carregada de `content/site.json`, e o teste `TestPrepublicationGateAcceptsConfigurableProjectBaseURL` prova que outro dominio HTTPS pode ser aceito sem alterar algoritmo. `portal-juridico.example` so pode ser tratado como placeholder enquanto `base_url_mode="lab_placeholder"`.

## 2026-06-09 — Gate candidato bloqueado a partir do arquivo permanente

Decisao: criar `batch_candidate_gates` como etapa intermediaria entre arquivo permanente de rascunhos e pre-publicacao, selecionando candidatos reais sem publicar.

Motivos:
- o arquivo de 600 rascunhos precisa virar continuidade operacional, nao ficar apenas como massa bruta;
- seleção de candidato nao pode ser confundida com URL publica, sitemap ou CTA visivel;
- a URL oficial ainda nao esta travada, entao o gate precisa respeitar `base_url_mode`;
- cada candidato deve existir no arquivo permanente, carregar CTA contextual e continuar vinculado a fonte matricial.

Consequencia: `data/editorial/batch_candidate_gates.jsonl`, `internal/batchcandidategates` e `./tools/check-batch-candidate-gates` entram no laboratorio. O gate exige 6 famílias, 3 intenções selecionadas por família, 100 registros mínimos no arquivo por lote, similaridade <=0.64, score humano mínimo, base URL flexível e flags públicas falsas. O próximo ciclo deve transformar candidatos selecionados em revisão jurídico-editorial por candidato, ainda sem render público.

## 2026-06-09 — URL oficial travada em wikijuridica.com.br

Decisao: travar a URL oficial do projeto como `https://wikijuridica.com.br` e remover o placeholder `portal-juridico.example` dos canonicals e gates vivos.

Motivos:
- o usuario definiu `wikijuridica.com.br` como dominio oficial do projeto;
- canonical, sitemap e robots precisam de base real antes de evoluir pre-publicacao;
- testes continuam lendo `content/site.json` para evitar algoritmo hardcoded, mas o contrato agora exige `official_configured`;
- URL oficial travada nao e autorizacao para publicar candidatos de lote ou rascunhos.

Consequencia: `content/site.json` passa para `base_url_mode="official_configured"`, `official_url_status="locked"` e `official_url_locked=true`. `content/pages.json`, `data/editorial/prepublication_gates.jsonl` e `data/editorial/batch_candidate_gates.jsonl` acompanham a base oficial. O proximo ciclo deve manter candidatos bloqueados e preparar pre-publicacao em lote apenas depois de fonte e revisao especificas.

## 2026-06-09 — Revisao juridico-editorial bloqueada de candidatos de lote

Decisao: criar `batch_candidate_reviews` como camada obrigatoria entre `batch_candidate_gates` e qualquer pre-publicacao em lote.

Motivos:
- selecionar candidato nao basta para preparar pagina juridica publica;
- cada candidato precisa de revisao juridico-editorial, CTA WhatsApp contextual e fonte matricial auditada;
- escala massiva nao pode depender de revisao manual pagina a pagina, mas a revisao algoritmica precisa deixar rastro por candidato;
- URL oficial travada nao remove os gates de fonte, qualidade, SEO e publicacao;
- promessa de resultado, CTA raso, path com dominio e fonte nao auditada precisam reprovar antes de qualquer render.

Consequencia: `data/editorial/batch_candidate_reviews.jsonl`, `internal/batchcandidatereviews`, `./tools/check-batch-candidate-reviews`, `internal/checks` e `tools/lab-cycle` entram no laboratorio. O gate exige 18 revisoes bloqueadas, uma por intencao selecionada, com `Origem`, `Gate` e `Intent` na mensagem de WhatsApp, matriz auditada, notas especificas e flags publicas falsas. O proximo ciclo deve transformar essas revisoes em pre-publication gates de lote com canonical oficial e `noindex`, ainda sem sitemap/publicacao.

## 2026-06-09 — Validacao global proporcional ao risco

Decisao: tratar `go test -count=1 ./...`, `./tools/check-all`, `./tools/lab-cycle` e equivalentes completos como validacao global de alto custo, nao como ritual automatico para toda alteracao pequena.

Motivos:
- validacao global consome tempo e pode atrasar ciclos localizados;
- engenharia agressiva exige prova suficiente, nao excesso de ritual;
- mudancas pequenas podem ser comprovadas com teste focado, check especifico, diff check e inspecao direta;
- mudancas amplas ou criticas ainda exigem prova global para evitar regressao em varias camadas.

Consequencia: o proximo ciclo deve escolher validacao proporcional. Rodar validacao global quando houver alteracao ampla, contrato central, risco P0/P1 critico, HTML/sitemap/canonical/robots/indexacao/performance, gerador em massa, preparacao de publicacao ou falha transversal. Em ciclos localizados, registrar no checkpoint os checks focados usados e por que eles cobrem o risco.

## 2026-06-09 — Pre-publicacao bloqueada para candidatos revisados

Decisao: criar `batch_prepublication_gates` como etapa posterior a `batch_candidate_reviews`, registrando canonical oficial, `noindex,follow`, title/meta e pendencias finais sem publicar.

Motivos:
- revisao juridico-editorial de candidato ainda nao equivale a pagina publica;
- a URL oficial ja esta travada e deve aparecer no canonical candidato;
- Googlebot nao deve receber candidatos enquanto fonte final, revisao SEO, manifesto publico e render/sitemap nao forem aprovados;
- pre-publicacao em lote precisa ser validada por candidato, nao por suposicao global.

Consequencia: `data/editorial/batch_prepublication_gates.jsonl`, `internal/batchprepublication`, `./tools/check-batch-prepublication-gates`, `internal/checks` e `tools/lab-cycle` entram no laboratorio. O ciclo usa validacao focada por politica proporcional: teste do contrato novo, tool especifica, storage contract, checks internos e diff check; validacao global fica reservada para alteracao ampla ou risco critico.

## 2026-06-09 — Especificidade de fonte por candidato pre-publicado

Decisao: criar `batch_source_specificity_resolutions` como camada obrigatoria depois de `batch_prepublication_gates`, cobrindo cada candidato com fonte final travada como referencia ou bloqueio explicito por fonte ampla.

Motivos:
- matriz de fonte auditada nao basta para dizer que todo candidato esta pronto;
- fonte institucional ampla nao pode ser mascarada como fonte final;
- candidatos com URL oficial especifica podem avancar para o proximo gate bloqueado sem scraping, ingestao ou publicacao;
- candidatos com fonte ampla precisam registrar motivo, detalhe necessario e permanecer fora de render/sitemap/publicacao.

Consequencia: `data/editorial/batch_source_specificity_resolutions.jsonl`, `internal/batchsourcespecificity`, `./tools/check-batch-source-specificity`, `internal/checks`, `content/storage_contract.json` e `tools/lab-cycle` entram no laboratorio. O gate exige 18 resolucoes, uma por candidato pre-publicado, fonte URL-level auditada, politica `reference_only_no_scraping_no_ingestion`, `candidate_robots=noindex,follow`, canonical oficial e flags publicas falsas. O ciclo usa validacao proporcional focada; `check-all` e `lab-cycle` ficam reservados para alteracao ampla ou risco transversal.

## 2026-06-09 — Manifesto publico bloqueado por candidato de lote

Decisao: criar `batch_public_manifest_gates` como camada bloqueada depois de `batch_source_specificity_resolutions`, cobrindo todos os candidatos e permitindo avanço interno para SEO/conteudo final apenas quando a fonte esta travada.

Motivos:
- fonte travada ainda nao autoriza publicacao, render ou sitemap;
- candidatos com fonte ampla nao podem entrar em revisao SEO como se estivessem prontos;
- o pipeline precisa separar backlog de fonte de backlog de SEO/conteudo;
- manifesto publico real deve ser posterior e mais restrito que este gate bloqueado.

Consequencia: `data/editorial/batch_public_manifest_gates.jsonl`, `internal/batchpublicmanifest`, `./tools/check-batch-public-manifest-gates`, `internal/checks`, `content/storage_contract.json` e `tools/lab-cycle` entram no laboratorio. O gate exige 18 registros, 7 com `public_manifest_blocked_seo_review_pending` e 11 com `public_manifest_blocked_source_specificity`, mantendo `index_policy=noindex`, `manifest_allowed=false`, `render_allowed=false`, `sitemap_allowed=false`, `publication_allowed=false` e `public_path=""`.

## 2026-06-09 — Rascunho autoral final bloqueado e intenção paga

Decisao: criar `batch_final_authorial_drafts` apenas para os candidatos com fonte travada e manifesto SEO pendente, e adicionar `paid-intent` como gate de negócio para impedir funil de gratuidade, curiosidade ou baixa intenção de contratação.

Motivos:
- rascunho final não é publicação, mas precisa virar artefato permanente para escalar conteúdo;
- CTA WhatsApp deve carregar origem, intenção, documentos e sinal de contratação particular;
- serviço jurídico comercial precisa priorizar busca com honorários/orçamento, valor envolvido, urgência e documentos concretos;
- termos de gratuidade, defensoria, justiça gratuita, estudo acadêmico, modelo pronto ou curiosidade não devem alimentar o lote comercial;
- inferência aceitável é textual e jurídico-econômica do termo/caso, não perfil pessoal sensível.

Consequencia: `data/editorial/batch_final_authorial_drafts.jsonl`, `internal/batchfinaldrafts`, `internal/paidintent`, `./tools/check-batch-final-authorial-drafts`, `./tools/check-paid-intent`, `internal/checks`, `content/storage_contract.json` e `tools/lab-cycle` entram no laboratorio. Todos os 7 rascunhos seguem `noindex`, sem render, sem sitemap, sem publicação e sem `public_path`; o gate pago deve ser refinado quando surgir falso positivo/negativo antes de escalar.

## 2026-06-09 — Agentes auxiliares planejados sem concorrência crítica

Decisao: no próximo ciclo e nos seguintes, usar agentes auxiliares é obrigatório quando houver duas ou mais frentes independentes, especialmente pesquisa de fontes oficiais, matriz de proveniência, testes, conteúdo bloqueado e debugging, mas sem concorrência no estado do repositório.

Motivos:
- a meta massiva exige acelerar pesquisa e produção sem perder validação;
- fontes oficiais e conteúdo por subtema podem ser divididos por área/fonte;
- concorrência no mesmo arquivo, gate, commit, fonte jurídica ou decisão crítica aumenta risco de conflito e mascaramento;
- o Codex principal deve manter responsabilidade por arquitetura, P0/P1, integração, validação, checkpoint e commit.

Consequencia: agentes devem produzir pesquisa, evidência, teste, edição disjunta ou rascunho em escopo isolado. Se houver escrita, ela precisa ser disjunta e só entra no repo após validação do Codex principal. O Codex principal valida evidência, revisa o diff, roda os checks relevantes, registra checkpoint e não publica nada sem gate. Para não perder contexto em compactação, todo agente usado em ciclo deve virar registro em `.agents/agent_context_ledger.jsonl` antes do checkpoint/commit.

## 2026-06-09 — Ledger persistente contra perda de contexto de agentes

Decisao: criar `.agents/agent_context_ledger.jsonl` e `./tools/check-agent-context-ledger` como contrato operacional para preservar contexto de subagentes entre compactações.

Motivos:
- compactação pode remover IDs, achados, riscos e decisões de integração dos agentes;
- pesquisa de fonte oficial e revisão de conteúdo precisam sobreviver ao próximo ciclo;
- agente auxiliar não pode virar prova invisível nem autorização implícita;
- o Codex principal precisa conseguir retomar sem refazer pesquisa ou confiar em memória solta.

Consequencia: cada agente usado deve registrar ciclo, ID real, apelido, tipo de tarefa, escopo, status, política de uso, resumo, evidências, riscos, decisão de integração e flags `repo_write_allowed`, `codex_validation_required=true`, `closed_before_checkpoint=true`. A política padrão é `reference_only_no_repo_write`; escrita por agente só é válida com `delegated_repo_write_codex_validated`, escopo disjunto, evidência do diff, riscos, fechamento antes do checkpoint e validação/integração pelo Codex principal. `internal/agentcontext` valida o ledger; `check-all` e `lab-cycle` passam a incluir esse gate.

Adendo do ciclo 46: em ciclo de escala com duas ou mais frentes independentes, usar agentes auxiliares em paralelo deixa de ser opcional. O Codex principal deve acionar o máximo permitido e pertinente pela OpenAI para pesquisa, teste, auditoria, documentação, conteúdo bloqueado ou edição disjunta, mantendo a chefia técnica nos críticos. Todo agente precisa deixar contexto durável em `.agents/agent_context_ledger.jsonl`; compactação, memória do chat ou checkpoint genérico não bastam.

## 2026-06-09 — Timing e otimização obrigatórios antes de commit

Decisao: teste lento não pode ser tratado como normal sem medição. Criar `cmd/profile-tests`, `internal/testprofile`, `./tools/profile-contract-tests` e fazer `tools/lab-cycle` medir cada etapa com `TIMING`, removendo duplicação de `check-all` e checks individuais já cobertos por `cmd/check all`.

Motivos:
- `internal/contract` chegou a 207,365s em medição por `go test -json`;
- gargalos reais eram revalidação editorial upstream e similaridade de 600 drafts, não falta de vontade de usar CPU;
- `lab-cycle` repetia `go test`, `check-all` e vários checks individuais;
- otimização precisa preservar contrato, não mascarar gate.

Consequencia: antes de commit com teste/gate lento, rodar perfil ou registrar timing. O ciclo otimizado usa `go test -count=1 ./...`, `go run ./cmd/check all` e ferramentas de laboratório não cobertas pelo check-all. `batchdrafts` evita alocação de mapa de união em Jaccard e validadores finais deixam de reconstruir índice upstream duas vezes no mesmo caminho. Medição pós-otimização de `./tools/profile-contract-tests`: `total_observed_seconds=17.02`, `slow_tests=0` com threshold de 5s.

## 2026-06-09 — Gate pago persistente e fonte geral como fonte ampla

Decisao: intenção comercial paga agora fica registrada em `batch_paid_intent_gates`, e fontes legais gerais como Código Civil, CDC e CLT compilada não destravam recortes específicos sozinhas.

Motivos:
- CTA com honorários não basta quando o tema dominante indica assistência pública, gratuidade provável ou autoatendimento administrativo;
- BPC/LOAS, CadÚnico, renda familiar, baixa renda e cumprimento de exigência precisam de bloqueio comercial explícito antes de escalar conteúdo; vulnerabilidade isolada nao basta para inferir assistencia publica;
- fonte ampla pode ser referência oficial, mas não substitui ato, súmula, regra, serviço ou orientação específica para o subtema;
- agentes auxiliares podem acelerar pesquisa oficial e rascunho bloqueado, mas o Codex principal precisa validar evidência, integração e gates críticos.

Consequencia: `./tools/check-paid-intent` valida o arquivo permanente `data/editorial/batch_paid_intent_gates.jsonl`; candidatos comerciais fortes continuam bloqueados para publicação, e candidatos com risco de assistência pública ou self-service ficam roteados para bloqueio comercial. `./tools/check-batch-source-specificity` trata `codigo_civil`, `codigo_consumidor` e `clt_compilada` como fontes amplas quando o recorte precisa de fonte específica. O próximo ciclo deve criar prontidão de expansão de candidatos, usando agentes sem concorrência crítica para pesquisar fontes oficiais e ampliar conteúdo bloqueado com validação pelo Codex principal.

## 2026-06-09 — Prontidao de expansao bloqueada por paid gate

Decisao: criar `batch_candidate_expansion_readiness` como camada entre o arquivo permanente de 600 rascunhos e a seleção candidata ampliada, com alvo inicial de 30 candidatos por família e bloqueio explícito quando faltar paid-intent por intenção.

Motivos:
- o projeto precisa sair de 18 candidatos rumo a dezenas/centenas por família sem publicar spam;
- `batch_draft_expansion_archive` já prova massa útil, mas não prova que cada intenção expandida tem paid gate, fonte específica e CTA aptos;
- CTA colado não pode salvar candidato fraco, e paid-intent ausente deve ser blocker acionável;
- readiness precisa sobreviver ao checkpoint e orientar agentes auxiliares sem concorrência no mesmo gate.

Consequencia: `data/editorial/batch_candidate_expansion_readiness.jsonl`, `internal/batchcandidateexpansion` e `./tools/check-batch-candidate-expansion-readiness` entram no laboratório. O gate registra 6 famílias com 30 alvos cada, mas mantém status `batch_candidate_expansion_blocked_paid_gate_missing` enquanto as intenções expandidas não tiverem `batch_paid_intent_gates`. Nenhum registro permite manifesto, render, sitemap, publicação ou `public_path`.

## 2026-06-09 — Paid gate em lote e bloqueio CTA-only

Decisao: gerar `batch_paid_intent_gates` para os 180 alvos de `batch_candidate_expansion_readiness`, adicionar `gate_scope` para separar rascunho final de prontidao de expansao, e bloquear candidato cujo sinal de contratacao paga aparece apenas no CTA/WhatsApp.

Motivos:
- paid-intent ausente e paid-intent existente mas reprovado sao diagnosticos diferentes;
- CTA contextual e importante, mas CTA colado nao pode salvar corpo informativo sem intencao de contratacao natural;
- o algoritmo precisa explicar se reprovou por CTA-only, sinal pago ausente, baixa pontuacao de negocio, gratuidade, pesquisa sem contratacao, assistencia publica dominante ou autoatendimento;
- `vulnerabilidade` isolada e termo juridico amplo e nao deve inferir assistencia publica sem BPC/LOAS, CadUnico, renda familiar, baixa renda, defensoria ou justica gratuita.

Consequencia: `data/editorial/batch_paid_intent_gates.jsonl` agora tem 180 registros bloqueados, `internal/paidintent` separa `paid_signals` do corpo e `cta_paid_signals` do CTA, `cmd/generate-paid-intent-gates` materializa o banco leve e `cmd/refresh-expansion-readiness` recalcula os contadores de prontidao. `batch_candidate_expansion_readiness` passa a usar `batch_candidate_expansion_blocked_paid_gate_failed` quando nao falta gate, mas ainda ha bloqueio comercial. O proximo ciclo deve reescrever/refinar em lote os candidatos bloqueados por CTA-only ou sinal pago ausente, sem liberar render, sitemap, publicacao ou `public_path`.

## 2026-06-09 — Refinamento pago em lote sem publicar

Decisao: criar `batch_paid_intent_refinements` e `internal/paidintentrefinement` para reescrever em lote apenas candidatos com paid-intent ausente ou CTA-only, movendo sinal de contratacao paga para o corpo informativo quando natural e mantendo o ledger bloqueado.

Motivos:
- CTA WhatsApp contextual e critico, mas sinal de honorarios somente no CTA nao basta para escalar conteudo;
- paid-intent ausente/CTA-only pode ser corrigido por algoritmo quando o tema permite contratacao particular online;
- BPC/assistencia publica dominante e autoatendimento administrativo continuam bloqueios comerciais, nao alvos de refinamento;
- script de laboratorio precisa ser idempotente para nao falhar quando o ciclo ja foi aplicado;
- readiness sem blocker antigo de paid/fonte precisa apontar o proximo gate real, nao ficar com blocker vazio.

Consequencia: `./tools/refine-paid-intent-drafts` refinou 140 registros na primeira aplicacao (135 no arquivo permanente de expansao e 5 rascunhos finais), `./tools/check-paid-intent-refinements` entrou no check-all, `batch_paid_intent_gates` passou a registrar 168 candidatos pagos bloqueados para publicacao, 6 bloqueios de assistencia publica e 6 bloqueios de autoatendimento. `batch_candidate_expansion_readiness` usa `batch_candidate_gate_pending` quando a familia esta livre de paid/source blockers antigos, mas continua `noindex`, sem manifesto, render, sitemap, publicacao ou `public_path`. O proximo ciclo executavel e ampliar `batch_candidate_gates` a partir dos candidatos pagos aprovados internamente, mantendo bloqueio publico e fonte especifica como gates.

## 2026-06-09 — Expansao candidata de 168 e refresh idempotente da cadeia bloqueada

Decisao: `batch_candidate_gates` deve ser expandido por algoritmo a partir de `batch_candidate_expansion_readiness` e `batch_paid_intent_gates`, e a cadeia downstream deve ser regenerada por `refresh-batch-candidate-pipeline`, nunca por edicao manual de JSONL em massa.

Motivos:
- ampliar de 18 para 168 candidatos sem propagar revisao, pre-publicacao, fonte especifica e manifesto quebra o contrato P0;
- paid-intent aprovado internamente nao equivale a publicacao, pois fonte especifica e revisao ainda podem bloquear;
- rascunhos finais bons devem ser preservados quando continuam elegiveis, mas BPC/assistencia publica e autoatendimento devem sair da cadeia candidata paga;
- a meta de 10k/milhoes exige ferramenta idempotente, nao ajuste manual lento.

Consequencia: `./tools/expand-batch-candidate-gates` seleciona 168 intenções pagas bloqueadas; `./tools/refresh-batch-candidate-pipeline` materializa 168 revisoes, 168 prepublication gates, 168 resolucoes de fonte e 168 manifestos, preservando 5 rascunhos finais elegiveis e bloqueando 163 por fonte especifica. Todos permanecem `noindex`, sem render, sitemap, publicacao ou `public_path`.

## 2026-06-09 — Expansao 318 com current separado do next target

Decisao: `batch_candidate_expansion_readiness` pode carregar o próximo alvo planejado pela estratégia, mas `batch_candidate_gates` só deve materializar o `current_candidate_count` da estratégia no ciclo atual.

Motivos:
- readiness com alvo 90 por família precisa existir para o próximo ciclo e para gerar paid gates antecipados;
- selecionar todos os paid-passed da readiness no mesmo ciclo saltaria de 318 para 468 candidatos sem checkpoint próprio;
- `ApplyStrategy` não pode trocar IDs sem recomputar paid counts, blockers, status e current count;
- expansão agressiva precisa ser rápida, mas cada salto de escala deve ser rastreável, validado e bloqueado para publicação.

Consequencia: `internal/batchexpansionapply` passou a recalcular cada readiness record após aplicar a estratégia; `internal/batchcandidatepromotion` limita a seleção ao `strategy.CurrentCandidateCount`; `batch_paid_intent_gates` cobre 480 registros incluindo próximos alvos; `batch_candidate_gates` permanece com 318 candidatos materializados; reviews, prepublication, source-specificity, public manifest e final drafts permanecem em 318, todos `noindex`, sem render, sitemap, publicação ou `public_path`. O próximo ciclo deve promover o alvo 90 das cinco famílias prontas para nova materialização validada, mantendo previdenciário bloqueado por paid-intent.

## 2026-06-09 — Lane previdenciaria informativa bloqueada

Decisao: previdenciario pode crescer por uma lane informativa/curiosa bloqueada, sem exigir alta intencao de pagamento, desde que a excecao fique restrita a `batch-previdenciario-digital` e nunca publique, renderize, entre em sitemap ou crie `public_path`.

Motivos:
- temas previdenciarios como BPC/LOAS, CadUnico, exigencia do INSS e autoatendimento podem ter utilidade humana e demanda real mesmo quando nao mostram alta intencao paga;
- tratar essa demanda como lixo comercial reduziria crescimento de familia juridica relevante;
- afrouxar a regra global criaria risco de spam, funil de gratuidade e selecao fraca nas familias comerciais;
- a excecao precisa ser status proprio, auditavel e bloqueada, nao mascaramento de paid-intent aprovado.

Consequencia: `internal/paidintent` cria `paid_intent_flexible_previdenciario_informational_blocked_publication` e centraliza elegibilidade em `paidintent.AllowsExpansion`, que aceita o status apenas em `batch-previdenciario-digital` e com flags publicas falsas. `internal/batchexpansionapply`, `internal/batchcandidateexpansion`, `internal/batchcandidatepromotion` e `internal/batchcandidatepipeline` usam esse helper. `batch_candidate_gates` sobe para 330 candidatos internos bloqueados; `batch_paid_intent_gates` cobre 510 alvos de laboratorio, com 42 previdenciarios informativos bloqueados; `batch_expansion_strategy` planeja proximo crescimento para 510 current total, sendo 90 nas cinco familias comerciais e 60 em previdenciario. Gratuidade explicita, defensoria/justica gratuita, "sem pagar", promessa de beneficio, promessa de resultado ou substituicao de canal publico continuam bloqueios P0.

## 2026-06-09 — Avanco explicito para 510 candidatos bloqueados

Decisao: criar `./tools/advance-batch-candidate-gates` para promover explicitamente o `next_candidate_target` planejado por `batch_expansion_strategy`, sem mudar o comportamento conservador de `./tools/expand-batch-candidate-gates`.

Motivos:
- o contrato atual separa current de next target para impedir salto implicito no mesmo checkpoint;
- o ciclo 47 precisava materializar 510 candidatos, mas isso deveria ser uma acao explicita, rastreavel e validada;
- reaproveitar `expand-batch-candidate-gates` como salto automatico quebraria a decisao anterior e poderia mascarar crescimento sem checkpoint;
- a meta massiva exige comando rapido de avancar lote, com teste e regeneracao downstream, nao edicao manual de JSONL.

Consequencia: `internal/batchcandidatepromotion` passa a ter `AdvanceToNextTargets`, mantendo `ExpandFromReadiness` como selecao current; `cmd/advance-batch-candidate-gates` e `tools/advance-batch-candidate-gates` materializam o next target quando a estrategia esta pronta. `batch_candidate_gates` sobe para 510 candidatos internos bloqueados: 90 em cada uma das cinco familias comerciais e 60 em previdenciario. A cadeia downstream sobe para 510 revisoes, prepublication gates, source-specificity, manifests e final drafts, todos `noindex`, sem render, sitemap, publicacao ou `public_path`. `batch_paid_intent_gates` passa a cobrir 590 alvos de laboratorio para o proximo crescimento planejado.

## 2026-06-09 — Flexibilidade previdenciaria por curiosidade qualificada

Decisao: previdenciario informativo nao precisa de alta intencao de pagamento para crescer internamente no laboratorio, desde que haja curiosidade qualificada, utilidade humana e bloqueio publico total.

Motivos:
- previdenciario tem demanda humana real em beneficios, CNIS, pericia, exigencia, prazo, documento e revisao, mesmo quando a pessoa ainda esta curiosa ou em fase administrativa;
- bloquear toda curiosidade previdenciaria reduziria crescimento de familia juridica relevante e impediria cobertura informativa util;
- afrouxar a regra geral criaria risco comercial em consumidor, familia, saude, sucessorio e trabalhista;
- a excecao precisa continuar auditavel, com status proprio e sem publicacao.

Consequencia: `batch-previdenciario-digital` pode usar `paid_intent_flexible_previdenciario_informational_blocked_publication` para crescer familias previdenciarias bloqueadas. Famílias comerciais continuam exigindo intencao paga particular ou bloqueio explicito. Gratuidade explicita, defensoria/justica gratuita, "sem pagar", promessa de beneficio, promessa de resultado e substituicao de canal publico continuam bloqueios P0. A lane previdenciaria permanece `noindex`, sem render, sitemap, publicacao ou `public_path`.

Adendo: famílias comerciais também são informativas. Não retirar o conteúdo informativo da plataforma jurídica; a intenção natural de contratação deve ser incorporada por cenário jurídico, documentos, risco econômico, urgência, honorários/orçamento e CTA contextual, sem transformar a página em oferta seca.

## 2026-06-09 — OAB, autoria configurada e atendimento jurídico digital

Decisao: o projeto deve tratar atendimento jurídico 100% digital, tudo online, sem sair de casa, envio remoto de documentos, WhatsApp contextual e atendimento a brasileiros fora do Brasil como modo real de prestação jurídica digital; isso não é promessa de resultado, é realidade brasileira quando usado como descrição objetiva do atendimento.

Pesquisa oficial feita no dia da sessao:
- `https://www.oab.org.br/leisnormas/legislacao/provimentos/205-2021`
- `https://www.oab.org.br/publicacoes/AbrirPDF?LivroId=0000004085`

Motivos:
- o Provimento OAB 205/2021 permite marketing jurídico compatível com a ética da OAB e define marketing de conteúdos jurídicos como criação/divulgação de conteúdo jurídico voltado a informar o público;
- o mesmo Provimento exige informação objetiva, verdadeira, sobriedade e veda captação, mercantilização, valores, gratuidade/descontos, expressões persuasivas, autoengrandecimento, promessa de resultados e casos concretos como oferta;
- o anexo do Provimento admite criação de conteúdo, artigos, ferramentas tecnológicas e chatbot para facilitar comunicação/coleta de dados sem suprimir a pessoalidade do advogado;
- o Código de Ética e Disciplina exige publicidade informativa, internet como veículo lícito com limites, e nome/OAB na publicidade profissional.

Consequencia: gates não devem reprovar termos como 100% digital, tudo online, sem sair de casa ou atendimento remoto quando forem descrição do fluxo. Devem reprovar garantia de êxito, liminar garantida, resultado certo, prazo prometido, comparação, captação indevida, caso concreto usado como oferta, sensacionalismo, valores, descontos e gratuidade. Autor dos conteúdos: Rafael Toledo, OAB/RJ 227191; a identidade fica em `content/site.json` como configuração versionada, não hardcoded em runtime.

## 2026-06-09 — Avanco 590 com refino semantico e archive esgotado explicito

Decisao: o avanço interno de 510 para 590 candidatos bloqueados só pode ocorrer depois de teste semântico específico contra repetição previdenciária e refresh completo da cadeia downstream.

Motivos:
- `--expect-total` evita salto acidental, mas não prova qualidade semântica;
- a lane previdenciária informativa pode crescer por curiosidade qualificada, mas não pode manter abertura, documentos, triagem e CTA repetidos;
- final drafts antigos podem ser reutilizados pelo pipeline se o reuso não detectar padrão legado;
- cinco famílias comerciais chegaram a 100/100 registros do archive, então fingir próximo crescimento sem novo archive seria pendência mascarada.

Consequencia: `batch_final_authorial_drafts` passa a exigir variação semântica mínima em previdenciário e o pipeline reconstrói finais previdenciários legados usando problema do leitor, documentos, risco e ação digital do rascunho selecionado. `batch_candidate_gates` sobe para 590 candidatos internos bloqueados, com cadeia downstream em 590 e flags públicas falsas. `batch_expansion_strategy` ganha `batch_expansion_strategy_blocked_archive_growth_required` para famílias com current igual ao archive observado; o próximo ciclo deve gerar mais `batch_draft_expansion_archive` permanente, bloqueado, semântico e validado antes de novo avanço.

## 2026-06-09 — Archive 780, anti-mascaramento e estratégia 770

Decisao: expandir o arquivo permanente bloqueado para 780 rascunhos, 130 por família, usando perfis semânticos de expansão e similaridade profile-aware, e registrar anti-mascaramento como regra explícita do próximo agente.

Motivos:
- o ciclo anterior expôs cinco famílias comerciais no limite de 100/100 do archive; crescer sem novo archive seria pendência mascarada;
- rascunho temporário de laboratório que passa nos gates e contém informação jurídica útil deve vir para o repositório como dado permanente bloqueado;
- apenas trocar faceta ou palavra não basta para escala; cada rodada acima de 100 precisa de eixo semântico novo, documento, risco, fonte, problema do leitor e CTA contextual;
- paid-intent não pode depender só do CTA; o corpo informativo precisa carregar sinal natural de contratação particular quando a família é comercial;
- teste verde não substitui leitura técnica do contexto, do texto gerado, dos blockers, da fonte e da semântica jurídica.

Consequencia: `cmd/generate-batch-drafts` ganha `--write-archive`; `internal/batchdraftgen` valida antes de escrever archive permanente; `internal/batchdrafts` passa a entender perfis de expansão na similaridade; `internal/batchcandidateexpansion` atualiza contadores reais de archive e similaridade; `data/editorial/batch_draft_expansion_archive.jsonl` fica com 780 registros; `batch_paid_intent_gates` cobre 770 alvos; `batch_paid_intent_refinements` registra 529 refinamentos; readiness e strategy ficam prontos para o próximo candidate gate. O estado público continua bloqueado: sem render, sitemap, publicação, `public_path` ou `index`.

Regra operacional: quando houver bug de contrato, falso positivo, falso negativo, dado stale, métrica suspeita ou conteúdo que passe no script mas pareça raso, mecânico, sem contexto ou comercial demais, a correção correta é investigar, refinar algoritmo/teste/dado e revalidar. É proibido baixar limite, remover blocker, renomear status ou aceitar script isolado como verdade P0/P1. O próximo ciclo deve materializar o avanço explícito de 590 para 770 candidatos bloqueados, regenerar a cadeia downstream e validar sem publicar.

## 2026-06-09 — Dados reais antes de inferência no algoritmo de conteúdo

Decisao: nenhum ajuste de similaridade, n-grama, título, termo, paid-intent, CTA ou expansão de lote pode ser feito por inferência no escuro. O laboratório deve primeiro expor dados reais do erro e só então alterar gerador, comparador ou gate.

Motivos:
- o ciclo de archive 780 mostrou que similaridade alta podia vir de campos concretos repetidos, não de limite baixo;
- título/termo repetido em várias seções gera n-grama mecânico e não pode ser tratado como diversidade semântica;
- algoritmo de conteúdo precisa entender área jurídica e contexto, evitando confundir tema de família/pensão com trabalhista por token solto;
- afrouxar limite ou filtrar token sem diagnóstico mascara spam, falso positivo ou falso negativo.

Consequencia: testes e diagnósticos de lote devem mostrar par/registro que falhou, `legal_area`, `source_matrix_id`, `unique_intent_id`, campos textuais, blocker, score e código gerador antes de refinamento. Se faltar dado, o próximo passo é instrumentar diagnóstico ou teste. Se o teste estiver certo, corrigir geração/conteúdo; se for falso positivo, fortalecer o comparador sem reduzir limite e mantendo teste negativo.

## 2026-06-09 — Lab aprovado vira repo permanente bloqueado

Decisao: artefato aprovado no laboratório não pode ficar apenas em `/tmp` quando tem valor para páginas futuras. Antes do commit do ciclo, rascunho, métrica, fonte ou conteúdo de lote que passou com segurança deve ser trazido para camada versionada do repositório, sempre bloqueado para publicação.

Motivos:
- rascunhos aprovados contêm informação jurídica e contexto útil para expansão futura;
- depender de `/tmp`, chat ou compactação desperdiça prova e quebra continuidade;
- mover para repo não significa publicar, renderizar, indexar ou criar sitemap;
- descarte de dado útil só é válido com prova e gate específico.

Consequencia: o ciclo 50 materializou no repo `batch_draft_expansion_archive` com 780 registros, métricas em `batch_generation_metrics`, `batch_paid_intent_gates` com 770, `batch_paid_intent_refinements` com 529, readiness e strategy em 6 famílias. Tudo permanece em camadas editoriais bloqueadas, sem `content/pages.json`, sem `public_path`, sem render, sem sitemap, sem publicação e sem `index`.

## 2026-06-09 — Archive 1.140 com diversidade de matriz e avanço 1.140

Decisao: expandir o arquivo permanente bloqueado para 1.140 rascunhos, 190 por família, somente depois de novo gate de diversidade por `source_matrix_id`, reparo de especificidade documental e validação do gerador acima de 160 por família.

Motivos:
- o archive de 960 ainda repetia 4-gramas dentro das mesmas matrizes de fonte, mesmo com intenção única;
- reduzir cue sem diagnóstico enfraquecia especificidade e criava risco de texto raso;
- a correção correta era fazer o gerador usar sinais reais do caso matriz, documento, fonte, risco e ação digital, sem baixar limite de similaridade;
- artefato validado no laboratório não pode ficar apenas em `/tmp` quando contém dado jurídico útil para páginas futuras;
- paid-intent stale depois de refinamento é bug de ordem, não blocker a mascarar; a cadeia deve ser regenerada até ficar idempotente antes de qualquer advance.

Consequencia: `internal/batchdrafts.ValidateSourceMatrixDiversity` passa a reprovar dominância de 4-grama e baixa diversidade por campo dentro de cada `source_matrix_id`; `internal/batchdraftgen` gera cues discriminativos por campo e acrescenta reparo de especificidade por área apenas quando o score acusa `low_specificity`; `data/editorial/batch_draft_expansion_archive.jsonl` fica com 1.140 registros; `batch_generation_metrics` registra 190 por família; `batch_paid_intent_gates` cobre 1.140; `batch_paid_intent_refinements` registra 827 e fica idempotente após regeneração; `batch_candidate_gates` avança para 1.140 com cadeia downstream completa em 1.140. O orçamento leve de `batch_candidate_gates` sobe para 32KB porque 190 IDs por família passam de 16KB, mas isso não é solução de escala infinita: antes de crescimento muito maior, o gate deve ser particionado por shard/tier ou registro candidato. O estado público continua bloqueado: sem render, sitemap, publicação, `public_path` ou `index`. O próximo ciclo deve gerar novo archive semântico acima de 190 por família, revalidar diversidade intra-matriz, regenerar paid/readiness/strategy e só então avançar além de 1.140.

## 2026-06-09 — Archive 1.320, concorrência de agentes e avanço 1.320

Decisao: expandir o arquivo permanente bloqueado para 1.320 rascunhos, 220 por família, e avançar `batch_candidate_gates` para 1.320 somente depois de dry-run limpo, persistência explícita, paid gates em 1.320, refinamento idempotente, readiness/strategy verdes e cadeia downstream bloqueada em 1.320.

Motivos:
- dry-run 220 isolado comprovou 1.320 rascunhos com similaridade máxima 0.60 sem alterar o repo;
- artefato aprovado em laboratório deve ser persistido no repo, não ficar apenas em `/tmp`;
- refinamento de paid-intent alterou o archive e exigiu regenerar paid gates antes de aceitar readiness;
- um subagente executou `git restore` e `rm` no workspace compartilhado sem autorização do Codex principal, apagando uma tentativa local de expansão; isso é falha operacional P0 de concorrência e não pode se repetir;
- `batch_candidate_gates` com 220 IDs por família ainda cabe em 32KB, mas a maior linha chegou a 22.173 bytes e confirma necessidade de particionamento antes de escala muito maior.

Consequencia: `data/editorial/batch_draft_expansion_archive.jsonl`, `batch_paid_intent_gates`, `batch_candidate_reviews`, `batch_prepublication_gates`, `batch_source_specificity_resolutions`, `batch_public_manifest_gates` e `batch_final_authorial_drafts` ficam com 1.320 registros bloqueados; `batch_paid_intent_refinements` registra 969 refinamentos; `batch_expansion_strategy` volta para `batch_expansion_strategy_blocked_archive_growth_required` em todas as famílias porque current=archive=220. Contratos de agentes passam a proibir `git restore`, `git checkout`, `rm`, reset ou limpeza no workspace compartilhado sem autorização explícita do Codex principal. O próximo ciclo deve particionar `batch_candidate_gates` ou provar limite seguro antes de crescer muito acima de 220 por família.

## 2026-06-10 — Candidate gates shardados, idempotentes e bloqueados

Decisao: particionar `batch_candidate_gates` em shards físicos antes de novo crescimento, mantendo o contrato lógico por 6 famílias e 1.320 candidatos bloqueados agregados.

Motivos:
- a maior linha de `batch_candidate_gates` em 1.320 chegou a 22.173 bytes e não escalaria para 10k+ sem linha gigante;
- baixar regra ou aceitar 32KB como solução permanente mascararia gargalo de storage;
- normalização não idempotente poderia remover metadados de shard e duplicar `gate_id` em execução futura;
- `WriteRecords` precisava validar o conjunto normalizado antes de substituir o JSONL, para não truncar arquivo em falha tardia;
- reviews/prepublication carregam `gate_id` físico, então mudar shards exige regenerar downstream e validar paridade.

Consequencia: `internal/batchcandidategates` passa a normalizar por `gate_group_id`, `shard_index`, `shard_count` e `selected_total`, com limite `MaxSelectedIntentIDsPerRecord=120`, validação global de `gate_id`, seleção duplicada, índice faltante/duplicado e total divergente. `WriteRecords` valida em memória e escreve via arquivo temporário/rename. `batch_candidate_gates.jsonl` passa de 6 para 12 linhas físicas, mantendo 1.320 candidatos lógicos bloqueados, maior linha 12.978 bytes e `record_max_bytes=16.384`. `batch_candidate_reviews` e `batch_prepublication_gates` foram regenerados para os `gate_id` shardados. O estado público continua bloqueado: sem render, sitemap, publicação, `public_path` ou `index`. O próximo ciclo deve crescer o archive acima de 220 por família, regenerar paid/refinement/readiness/strategy e avançar candidatos em shards sem ultrapassar orçamento leve.

## 2026-06-10 — Expansao bloqueada para 1.500 candidatos e similaridade otimizada sem afrouxar gate

Decisao: crescer o arquivo permanente bloqueado para 250 rascunhos por família, total 1.500, e avançar candidate gates para 1.500 em shards, mantendo publicação bloqueada e reduzindo custo de similaridade sem mudar o score aceito.

Motivos:
- o contrato de crescimento atual limita avanço a +30 por família, então 250 por família era o próximo degrau correto depois de 220;
- rascunho aprovado em laboratório precisa ir para repo permanente quando tem valor futuro, portanto `batch_draft_expansion_archive` e métricas foram atualizados, não deixados em `/tmp`;
- `refine-paid-intent-drafts` altera o archive, então paid gates precisam ser recalculados depois do refinamento;
- readiness calculada antes do advance fica stale depois que candidate gates sobem de 220 para 250; por isso readiness e strategy devem ser rodadas novamente após `advance-batch-candidate-gates`;
- `MaximumPairSimilarityDetail` era o gargalo dominante em 1.500 registros; pular Jaccard semântico apenas quando `source_matrix_id` difere é equivalente porque `maximumSimilarityScore` já retorna somente `textScore` nesse caso.

Consequencia: `batch_draft_expansion_archive`, `batch_paid_intent_gates`, `batch_candidate_reviews`, `batch_prepublication_gates`, `batch_source_specificity_resolutions`, `batch_public_manifest_gates` e `batch_final_authorial_drafts` ficam com 1.500 registros bloqueados; `batch_candidate_gates` fica com 18 shards físicos; `batch_paid_intent_refinements` fica com 1.114 registros. `internal/batchdrafts` mantém threshold de similaridade, adiciona teste para a equivalência de matrizes diferentes, aumenta cache de fingerprints para 32 worksets e reduz o perfil de contratos para cerca de 29s. O estado público continua bloqueado: sem alteração em `content/pages.json`, sem diff público, sem `render_allowed=true`, `sitemap_allowed=true`, `publication_allowed=true` ou `public_path`. O próximo ciclo deve crescer para 280 por família, total 1.680, com a mesma ordem: dry-run, write archive/metrics, paid, refinement, paid, readiness, strategy, advance esperado, readiness/strategy pós-advance, pipeline downstream, checks e checkpoint.

## 2026-06-10 — Expansao bloqueada para 1.680 com diversidade por rodada

Decisao: crescer para 280 rascunhos por família, total 1.680, somente depois de corrigir o gerador para variar documento, risco e ação digital por facet e perfil de rodada.

Motivos:
- o dry-run inicial de 280 reprovou corretamente por `generation_batch_draft_matrix_document_diversity_low` e `generation_batch_draft_matrix_risk_diversity_low`, com várias matrizes em 0.38/0.39;
- baixar threshold de diversidade mascararia conteúdo mecânico, então a correção precisava melhorar o algoritmo;
- o limite real era a assinatura inicial dos campos, ainda dominada pelo facet, enquanto o perfil de rodada aparecia tarde demais;
- combinar pista do facet com pista da rodada no início de `DocumentContext`, `RiskContext` e `DigitalAction` aumenta diversidade sem remover n-grama, sem reduzir score e sem publicar nada.

Consequencia: `internal/batchdraftgen` passa a misturar pista primária e secundária em perfis de rodada; `internal/contract` ganha teste para geração diversa em 280 por família. O dry-run de 280 passou com `generated_drafts=1680`, `rewritten=1680` e `max_similarity=0.61`; o archive permanente, paid gates, candidate gates, reviews, prepublication, source-specificity, manifest e final drafts ficam em 1.680 registros bloqueados. `batch_candidate_gates` continua em shards físicos dentro do orçamento leve. O estado público continua bloqueado: sem alteração em `content/pages.json`, sem diff público e sem flags públicas verdadeiras. O próximo ciclo deve crescer para 310 por família, total 1.860, mantendo teste de diversidade, paid/refinement/readiness/strategy, advance esperado, refresh pós-advance, pipeline e checks completos.

## 2026-06-10 — Expansao bloqueada para 1.860 e readiness leve derivado do archive

Decisao: crescer para 310 rascunhos por família, total 1.860, e corrigir `batch_candidate_expansion_readiness` para não armazenar a lista completa de intenções por família.

Motivos:
- o dry-run e a escrita permanente de 310 por família passaram com `generated_drafts=1860`, `rewritten=1860` e `max_similarity=0.61`;
- rascunho aprovado em laboratório deve ir para o repo permanente bloqueado, portanto archive e métricas foram materializados antes do commit;
- `check-storage-contract` reprovou corretamente `batch_candidate_expansion_readiness:1` por linha JSONL acima de 32KB, mostrando que guardar 310 IDs completos no readiness era bug de escala;
- aumentar orçamento ou ignorar storage mascararia o problema e voltaria em 2.040/10k+;
- a lista completa já existe no archive permanente e pode ser derivada por `batch_id` + `target_candidate_count` sem perder validação.

Consequencia: `batch_draft_expansion_archive`, paid gates, candidate gates, reviews, prepublication, source-specificity, public manifest e final drafts ficam com 1.860 registros bloqueados. `batch_candidate_expansion_readiness` passa a gravar `expansion_candidate_selector=archive_prefix_by_batch` e amostra curta, enquanto `batchcandidateexpansion.CandidateIntentIDs`, `batchcandidatepromotion` e `paidintent` derivam os alvos completos do archive. Testes novos impedem readiness pesado e provam promoção por alvos derivados. O estado público continua bloqueado: sem alteração em `content/pages.json`, sem diff público, sem `render_allowed=true`, `sitemap_allowed=true`, `publication_allowed=true` ou `public_path`. O próximo ciclo deve crescer para 340 por família, total 2.040, mantendo storage leve, diversidade, paid/refinement/readiness/strategy, advance esperado, refresh pós-advance, pipeline, checks completos e otimização do gargalo de similaridade/readiness.
