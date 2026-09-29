# ARCHITECTURE.md

## Decisao de base

O projeto usa Go e biblioteca padrao como base tecnica. A escolha privilegia binario proprio, tipagem estatica, concorrencia nativa, `net/http`, `encoding/xml`, `html` e `testing`, sem SDK externo e sem framework frontend.

## Modulos

- `cmd/build`: gera artefatos publicos em `public/`.
- `cmd/check`: executa validadores locais.
- `cmd/profile-tests`: executa perfil de testes Go via `go test -json` para ranquear gargalos antes de commit.
- `cmd/server`: entrega paginas com geracao on demand propria.
- `internal/content`: modelos e carregamento do manifesto de paginas.
- `internal/manualresearch`: banco leve de pesquisa editorial manual de termos de alta intencao digital.
- `internal/contentbriefs`: briefs editoriais iniciais, nao-template e nao publicaveis.
- `internal/scale`: plano finito de blueprints para pelo menos 10 mil paginas sem publicar conteudo durante P0.
- `internal/cta`: politica propria de CTA WhatsApp subordinada a fonte, revisao e aprovacao editorial.
- `internal/router`: rotas canonicas limpas e mapeamento para cache/saida.
- `internal/render`: HTML completo textual, sem hidratacao.
- `internal/ondemand`: geracao sob demanda e cache local controlado pelo projeto.
- `internal/seo`: title, meta description, canonical e robots.
- `internal/crawl`: robots.txt e politica configuravel de bots.
- `internal/sitemap`: sitemap index e sitemap particionado.
- `internal/quality`: gates contra duplicidade, conteudo raso, falta de fonte e falta de revisao.
- `internal/editorial`: estados editoriais e politica index/noindex.
- `internal/editorialdrafts`: persistencia de rascunhos em banco leve, sempre sem rota publica.
- `internal/reviewqueue`: fila editorial `needs_review` com historico e publicacao bloqueada.
- `internal/approvals`: aprovacao editorial separada de publicacao, ainda sem URL publica.
- `internal/publicationblockers`: manifesto de requisitos faltantes antes de qualquer URL publica.
- `internal/sourceblockers`: bloqueios por termo quando a fonte atual ainda nao e especifica o suficiente para aprovacao/publicacao.
- `internal/legal`: controle de conteudo juridico.
- `internal/sources`: contratos de fontes oficiais.
- `internal/storage`: banco leve proprio em JSONL para termos juridicos, auditorias, snapshots, rascunhos e manifesto publicado.
- `internal/termintents`: candidatos de termos juridicos com demanda humana, fonte oficial e adequacao a contratacao 100% digital.
- `internal/termpromotion`: ranking e promocao controlada de candidatos para seeds `draft_only`, com diversidade de areas e penalizacao de sinais fracos.
- `internal/batchdraftarchive`: arquivo permanente bloqueado de rascunhos de lote que passaram no laboratorio e ainda nao podem virar pagina publica.
- `internal/batchcandidategates`: selecao bloqueada de candidatos a partir do arquivo permanente, com base URL configuravel e publicacao falsa.
- `internal/batchcandidatereviews`: revisao juridico-editorial bloqueada de candidatos de lote, com CTA WhatsApp contextual e matriz de fonte auditada.
- `internal/batchprepublication`: gates de pre-publicacao bloqueada por candidato revisado, com canonical oficial e `noindex`.
- `internal/batchfinaldrafts`: rascunhos autorais finais bloqueados para candidatos com fonte travada e manifesto SEO pendente.
- `internal/paidintent`: gate persistente de intenção comercial paga, bloqueando gratuidade explícita, não pagamento, promessa indevida e status comercial fraco; separa sinal pago do corpo e do CTA, exige contratação particular online nas famílias comerciais e mantém uma lane previdenciária informativa bloqueada por `paid_intent_flexible_previdenciario_informational_blocked_publication`.
- `internal/paidintentrefinement`: refinador em lote para mover sinal de contratação paga do CTA para o corpo informativo quando natural, persistindo ledger bloqueado e mantendo idempotência em no-op.
- `cmd/refresh-editorial-drafts`: regeneracao segura de rascunhos persistidos quando o algoritmo de escrita e refinado.
- `internal/provenance`: contrato de proveniencia por payload antes de qualquer conteudo.
- `internal/architecture`: validacao de estrutura e proibicoes P0.
- `internal/agentcontext`: ledger persistente de contexto de agentes auxiliares para sobreviver a compactação, exigir ID real/escopo/evidência/risco/decisão de integração e impedir uso de pesquisa ou escrita delegada sem validação principal.
- `internal/testprofile`: parser próprio de eventos `go test -json` para timing de testes lentos sem dependência externa.
- `tools`: scripts locais obrigatorios.

