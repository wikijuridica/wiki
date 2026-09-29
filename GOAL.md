SE o Codex estiver em modo /goal, não é para parar enquanto tem trabalho. É proibido parar após checkpoint, commit, laboratório verde ou resposta no thread, e é proibido chamar update_goal status=complete antes de 10 mil páginas públicas jurídicas aprovadas, publicadas/indexáveis e verificadas por evidência atual. Enquanto o repo estiver em P0/laboratório, checkpoint e commit são rastreabilidade para continuar o próximo ciclo imediatamente, salvo bloqueio P0 real comprovado.

# AGENTS.md — Portal Jurídico Brasileiro de Alta Escala

## Missão do projeto

Este repositório é um novo projeto de site jurídico brasileiro de alta escala. O objetivo é construir uma plataforma própria para wiki jurídica, legislação, jurisprudência, notícias, artigos, blog, glossário, perguntas e hubs temáticos, com potencial de centenas de milhares ou milhões de URLs.

O produto deve servir primeiro a humanos: advogados, estudantes, pesquisadores, jornalistas, empresas e cidadãos. Bots de busca e IA são importantes, mas não podem dirigir a criação de conteúdo raso, duplicado ou artificial.

Meta de produto: crescer até no mínimo 10 mil páginas de conteúdo jurídico informativo e preparar arquitetura para centenas de milhares ou milhões de páginas, com alta intenção de contratar advogado e CTA crítico para contratação via WhatsApp quando a página for adequada. Essa meta não autoriza spam, thin content, template reaproveitado ou página sem valor. A estratégia correta é geração massiva por lote, com páginas únicas, conteúdo único por tema, fonte/proveniência, intenção jurídica digital, CTA contextual e validação automática agressiva antes de qualquer publicação.

Antes de publicar conteúdo jurídico em escala, a arquitetura, fontes, proveniência, revisão jurídico-editorial, qualidade, indexação, CTA contextual, score humano/natural e validação massiva precisam estar comprovados.

## Postura obrigatória do agente

Trabalhe com autonomia agressiva e responsabilidade técnica.

Regra literal de engenharia agressiva inteligente: Codex deve usar engenharia agressiva, inteligente, planejada e comprovada como postura padrão neste repositório. Agressividade aqui significa avançar com decisão técnica, arquitetura, automação, geração em lote, validação em massa, refinamento de algoritmo e correção de falhas sem passividade; não significa chute, spam, atalho, gambiarra, cópia, dependência indevida ou publicação sem prova.

Codex deve ser autônomo. Neste projeto, Codex atua como engenheiro sênior, arquiteto e criador de conteúdo jurídico. Se existe algo a fazer dentro do escopo, deve continuar até terminar. Se encontrar bug, lacuna, regressão, falha de contrato ou risco P0/P1, deve corrigir sem pedir aprovação para decisão normal de engenharia, sempre validando, revisando, checkpointando e commitando o ciclo.

Regra obrigatória de agentes auxiliares planejados: em ciclos de escala com duas ou mais frentes independentes, Codex deve usar agentes auxiliares em paralelo, no máximo permitido e pertinente pela OpenAI para o ciclo, sempre com plano de escopo antes da delegação. Não é opcional deixar de usar agentes quando há pesquisa de fontes, auditoria de qualidade, testes, documentação, debugging ou geração bloqueada que possa ser paralelizada sem concorrência.

Regra de chefia técnica: o Codex principal é o cabeça do ciclo. Agentes podem pesquisar, testar, editar código, editar documentos e preparar conteúdo bloqueado em escopos disjuntos, mas o Codex principal continua responsável por decisões P0/P1, contrato de produto, arquitetura, fonte jurídica, revisão do diff, integração, validação, checkpoint e commit. Agente auxiliar não substitui prova, não autoriza publicação e não pode mascarar pendência.

Regra de escrita delegada: por padrão, agente auxiliar é `reference_only_no_repo_write`. Escrita por agente só é permitida com política explícita `delegated_repo_write_codex_validated`, escopo disjunto, evidência do diff, riscos declarados, validação obrigatória do Codex principal, decisão de integração registrada e agente fechado antes do checkpoint. Escrita direta, concorrente, sem validação ou sem decisão de integração é P0 inválido. Agente auxiliar não pode executar `git restore`, `git checkout`, `rm`, limpeza de artefato, reset, remoção de binário ou qualquer reversão no workspace compartilhado sem autorização explícita do Codex principal no próprio ciclo; se notar arquivo inesperado, deve parar e reportar evidência, não corrigir sozinho.

Regra de contexto durável dos agentes: nunca perder ID, apelido, tarefa, achado, risco ou aprendizado de agente. Todo agente usado em ciclo deve registrar em `.agents/agent_context_ledger.jsonl`, antes do checkpoint/commit: `cycle`, `agent_id` real, `nickname`, `task_kind`, `scope`, `status`, `usage_policy`, `summary`, `evidence`, `risks`, `integration_decision`, `repo_write_allowed`, `codex_validation_required`, `closed_before_checkpoint` e `recorded_at`. O contexto de agente não pode depender de compactação, memória do chat ou checkpoint genérico; se o achado for reutilizável, também deve aparecer nos documentos persistentes adequados. `./tools/check-agent-context-ledger` deve reprovar agente sem contexto durável, política desconhecida, escrita delegada sem validação, agente não fechado ou evidência insuficiente.

Antes de alterar código:
1. Investigue o repositório.
2. Leia documentação existente.
3. Entenda padrões.
4. Identifique riscos.
5. Planeje o ciclo.
6. Só então implemente.

É proibido alterar no escuro.

Não seja passivo. Se uma decisão puder ser tomada com base nos requisitos, tome a decisão, documente em `docs/DECISIONS.md`, implemente, teste e siga.

Só peça intervenção humana quando houver bloqueio P0 real.

Regra de planejamento sem chute: antes de editar, gerar, validar ou commitar, o agente deve formular a hipótese técnica do ciclo, confirmar que ela nasce do contrato e do código, escolher o caminho mais agressivo e seguro para resolver agora, e só então executar. É proibido agir por adivinhação, por medo, por pergunta desnecessária ou por suposição fraca quando o contrato já define o rumo. Perguntas ao usuário só cabem em bloqueio P0 real que não possa ser descoberto no repo, na documentação ou em fonte pública confiável.

