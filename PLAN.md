# Plano de Desenvolvimento — PoE Build Progression Overlay

## 1. Objetivo

Desenvolver uma aplicação desktop leve para Windows, escrita em Go, que exiba dentro do Path of Exile um guia de progressão de build.

A aplicação importará builds já criadas no Path of Building por meio de links do `pobb.in`, Pastebin compatível ou código PoB direto. O objetivo não é recriar o Path of Building, mas aproveitar as etapas já configuradas pelo autor da build para mostrar:

- progressão da árvore passiva;
- diferentes árvores salvas, como `Ato 1`, `Ato 2`, `Level 40` e `Endgame`;
- skill sets correspondentes a cada etapa;
- gemas adicionadas, removidas ou substituídas;
- visualização gráfica da árvore;
- progresso atual salvo localmente.

A interface permanecerá escondida e será exibida somente quando o usuário pressionar `Ctrl + B`.

---

## 2. Escopo inicial

### Incluído

- Aplicação residente na bandeja do Windows.
- Overlay aberta por `Ctrl + B`.
- Fechamento por `Ctrl + B` novamente ou `Esc`.
- Importação de links `pobb.in`.
- Importação de Pastebin compatível.
- Importação de código PoB direto.
- Extração das árvores e skill sets salvos na build.
- Preservação dos nomes e da ordem das etapas.
- Navegação entre etapas.
- Comparação entre etapa anterior e atual.
- Exibição gráfica da árvore passiva.
- Exibição das gemas e links de cada etapa.
- Armazenamento local da build processada.
- Atualização manual a partir do link original.
- Inicialização rápida sem novo download ou parsing.

### Fora do escopo

- Price Check.
- Live Search.
- Macros de chat.
- Tracker de currency.
- Cálculo de DPS.
- Simulação de itens.
- Cálculos defensivos.
- Motor de modificadores do PoB.
- Importação automática do personagem atual.
- Alteração ou criação de builds dentro do aplicativo.
- Reimplementação completa do Path of Building.

---

## 3. Stack tecnológica

| Área | Tecnologia |
|---|---|
| Linguagem | Go |
| Interface | Gio |
| Integração Windows | Win32 com `golang.org/x/sys/windows` |
| Banco local | SQLite |
| HTTP | `net/http` |
| Parsing XML | `encoding/xml` |
| Base64 | `encoding/base64` |
| Descompactação | `compress/zlib` |
| Configuração | SQLite |
| Plataforma inicial | Windows |

### Restrições

- Não usar Chromium, WebView ou Electron.
- Não usar React, Vue, Svelte, HTML ou JavaScript.
- Não renderizar enquanto a overlay estiver escondida.
- Não fazer download ao abrir uma build já importada.
- Não executar parsing pesado na thread da interface.
- Não depender do Path of Building instalado.

---

## 4. Comportamento da overlay

### Hotkey

```text
Ctrl + B
```

Comportamento:

```text
Overlay fechada + Ctrl+B → abre
Overlay aberta + Ctrl+B  → fecha
Overlay aberta + Esc     → fecha
```

O `Esc` somente será capturado enquanto a overlay estiver visível.

### Primeira abertura

```text
┌──────────────────────────────────────┐
│ Importar build                       │
│                                      │
│ [ Cole o link pobb.in ou Pastebin ]  │
│                                      │
│             [ Importar ]             │
└──────────────────────────────────────┘
```

### Aberturas seguintes

Se existir uma build ativa salva:

```text
Ctrl+B
→ carregar dados normalizados do SQLite
→ abrir diretamente na última etapa selecionada
```

Não realizar rede, descompactação ou parsing XML nesse fluxo.

---

## 5. Fluxo de importação

```text
Link ou código PoB
→ identificar a origem
→ baixar conteúdo bruto, quando necessário
→ extrair o código da build
→ decodificar Base64 URL-safe
→ descompactar zlib
→ interpretar XML
→ extrair árvores e skill sets
→ normalizar os dados
→ calcular diferenças entre etapas
→ preparar dados gráficos
→ salvar no SQLite
```

### Fontes aceitas

1. `https://pobb.in/<id>`
2. Pastebin compatível com exportação do PoB
3. Código PoB colado diretamente

A importação deve detectar automaticamente o tipo de entrada.

### Validações

- URL válida.
- Resposta HTTP válida.
- Conteúdo não vazio.
- Base64 válida.
- Conteúdo zlib válido.
- XML compatível.
- Pelo menos uma árvore ou um skill set disponível.

---

## 6. Modelo de progressão

O aplicativo deve preservar exatamente os nomes e a ordem das etapas criadas pelo autor da build.

Exemplos:

```text
Ato 1
Ato 2
Ato 3
Normal Lab
Level 50
Início dos mapas
Endgame
```