## Geracao on demand obrigatoria

Rotas publicas devem poder ser geradas sob demanda por `internal/ondemand`, sem Next.js e sem mecanismo terceirizado de ISR/SSR. O gerador resolve uma rota canonica, renderiza HTML completo no primeiro response e grava cache local. A segunda chamada pode servir do cache do proprio projeto.

O build estatico continua permitido como artefato operacional, mas nao substitui o requisito de geracao on demand propria.

## URL base e canonical

`content/site.json` define a base de canonical, robots e sitemap. O dominio oficial do projeto esta travado como `https://wikijuridica.com.br`, com `base_url_mode="official_configured"`, `official_url_status="locked"` e `official_url_locked=true`.

Validadores devem usar a base configurada, nao uma constante de dominio. O contrato continua exigindo HTTPS absoluto, path limpo e canonical correspondente a rota; a URL oficial travada nao libera publicacao de candidatos, que ainda dependem de fonte, revisao, qualidade, SEO, CTA e manifesto publico finito.

## HTML publico leve

Leveza e requisito de indexacao. O renderizador deve entregar HTML textual completo, com CSS minimo e sem JavaScript, bundle, WebAssembly, import map, `modulepreload`, payload de framework ou marcador de hidratacao. Isso protege Googlebot, OAI-SearchBot e outros bots valiosos, alem de reduzir custo operacional em escala massiva.

`internal/checks` reprova HTML publico acima do orcamento, CSS inline excessivo, referencias a runtime cliente e marcadores de Next.js, React, Vue/Svelte/Astro/Angular, Vite ou Webpack. Qualquer excecao exigiria ADR de dependencia e continuaria bloqueada para pagina publica indexavel enquanto P0 estiver ativo.

## Escala e fábrica de conteúdo

Durante P0, a plataforma deve preparar escala massiva sem publicar spam. `content/scale_plan.json` define blueprints finitos para pelo menos 10 mil paginas planejadas, mas a arquitetura deve suportar centenas de milhares ou milhões de URLs por geração on demand própria. O desbloqueio de conteudo exige P2: fonte oficial documentada, proveniencia, autoria, revisão automatizada/algorítmica comprovada, intenção única, qualidade e indexacao coerente.

O plano de escala e um contrato de capacidade e uma fábrica de conteúdo validado, nao um gerador de spam para Google. O modulo `scale` deve ajudar a medir alcance futuro, montar lotes finitos por família jurídica, diversificar intenção e bloquear publicacao em massa enquanto fontes, scoring, CTA e validadores nao estiverem maduros.

Eixo obrigatório implementado: `scalable_content_batches`. Essa camada registra lotes com centenas de milhares de intenções candidatas planejadas, score de naturalidade, fonte, CTA contextual, similaridade intra-lote e decisão de bloqueio. O pipeline deve gerar, pontuar, reescrever e revalidar em lote.

A arquitetura deve favorecer engenharia agressiva inteligente: processamento em lote, validação agregada, diagnósticos específicos, reexecução rápida, refinamento de algoritmo e prova por dados. O runtime público deve continuar barato, mas o laboratório pode consumir CPU de forma agressiva para provar escala e qualidade antes de qualquer publicação.

## CTA WhatsApp

`content/cta_policy.json` registra a arquitetura de CTA proprio por WhatsApp para paginas de alta intencao de contratar advogado. O CTA nao pode aparecer como atalho para publicar conteudo sem fonte ou sem revisao; ele depende de pagina juridica aprovada, proveniencia e revisao editorial.

