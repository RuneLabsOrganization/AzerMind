# Dungeons & Raids: plano de renovação da interface

Status: implementação autorizada; etapas 1–5 verificadas automaticamente; integração em andamento.

## Objetivo e limites

Aplicar o padrão visual de Items à lista de instâncias e aos detalhes de bosses
e loot. Facilitar encontrar uma instância, consultar suas referências e anexar
uma seleção ao rascunho. Preservar as mudanças existentes em Items, Settings,
guias, compositor, foco do chat e leilão.

Escopo principal: addon/AzerMind/Instances.lua, testes específicos e documentação.
Reutilizar componentes existentes de Theme; qualquer alteração compartilhada
precisa de justificativa e regressões das telas já concluídas. Não substituir
InstancesData, inventar expansões/dificuldades ou modificar SavedVariables.

## Diagnóstico confirmado no código

- Lista de 680x620 e detalhes de 820x620 ainda usam Surface(window), sem a
  composição de moldura e painéis adotada no novo Items.
- Na lista, títulos/metadados com largura 490 dividem a linha com Details:
  reservar áreas distintas evita sobreposição e cortes pouco claros.
- Contagem de resultados e seleção estão misturadas; ações de seleção e
  paginação competem na parte inferior.
- Show selected only aplica primeiro os filtros normais: seleções escondidas
  continuam inacessíveis nessa revisão.
- Busca atual aceita uma sequência literal em nome, localização e bosses;
  não pesquisa loot. Perto do nível usa margem de cinco níveis e é referência,
  não confirmação de acesso.
- Detalhes já oferecem bosses, loot, NPC, mapa, cópia e retorno à lista.
  Melhorar sua apresentação sem remover essas funções.

Diagnóstico visual real ainda pendente: o código não comprova aparência no WoW.

## Direção recomendada

1. Apenas trocar bordas: rápido, mas mantém os problemas de navegação.
2. **Lista e detalhes separados, ambos renovados (recomendado):** mantém uma
   lista legível e dá espaço para bosses e loot, preservando o fluxo atual.
3. Tudo em três colunas: facilita consulta contínua, mas comprime nomes e
   controles em telas menores. Não adotar nesta iteração.

Lista: dimensão inicial semelhante a Items, ajustada aos limites reais da tela.
Cabeçalho com título e fechar; navegação lateral com Todos/Dungeons/Raids e
contagens; busca e filtros no topo; resultados ao centro; paginação separada;
seleção e anexação no rodapé fixo. Mobs & NPCs fica como ação secundária.
Não criar uma árvore de expansões ou regiões sem dados confiáveis.

Detalhes: moldura correspondente; título, nível e localização compactos;
bosses à esquerda e loot à direita, com busca e paginações independentes.
Fonte e ressalvas em uma área discreta de informações, consultável sem
poluir cada linha. Créditos obrigatórios continuam disponíveis.

## Etapas e critérios de aceite

### 1. Estrutura visual da lista

- [ ] Aplicar moldura, fundo, margens, tipografia e botões do novo padrão.
- [ ] Reservar regiões de navegação, busca, lista, paginação e rodapé.
- [ ] Uma única ação de fechar; adaptar escala sem ocultar o rodapé.
- [ ] Manter nomes longos fora da área de Details e checkbox.

Verificação: limites e sobreposições em 900x600, 1280x720 e 1920x1080;
nomes/localizações longos; inspeção no WoW de bordas, fontes e camadas.

### 2. Busca e filtros

- [ ] Todos/Dungeons/Raids com contagens e seleção evidente.
- [ ] Localização em seletor; Perto do meu nível com explicação da margem.
- [ ] Busca por palavras em instância, boss, localização e loot registrado;
  identificar por que um resultado foi encontrado quando for boss ou loot.
- [ ] Mostrar filtros ativos e permitir limpar um ou todos sem apagar seleção.
- [ ] Estado sem resultados com recuperação; informar nível indisponível sem
  apresentar resultados como recomendação personalizada.
- [ ] Ordenação previsível por nome e nível de referência, desconhecidos ao fim.

Verificação: filtros combinados, acentos/nomes reais conforme dados existentes,
busca de boss e item, nível ausente, reset e nenhum resultado. Não baixar dados
ou depender de GetItemInfo para a busca básica de loot registrado.

### 3. Resultados e navegação