Não inventar progressão quando a build tiver apenas uma árvore final.

### Estrutura principal

```go
type Build struct {
    ID            string
    Name          string
    SourceType    string
    SourceURL     string
    SourceHash    string
    CurrentStage  string
    ImportedAt    time.Time
    UpdatedAt     time.Time
}

type BuildStage struct {
    ID             string
    BuildID        string
    Name           string
    Order          int
    CharacterLevel *int
    PassiveNodes   []int
    NewNodes       []int
    RemovedNodes   []int
    SkillGroups    []SkillGroup
    Notes          string
}

type SkillGroup struct {
    ID       string
    Label    string
    Enabled  bool
    Slot     string
    Gems     []Gem
}

type Gem struct {
    Name         string
    Level        int
    RequiredLevel *int
    Quality      int
    Enabled      bool
    IsSupport    bool
}
```

---

## 7. Relação entre árvores e skill sets

Uma build pode possuir várias árvores e vários skill sets. Nem sempre os nomes estarão perfeitamente associados.

A estratégia inicial será:

1. Preservar a ordem original das árvores.
2. Preservar a ordem original dos skill sets.
3. Associar por identificador quando o XML fornecer relação explícita.
4. Associar por nome quando houver nomes equivalentes.
5. Usar posição correspondente como fallback.
6. Permitir correção manual da associação na interface.

Exemplo:

```text
Árvore: Ato 3
Skill set: Ato 3
```

Caso não exista associação confiável, a interface deve informar isso sem inventar dados.

---

## 8. Interface principal

```text
┌──────────────────────────────────────────────┐
│ Build: Static Strike Slayer                  │
│ [←] Ato 3                         [Ato 4 →]   │
├──────────────────────────────────────────────┤
│                                              │
│       VISUALIZAÇÃO GRÁFICA DA ÁRVORE         │
│                                              │
├──────────────────────────────────────────────┤
│ Novos passivos nesta etapa                   │
│ • Heart of the Warrior                       │
│ • Barbarism                                  │
│ • Life Mastery                               │
├──────────────────────────────────────────────┤
│ Gemas                                        │
│ + Static Strike                              │
│ + Melee Physical Damage Support              │
│ - Ground Slam                                │
└──────────────────────────────────────────────┘
```

### Navegação

- Etapa anterior.
- Próxima etapa.
- Seleção direta em lista.
- Indicador da etapa atual.
- Última etapa escolhida persistida localmente.

---

## 9. Comparação entre etapas

Para cada etapa, pré-calcular a diferença em relação à anterior.

### Árvore

```text
Nós atuais - nós anteriores = nós novos
Nós anteriores - nós atuais = nós removidos
```

Mostrar:

- nós já existentes;
- nós adicionados na etapa atual;
- nós removidos, quando houver;
- caminho relevante até os novos nós.

### Gemas

Comparar:

- nova gema;
- gema removida;
- gema substituída;
- mudança de grupo;
- mudança de link;
- mudança da skill principal;
- mudança de ativação.

O aplicativo não deve interpretar se uma mudança é melhor ou pior. Apenas representar o que foi definido pelo autor da build.

---

## 10. Visualização gráfica da árvore

### Objetivo

Permitir que o jogador veja visualmente onde investir os próximos pontos, sem abrir o Path of Building.

### Dados necessários

- ID de cada nó.
- Posição do nó.
- Conexões entre nós.
- Grupo visual.
- Tipo do nó.
- Nome do nó.
- Informação de mastery.

Os dados estruturais da árvore devem ser armazenados localmente e versionados por versão do jogo.

### Estados visuais

| Estado | Representação |
|---|---|
| Nó da etapa anterior | discreto |
| Nó novo da etapa atual | destaque principal |
| Nó futuro | destaque secundário |
| Nó removido | marca específica |
| Mastery | símbolo diferenciado |
| Conexão ativa | linha destacada |

### Interações

- Zoom pelo scroll.
- Pan pelo arraste.
- Centralização automática nos nós novos.
- Botão para ajustar à área relevante.
- Tooltip com nome do nó.
- Alternância opcional entre recorte relevante e árvore completa.

### Estratégia de performance

- Pré-calcular os limites gráficos de cada etapa.
- Pré-calcular as conexões visíveis.
- Renderizar apenas elementos dentro da viewport.
- Não redesenhar quando não houver mudança.
- Não carregar texturas grandes durante a abertura normal.

---

## 11. Skills e gemas

A área de gemas deve mostrar os grupos exatamente como vieram da build.

Exemplo:

```text
Skill principal — 4 links
Static Strike
Melee Physical Damage Support
Faster Attacks Support
Fortify Support
```

### Destaques da etapa