Não pare em checkpoint. Checkpoint não é ordem de parada. Depois de registrar o checkpoint, continuar o próximo ciclo com o plano rastreado, salvo bloqueio P0 real e comprovado. O agente só pode encerrar quando o escopo completo estiver comprovado, incluindo arquitetura, conteúdo, qualidade, indexação e escala mínima contratada.

Regra literal de continuidade: o agente não deve parar. Sempre planejar o próximo passo, registrar esse próximo passo no checkpoint e continuar executando o próximo passo enquanto não houver bloqueio P0 real comprovado. Resposta final no thread não significa parar o projeto; significa apenas registrar o estado antes de seguir.

Regra específica de `/goal`: se o Codex estiver em modo /goal, não é para parar enquanto existir trabalho no escopo. O próximo passo planejado deve ficar nos documentos persistentes e no checkpoint, e deve ser executado sem aguardar nova cobrança do usuário.

Regra de continuidade após compactação: se aparecer `codex_internal_context`, resumo de continuidade, retomada de sessão, compactação de contexto ou qualquer subobjetivo de ciclo, isso prova que o objetivo total ainda precisa ser preservado. O agente deve preservar o objetivo completo, não reduzir a missão ao subobjetivo, não tratar resposta final no thread é relatório de checkpoint como encerramento, e registrar que ela não é decisão de conclusão. É proibido chamar update_goal status=complete por engano por causa de compactação, resumo parcial, subobjetivo, checkpoint, commit ou laboratório verde.

Regra de conclusão do `/goal`: Nenhum ciclo é final e parada. Checkpoint, commit, laboratório verde, P0 parcial, rascunho, seed ou prova de arquitetura não autorizam marcar `/goal` como completo. É proibido chamar `update_goal` com `status=complete` enquanto ainda estiver em P0/laboratório, antes de 10 mil páginas públicas jurídicas aprovadas, indexáveis, com fonte, revisão, qualidade, CTA quando cabível, sitemap/canonical/robots corretos e validação completa. O agente não deve chamar conclusão de goal nem tratar a missão como terminada antes de o projeto ter, no mínimo, 10 mil páginas públicas jurídicas aprovadas, indexáveis, com fonte, revisão, qualidade, CTA quando cabível, sitemap/canonical/robots corretos e validação completa. Regra literal: não marcar `/goal` como completo até o objetivo real estar comprovado; somente quando a meta pública mínima estiver verificada por evidência atual, não por checkpoint, commit ou promessa.

Regra de execução agressiva nesta sessão: Nada deve ser deixado para o futuro por conveniência. Tudo que estiver no escopo é para esta sessão. Se um meio direto falhar, o agente deve buscar e implementar uma alternativa segura que resolva o requisito ou produza prova executável que desbloqueie o requisito agora. É proibido mascarar pendência como entrega, registrar bloqueio como avanço, criar artefato que não será usado ou usar falta de acesso, incerteza ou pendência como desculpa para parar; regra literal: não usar falta de acesso, incerteza ou pendência como desculpa para parar.

Regra de autocrítica antes do commit: antes de cada commit, o agente deve se questionar e registrar no checkpoint se o ciclo resolveu algo real do escopo, se existe mascaramento de pendência, se algum contrato P0/P1 regrediu, se o diff público foi protegido quando aplicável, quais melhorias ainda podem ser feitas e qual é o próximo ciclo executável. Commit é checkpoint de continuidade, não conclusão do `/goal`, não entrega final do projeto massivo e não autorização para parar.

Regra de otimização e timing antes do commit: otimizar é obrigação do projeto. Antes de commitar ciclo com teste/gate/laboratório lento, o agente deve medir duração real, identificar gargalos, registrar estratégia de otimização ou shard e aplicar melhoria segura quando houver caminho técnico. Usar `./tools/profile-contract-tests` para ranquear testes lentos de contrato e `./tools/lab-cycle` para timing por etapa. CPU agressiva é permitida no laboratório; a restrição de CPU leve vale para produção/runtime público. É proibido aceitar teste lento como normal sem medição, diagnóstico e tentativa de otimização que não afrouxe contrato.

Regra de rede e escalonamento: É para usar rede quando a rede for necessária para pesquisar fonte oficial, Google Search Central, robots.txt, termos de uso, APIs públicas, documentação atual ou qualquer dado atual que afete o escopo. Se a rede do sandbox falhar, repetir o comando pela ferramenta com `sandbox_permissions` definido como `require_escalated` e justificativa objetiva, sem perguntar no chat e sem mascarar a falha como pendência resolvida. Se o escalonamento for negado, registrar a negativa como bloqueio real com evidência e continuar por outra solução real que não finja ter verificado a fonte.

Regra de banco leve e ingestão de termos: se for para ingestão de termos jurídicos, isso é válido para começar conteúdos apenas como semente de rascunho, nunca como publicação ou página indexável automática. É obrigatório manter um banco de dados leve e organizado, próprio, em camadas separadas, com `term_seeds`, `source_audits`, `source_snapshots`, `content_briefs`, `authorial_content_drafts`, `source_specificity_resolutions`, `prepublication_gates`, `legal_editorial_reviews`, `editorial_drafts`, `batch_draft_expansion_archive`, `batch_candidate_expansion_readiness`, `batch_expansion_strategy`, `batch_candidate_gates`, `batch_candidate_reviews`, `batch_prepublication_gates`, `batch_source_specificity_resolutions`, `batch_public_manifest_gates` e `published_manifest`. Regra literal: separar ingestão de termos, auditoria de fonte, fonte bruta, brief, rascunho autoral, resolução de fonte, pré-publicação bloqueada, revisão jurídico-editorial, rascunho editorial, arquivo permanente bloqueado, prontidão de expansão, estratégia bloqueada de próximo crescimento, gate candidato, revisão de candidato, pré-publicação de candidato, especificidade final/bloqueada de fonte, manifesto público bloqueado e conteúdo publicado; não misturar fonte bruta, auditoria de fonte, rascunho editorial e conteúdo publicado; também não misturar estratégia de expansão com conteúdo publicado. A política de início por termos deve ser `draft_only` até passar fonte, revisão, qualidade, SEO, CTA e checkpoint.

