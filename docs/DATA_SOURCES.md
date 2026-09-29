# DATA_SOURCES.md

Antes de ingestao, cada fonte oficial precisa de documento em `docs/data-sources/`.

Estado inicial:
- Portal da Legislacao / Planalto: URL oficial confirmada como `https://legislacao.presidencia.gov.br/` por pagina publica `gov.br`, documentada em `docs/data-sources/planalto.md`, sem ingestao automatica por falha de alcance HTTP local.
- Camara Dados Abertos: documentado preliminarmente em `docs/data-sources/camara-dados-abertos.md`, sem ingestao automatica.
- Senado Dados Abertos: documentado preliminarmente em `docs/data-sources/senado-dados-abertos.md`, sem ingestao automatica.
- LexML: documentado preliminarmente em `docs/data-sources/lexml.md`, sem ingestao automatica.
- CNJ/Datajud: documentado preliminarmente em `docs/data-sources/cnj-datajud.md`, sem ingestao automatica.
- CNJ Atos / Resolução 35/2007: documentado em `docs/data-sources/cnj-atos-resolucao-35.md`, usado apenas como referência oficial específica para atos notariais de família/sucessório, sem ingestão automática.
- STF: documentado preliminarmente em `docs/data-sources/stf.md`, sem ingestao automatica.
- STJ: documentado preliminarmente em `docs/data-sources/stj.md`, sem ingestao automatica.
- CJF: documentado preliminarmente em `docs/data-sources/cjf.md`, sem ingestao automatica.
- APIs auxiliares candidatas: documentadas em `docs/data-sources/api-candidates.md`, sem ingestao automatica.
- Pesquisa de demanda e intencao: documentada em `docs/data-sources/search-demand.md`; Google Trends e Google Search Central sao caminhos seguros para sinal direcional/metodologico, nao fonte juridica e nao autorizacao de publicacao automatica.

Nenhum scraping cego esta autorizado neste ciclo.

Antes de qualquer conteudo juridico em escala, a fonte correta deve ser pesquisada e documentada. A ausencia de fonte documentada impede publicacao indexavel, mesmo que exista demanda comercial ou CTA.

O registro operacional fica em `content/source_registry.json`. Durante P0, todas as fontes devem manter `ingestion_enabled=false`.

As fontes listadas sao referencia e proveniencia. Elas nao autorizam scraping, clonagem, espelhamento ou criacao de paginas mecanicas. O portal deve escrever conteudo proprio, natural e unico a partir de pesquisa critica.

## Banco leve separado

Ingestao de termos juridicos pode iniciar a producao de conteudo apenas como rascunho (`draft_only`). Esses termos ficam em `data/terms/legal_terms.jsonl`, separados da auditoria de fonte, snapshots oficiais, rascunho editorial e manifesto publicado.

O contrato de armazenamento fica em `content/storage_contract.json` e e validado por `./tools/check-storage-contract`. Nenhuma fonte auditada, termo juridico ou snapshot autorizado pode ser tratado como conteudo publico pronto. O caminho correto e: termo com proveniencia -> rascunho editorial proprio -> revisao -> qualidade -> SEO -> publicacao.

As sementes iniciais ficam em `data/terms/legal_terms.jsonl` e sao validadas por `./tools/check-term-seeds`. Isso permite iniciar laboratorio editorial por termos sem gerar paginas para Googlebot.

Os candidatos de alta intencao ficam em `data/terms/intent_candidates.jsonl` e sao validados por `./tools/check-term-intent-candidates`. Cada candidato precisa separar demanda humana, fonte juridica oficial, adequacao a atendimento 100% digital e bloqueio de publicacao.

A pesquisa editorial manual fica em `data/research/high_intent_terms.jsonl` e e validada por `./tools/check-manual-keyword-research`. Ela registra termos de alta intencao de contratacao juridica digital, usando Google Trends como orientacao direcional e fontes oficiais como autoridade/proveniencia. Isso substitui dependencia de script fraco para descoberta de termos.

