# CNJ Datajud

- nome da fonte: API Publica Datajud
- orgao: Conselho Nacional de Justica
- URL base: `https://api-publica.datajud.cnj.jus.br/`
- documentacao pesquisada: `https://datajud-wiki.cnj.jus.br/api-publica/acesso/`
- tipo de dado: metadados de processos publicos dos tribunais brasileiros
- formato: API publica com chave publica indicada na documentacao do CNJ
- atualizacao: pendente de auditoria por tribunal/alias
- licenca/termo: pendente de auditoria do termo de uso e restricoes de dados pessoais/processos sigilosos
- robots.txt: nao substitui termo de API; pendente de verificacao operacional
- limites: pendente de medicao; paginação e filtros precisam de laboratorio proprio
- campos disponiveis: numeroProcesso, classe, assuntos, orgao julgador, movimentacoes e metadados conforme wiki
- riscos: privacidade, LGPD, processos sigilosos, dados pessoais, golpes e uso comercial indevido
- estrategia de cache: cache minimo, com expurgo e logs de proveniencia; sem publicar dados sensiveis
- estrategia de proveniencia: tribunal/alias, query, data, chave publica vigente e hash do retorno
- estrategia de deduplicacao: numero CNJ e tribunal/alias

Estado: candidata oficial sensivel; ingestao bloqueada.