CTA WhatsApp e critico para o produto: as paginas devem ser informativas, mas desenhadas para alta intencao de contratar advogado quando o contexto for adequado. A criticidade comercial nao remove os gates juridicos.

Todo CTA de WhatsApp deve carregar mensagem contextual de origem. A mensagem precisa incluir path/canonical ou rota candidata, intencao unica/termo e resumo da demanda para o atendimento saber de onde a pessoa veio e qual triagem inicial faz sentido.

## Proveniencia por payload

Antes de qualquer dado oficial virar conteudo, o payload precisa de registro com fonte, URL oficial, data de acesso, hash SHA-256, snapshot de robots, snapshot de termos, campos usados e finalidade. Fonte pesquisada nao equivale a conteudo aprovado.

## Banco leve de termos e ingestao

Ingestao de termos juridicos e valida para iniciar conteudos somente como semente de rascunho. O armazenamento e proprio, leve e separado em `content/storage_contract.json`, usando arquivos JSONL em `data/` e biblioteca padrao Go.

Camadas obrigatorias:
- `term_seeds`: termos juridicos para iniciar rascunhos, sem texto oficial bruto e sem texto editorial publico;
- `term_intent_candidates`: candidatos priorizados por demanda humana e contratacao online, ainda sem publicacao;
- `manual_keyword_research`: pesquisa editorial manual de alta intencao digital, com Trends como orientacao e fontes oficiais como autoridade;
- `term_seeds` promovidos: seeds com `candidate_id`, evidencia de demanda, modo `digital_only` e CTA alto, mas ainda `draft_only`;
- `source_audits`: auditoria de robots, termos de uso, alcance HTTP e decisao de bloqueio;
- `batch_source_url_audits`: auditoria URL-a-URL das fontes da matriz de lote, com hash da URL, robots/termos revisados, uso apenas referencial e bloqueio de scraping/ingestao/publicacao;
- `source_snapshots`: snapshots autorizados, pequenos, com hash e proveniencia;
- `editorial_drafts`: texto editorial proprio em PT-BR, sempre noindex ate aprovacao;
- `content_briefs`: brief inicial natural e especifico por termo, sem URL publica;
- `authorial_content_drafts`: rascunhos autorais derivados de briefs, com anti-template, CTA digital contextual e publicacao bloqueada;
- `source_specificity_blockers`: manifesto que impede aprovacao/publicacao quando o termo ainda precisa fonte primaria, norma especifica ou recorte juridico;
- `source_specificity_resolutions`: manifesto de fontes especificas resolvidas para pre-publicacao, ainda sem URL publica;
- `prepublication_gates`: contrato SEO/crawl para rota candidata finita, com render, sitemap e publicacao bloqueados;
- `legal_editorial_reviews`: revisao juridico-editorial bloqueada com CTA WhatsApp contextual em rascunho;
- `scalable_content_batches`: lotes massivos de intenções únicas e rascunhos autorais, bloqueados quando houver spam, template ou score humano insuficiente;
- `human_content_score`: score de naturalidade/IA-like/mecânico para revisão algorítmica, reescrita e auditoria;
- `batch_drafts`: rascunhos de amostra por lote massivo, com score e reescrita comprovada, ainda sem render, sitemap ou publicacao;
- `batch_draft_expansion_archive`: arquivo permanente bloqueado de rascunhos validados em laboratorio, preservado para expansao futura ate prova contraria;
- `batch_candidate_expansion_readiness`: prontidao bloqueada de expansao por familia, conectando 600 rascunhos permanentes a alvos de 30+ candidatos e diferenciando paid-intent ausente, paid-intent existente mas reprovado, fonte ampla, proximo gate pendente e flags publicas;
- `batch_candidate_gates`: gate permanente de candidatos selecionados do arquivo, agora expansivel por `paidintent.AllowsExpansion` para 510 intenções internas, ainda sem render, sitemap, publicacao ou `public_path`;
- `batch_candidate_reviews`: revisao juridico-editorial bloqueada de cada candidato de lote selecionado, com fonte matricial auditada e CTA WhatsApp de origem rastreavel;
- `batch_prepublication_gates`: pre-publicacao bloqueada de cada candidato revisado, com canonical oficial, title/meta, `noindex` e fonte final ainda pendente;
- `batch_source_specificity_resolutions`: resolucao de fonte por candidato pre-publicado, marcando URL especifica auditada ou bloqueio explicito por fonte ampla, sem liberar render/sitemap/publicacao;
- `batch_public_manifest_gates`: manifesto publico bloqueado por candidato, permitindo SEO/conteudo pendente apenas quando a fonte esta travada e mantendo fonte ampla bloqueada;
- `batch_final_authorial_drafts`: rascunhos autorais finais bloqueados, com fonte travada, score humano/naturalidade, CTA contextual e intenção comercial paga, ainda sem render, sitemap ou publicação;
- `batch_paid_intent_gates`: gate bloqueado de intencao comercial paga por rascunho final ou alvo de expansao, com `gate_scope`, `paid_signals` do corpo, `cta_paid_signals` do CTA, status comercial especifico para CTA-only/curiosidade/gratuidade/autoatendimento e status previdenciario informativo restrito a `batch-previdenciario-digital`;
- `batch_paid_intent_refinements`: ledger bloqueado de refinamentos em lote, registrando status original, status apos reescrita, campo alterado, sinal pago no corpo, score humano e flags publicas falsas;
- `batch_generation_metrics`: métricas agregadas de geração/refino por lote, provando volume, reescrita, score e similaridade sem criar URL pública;
- `batch_source_matrix`: matriz de fontes oficiais por subtema, usada como referência/proveniência sem scraping e sem publicação;
- `batch_source_url_audits`: auditoria das URLs da matriz, separada da camada editorial, exigindo cobertura de cada URL por `matrix_id` antes de escalar rascunhos;

