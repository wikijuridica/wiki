# LAB_VALIDATION.md

## Regra

O projeto deve operar como laboratorio: testar, validar, refinar, testar novamente e somente entao registrar checkpoint. Nada de chute.

Laboratorio deve usar engenharia agressiva inteligente. O agente deve planejar a hipótese do ciclo, rodar validação em massa quando o escopo for massa, usar CPU disponível para acelerar prova e refinar algoritmo antes de expor qualquer conteúdo público. Passividade, pergunta desnecessária, commit sem autocrítica e validação pequena para decisão massiva são falhas de laboratório.

Otimização é regra do laboratório. Antes de commitar ciclo com teste ou gate lento, medir duração real, identificar gargalos e aplicar melhoria segura quando houver caminho técnico. `./tools/profile-contract-tests` ranqueia testes lentos de contrato por `go test -json`; `./tools/lab-cycle` imprime `TIMING start/pass/fail` por etapa. Se um teste pesado continuar necessário, registrar estratégia de shard ou índice em memória no checkpoint.

Em ciclos com duas ou mais frentes independentes, o agente principal deve acionar agentes auxiliares em paralelo, no máximo permitido e pertinente para o ciclo, para acelerar pesquisa de fontes oficiais, matriz de proveniência, criação de conteúdo bloqueado, testes, edição localizada, revisão documental e debugging. A divisão deve ser planejada por tarefas independentes, sem concorrência no mesmo arquivo, gate, commit, fonte jurídica ou decisão crítica. A saída de agente é insumo isolado até o Codex principal validar. Antes de checkpoint/commit, registrar `.agents/agent_context_ledger.jsonl` com ID real, escopo, status, política de uso, evidências, riscos, decisão de integração e fechamento, para compactação não apagar contexto.

Validação deve ser proporcional ao risco. Validação global completa é cara em tempo e só deve rodar quando houver alteração ampla, mudança em contrato central, risco P0/P1 crítico, modificação de HTML/sitemap/canonical/robots/indexação/performance, gerador em massa, preparação de publicação ou falha que possa contaminar várias camadas. Em ciclo pequeno/localizado, use teste focado, check específico, inspeção do diff/artefato e `git diff --check`.

## Comando global

`./tools/lab-cycle`

Esse comando combina:
- `go test -count=1 ./...`;
- `go run ./cmd/check all`, que cobre os checks nomeados sem repetir `go test`;
- `./tools/generate-batch-drafts --samples-per-batch 10 --checked-at 2026-06-09`;
- `./tools/lab-term-draft`;
- `./tools/lab-content-quality`;
- `go run ./cmd/build public`;
- `go list -m all`;
- `git diff --check`;
- busca por residuos `.py` e `.pyc`.

Use `./tools/lab-cycle` como prova global em momentos críticos ou alterações de grande alcance. Não transforme esse comando em ritual automático para toda edição pequena: isso aumenta custo sem melhorar a prova. O checkpoint deve explicar por que a validação escolhida foi suficiente; se a validação global foi pulada, registrar os checks focados usados. O comando global não deve chamar `./tools/check-all` nem repetir checks individuais já cobertos por `cmd/check all`; duplicação sem ganho é falha de otimização.

O gate `./tools/check-performance-budget` deve reprovar HTML publico pesado, `<script>`, runtime cliente, bundle JavaScript, WebAssembly, mapas, `modulepreload`, import map, marcadores de hidratacao e CSS inline excessivo. Leveza e parte da prova de indexacao para Googlebot e bots valiosos.

`./tools/check-agent-context-ledger` valida que agentes auxiliares usados no ciclo foram registrados em `.agents/agent_context_ledger.jsonl`, com política `reference_only_no_repo_write` ou `delegated_repo_write_codex_validated`, sem escrita concorrente, com evidência, risco, decisão de integração, fechamento antes do checkpoint e validação obrigatória pelo Codex principal. Sem esse ledger, achado de subagente não conta como prova de checkpoint; escrita delegada sem validação principal reprova.

`./tools/lab-content-quality` usa arquivos temporarios em `/tmp` para validar texto natural versus texto mecanico. Isso e laboratorio, nao publicacao. Ele deve detectar conteudo raso, keyword stuffing e permutacao antes que qualquer pagina seja exposta ao Googlebot.

