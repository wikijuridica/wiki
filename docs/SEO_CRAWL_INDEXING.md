# SEO_CRAWL_INDEXING.md

## Indexacao

Toda pagina com `status != published` ou `index_policy != index` recebe `noindex,follow` e fica fora de sitemap.

Toda pagina indexavel precisa de:
- HTML textual completo;
- HTML leve, sem JavaScript, sem runtime cliente, sem hidratacao, sem bundle e sem CSS inline excessivo;
- `<title>` unico;
- meta description unica;
- canonical absoluto;
- meta robots `index,follow`;
- links internos rastreaveis;
- conteudo util e nao duplicado.

## URL base oficial

O dominio oficial do projeto e `wikijuridica.com.br`. `content/site.json` deve declarar `base_url="https://wikijuridica.com.br"`, `base_url_mode="official_configured"`, `official_url_status="locked"` e `official_url_locked=true`.

Os testes devem validar base HTTPS, canonical absoluto, path limpo e correspondencia entre canonical e rota usando a base configurada. Eles nao devem voltar a `portal-juridico.example` nem hardcodar dominio de laboratorio no algoritmo. URL oficial travada nao libera publicacao: candidatos e pre-publicacao continuam `noindex` e bloqueados ate fonte, revisao, qualidade, CTA, sitemap e manifesto publico finito passarem.

## Orcamento de HTML

Pagina publica indexavel deve ser facil de rastrear. O contrato atual reprova HTML publico acima de 50 KB, `<script>`, referencias `.js/.mjs/.wasm`, `modulepreload`, import maps, payloads de framework e marcadores de hidratacao. O objetivo e manter o primeiro response barato, textual e previsivel para Googlebot, OAI-SearchBot e bots valiosos.

## Orcamento de Search appearance

Pesquisa oficial feita em 2026-06-09 na Central da Pesquisa Google:
- Conteudo util para pessoas: `https://developers.google.com/search/docs/fundamentals/creating-helpful-content`
- Crawling e indexing: `https://developers.google.com/search/docs/crawling-indexing`
- Canonicalizacao: `https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls`
- Robots meta: `https://developers.google.com/search/docs/crawling-indexing/robots-meta-tag`
- Links de titulo: `https://developers.google.com/search/docs/appearance/title-link?hl=pt-BR`
- Metadescricoes/snippets: `https://developers.google.com/search/docs/appearance/snippet?hl=pt-br`
- Requisitos tecnicos minimos: `https://developers.google.com/search/docs/essentials/technical`

Pontos contratuais:
- o Google informa que nao ha limite fixo oficial para tamanho de `<title>`;
- o Google informa que nao ha limite fixo oficial para metadescricoes;
- ambos podem ser truncados conforme a largura do dispositivo;
- Googlebot precisa conseguir acessar a pagina, receber HTTP 200 e encontrar conteudo indexavel que nao viole politicas de spam.
- conteudo criado principalmente para manipular ranking, raso, mecanico, massificado ou sem valor adicional deve ser tratado como falha editorial antes de virar URL.

Decisao do projeto: usar orcamento conservador interno, contado por caracteres Unicode, para reduzir truncagem e texto ruim:
- `title`: 20 a 65 caracteres Unicode;
- metadescricao: 70 a 160 caracteres Unicode;
- `max-snippet:160` em paginas indexaveis.

Quando nao houver dados atuais suficientes sobre Googlebot, snippets, links de titulo, metadados ou superficie de busca, o agente deve pesquisar a Central da Pesquisa Google no dia da sessao e registrar a fonte consultada.

## Crawl

`content/crawl_policy.json` configura bots. Googlebot, OAI-SearchBot, GPTBot e regra geral sao separados. Busca interna e parametros ficam bloqueados em robots.txt e tambem usam `noindex` quando renderizados.

Robots.txt sozinho nao e usado como substituto de `noindex`.