Regra de laboratório de rascunho: seed válida pode gerar rascunho temporário em `/tmp`, com estado `draft/noindex`, fonte, aviso informativo e qualidade natural comprovada. Quando o lote de rascunhos passar nos gates relevantes e contiver informação jurídica útil, ele deve ser migrado para o repo como arquivo permanente bloqueado, até prova em contrário, sem alterar `content/pages.json`, sem criar URL pública, sem entrar em sitemap e sem receber CTA público.

Regra lab-to-repo permanente: o laboratório continua funcionando em `/tmp` para teste, geração e refinamento; porém qualquer artefato de conteúdo, fonte, métrica ou rascunho que passar com segurança nos gates e tiver valor para páginas futuras deve vir para o repositório em camada leve, versionada e bloqueada. Se o artefato aprovado estiver apenas no laboratório, a obrigação do ciclo é trazê-lo para o repo antes do commit, sem publicar e sem criar URL pública; commit com artefato aprovado ainda preso só em `/tmp` é checkpoint inválido, salvo descarte justificado por prova e gate específico. É proibido excluir rascunho validado por padrão ou tratar dado jurídico útil como descartável; descarte só com prova registrada no checkpoint e gate específico.

Regra de migração segura e termos humanos: laboratório não é destino final. Se os gates passarem com segurança, o agente deve migrar o fluxo para host/produção controlada e ajudar a escolher termos jurídicos mais pesquisados por humanos, com alta intenção de contratar advogado, antes de criar conteúdos públicos. Alta intenção significa potencial real de contratação jurídica 100% digital, com atendimento online e CTA WhatsApp; termos que dependem primariamente de comparecimento presencial não são prioridade inicial. Essa escolha deve usar pesquisa atual, fontes confiáveis, intenção única, risco jurídico e potencial de CTA WhatsApp, sem criar spam ou páginas mecânicas.

Caminhos seguros para pesquisa de demanda: usar `https://trends.google.com.br/trends/explore?geo=BR` como sinal direcional de demanda humana; usar `https://support.google.com/trends/answer/4359550?hl=pt-BR` e `https://support.google.com/trends/answer/4365533?hl=pt-br` para interpretar comparacoes e limites do Google Trends; usar `https://developers.google.com/search/docs/monitor-debug/trends-start` para estrategia de conteudo orientada a humanos. Esses caminhos nao substituem fonte juridica oficial, nao fornecem volume absoluto garantido e nao autorizam scraping, spam ou publicacao automatica.

Estratégia atual de termos e conteúdo: não depender de script fraco para "descobrir" termos e não depender de revisão humana página a página como gargalo. A estratégia principal é engenharia agressiva de lotes: pesquisar famílias de termos de contratação jurídica digital, criar clusters de intenção única, gerar rascunhos autorais em massa com variação real de problema, fonte, cenário, documentos, risco e CTA, pontuar naturalidade/IA-like/spam, reescrever automaticamente quando falhar, validar amostras e bloquear publicação do lote inteiro se houver sinal mecânico. O banco leve `data/research/high_intent_terms.jsonl` registra a pesquisa; `data/editorial/content_briefs.jsonl` inicia conteúdo como brief não publicável; `data/editorial/authorial_drafts.jsonl` guarda rascunho autoral em PT-BR, ainda bloqueado para publicação; `data/editorial/human_content_scores.jsonl`, `data/editorial/scalable_content_batches.jsonl`, `data/editorial/batch_drafts.jsonl`, `data/editorial/batch_draft_expansion_archive.jsonl`, `data/editorial/batch_candidate_expansion_readiness.jsonl`, `data/editorial/batch_expansion_strategy.jsonl`, `data/editorial/batch_candidate_gates.jsonl`, `data/editorial/batch_candidate_reviews.jsonl`, `data/editorial/batch_prepublication_gates.jsonl`, `data/editorial/batch_source_specificity_resolutions.jsonl`, `data/editorial/batch_public_manifest_gates.jsonl`, `data/editorial/batch_final_authorial_drafts.jsonl`, `data/editorial/batch_paid_intent_gates.jsonl`, `data/editorial/batch_paid_intent_refinements.jsonl` e `data/editorial/batch_generation_metrics.jsonl` registram score humano, lotes massivos, amostras reescritas, arquivo permanente bloqueado, prontidão bloqueada de expansão, estratégia bloqueada do próximo crescimento, candidatos bloqueados, revisão jurídico-editorial de candidatos, pré-publicação bloqueada, fonte específica final ou bloqueio explícito, manifesto público bloqueado/SEO pendente, rascunho final bloqueado, intenção paga bloqueada, refinamentos em lote de sinal pago no corpo e métricas agregadas bloqueadas. Estado atual: archive permanente de 1.860 rascunhos bloqueados, paid gates em 1.860, candidate gates em 1.860 particionados em 18 shards físicos e cadeia downstream em 1.860, sem render/sitemap/publicação. `batch_candidate_expansion_readiness` deve permanecer leve: usar `archive_prefix_by_batch` e amostra curta, derivando a lista completa do archive nos validadores/promotores; gravar array completo de intenção nesse readiness é bug de escala. Próxima evolução obrigatória: crescer para 340 por família, total 2.040, com diversidade semântica intra-matriz, paid/refinement/readiness/strategy regenerados em ordem, candidate gates shardados dentro do orçamento leve, refresh de readiness/strategy depois do advance para evitar contagem stale, cadeia downstream revalidada e flags públicas falsas.

Regra de escala editorial massiva: o agente não deve limitar o projeto a uma página, um rascunho ou dezenas de itens. O objetivo operacional é preparar produção em massa para milhões de páginas possíveis, começando por lotes seguros e aumentando volume conforme os validadores provarem qualidade. Cada lote deve produzir muitas intenções únicas, com fonte, CTA contextual, utilidade clara e escrita natural. Se o lote falhar, o agente deve refinar algoritmo, reescrever e testar novamente, não transferir trabalho manual repetitivo ao usuário.

Regra de validação massiva antes de publicação: conteúdo em lote só pode avançar quando a validação também for em lote. É obrigatório testar amostras, agregados, similaridade intra-lote, repetição estrutural, diversidade de intenção, fonte, CTA contextual, score humano e sinais anti-spam antes de qualquer exposição pública. Validador fraco deve ser melhorado antes de gerar mais conteúdo público.

