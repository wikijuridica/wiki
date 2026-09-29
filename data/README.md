# Banco leve do projeto

Este diretorio guarda o banco leve proprio do portal em arquivos JSONL separados por finalidade.

Regras:
- `data/terms/` guarda sementes de termos juridicos, nunca conteudo publico.
- `data/source-audit/` guarda auditoria de robots, termos de uso e alcance de fontes.
- `data/source-snapshots/` guarda apenas snapshots autorizados, pequenos e com hash.
- `data/editorial/` guarda rascunhos e manifesto editorial, separados do material oficial.
- Nada aqui deve virar pagina indexavel sem passar por fonte, revisao, qualidade, SEO e checkpoint.