Ferramentas operacionais de escala: `./tools/expand-batch-candidate-gates` recalcula a seleção current sem salto implícito; `./tools/advance-batch-candidate-gates` promove explicitamente o `next_candidate_target` planejado pela estratégia; `./tools/refresh-batch-candidate-pipeline` propaga a seleção para revisões, pre-publicação, fonte específica, manifesto e rascunhos finais bloqueados. As ferramentas devem ser idempotentes e não podem publicar: elas travam fonte por matriz quando existe URL oficial específica auditada, geram rascunhos finais bloqueados para manifests elegíveis, preservam apenas rascunhos antigos que ainda passam score/qualidade e mantêm os demais candidatos em `final_source_blocked_needs_specific_url`.
- `published_manifest`: manifesto leve de conteudo aprovado, sem substituir o renderizador.

Camada operacional fora do banco de conteúdo:
- `.agents/agent_context_ledger.jsonl`: registra subagentes por ciclo, ID real, escopo, status, política de uso, evidência, riscos, decisão de integração, validação obrigatória do Codex principal e fechamento antes do checkpoint. Não é conteúdo jurídico, não autoriza publicação e não substitui validação do Codex principal.

Regra P0: termos podem iniciar `draft_only`; nenhuma linha do banco vira pagina indexavel sem fonte, revisao, qualidade, SEO, intencao unica e checkpoint.

## Laboratorio

Toda mudanca P0/P1 deve passar por ciclo de laboratorio: escrever ou ajustar teste, rodar validacao, refinar, testar novamente e inspecionar artefatos. `tools/lab-cycle` combina `go test -count=1 ./...`, `go run ./cmd/check all`, build, auditoria de dependencias, diff check e busca por residuos Python, sem duplicar checks individuais ja cobertos.

O laboratorio pode gerar primeiro em `/tmp`, mas resultado validado, juridicamente util e reutilizavel deve ser trazido para o repo como camada permanente bloqueada. Isso evita depender de contexto compactado ou diretorio temporario para continuar a fabrica de conteudo, sem confundir rascunho aprovado em laboratorio com pagina publicada.

Antes de commit, o laboratório deve registrar autocrítica: hipótese do ciclo, prova obtida, riscos, melhorias possíveis, motivo pelo qual o commit é apenas checkpoint e próximo ciclo planejado. Essa autocrítica impede commit tratado como conclusão do projeto massivo.