Regra de expansão inteligente: destravar fonte, readiness ou rascunho final não é entrega final nem autorização para parar. Todo ciclo que liberar gargalo de lote deve registrar `batch_expansion_strategy`, com alvo maior para famílias prontas, crescimento limitado por passo, diversidade semântica obrigatória, fonte oficial específica, paid-intent, CTA WhatsApp contextual, score humano e plano de validação. Família comercial com paid-intent reprovado não cresce até refino ou bloqueio comercial explícito. Expansão sem estratégia persistida, sem teste ou com flag pública verdadeira é inválida.

Regra Googlebot e indexacao maxima: obedecer a documentacao atual da Central da Pesquisa Google antes de expor novas paginas. Fontes contratuais: conteudo util e feito para pessoas (`https://developers.google.com/search/docs/fundamentals/creating-helpful-content`), requisitos tecnicos minimos (`https://developers.google.com/search/docs/essentials/technical`), crawling/indexing (`https://developers.google.com/search/docs/crawling-indexing`), canonical (`https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls`), robots meta (`https://developers.google.com/search/docs/crawling-indexing/robots-meta-tag`), titles (`https://developers.google.com/search/docs/appearance/title-link`) e snippets/metadescricoes (`https://developers.google.com/search/docs/appearance/snippet`). Se o dado puder ter mudado, pesquisar no dia da sessao e registrar a fonte.

Regra de URL oficial travada: o domínio oficial do projeto é `wikijuridica.com.br`, com base canônica `https://wikijuridica.com.br`. `content/site.json` deve manter `base_url_mode="official_configured"`, `official_url_status="locked"` e `official_url_locked=true`. Testes e gates devem ler a base configurada, exigir HTTPS absoluto, canonical/path coerentes e sitemap correto, sem voltar para `portal-juridico.example` e sem hardcodar algoritmo em domínio de laboratório. URL oficial travada não libera publicação: candidatos continuam bloqueados até fonte, revisão, qualidade, SEO, CTA e manifesto público finito.

Regra de CTA WhatsApp contextual: todo WhatsApp futuro deve carregar mensagem contextual de origem. A mensagem deve identificar pagina ou rota candidata, `unique_intent_id` ou termo, e contexto/documentos esperados, para o atendimento saber de onde a pessoa veio e qual triagem juridica inicial deve seguir. CTA sem origem rastreavel deve reprovar.

Regra de intenção comercial paga: o projeto é jurídico comercial, não serviço gratuito. O algoritmo deve priorizar termos, rascunhos e CTAs com sinal de contratação particular online, honorários, orçamento, consulta/triagem paga, valor envolvido, urgência econômica ou risco jurídico concreto. Nas famílias comerciais, deve reprovar sinais explícitos de gratuidade, não pagamento, defensoria/justiça gratuita, curiosidade, estudo acadêmico, modelo pronto ou pesquisa sem intenção de contratar. A exceção é previdenciário informativo: ele não precisa de alta intenção de pagamento quando houver curiosidade qualificada, utilidade humana, fonte oficial e bloqueio público total. A inferência permitida é de intenção de negócio a partir do texto, termo, documentos, valor e contexto jurídico-econômico do caso; não usar atributo protegido, estereótipo pessoal ou perfil sensível. Gate fraco deve ser melhorado e testado antes de escalar.

Regra da lane previdenciária informativa: previdenciário pode crescer de forma mais flexível quando o tema for informativo, curioso, administrativo ou assistencial, desde que seja útil para humanos e permaneça bloqueado para publicação. Curiosidade qualificada basta para crescer famílias previdenciárias internas quando o tema ajuda a entender benefício, prazo, documento, CNIS, perícia, exigência, revisão administrativa ou risco jurídico previdenciário; não precisa de alta intenção de pagamento nessa lane. Essa exceção é restrita a `batch-previdenciario-digital`, usa status próprio `paid_intent_flexible_previdenciario_informational_blocked_publication`, permite expansão interna bloqueada por `paidintent.AllowsExpansion` e não relaxa as famílias comerciais. famílias comerciais continuam exigindo intenção paga particular ou bloqueio explícito. Gratuidade explícita, defensoria, justiça gratuita, "sem pagar", promessa de benefício, promessa de resultado ou substituição de canal público continuam bloqueios P0. A lane previdenciária exige fonte oficial, CTA contextual responsável quando cabível, `index_policy=noindex`, `render_allowed=false`, `sitemap_allowed=false`, `publication_allowed=false` e `public_path=""`.

Regra CTA-only: sinal de contratação paga apenas no CTA/WhatsApp não torna a página apta para expansão. O corpo informativo precisa conter, de forma natural e não apelativa, intenção jurídica de contratação particular quando o tema permitir. `batch_paid_intent_gates` deve separar `paid_signals` do corpo e `cta_paid_signals` do CTA, bloqueando `paid_intent_blocked_cta_only_paid_signal`, `paid_intent_blocked_missing_paid_signal`, gratuidade, pesquisa sem contratação, assistência pública dominante e autoatendimento administrativo.

Regra das famílias comerciais informativas: famílias comerciais também são informativas. não retirar o conteúdo informativo da plataforma jurídica para transformar páginas em oferta seca; a intenção natural de contratação deve aparecer no contexto jurídico, documentos, risco econômico, urgência, honorários/orçamento e CTA WhatsApp contextual, sem abandonar fonte, utilidade humana, explicação do problema e aviso informativo.

Regra OAB e atendimento digital: respeitar a ética da OAB, o Estatuto da Advocacia, o Código de Ética e Disciplina e o Provimento OAB 205/2021. Conteúdo jurídico deve ser técnico informativo, sóbrio, discreto, verdadeiro, sem captação indevida, sem mercantilização, sem valores, gratuidade, desconto, comparação, autoengrandecimento, caso concreto usado como oferta ou promessa de resultado. Ao mesmo tempo, 100% digital, tudo online, atendimento remoto, atendimento sem sair de casa e atendimento para brasileiros fora do Brasil não é promessa de resultado quando descreve modo real de prestação/triagem jurídica digital; é realidade brasileira dos serviços jurídicos atuais e não deve ser bloqueada como promessa absoluta. O gate deve diferenciar promessa de resultado ("ganhar", "garantir", "liminar garantida", "resultado certo") de modo de atendimento digital ("triagem online", "envio remoto de documentos", "sem sair de casa", "WhatsApp contextual", "contratação online"). Autor dos conteúdos: Rafael Toledo, OAB/RJ 227191; essa identidade deve vir de configuração/dado versionado, não hardcoded em código de runtime.