`./tools/check-cpu-budget` vale para runtime publico/producao. Ele nao proibe testes, build, laboratorio, auditorias ou validacoes em massa mais pesadas quando forem necessarias para provar qualidade. No laboratorio, usar CPU de forma agressiva e aceitavel quando acelera score, refinamento, descoberta de falha ou prova de escala; o gate impede que caminhos de atendimento publico gastem CPU com execucao externa, rede, sleeps ou loops sem limite.

`./tools/lab-term-draft` usa seeds juridicas aprovadas pelo contrato de `term_seeds` para criar rascunho temporario em `/tmp`. O rascunho deve ser `draft/noindex`, nao pode escrever em `content/pages.json`, nao pode entrar em sitemap e nao pode receber CTA.

`./tools/persist-term-drafts` e ferramenta de ciclo para gravar rascunho validado na camada `editorial_drafts`; `./tools/check-editorial-drafts` e o gate permanente no laboratorio para garantir que o draft persistido continua fora da publicacao.

`./tools/queue-editorial-review` move rascunhos persistidos para fila de revisao `needs_review`, sem publicacao. `./tools/check-review-queue` exige `publication_allowed=false`, `public_path` vazio, fonte, autoria e historico.

`./tools/approve-editorial-review` registra aprovacao editorial de laboratorio; `./tools/check-approvals` garante que essa aprovacao nao cria rota publica nem libera indexacao.

`./tools/block-publication` cria manifesto de publicacao bloqueada; `./tools/check-publication-blockers` garante que aprovacao editorial ainda nao libera URL publica.

Quando o laboratorio passar com seguranca, o ciclo seguinte deve migrar para host/produção controlada e priorizar termos juridicos pesquisados por humanos e com alta intencao de contratacao online, sem spam.

Alta intencao de contratacao online exige que o fluxo possa ser 100% digital: WhatsApp, envio remoto de documentos, triagem remota e contratacao de advogado sem depender de atendimento presencial como caminho principal.

Regra OAB de laboratório: pesquisar e respeitar a ética da OAB antes de alterar contrato de publicidade, conteúdo informativo ou CTA. O Provimento OAB 205/2021 permite marketing jurídico compatível com ética e marketing de conteúdo jurídico voltado a informar o público; o Código de Ética exige sobriedade, discrição, caráter informativo e vedação à captação indevida. Portanto, 100% digital, tudo online, sem sair de casa, envio remoto de documentos, WhatsApp contextual e atendimento a brasileiros fora do Brasil não é promessa de resultado quando descreve modo de atendimento; é realidade brasileira e não deve ser bloqueada como promessa absoluta. Promessa de resultado continua sendo garantia de êxito, decisão, liminar, prazo, vantagem, caso concreto como oferta ou linguagem apelativa. Autor dos conteúdos: Rafael Toledo, OAB/RJ 227191, sempre vindo de configuração/dado versionado e não hardcoded.

Os caminhos seguros de pesquisa de demanda ficam em `docs/data-sources/search-demand.md`. O laboratorio deve validar `term_intent_candidates` antes de transformar qualquer termo pesquisado em seed, rascunho ou pauta publica.

Para termos de alta intencao, a estrategia preferida e pesquisa editorial manual na web, nao script de descoberta. O laboratorio deve validar `manual-keyword-research`, `content-briefs` e `authorial-content-drafts`, revisando se os briefs e rascunhos possuem angulo unico, nao repetem molde e nao tentam publicar. Google Trends orienta demanda, mas nao decide sozinho.

Promocao de candidato para seed deve passar por ranking refinavel, nao por lista fixa. O score deve considerar area, risco, fonte oficial, adequacao digital, CTA WhatsApp e penalidades para fluxo presencial ou evidencia fraca. Quando falso positivo ou falso negativo aparecer, escrever teste e ajustar o algoritmo antes de continuar.

Rascunho autoral deve passar por algoritmo anti-template antes de qualquer novo lote: abertura repetida, conjunto de secoes reaproveitado, heading generico, CTA raso, fonte ausente ou path publico devem reprovar. Contagem de palavras isolada nao basta para liberar conteudo.

Se a inspeção mostrar grafia mecanica ou sem acento em rascunho PT-BR, tratar como falha de laboratorio. Refinar `internal/draftlab`, regenerar `data/editorial/drafts.jsonl`, atualizar `data/editorial/review_queue.jsonl` e rodar novamente os checks antes do commit.