Briefs iniciais de conteudo ficam em `data/editorial/content_briefs.jsonl` e sao validados por `./tools/check-content-briefs`. Eles iniciam a construcao de conteudo, mas nao publicam, nao criam URL, nao entram em sitemap e nao autorizam CTA publico.

Rascunhos autorais ficam em `data/editorial/authorial_drafts.jsonl` e sao validados por `./tools/check-authorial-content-drafts`. Eles transformam briefs em texto proprio natural, mas continuam sem publicacao, sem URL publica e sem sitemap ate fonte especifica, revisao, SEO/crawl e qualidade passarem.

Quando um candidato e promovido para `term_seeds`, ele deve preservar `candidate_id`, URL de evidencia de demanda, grupo de consulta, modo `digital_only` e intencao alta de CTA WhatsApp. Essa promocao nao cria pagina, nao cria CTA publico e nao libera sitemap.

Bloqueios de fonte especifica ficam em `data/editorial/source_blockers.jsonl` e sao validados por `./tools/check-source-specificity-blockers`. Eles registram a fonte atual, os tipos de fonte ainda exigidos e a proxima direcao de pesquisa para cada termo priorizado.

Resolucoes de fonte especifica ficam em `data/editorial/source_resolutions.jsonl` e sao validadas por `./tools/check-source-specificity-resolutions`. Elas registram URLs oficiais especificas, tipo de fonte, uso editorial permitido e gates restantes, mas nao armazenam texto oficial bruto e nao liberam publicacao.

Gates de pre-publicacao ficam em `data/editorial/prepublication_gates.jsonl` e sao validados por `./tools/check-prepublication-gates`. Eles conectam fonte resolvida, blocker ainda ativo e contrato SEO/crawl candidato, sem render publico ou sitemap.

Revisoes juridico-editoriais ficam em `data/editorial/legal_reviews.jsonl` e sao validadas por `./tools/check-legal-editorial-reviews`. Elas conectam rascunho autoral, fonte resolvida e pre-publicacao, preparando CTA WhatsApp como rascunho interno sem promessa e sem publicacao.

Rascunhos persistidos ficam em `data/editorial/drafts.jsonl`, separados de snapshots oficiais e de paginas publicas. Eles devem permanecer `draft/noindex` ate fonte, revisao, qualidade e decisao editorial completa.

A fila de revisao fica em `data/editorial/review_queue.jsonl`. Ela registra autoria, motivo, historico e estado `needs_review`, mas nao libera publicacao nem cria URL.

Aprovacoes editoriais ficam em `data/editorial/approved_drafts.jsonl`. Mesmo aprovadas editorialmente, continuam separadas do manifesto publicado ate fonte especifica, revisao completa, qualidade, SEO, CTA e escala segura.

O manifesto de bloqueio fica em `data/editorial/publication_blockers.jsonl`. Ele lista requisitos faltantes para publicar e orienta a priorizacao de termos por demanda humana, nao por permutacao de keyword.

A pesquisa de demanda de termos deve separar evidencia de busca humana, fonte juridica e intencao comercial. Para a fase inicial, alta intencao significa contratacao juridica online, sem requisito presencial como padrao. Termos de alta demanda mas baixa adequacao a atendimento digital devem ser registrados como menor prioridade, nao mascarados como CTA forte.

## Lotes de conteúdo

Dados de lote ficam em banco leve separado, `data/editorial/scalable_content_batches.jsonl`, sem virar sitemap automaticamente. Cada lote precisa registrar:
- família jurídica;
- número de intenções únicas planejadas;
- fontes oficiais por subtema;
- critérios de diferenciação entre páginas;
- score mínimo humano/natural;
- limite de similaridade intra-lote;
- CTA contextual por origem;
- decisão de publicação bloqueada ou liberada.

