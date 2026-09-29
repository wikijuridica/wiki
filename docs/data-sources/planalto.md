# Portal da Legislacao / Planalto

- nome da fonte: Portal da Legislacao / Planalto
- orgao: Presidencia da Republica / Governo Federal
- URL base oficial confirmada: `https://legislacao.presidencia.gov.br/`
- URL de atos: `https://legislacao.presidencia.gov.br/atos/?...`
- pagina publica confiavel usada para confirmacao: `https://www.gov.br/pt-br/servicos/pesquisa-de-legislacao-portal-da-legislacao`
- URL historica de texto normativo vinculada a atos: `https://www.planalto.gov.br/ccivil_03/`
- URL historica alternativa indicada por pagina de multiplas escolhas: `https://www.planalto.gov.br/ccivil_03.old/`
- tipo de dado: legislacao federal e paginas oficiais relacionadas
- formato: HTML e outros formatos oficiais conforme pagina
- atualizacao: portal de pesquisa oficial informa acesso a atos normativos desde 1808; ingestao continua bloqueada
- licenca/termo: pagina gov.br de servico localizou o canal oficial, mas nao aprova coleta automatica
- robots.txt: auditoria local em `https://legislacao.presidencia.gov.br/robots.txt` teve timeout; `https://www.planalto.gov.br/robots.txt` resetou conexao; `https://www4.planalto.gov.br/robots.txt` tambem teve timeout
- limites: nao definidos neste ciclo
- campos disponiveis: pendente de mapeamento
- riscos: alteracao de HTML, termos de uso, duplicidade normativa, citacao desatualizada
- estrategia de cache: somente apos auditoria de termos e campos
- estrategia de proveniencia: registrar URL oficial, data de verificacao, tipo de dado e hash de conteudo quando ingerido
- estrategia de deduplicacao: canonical por identificador normativo e hash normalizado

Estado: URL oficial confirmada por fonte publica gov.br; auditoria HTTP local ainda bloqueia ingestao automatica.

## Auditoria de ancoras especificas — 2026-06-09

Para o lote P0, artigos especificos de CLT, Codigo Civil e CDC podem ser registrados como URL de referencia bloqueada quando:

- a URL permanece no dominio oficial `www.planalto.gov.br`;
- o uso fica `reference_only_no_scraping_no_ingestion`;
- a matriz tambem preserva a fonte ampla original;
- o audit URL-level registra hash, robots, termos e que nao houve coleta de conteudo;
- render, sitemap, publicacao e `public_path` continuam bloqueados.

No ciclo 45, a tentativa HTTP local/escalonada para a CLT compilada teve timeout, e a pagina aberta por ferramenta web retornou multipla escolha apontando o caminho `ccivil_03.old`. Por isso, as ancoras especificas foram registradas apenas como referencia oficial bloqueada, nunca como ingestao automatica ou texto copiado.