Antes de aprovar qualquer draft priorizado, validar `source_specificity_blockers`. O laboratorio deve bloquear termo que usa apenas fonte ampla, noticia institucional ou canal administrativo quando ainda falta norma, artigo, regra ou recorte juridico especifico.

Quando uma fonte especifica for encontrada, registrar `source_specificity_resolutions` e rodar `./tools/check-source-specificity-resolutions`. A resolucao aproxima o rascunho da revisao, mas permanece `noindex`, sem URL publica e sem CTA publico ate os demais gates.

Antes de renderizar uma rota candidata, registrar `prepublication_gates` e rodar `./tools/check-prepublication-gates`. O gate deve provar title/meta/canonical dentro do orcamento do Google Search, mas manter render, sitemap e publicacao bloqueados.

Antes de qualquer CTA visivel, registrar `legal_editorial_reviews` e rodar `./tools/check-legal-editorial-reviews`. CTA WhatsApp deve ser contextual, pedir documentos para triagem e reprovar qualquer promessa de resultado, prazo ou liminar.

Antes de qualquer lote massivo, registrar `scalable_content_batches` e rodar `./tools/check-scalable-content-batches`. O lote precisa ter validação em massa, score humano mínimo, limite IA-like, limite de similaridade, CTA contextual, fontes oficiais por família e publicação bloqueada.

Antes de qualquer reescrita/publicação derivada de lote, registrar `human_content_score` e rodar `./tools/check-human-content-score`. Score baixo ou risco IA-like alto exige reescrita e nova validação, nunca publicação ou correção manual repetitiva pelo usuário.

Antes de ampliar produção, registrar `batch_drafts` e rodar `./tools/check-batch-drafts`. Cada lote deve ter amostras suficientes, score calculado, reescrita automática comprovada quando houve falha inicial, baixa similaridade intra-lote e publicação bloqueada.

Antes de considerar o gerador pronto para volume maior, rodar `./tools/check-batch-draft-generation` e `./tools/generate-batch-drafts`. O gerador deve ser determinístico, produzir amostras temporárias em `/tmp`, persistir métricas agregadas, provar reescrita automática e manter `render_allowed=false`, `sitemap_allowed=false`, `publication_allowed=false` e `public_path=""`.

Quando uma massa gerada em `/tmp` passar nos gates e tiver utilidade jurídica para páginas futuras, ela deve ser trazida para `data/editorial/batch_draft_expansion_archive.jsonl` e validada por `./tools/check-batch-draft-expansion-archive`. Laboratório validado não deve ser descartado por padrão; repo permanente bloqueado é o caminho de continuidade. Se algo aprovado ainda estiver só no laboratório, o ciclo deve migrar esse artefato para repo antes do commit, mantendo `noindex`, sem render, sitemap, publicação ou URL pública.

Antes de ampliar a seleção de candidatos por família, rodar `./tools/check-batch-candidate-expansion-readiness`. O gate deve provar que a família tem massa permanente suficiente, CTA contextual, score humano, baixa similaridade, paid-intent por intenção, fonte específica ou bloqueio acionável, `index_policy=noindex`, manifesto falso e flags públicas falsas. Se faltar paid gate para as intenções expandidas, o estado correto é `batch_candidate_expansion_blocked_paid_gate_missing`; se o gate existir mas reprovar, o estado correto é `batch_candidate_expansion_blocked_paid_gate_failed`. Nenhum dos dois pode ser mascarado como readiness pronta.

Depois que readiness e fonte deixarem de ser gargalo para uma família, rodar `./tools/refresh-batch-expansion-strategy` e `./tools/check-batch-expansion-strategy`. A estratégia deve planejar crescimento real do próximo lote, mas limitado e validável: só famílias com paid-intent e fonte prontos podem aumentar alvo; famílias com paid-intent reprovado preservam bloqueio. Estratégia não é publicação e deve manter `noindex`, sem manifesto, render, sitemap, publicação ou `public_path`.

Antes de preparar pré-publicação por lote, rodar `./tools/check-batch-candidate-gates`. O gate deve selecionar candidatos reais do arquivo permanente, acompanhar a base oficial configurada em `content/site.json`, exigir CTA contextual e impedir render, sitemap, publicação ou `public_path`.