Scores humanos/naturalidade ficam em `data/editorial/human_content_scores.jsonl`. Esse arquivo guarda metadados de score e decisão de reescrita, não texto oficial bruto nem página pública.

Rascunhos de lote ficam em `data/editorial/batch_drafts.jsonl`. Eles podem guardar texto editorial próprio de amostra para laboratório, mas continuam `noindex`, sem render, sem sitemap, sem `public_path` e sem publicação.

Arquivo permanente de expansão fica em `data/editorial/batch_draft_expansion_archive.jsonl`. Ele preserva rascunhos massivos que passaram nos gates de laboratório, com fonte, score, reescrita e bloqueio público. Esse arquivo não é manifesto publicado, não guarda texto oficial bruto e não autoriza render, sitemap, CTA público ou indexação.

Gates candidatos de lote ficam em `data/editorial/batch_candidate_gates.jsonl`. Eles selecionam intenções existentes no arquivo permanente para análise posterior, mas continuam bloqueados: sem HTML, sem sitemap, sem `public_path`, sem CTA público e sem URL oficial travada.

Métricas de geração/refino em lote ficam em `data/editorial/batch_generation_metrics.jsonl`. Elas registram quantidade gerada, amostras aprovadas, reescritas, menor score humano, maior risco IA-like, similaridade máxima e próximo passo de validação. Esse arquivo é banco leve de rastreabilidade, não manifesto público.

Matriz de fontes por lote fica em `data/editorial/batch_source_matrix.jsonl`. Cada registro conecta um subtema de alta intenção a URLs oficiais usadas como referência/proveniência, com política `reference_only_no_scraping`. A matriz não autoriza cópia de texto oficial, não faz scraping e não cria página pública.

Auditoria URL-level da matriz fica em `data/source-audit/batch_source_urls.jsonl`, separada do conteúdo editorial. Cada registro cobre uma URL oficial única com hash `urlsha256`, `matrix_ids`, robots/termos revisados, política `reference_only_no_scraping_no_ingestion` e bloqueio explícito de scraping, ingestão, render, sitemap e publicação.

No ciclo atual, `https://atos.cnj.jus.br/atos/detalhar/179` foi auditada como URL específica da Resolução CNJ 35/2007 e vinculada aos subtemas de divórcio/partilha/inventário extrajudicial. A fonte aumenta especificidade da matriz, mas continua referência: não autoriza copiar texto oficial, fazer scraping, ingerir payload, renderizar página ou publicar sitemap.

Resolucao de especificidade por candidato fica em `data/editorial/batch_source_specificity_resolutions.jsonl`. Cada registro cobre um candidato de `batch_prepublication_gates` e decide se a fonte auditada ja e especifica o bastante para referencia final (`final_source_locked_reference_only`) ou se continua bloqueada por fonte ampla (`final_source_blocked_needs_specific_url`). Mesmo fonte travada nao autoriza scraping, ingestao, render, sitemap, publicacao ou `public_path`.

Manifesto publico bloqueado por candidato fica em `data/editorial/batch_public_manifest_gates.jsonl`. Ele ainda nao e `published_manifest`: registra quais candidatos com fonte travada podem seguir para rascunho/SEO final bloqueado e quais continuam parados por fonte ampla. Todo registro permanece `noindex`, `manifest_allowed=false`, `render_allowed=false`, `sitemap_allowed=false`, `publication_allowed=false` e `public_path=""`.

Termos em massa não podem nascer de combinação infinita de cidade, palavra-chave e área. A escala deve vir de problemas jurídicos reais, etapas processuais/administrativas, documentos, riscos, fontes e intenções digitais distintas.

Pesquisa de fonte em escala deve ser planejada por famílias e subtemas, nao improvisada por página. O lote deve carregar mapa de fontes suficientes para cada grupo de intenção, com fonte oficial/proveniência antes de rascunho público. Ausência de fonte específica bloqueia o lote, mesmo que a demanda e o CTA sejam fortes.