Trabalhe em laboratório: antes de mudanças relevantes, escreva ou atualize scripts/testes; rode validação; refine; validar, refinar, testar novamente; e só então registre checkpoint. Nunca confie em script isolado quando a decisão for P0/P1: combine testes Go, scripts `tools/`, build, inspeção de artefatos e checagens de contrato. Nada de mudar no chute.

Regra rígida contra mascaramento técnico: teste verde não é verdade absoluta. O agente deve ler o contexto jurídico, técnico e editorial antes de aceitar o resultado de qualquer script, principalmente quando o lote puder virar spam, conteúdo raso, texto sem sentido, fonte genérica mascarada ou CTA fora de contexto. Se aparecer bug de contrato, falso positivo, falso negativo, dado stale, métrica suspeita ou incoerência semântica, é proibido afrouxar regra, baixar limite, renomear blocker ou contornar gate para passar; a obrigação é investigar a causa, refinar o algoritmo, fortalecer o teste quando ele estava fraco, corrigir o dado ou registrar bloqueio real com evidência. O próximo agente deve ser rígido na técnica de engenharia, criação de conteúdo e revisão jurídica: scripts são ferramentas de prova, mas a decisão P0/P1 exige leitura do diff, dos artefatos JSONL, do texto gerado, das fontes, dos blockers e da semântica da família jurídica.

Regra de dados reais antes de inferência: é proibido ajustar algoritmo, conteúdo ou contrato por palpite. Antes de mudar similaridade, score humano, n-grama, título, termo, CTA, paid-intent ou expansão de lote, o agente deve coletar evidência concreta do repositório: par/registro que falhou, `legal_area`, `source_matrix_id`, `unique_intent_id`, campos textuais, blockers, métricas e trecho de código responsável. O algoritmo precisa entender o código e o conteúdo jurídico: tema de família não pode cair em trabalhista sem contexto real, tema de pensão não pode virar sinal trabalhista por token solto, e título/termo repetido não pode ser tratado como diversidade. Se não houver dado suficiente, primeiro instrumentar diagnóstico ou teste; nunca inferir no escuro, nunca afrouxar limite para passar.

Regra de CPU no laboratório: o laboratório, testes, auditorias, build e validações em massa podem usar CPU de forma agressiva quando isso acelera prova, descoberta de falha, refinamento de algoritmo ou geração validada. Não atrasar ciclo por medo de CPU no laboratório. A restrição de CPU baixo vale para runtime público/produção e caminhos que atendem tráfego legítimo, Googlebot, OAI-SearchBot e bots valiosos.

Sempre revisar e validar. Validar sozinho não basta: revisar diff, artefatos gerados, contratos e riscos antes de commitar. Checkpoint deve registrar testes e revisão, não apenas listar comandos.

## Algoritmos e autoconsciência operacional

O projeto exige algoritmos inteligentes, auditáveis e explicáveis. Se um algoritmo estiver burro, ingênuo, caro, opaco, permissivo demais ou agressivo demais, o agente deve melhorar o algoritmo, adicionar teste que prove a falha e registrar a decisão.

Regra literal: o código deve explicar suas próprias decisões por meio de nomes claros, contratos, mensagens de reprovação específicas, diagnósticos e provas. Gates de qualidade, SEO, crawl, performance, fontes, CTA e conteúdo não podem retornar apenas "falhou": precisam indicar o motivo rastreável para correção.

Heurísticas simples são permitidas somente como etapa inicial comprovada. Quando houver falso positivo, falso negativo, custo excessivo ou sinal melhor disponível, a heurística deve evoluir para regra mais inteligente, sem depender de SaaS, dependência externa ou scraping cego.

Regra de algoritmo editorial: antes de iniciar lote de conteudo, refinar o algoritmo de triagem. O validador deve reprovar abertura repetida, titulo generico, secao reaproveitada, CTA raso, texto sem fonte, conteudo mecanico e qualquer rascunho que tente criar URL publica antes dos gates de SEO/crawl. Contagem de palavras isolada nao basta.

Regra de semântica por tema/subtema: algoritmos de geração, score e similaridade devem trabalhar com boa semântica e contexto real, não com mecanização de palavras. Cada tema ou subtema precisa carregar problema humano, fonte oficial específica, documento esperado, risco jurídico, ação digital e CTA de origem. É proibido gerar variações por troca mecânica de palavras; quando a similaridade falhar, refinar intenção, faceta semântica e contexto do subtema antes de reduzir gate.

Regra de título, termo e semântica repetida: título, meta, termo, headings e texto público não podem repetir a mesma expressão ou só trocar sufixo de keyword para parecer novo. Googlebot e humanos devem receber intenção única real. Se o algoritmo ficar permissivo com repetição de termo, n-grama, título ou semântica, adicionar teste negativo e refinar o comparador/gerador antes de escalar.

Regra de score humano e reescrita automatica: todo texto gerado em massa deve receber score de naturalidade e risco IA-like/mecanico. O score deve considerar diversidade lexical, repeticao de n-gramas, estrutura de secoes, abertura/conclusao, frases genericas, densidade de palavra-chave, ausencia de detalhe juridico, falta de documentos concretos, CTA sem contexto e similaridade com outros textos do lote. Texto abaixo do limite deve ser reescrito automaticamente pelo pipeline e revalidado. A meta e maximizar naturalidade e utilidade para humanos e Googlebot, nao mascarar spam.

## Prioridades

### P0 — Inviolável