Depois de expandir `batch_candidate_gates`, rodar `./tools/refresh-batch-candidate-pipeline`. O refresh deve propagar a seleção para revisões, pre-publication, fonte específica, manifesto e rascunhos finais bloqueados. Se a seleção subir de 18 para 168, 318, 330, 468, 510, 590 ou qualquer novo patamar validado, os artefatos downstream também devem cobrir o mesmo current materializado. Rascunhos finais só podem preservar candidatos ainda elegíveis por fonte travada e `paidintent.AllowsExpansion`. Falha de contagem downstream é falha P0, não detalhe cosmético.

Antes de transformar candidatos de lote em pré-publicação, rodar `./tools/check-batch-candidate-reviews`. Cada candidato selecionado precisa de revisão jurídico-editorial bloqueada, fonte matricial auditada e CTA WhatsApp de origem rastreável. URL oficial travada não dispensa revisão; promessa de resultado, path com domínio, fonte sem auditoria ou flag pública verdadeira bloqueiam o lote.

Antes de qualquer rota candidata de lote se aproximar de renderização, rodar `./tools/check-batch-prepublication-gates`. Esse gate registra canonical oficial e `noindex,follow`, mas mantém fonte final, revisão SEO e manifesto público como pendências. Render, sitemap, publicação e `public_path` devem continuar falsos.

Antes de qualquer manifesto público bloqueado por lote, rodar `./tools/check-batch-source-specificity`. Esse gate cobre todos os candidatos de pré-publicação e separa `final_source_locked_reference_only` de `final_source_blocked_needs_specific_url`. Fonte ampla não pode ser mascarada como pronta; fonte travada continua apenas referência, sem scraping, ingestão, render, sitemap, publicação ou `public_path`.

Antes de gerar rascunho final ou revisão SEO de candidato de lote, rodar `./tools/check-batch-public-manifest-gates`. Esse gate nao publica: ele apenas separa candidatos com fonte travada que ainda precisam SEO/conteudo final de candidatos bloqueados por fonte ampla. Candidato com `final_source_blocked_needs_specific_url` nao pode entrar em revisão SEO como se a fonte estivesse pronta.

Antes de ampliar rascunhos finais de lote, rodar `./tools/check-batch-final-authorial-drafts` e `./tools/check-paid-intent`. O primeiro gate exige rascunho autoral natural, fonte travada, CTA contextual, aviso informativo e bloqueio total de render/sitemap/publicação. O pipeline deve regenerar rascunho final que falhar score humano/qualidade, mesmo que path, title e manifest ainda batam. O segundo gate valida `batch_paid_intent_gates`: famílias comerciais exigem contratação particular online, honorários/orçamento e contexto jurídico-econômico no corpo informativo, não apenas no CTA; previdenciário pode usar lane informativa/curiosa bloqueada quando o batch for `batch-previdenciario-digital`. famílias comerciais também são informativas: não retirar o conteúdo informativo da plataforma jurídica; a intenção natural de contratação deve ser adicionada sem remover fonte, explicação, documentos, risco e aviso informativo. Previdenciário não precisa de alta intenção de pagamento para laboratório bloqueado: curiosidade qualificada sobre benefício, CNIS, perícia, exigência, prazo, documento ou revisão pode crescer famílias previdenciárias com status `paid_intent_flexible_previdenciario_informational_blocked_publication`, enquanto famílias comerciais continuam exigindo intenção paga ou bloqueio explícito. Gratuidade explícita, defensoria/justiça gratuita, não pagamento, estudo, modelo pronto, promessa de benefício e substituição de canal público continuam bloqueios.

Quando paid-intent comercial reprovar por sinal pago ausente ou CTA-only, rodar `./tools/refine-paid-intent-drafts`, depois `./tools/generate-paid-intent-gates`, `./tools/check-paid-intent-refinements`, `./tools/check-paid-intent` e `./tools/refresh-expansion-readiness`. O refinador deve ser idempotente: se o ciclo ja foi aplicado e nao houver pendencia, `refinements=0` deve passar. O refinador nao resgata temas de gratuidade, defensoria/justiça gratuita, não pagamento ou promessa de benefício. Previdenciário informativo não deve ser "resgatado" para simular alta intenção paga: ele cresce por status próprio bloqueado e continua sem render, sitemap, publicação ou `public_path`.

Fonte legal geral não basta para destravar recorte específico. CLT compilada, Código Civil e CDC são referências oficiais importantes, mas devem ser tratados como fonte ampla quando o recorte exige ato, procedimento, orientação, súmula, serviço ou regra específica. `./tools/check-batch-source-specificity` deve reprovar tentativa de marcar esses códigos gerais como `final_source_locked_reference_only` sem complemento específico auditado.

