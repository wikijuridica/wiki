# ROADMAP_P0_P5.md

## P0/P1 rastreado no ciclo 1

- Base Go sem dependencias externas.
- Contrato proibindo Next.js e exigindo geracao on demand propria.
- Contrato de continuidade: checkpoint nao encerra trabalho.
- Plano finito para no minimo 10 mil paginas futuras, bloqueadas para publicacao durante P0.
- Pipeline de lotes massivos para centenas de milhares ou milhões de páginas possíveis, sem spam, com intenção única, score humano e CTA contextual.
- Politica propria de CTA WhatsApp, subordinada a fonte e revisao.
- Laboratorio multi-validacao em `tools/lab-cycle`.
- HTML textual completo.
- Canonical, robots, sitemap index e sitemap particionado.
- Busca interna noindex.
- Gates de qualidade e duplicidade.
- Scripts locais obrigatorios.

## Proximos ciclos

- Continuar P0: fortalecer arquitetura de escala, cache, roteamento, validadores e auditoria antes de publicar conteudo juridico em escala.
- Continuar P0: expandir `scalable_content_batches` e `human_content_score` para sair de rascunho unitário, gerar lotes de drafts em massa e validar reescrita automática antes de qualquer publicação.
- Continuar P0: ampliar `batchdraftgen` de 5 amostras por família para centenas e depois milhares por família, medindo similaridade, fonte específica por subtema, CTA contextual e custo de laboratório.
- Continuar P0: transformar as métricas de `batch_generation_metrics` em gate para pré-publicação bloqueada por lote, sem publicar até fonte específica, revisão jurídica, SEO/crawl e render leve passarem.
- Continuar P0: expandir `batch_source_matrix` com URLs oficiais por subtema, revisão de robots e estratégia sem scraping para cada família de alta intenção digital.
- Continuar P0: usar `advance-batch-candidate-gates`, `expand-batch-candidate-gates` e `refresh-batch-candidate-pipeline` para subir candidatos bloqueados de 510 para 590 e depois milhares, preservando paid-intent comercial nas cinco famílias, lane previdenciária informativa bloqueada, fonte específica, revisão, CTA contextual e flags públicas falsas.
- Continuar P0: reduzir os 66 candidatos ainda bloqueados por fonte ampla pesquisando URLs oficiais específicas por subtema, começando por trabalhista/TST, consumidor financeiro/BCB e família que ainda depende só de fonte geral.
- Continuar P0: transformar autocrítica pré-commit e validação massiva em contrato testável, para impedir commit tratado como conclusão do `/goal`.
- Continuar P0 em laboratorio: validar, refinar, testar novamente e inspecionar artefatos.
- P2: ampliar contratos de tipos de pagina e fontes oficiais.
- P3: medir concorrencia, cache e geracao incremental para alto volume.
- P4: historico editorial persistente.
- P5: busca propria, grafo juridico e recomendacoes internas.
