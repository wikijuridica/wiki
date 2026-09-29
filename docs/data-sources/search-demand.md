# Pesquisa de demanda e intencao

Esta camada nao e fonte juridica oficial. Ela existe para priorizar termos pesquisados por humanos antes de criar conteudo, sem scraping e sem publicar automaticamente.

## Google Trends Explore

- nome da fonte: Google Trends Explore
- orgao/servico: Google
- URL base segura: `https://trends.google.com.br/trends/explore?geo=BR`
- documentacao pesquisada: `https://support.google.com/trends/answer/4359550?hl=pt-BR`
- FAQ pesquisada: `https://support.google.com/trends/answer/4365533?hl=pt-br`
- uso permitido: comparar ate 5 grupos de termos por vez como sinal direcional de demanda humana no Brasil
- uso proibido: scraping, automacao agressiva, afirmar volume absoluto, publicar conteudo so porque um termo subiu
- interpretacao: dados sao amostrados, anonimizados, agregados e normalizados; servem como um ponto de dados entre outros
- estrategia de cache: registrar apenas URL de comparacao, data de consulta e decisao editorial; nao coletar massa de dados
- status: seguro como evidencia direcional, bloqueado para ingestao automatica

## Google Search Central sobre Trends

- nome da fonte: Google Search Central - Get started with Google Trends
- URL: `https://developers.google.com/search/docs/monitor-debug/trends-start`
- uso permitido: orientar estrategia de conteudo e priorizacao quando o termo fizer sentido para o publico do projeto
- regra critica: nao escrever sobre algo apenas porque esta em tendencia; o tema precisa ser util, confiavel, humano e alinhado ao publico
- status: seguro como contrato metodologico

## Fontes oficiais juridicas para lastro

Demanda nao substitui fonte juridica. Cada candidato de termo deve apontar tambem para fonte oficial ou institucional adequada, por exemplo:

- CNJ/Datajud: `https://datajud-wiki.cnj.jus.br/api-publica/acesso/`
- Dados Abertos da Camara: `https://dadosabertos.camara.leg.br/swagger/api.html`
- Dados Abertos do STJ: `https://dadosabertos.web.stj.jus.br/`
- e-Notariado/CNJ: `https://www.cnj.jus.br/e-notariado-completa-tres-anos-com-mais-de-15-milhao-de-atos-online/`
- INSS/gov.br: `https://www.gov.br/inss/`
- ANS/gov.br: `https://www.gov.br/ans/`
- MJ/Senacon/gov.br: `https://www.gov.br/mj/`

Essas fontes servem para proveniencia, contexto e pesquisa juridica. Nao autorizam clonagem, espelhamento ou geracao mecanica de paginas.