Ancora de artigo em fonte oficial pode ser complemento especifico quando registrada no audit URL-level, mas so como referencia bloqueada. Se a rede local nao confirmar o conteudo por timeout, registrar o status real, preservar `reference_only_no_scraping_no_ingestion` e manter publicacao bloqueada ate revisao posterior; nao copiar texto, nao inferir ementa e nao promover HTML publico por causa da ancora.

Antes de ampliar subtemas, rodar `./tools/check-batch-source-matrix`. A matriz deve provar fonte oficial específica por subtema, política sem scraping e cobertura de cada draft gerado. Se a fonte estiver genérica, ausente ou sem revisão de robots, o lote fica bloqueado.

Antes de ampliar rascunhos por família, rodar `./tools/check-batch-source-url-audits`. Toda URL da matriz deve estar auditada por hash, robots/termos, política sem scraping/ingestão e vínculo com os subtemas que a usam. Falta de auditoria URL-level bloqueia o lote inteiro.

Se a similaridade subir em volume maior, investigar a semântica: verificar se os textos diferem em problema, documento, fonte, risco e ação digital. Não resolver similaridade com troca mecânica de palavras ou redução de limite.

## Politica

Nenhum script isolado e prova suficiente para mudanca P0/P1 ampla. Use o laboratorio como conjunto proporcional de provas e inspecione os artefatos quando a mudanca afetar HTML, sitemap, robots, canonical, indexacao, qualidade ou conteudo. Mudança localizada deve priorizar validação focada, desde que cubra diretamente o risco alterado.

Laboratorio verde nao autoriza mascarar bug de contrato. Antes de commit com conteudo, arquivo editorial, paid-intent, source-specificity, candidate gate ou strategy, revisar tecnicamente o contexto: diff, dados JSONL, motivo de blocker, amostras de texto, fonte, CTA e semantica da familia. Se houver falso positivo, falso negativo, dado stale, texto raso, similaridade suspeita, fonte ampla tratada como especifica ou conteudo comercial demais, corrigir algoritmo/teste/dado e rodar novamente. Nao baixar regra, nao remover blocker, nao renomear status nem aceitar "passou no script" como prova isolada.

Antes de refinar algoritmo no laboratório, capturar dados reais do erro. Para similaridade, n-grama, título repetitivo, vazamento de área ou paid-intent suspeito, registrar ou imprimir o par/registro, `legal_area`, `source_matrix_id`, `unique_intent_id`, campos textuais e motivo de blocker. Ajuste sem essa evidência é chute e falha de contrato. Se o diagnóstico mostrar que o teste está certo, corrigir geração/conteúdo; se mostrar falso positivo, fortalecer o comparador sem baixar limite e mantendo teste negativo.

Validar nao basta. Todo ciclo deve revisar o diff, os artefatos gerados, os riscos e o contrato antes de commitar. A revisao deve procurar atalhos, spam, conteudo mecanico, regressao de P0 e dependencia indevida.

Antes de commitar, o agente deve fazer autocrítica explícita e registrar no checkpoint:
- o que este ciclo realmente resolveu;
- por que o commit não é conclusão do `/goal`;
- se existe pendência mascarada como entrega;
- se o diff público foi protegido quando aplicável;
- o que pode ser melhorado no próximo ciclo;
- qual é o próximo passo executável.

Commit sem plano de continuidade e sem autocrítica é inválido para este projeto.

Se qualquer validacao falhar:
1. identificar causa raiz;
2. corrigir;
3. rodar novamente;
4. registrar falha e correcao no checkpoint;
5. commitar checkpoint e artefatos do ciclo;
6. continuar o proximo ciclo quando nao houver bloqueio P0 real.

## Continuidade

O agente nao deve parar ao registrar checkpoint. Todo ciclo deve terminar com proximo passo acionavel, e esse proximo passo deve ser executado em seguida quando nao houver bloqueio P0 real comprovado.

## Commit por ciclo

Depois de validacoes relevantes passarem, fazer commit do checkpoint e dos artefatos do ciclo. O projeto nao deve depender de chat, contexto compactado ou memoria externa para continuidade.

Antes do commit, executar `date`, conferir a hora local e registrar o ciclo numerado no checkpoint. Isso mantem a ordem de auditoria mesmo apos compactacao de contexto.