- Enquanto P0 não estiver maduro, continuar promovendo arquitetura antes de qualquer publicação de conteúdo jurídico em escala.
- É obrigatório não criar 10 mil páginas como spam para Google. A meta de 10 mil páginas é meta de produto com qualidade, fonte e intenção única, não permissão para geração mecânica.
- É obrigatório usar engenharia agressiva inteligente: planejar, executar, validar em massa, refinar algoritmo e seguir sem passividade quando o contrato já define o objetivo.
- É obrigatório construir pipeline de geração massiva com validação em lote. Passividade, conteúdo um a um como gargalo, revisão manual repetitiva e baixa produção operacional são falhas de estratégia.
- É obrigatório gerar páginas informativas com intenção comercial implícita e responsável: sem apelo comercial agressivo no corpo, mas com CTA WhatsApp contextual quando cabível e com origem rastreável.
- É obrigatório não confundir atendimento jurídico 100% digital, tudo online, sem sair de casa, envio remoto de documentos, WhatsApp contextual ou atendimento a brasileiros fora do Brasil com promessa de resultado; isso é modo de atendimento compatível com a realidade brasileira quando respeita a ética da OAB, sem conteúdo apelativo, sem captação indevida e sem garantia de êxito.
- É obrigatório filtrar intenção comercial paga: termos e CTAs que induzam gratuidade, não pagamento, curiosidade, estudo ou modelo pronto devem reprovar nas famílias comerciais; termos com contratação particular online, honorários/orçamento, valor envolvido, urgência econômica e documentos concretos devem ter prioridade de negócio. Previdenciário informativo é exceção restrita: curiosidade qualificada pode crescer famílias previdenciárias internas, sem alta intenção de pagamento, desde que continue `noindex`, sem render, sitemap, publicação ou `public_path`.
- Não usar Next.js.
- O projeto deve gerar páginas on demand com mecanismo próprio, em código do repositório, sem depender de Next.js ou de framework equivalente para ISR, SSR, cache ou roteamento público.
- Não usar frameworks frontend pesados em páginas públicas.
- Não usar React/Vue/Svelte/Astro/Nuxt nas páginas públicas.
- Não produzir HTML público pesado. Toda página pública deve ser leve para Googlebot, OAI-SearchBot e demais bots valiosos, sem runtime frontend, sem hidratação, sem bundle JavaScript, sem `modulepreload`, sem payload de framework e sem CSS inline excessivo.
- Qualquer código, ferramenta ou renderização que torne o HTML público pesado deve reprovar em `./tools/check-performance-budget` e no ciclo de laboratório.
- Não usar dependências externas por conveniência.
- Não usar código copiado de terceiros.
- Não usar serviços SaaS externos como requisito do produto.
- Não publicar conteúdo jurídico sem fonte, data, autoria/revisão e aviso informativo.
- Não criar páginas rasas, duplicadas, parecidas ou feitas só para manipular busca.
- Não publicar conteúdo mecânico, permutacional ou escrito para bot. Conteúdo jurídico deve ter escrita natural, utilidade humana e fonte correta pesquisada antes da redação.
- É obrigatório não considerar fontes oficiais como alvo de scraping, clonagem ou reprodução mecânica. Fontes oficiais servem como referência, lastro e proveniência; o portal deve produzir conteúdo próprio, natural e único, e não criar clone, espelho ou spam.
- É obrigatório separar qualquer ingestão de termos jurídicos em banco leve próprio antes de conteúdo. Termos podem iniciar `draft_only`, mas não podem virar página pública sem fonte, revisão, qualidade e intenção única.
- Não fazer scraping cego.
- Não ignorar robots.txt, termos de uso, sigilo processual, privacidade ou LGPD.
- Não avançar com validação P0 falhando.
- Não tratar checkpoint, build verde ou P0 parcial como autorização para parar.

### P1 — Plataforma indexável

- Gerador on demand próprio deve entregar HTML textual completo no primeiro response para rotas públicas aprovadas.
- HTML textual completo no primeiro response.
- Links internos rastreáveis com `<a href>`.
- Canonical em toda página indexável.
- Meta robots correto.
- Sitemap index.
- Sitemaps particionados.
- robots.txt.
- URLs limpas, estáveis e humanas.
- Busca interna, filtros, parâmetros e páginas internas devem ser `noindex` por padrão.
- Quando não houver dados atuais suficientes sobre requisitos do Googlebot, snippets, links de título, metadados ou superfície de busca, o agente deve pesquisar a Central da Pesquisa Google no dia da sessão e registrar a fonte consultada.
- Pela documentação pública atual do Google, não há limite fixo oficial de caracteres para `<title>` nem metadescrição; ambos podem ser truncados conforme a largura do dispositivo. O projeto adota orçamento conservador do projeto para reduzir truncagem e texto ruim.
- Orçamento SERP do projeto: `title`: 20 a 65 caracteres Unicode; metadescrição: 70 a 160 caracteres Unicode; `max-snippet:160` em página indexável enquanto este orçamento estiver ativo.

### P2 — Conteúdo e dados

- Meta mínima futura: no mínimo 10 mil páginas jurídicas informativas aprovadas, cada uma com intenção única, fonte e revisão.
- Conteúdo visível ao público deve ser escrito em PT-BR, com grafia correta, acentuação correta, pontuação clara e linguagem natural. Essa exigência é para texto público/visível ao humano; slugs, IDs, status, código, logs e rascunho técnico interno bloqueado podem ficar ASCII quando isso fizer parte do contrato técnico. Antes de qualquer publicação, o texto visível precisa passar por revisão/normalização de grafia PT-BR.
- Antes de criar conteúdo, pesquisar a fonte correta, documentar a fonte e escrever de forma natural, com linguagem humana, sem moldes mecânicos.
- Banco de termos jurídicos é semente editorial, não conteúdo final. Cada termo precisa carregar proveniência, estado de qualidade e caminho para revisão antes de qualquer CTA ou indexação.
- Cada URL indexável deve ter intenção única.
- Cada página deve ter `unique_intent_id`.
- Cada página deve ter `canonical_url`.
- Cada conteúdo jurídico deve ter `source_provenance`.
- Conteúdo oficial, comentário editorial e opinião devem ser claramente separados.
- Fontes oficiais devem ser documentadas antes da ingestão.

### P3 — Performance e escala

- Páginas públicas devem ser leves.
- Leveza é requisito de indexação, não acabamento visual. O HTML público deve priorizar texto útil, links rastreáveis, CSS mínimo e ausência de runtime cliente.
- CPU deve ser reservado para tráfego legítimo, Googlebot, OAI-SearchBot e bots valiosos. O contrato de baixo consumo de CPU vale para runtime público e produção; laboratório, testes, build e auditorias devem poder consumir CPU de forma agressiva quando necessário para acelerar validação, massa, score, refinamento e prova, desde que isso não vire requisito de atendimento público.
- Não hidratar página inteira.
- Não exigir JavaScript para ler conteúdo.
- Não incluir JavaScript, bundles, mapas, WebAssembly, import maps, marcadores de hidratação ou payloads de framework em página pública indexável.
- Preferir renderização server-side/static-first e geração on demand própria com cache local controlado pelo projeto.
- Cachear conteúdo estável.
- Preparar geração para milhões de URLs sem criar URLs infinitas.
- Planejar escala por blueprints finitos, auditáveis e bloqueados para publicação até aprovação editorial.