```text
Adicionar
+ Static Strike
+ Fortify Support

Remover
- Ground Slam

Próxima mudança
Ato 4
```

Quando disponível, mostrar:

- nível mínimo exigido pela gema;
- nível configurado na build;
- qualidade;
- gema ativa ou desativada;
- suporte ou skill ativa;
- grupo de links.

O nível mínimo da gema deve vir dos dados locais da versão do jogo, não de inferência pelo nome da etapa.

---

## 12. Persistência local

Usar SQLite com migrations versionadas.

### Tabelas iniciais

```text
app_settings
builds
build_sources
build_stages
build_stage_nodes
build_skill_groups
build_gems
build_progress
passive_tree_versions
passive_tree_nodes
passive_tree_connections
window_state
```

### Dados salvos

- URL original.
- Tipo de fonte.
- Hash do conteúdo.
- Conteúdo original opcionalmente compactado.
- Dados normalizados.
- Diferenças entre etapas.
- Etapa atual.
- Associação entre árvore e skill set.
- Zoom e posição da árvore.
- Tamanho e posição da overlay.

---

## 13. Otimização de carregamento

### Importação

Operações mais pesadas ficam restritas ao momento de importação ou atualização:

- download;
- Base64;
- zlib;
- XML;
- normalização;
- comparação entre etapas;
- preparação gráfica.

### Abertura normal

```text
Ctrl+B
→ buscar build ativa
→ buscar etapa atual
→ carregar nós e gemas já processados
→ exibir overlay
```

### Regras

- Não consultar rede na abertura.
- Não reprocessar XML na abertura.
- Não recalcular diferenças na abertura.
- Não reconstruir toda a árvore quando apenas trocar de aba.
- Manter em memória somente a build ativa.
- Carregar outras builds sob demanda.
- Usar consultas indexadas.

---

## 14. Arquitetura do projeto

```text
poe-build-overlay/
├── cmd/
│   └── poe-build-overlay/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── state.go
│   │   └── lifecycle.go
│   │
│   ├── overlay/
│   │   ├── controller.go
│   │   ├── window.go
│   │   └── build_view.go
│   │
│   ├── ui/
│   │   ├── theme/
│   │   ├── widgets/
│   │   └── tree/
│   │
│   ├── platform/
│   │   └── windows/
│   │       ├── hotkeys.go
│   │       ├── window.go
│   │       ├── monitor.go
│   │       └── tray.go
│   │
│   ├── importers/
│   │   ├── importer.go
│   │   ├── pobbin.go
│   │   ├── pastebin.go
│   │   └── direct_code.go
│   │
│   ├── pob/
│   │   ├── decoder.go
│   │   ├── parser.go
│   │   ├── model.go
│   │   └── normalizer.go
│   │
│   ├── builds/
│   │   ├── service.go
│   │   ├── stages.go
│   │   ├── skills.go
│   │   └── diff.go
│   │
│   ├── passive_tree/
│   │   ├── loader.go
│   │   ├── model.go
│   │   ├── viewport.go
│   │   └── renderer.go
│   │
│   ├── storage/
│   │   ├── database.go
│   │   ├── migrations.go
│   │   └── repositories/
│   │
│   └── config/
│
├── assets/
│   ├── icons/
│   └── tree/
│
├── migrations/
├── go.mod
└── README.md
```

---

## 15. Interfaces internas

### Importador

```go
type BuildImporter interface {
    Supports(input string) bool
    Import(ctx context.Context, input string) ([]byte, error)
}
```

### Parser

```go
type PoBParser interface {
    Parse(data []byte) (ParsedBuild, error)
}
```

### Repositório

```go
type BuildRepository interface {
    Save(ctx context.Context, build Build) error
    FindByID(ctx context.Context, id string) (Build, error)
    FindActive(ctx context.Context) (Build, error)
    SetCurrentStage(ctx context.Context, buildID, stageID string) error
}
```

### Overlay

```go
type OverlayController interface {
    ToggleBuildOverlay()
    Hide()
    IsVisible() bool
}
```

---

## 16. Tratamento de erros

A interface deve tratar claramente:

- link inválido;
- serviço indisponível;
- build removida;
- código PoB inválido;
- Base64 inválida;
- zlib inválido;
- XML incompatível;
- build sem árvores;
- build sem skill sets;
- árvore sem nome;
- associação ambígua entre árvore e skill set;
- dados da árvore incompatíveis com a versão da build;
- erro de banco;
- hotkey indisponível.

Exemplo:

```text
Não foi possível importar a build.
O link não retornou um código PoB válido.
```

---

## 17. Performance

### Objetivos

- CPU próxima de zero quando ocioso.
- Nenhuma renderização com a overlay escondida.
- Abertura rápida usando somente dados locais.
- Navegação imediata entre etapas já carregadas.
- Importação sem bloquear a interface.
- Uso de memória previsível mesmo em builds grandes.