- [ ] Nome em destaque; tipo, nível e localização em linha secundária curta.
- [ ] Checkbox e realce de seleção; Details com área independente.
- [ ] Contagem de resultados separada de intervalo/página e total selecionado.
- [ ] Tooltip revela texto cortado e instruções úteis.
- [ ] Abrir detalhes e voltar preserva busca, filtros, página e seleção.
- [ ] Reutilizar linhas; abrir fonte/NPC não altera seleção acidentalmente.

Verificação: extremos da paginação, poucos/muitos resultados, estabilidade da
ordenação, clique esquerdo/direito e percurso de ida/volta para detalhes.

### 4. Tela de bosses e loot

- [ ] Aplicar a mesma moldura e hierarquia visual da lista.
- [ ] Lista de bosses com seleção clara; distinguir Trash Drops quando houver.
- [ ] Loot com ícone, nome e metadados realmente disponíveis; sem exigir cache
  carregado nem inventar atributos, taxas de drop ou elegibilidade.
- [ ] Busca de loot com estado vazio, contagem e paginação legíveis.
- [ ] Separar Entrada no mapa, NPC, Copiar resumo e Voltar conforme contexto.
- [ ] Sem coordenadas/ID/loot: explicar o dado ausente e desabilitar só a ação
  que depende dele. Mostrar fonte e caráter parcial da referência.

Verificação: mais de sete bosses e mais de cinco loots, ausência de bosses,
boss sem loot, Trash Drops sem NPC, item sem ID e metadados não carregados.
Preservar tooltips, menus de item, mapa e retorno; nenhuma ação de /target.

### 5. Seleção e anexação

- [ ] Agrupar Selecionar página, Selecionar resultados e Limpar seleção.
- [ ] Revisar selecionados mostra toda a seleção, mesmo fora dos filtros;
  ao sair, restaura a navegação anterior.
- [ ] Informar quantos selecionados estão escondidos pelos filtros.
- [ ] Continuar anexando links compactos de instância, sem despejar resumo,
  bosses e loot automaticamente no compositor.
- [ ] Preservar vínculo com a conversa original, bloqueio de conversa trocada
  e comportamento de inserir no rascunho sem enviar mensagem.
- [ ] Conferir limites existentes do rascunho; se necessário, aplicar limite
  explícito e atômico, com mensagem clara e sem truncar silenciosamente.

Verificação: seleção em várias páginas/filtros, revisão, reset, conversa
trocada, seleção indisponível e conteúdo exato anexado; nenhum envio automático.

### 6. Revisão integrada e entrega

- [ ] Executar o percurso lista > busca > detalhes > boss > loot > voltar >
  revisar selecionados > anexar, incluindo Dungeon e Raid.
- [ ] Rodar testes específicos após cada etapa e corrigir falhas antes da próxima.
- [ ] Rodar suíte completa, verificações Lua e diff final ao integrar.
- [ ] Conferir que o diff não alterou as telas concluídas anteriormente.
- [ ] Registrar testes no fluxo normal/release e atualizar documentação de uso.
- [ ] Na implementação autorizada: commits por etapa, instalação com backup,
  conferência dos arquivos e push sem sobrescrever alterações remotas.
- [ ] Conferir capturas reais no WoW: lista, filtros, nomes longos, detalhes,
  loot vazio e rodapé em tamanho menor. Registrar pendências separadamente.

## Regra de acompanhamento

Cada etapa registra mudança, resultado dos testes e pendências antes de avançar.
Testes de geometria e comportamento não aprovam texturas, fontes ou camadas.
Não declarar aprovação visual sem evidência real do cliente. Implementação autorizada pelo usuário em 2026-11-16.


## Registro de implementação

- Etapa 1: moldura/leather, navegação lateral, rodapé e áreas separadas. Os 13
  testes de referência/refinamento passaram antes de avançar.
- Etapas 2–3: palavras, boss/loot, filtros individuais, ordenação e paginação.
  Os sete testes de compatibilidade de instâncias passaram. A busca por loot
  amplia resultados de palavras genéricas; teste de faixa de nível agora usa
  o nome completo Wailing Caverns para identificar a instância sem ambiguidade.
- Etapa 4: detalhes com moldura correspondente, informações de fonte separadas,
  controles de mapa/NPC preservados e títulos fora da área das ações.
- Etapa 5: revisão independente dos filtros, restauração da página, contagem de
  ocultos, reconciliação de indisponíveis e limite atômico de 36.000 bytes.
- Cinco novas regressões passaram: busca/filtros, revisão/anexação, ordenação,
  geometria/reuso de linhas, telas suportadas e ausência de metadados/IDs.
- Revisão visual no cliente: PENDENTE. Não confundir geometria simulada com
  aprovação de texturas, fontes e camadas no WoW.