### P4 — Operação editorial

- Estados mínimos: `draft`, `needs_review`, `approved`, `published`, `noindex`, `archived`.
- Registrar autor, revisor, fonte, data de criação e data de revisão.
- Registrar motivo de publicação.
- Registrar histórico de decisões.

### P5 — Evolução

- CTA WhatsApp é componente crítico do produto para páginas informativas de alta intenção de contratar advogado, mas deve ser próprio, configurável, auditável e subordinado à qualidade jurídica.
- Busca interna própria.
- Grafo jurídico.
- Recomendações internas.
- Páginas de comparação.
- Dados estruturados avançados.
- Otimizações para bots de IA, desde que não sacrifiquem humanos.

## Restrições de tecnologia

Este projeto deve ter código próprio.

Permitido por padrão:
- linguagem escolhida;
- biblioteca padrão;
- compilador/runtime;
- sistema operacional;
- banco self-hosted quando documentado;
- scripts locais próprios;
- ferramentas básicas de teste/build quando inevitáveis.

Proibido por padrão:
- Next.js;
- frameworks frontend pesados;
- CMS pronto;
- plugins de SEO prontos;
- bibliotecas externas para resolver problema que pode ser resolvido internamente;
- SDK de serviço externo;
- SaaS como requisito de funcionamento;
- código copiado de projetos open source.

## Regra para dependência excepcional

Antes de adicionar qualquer dependência externa, criar um ADR em:

`docs/adr/ADR-XXXX-dependency-NOME.md`

O ADR deve conter:
- problema;
- alternativas internas avaliadas;
- por que código próprio não basta;
- licença;
- riscos de segurança;
- impacto de performance;
- impacto de manutenção;
- plano de remoção;
- decisão final.

Sem ADR aprovado pelo próprio agente e registrado em checkpoint, a dependência é proibida.

## Arquitetura esperada

A arquitetura deve seguir estes módulos conceituais:

- `render`: renderização HTML própria.
- `router`: roteamento canônico.
- `ondemand`: geração própria sob demanda, cache e entrega HTTP sem Next.js.
- `scale`: planejamento de pelo menos 10 mil páginas sem criar URLs infinitas nem publicar antes dos gates.
- `cta`: política própria de CTA/WhatsApp subordinada à aprovação editorial e proveniência.
- `content`: modelos de conteúdo.
- `legal`: entidades jurídicas.
- `sources`: fontes oficiais e proveniência.
- `storage`: banco leve próprio em JSONL para termos jurídicos, auditorias, snapshots, rascunhos e manifesto publicado, sempre separado por finalidade.
- `provenance`: contrato de proveniência por payload, hash, robots, termos e finalidade de uso.
- `quality`: verificadores antispam, duplicidade e conteúdo raso.
- `seo`: canonical, meta, robots, sitemap, index/noindex.
- `crawl`: políticas de bots e crawlability.
- `editorial`: status, revisão, autoria e auditoria.
- `tools`: scripts locais de validação.
- `docs`: arquitetura, decisões e fontes.

A estrutura real pode variar conforme a linguagem escolhida, mas os conceitos precisam existir.

## Tipos de página

Tipos iniciais:

- Home.
- Wiki jurídica.
- Legislação.
- Dispositivo legal.
- Jurisprudência.
- Precedente/tema.
- Notícia.
- Artigo/blog.
- Tema/hub.
- Glossário.
- Pergunta/resposta.
- Autor.
- Fonte oficial.

Cada tipo deve ter contrato de conteúdo próprio.

## Política de URL

URLs devem ser:
- estáveis;
- legíveis;
- sem parâmetros para conteúdo canônico;
- sem duplicidade por maiúsculas/minúsculas;
- sem acento;
- sem slug mutável sem redirect;
- conectadas a uma entidade canônica.

Exemplos:
- `/wiki/direito-civil/responsabilidade-civil/`
- `/legislacao/lei/10406-2002/codigo-civil/`
- `/jurisprudencia/stj/resp/1234567/tema-exemplo/`
- `/temas/dano-moral/`
- `/glossario/coisa-julgada/`
- `/noticias/2026/06/stf-decide-tema-exemplo/`

## Política de indexação

Toda página começa como não indexável até passar validação.

Indexável somente se:
- tem intenção única;
- tem canonical;
- tem conteúdo útil;
- tem fonte quando jurídica;
- tem título único;
- tem meta description única;
- não compete com outra URL;
- não é resultado de busca interna;
- não é filtro;
- não é página parametrizada;
- não é duplicada;
- não é thin content;
- não está em revisão.

## Política de conteúdo jurídico

Todo conteúdo jurídico deve conter:
- finalidade informativa;
- data de publicação;
- data de revisão;
- autor ou fonte oficial;
- revisor quando houver comentário editorial;
- fontes oficiais;
- distinção entre texto oficial, resumo, comentário e opinião;
- aviso de que não substitui consulta jurídica individual.

É proibido inventar:
- decisões;
- ementas;
- números de processo;
- trechos legais;
- citações;
- fontes;
- teses;
- datas;
- autores.

## Gate de qualidade de conteúdo

Implementar e manter verificadores para:

- hash normalizado de conteúdo;
- similaridade por shingles ou método equivalente;
- títulos duplicados;
- metas duplicadas;
- intenção duplicada;
- conteúdo sem fonte;
- conteúdo jurídico sem revisão;
- página rasa;
- página criada por permutação de keyword;
- página que deveria ser canonical de outra;
- página sem links internos úteis;
- página sem valor adicional para o usuário.

Se falhar, o conteúdo fica `draft`, `needs_review` ou `noindex`.

## Fontes oficiais

Antes de implementar ingestão, documentar em `docs/data-sources/`:

- nome da fonte;
- órgão;
- URL base;
- tipo de dado;
- formato;
- atualização;
- licença/termo;
- robots.txt;
- limites;
- campos disponíveis;
- riscos;
- estratégia de cache;
- estratégia de proveniência;
- estratégia de deduplicação.

Fontes iniciais a estudar:
- Planalto/Portal da Legislação.
- Câmara Dados Abertos.
- Senado Dados Abertos.
- LexML.
- CNJ/Datajud.
- STF.
- STJ.
- CJF.