### Benchmark

Medir:

- tempo de inicialização;
- tempo entre `Ctrl+B` e overlay visível;
- uso de memória ocioso;
- uso de CPU ocioso;
- tempo de importação de build grande;
- tempo de carregamento de build processada;
- tempo de troca entre etapas;
- renderização da árvore completa;
- renderização apenas da região relevante.

---

## 18. Fases de desenvolvimento

## Fase 1 — Fundação desktop

- Criar projeto Go.
- Configurar Gio.
- Criar janela sem bordas.
- Implementar `Ctrl+B` como toggle.
- Implementar fechamento por `Esc`.
- Criar system tray.
- Salvar posição e tamanho da janela.
- Garantir ausência de renderização quando escondida.

### Critério de conclusão

A aplicação inicia na bandeja e abre ou fecha uma overlay vazia sem interferir no jogo.

---

## Fase 2 — Persistência local

- Integrar SQLite.
- Criar migrations.
- Criar repositórios.
- Salvar configurações da janela.
- Salvar build ativa e etapa atual.

### Critério de conclusão

O estado permanece após reiniciar o aplicativo.

---

## Fase 3 — Importação PoB

- Suportar código PoB direto.
- Implementar importação de `pobb.in`.
- Implementar Pastebin compatível.
- Decodificar Base64.
- Descompactar zlib.
- Interpretar XML.
- Extrair árvores.
- Extrair skill sets.
- Preservar nomes e ordem.
- Salvar conteúdo normalizado.

### Critério de conclusão

Uma build com múltiplas árvores e skill sets pode ser importada e reaberta sem rede.

---

## Fase 4 — Navegação de progressão

- Listar etapas.
- Implementar anterior e próxima.
- Implementar seleção direta.
- Salvar etapa atual.
- Associar árvore e skill set.
- Permitir correção manual da associação.

### Critério de conclusão

O usuário consegue percorrer todas as etapas importadas na ordem definida pelo autor.

---

## Fase 5 — Comparação entre etapas

- Calcular nós adicionados e removidos.
- Calcular gemas adicionadas e removidas.
- Identificar alterações nos grupos de links.
- Pré-calcular diferenças durante importação.
- Exibir resumo da etapa atual.

### Critério de conclusão

A overlay mostra claramente o que mudou em relação à etapa anterior.

---

## Fase 6 — Árvore gráfica

- Obter dados estruturais da árvore passiva.
- Versionar dados da árvore.
- Mapear IDs para posições e conexões.
- Desenhar nós e caminhos com Gio.
- Implementar destaque por estado.
- Implementar zoom e pan.
- Centralizar nos nós novos.
- Implementar recorte relevante.
- Otimizar renderização por viewport.

### Critério de conclusão

A etapa atual exibe uma árvore gráfica rápida e legível, destacando os novos passivos.

---

## Fase 7 — Skills e gemas

- Renderizar grupos de gemas.
- Mostrar links.
- Mostrar novas gemas.
- Mostrar gemas removidas.
- Mostrar nível mínimo quando disponível.
- Mostrar próxima alteração.

### Critério de conclusão

O usuário consegue seguir a progressão de skills sem abrir o Path of Building.

---

## Fase 8 — Atualização e gerenciamento de builds

- Atualizar pelo link original.
- Comparar hash antes de reprocessar.
- Preservar etapa atual quando possível.
- Permitir múltiplas builds salvas.
- Selecionar build ativa.
- Excluir build local.

### Critério de conclusão

O usuário consegue manter e alternar builds sem repetir importações desnecessárias.

---

## Fase 9 — Estabilização

- Criar logs locais com rotação.
- Tratar corrupção do banco.
- Testar DPI.
- Testar múltiplos monitores.
- Testar janela em fullscreen, borderless e windowed.
- Medir CPU, memória e latência.
- Criar pacote portátil ou instalador.
- Adicionar aviso de não afiliação com a GGG.

### Critério de conclusão

A aplicação está estável, rápida e pronta para distribuição inicial.

---

## 19. Critério final do MVP

O MVP estará concluído quando o usuário puder:

1. Iniciar a aplicação na bandeja.
2. Pressionar `Ctrl+B`.
3. Colar um link `pobb.in`.
4. Importar uma build com múltiplas etapas.
5. Fechar e reabrir a overlay sem novo download.
6. Navegar por `Ato 1`, `Ato 2`, `Level 40`, `Endgame` ou outros nomes definidos pelo autor.
7. Visualizar graficamente os passivos da etapa.
8. Identificar os novos nós em relação à etapa anterior.
9. Ver as gemas e links correspondentes à etapa.
10. Fechar a overlay por `Ctrl+B` ou `Esc`.