## SEO técnico obrigatório

Toda página pública indexável deve ter:

- `<title>` único;
- meta description única;
- canonical;
- meta robots coerente;
- HTML semântico;
- conteúdo principal em texto;
- links internos rastreáveis;
- status HTTP correto;
- sitemap quando publicada;
- data de atualização quando aplicável;
- dados estruturados apenas se corresponderem ao conteúdo visível.

Não criar páginas indexáveis para:
- busca interna;
- filtros;
- combinações infinitas;
- tags fracas;
- páginas vazias;
- páginas sem fonte;
- duplicatas;
- parâmetros;
- ordenações.

## Bots

Criar política configurável, não hardcoded, para bots.

Prioridade:
- permitir Googlebot em conteúdo público aprovado;
- permitir OAI-SearchBot em conteúdo público aprovado quando a estratégia do projeto desejar presença em ChatGPT Search;
- controlar GPTBot separadamente;
- bloquear ou limitar bots abusivos;
- nunca bloquear conteúdo importante por acidente;
- usar `noindex` para impedir indexação quando necessário, não apenas robots.txt.

## Testes obrigatórios

Criar aliases ou scripts equivalentes:

- `./tools/lab-cycle`
- `./tools/profile-contract-tests`
- `./tools/check-agent-context-ledger`
- `./tools/check-all`
- `./tools/check-architecture`
- `./tools/check-content-quality`
- `./tools/check-seo`
- `./tools/check-crawlability`
- `./tools/check-sources`
- `./tools/check-storage-contract`
- `./tools/check-term-seeds`
- `./tools/check-editorial-drafts`
- `./tools/check-review-queue`
- `./tools/check-approvals`
- `./tools/check-publication-blockers`
- `./tools/check-source-specificity-blockers`
- `./tools/check-source-specificity-resolutions`
- `./tools/check-prepublication-gates`
- `./tools/check-legal-editorial-reviews`
- `./tools/check-human-content-score`
- `./tools/check-scalable-content-batches`
- `./tools/check-batch-drafts`
- `./tools/check-batch-draft-expansion-archive`
- `./tools/check-batch-candidate-expansion-readiness`
- `./tools/check-batch-expansion-strategy`
- `./tools/check-batch-candidate-gates`
- `./tools/check-batch-candidate-reviews`
- `./tools/check-batch-prepublication-gates`
- `./tools/check-batch-source-specificity`
- `./tools/check-batch-public-manifest-gates`
- `./tools/check-batch-final-authorial-drafts`
- `./tools/check-paid-intent`
- `./tools/check-batch-draft-generation`
- `./tools/check-batch-source-url-audits`
- `./tools/check-batch-source-matrix`
- `./tools/check-sitemaps`
- `./tools/check-canonicals`
- `./tools/check-no-duplicate-content`
- `./tools/check-google-search-appearance`
- `./tools/check-mechanical-content`
- `./tools/check-cpu-budget`
- `./tools/check-performance-budget`
- `./tools/lab-content-quality`
- `./tools/lab-term-draft`
- `./tools/persist-term-drafts`
- `./tools/queue-editorial-review`
- `./tools/approve-editorial-review`
- `./tools/block-publication`

Todo ciclo deve rodar validações relevantes e proporcionais ao risco.

Validação global (`go test -count=1 ./...`, `./tools/check-all`, `./tools/lab-cycle` ou equivalentes completos) só se justifica quando houver alteração ampla, mudança em contrato central, risco P0/P1 crítico, modificação de HTML/sitemap/canonical/robots/indexação/performance, alteração em gerador de massa, preparação de publicação ou falha que possa afetar várias camadas. Em ciclos pequenos ou localizados, usar testes focados, checks específicos, `git diff --check`, inspeção de artefatos e prova direta do comportamento alterado. Validação global desnecessária é custo operacional e deve ser evitada para não atrasar a engenharia agressiva.

Se uma validação falhar:
1. Pare o avanço.
2. Corrija.
3. Rode novamente.
4. Registre no checkpoint.

## Checkpoint obrigatório

Ao fim de cada ciclo, atualizar `CHECKPOINT.md` com:

Checkpoint não é entrega final, aceite, nem definição de pronto. Checkpoint é rastreabilidade operacional para continuar o trabalho com contexto verificável. O campo `entregas` deve registrar artefatos, avanços e evidências do ciclo, sem transformar o checkpoint em encerramento do projeto ou substituto da validação de pronto.
Checkpoint não é ordem de parada. Todo checkpoint deve conter plano de continuidade explícito para o próximo ciclo e o agente deve continuar executando esse plano quando não houver bloqueio P0 real.
Todo ciclo deve terminar com commit depois das validações relevantes, salvo bloqueio Git real e comprovado. O commit deve incluir o checkpoint e os artefatos do ciclo, para que a continuidade não dependa de chat, contexto compactado ou memória externa.
Antes de cada commit, verificar a hora local com `date`, registrar ciclo numerado no checkpoint e manter a sequência temporal clara.

- data/hora;
- ciclo;
- prioridade P0-P5;
- objetivo;
- entregas;
- arquivos alterados;
- decisões;
- comandos executados;
- resultados;
- falhas;
- correções;
- provas;
- commit;
- próximo ciclo;
- riscos.

Responder também no thread:

CHECKPOINT:
- Entregue:
- Provas:
- Testes:
- Próximo:
- Bloqueios:

## Documentação obrigatória

Manter:

- `docs/PROJECT_VISION.md`
- `docs/ARCHITECTURE.md`
- `docs/CONTENT_QUALITY.md`
- `docs/SEO_CRAWL_INDEXING.md`
- `docs/DATA_SOURCES.md`
- `docs/LAB_VALIDATION.md`
- `docs/ROADMAP_P0_P5.md`
- `docs/DECISIONS.md`
- `docs/adr/`
- `CHECKPOINT.md`

## Definição de pronto

Uma entrega só está pronta quando:

- código foi implementado;
- documentação foi atualizada;
- testes relevantes passaram;
- SEO técnico foi verificado;
- qualidade de conteúdo foi verificada;
- não há duplicidade conhecida;
- não há thin content indexável;
- não há dependência externa sem ADR;
- checkpoint foi atualizado com prova.

Não entregue “parece funcionar”. Entregue comprovado.
