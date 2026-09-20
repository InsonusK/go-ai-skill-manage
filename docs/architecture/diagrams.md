# Диаграммы потока данных sync

Дополняют [автоматический граф пакетов](../features/diagrams/sync.svg)
(`make diagrams`, строится из реальных Go imports и не расходится с кодом).
Диаграммы на этой странице написаны вручную в Mermaid и объясняют порядок
вызовов и контракты данных — то, что граф импортов пакетов не показывает.

## Последовательность SyncService.Run

```mermaid
sequenceDiagram
    participant Main as main (composition root)
    participant CLI as command.App
    participant Sync as SyncService
    participant Mgr as SourceManager<br/>(SourceProvider)
    participant SrcMap as SourceMap
    participant Det as SourceSelector<br/>(discovery.Detector)
    participant Cat as Catalog
    participant Rel as RelationExpander
    participant SkMap as discovery.BuildSkillMap
    participant Plan as SyncPlanner<br/>(planning.Planner)
    participant Writer as PlanWriter

    Main->>Mgr: NewSourceManager(providers)
    Main->>CLI: Execute(ctx, opts, cwd)
    CLI->>Sync: Run(req)
    loop каждый source из req.Sources
        Sync->>Mgr: Acquire(spec, options)
        alt SourceKey уже в кеше
            Mgr-->>Sync: тот же *Repository (провайдер не вызывается)
        else первый запрос для этого SourceKey
            Mgr-->>Sync: repo, err (провайдер вызван, добавлен в кеш)
        end
        Sync->>SrcMap: Put(repo)
        Sync->>Det: Select(repo, spec)
        Det-->>Sync: []*Skill
        loop каждый найденный skill
            Sync->>Cat: discovery.Add(cat, skill)
        end
    end
    Sync->>Rel: Expand(cat, addRelations, skipFolders)
    Rel-->>Sync: err (или доращённый Catalog)
    Sync->>SkMap: BuildSkillMap(cat)
    SkMap-->>Sync: *SkillMap
    loop каждый target из req.Targets
        Sync->>Plan: Plan(cat, skillMap, sourceMap, req, target)
        Plan-->>Sync: TargetPlan
    end
    alt dry run
        Sync-->>CLI: Result (без записи)
    else обычный запуск
        loop каждый TargetPlan
            Sync->>Writer: Apply(plan)
        end
        Sync-->>CLI: Result
    end
    CLI-->>Main: exit code
    Main->>Mgr: Close() (defer, всегда выполняется)
    Mgr->>Mgr: закрывает каждый закешированный Repository,<br/>в порядке, обратном получению
```

`SourceManager` живёт **дольше одного `Run`** — его создаёт и закрывает `main`,
не `SyncService.Run`: `Acquire` дедуплицирует по `SourceKey` (тип+путь+ветка),
так что два `sources:` с одним источником, но разными `subpath`/`tags`, скачиваются
один раз. `SourceMap` и `SkillMap`, в отличие от этого, строятся **один раз за
запуск** и переиспользуются без изменений в цикле по `req.Targets` —
`SkillMap.Destinations()` возвращает копию, которую `OutputLayout` расширяет
вложениями конкретной цели, не трогая общий реестр.

## От исходного описания пайплайна к реальным вызовам

Ниже — те же восемь шагов, которыми обычно описывают `sync`, сопоставленные с
функциями, которые их реализуют.

```mermaid
flowchart TD
    A["1. Прочитать sources из ai-skills.yaml"] --> B
    B["2. Скачать не-local source в temp dir"] --> C
    C["3. Построить SourceMap"] --> D
    D["4. Для каждого source: SourceSelector.Select<br/>по subpath/tags, flat-skill vs dir-skill"] --> E
    E["5. Скопировать найденные скилы,<br/>построить SkillMap"] --> F
    F["6. Применить target.default.adapters"] --> G
    G["7. Для каждого target≠default:<br/>копия + свои adapters"] --> H
    H["8. Записать каждый target в target.path"]

    A -.->|"config.Parse + config.Resolve"| A1[/"model.Request"/]
    B -.->|"SourceManager.Acquire (deduped by SourceKey)<br/>repository.Local / repository.Fetcher"| B1[/"model.Repository"/]
    C -.->|"SyncService.Run<br/>sources.Put(repo)"| C1[/"model.SourceMap"/]
    D -.->|"discovery.Detector.Select"| D1[/"[]model.Skill"/]
    E -.->|"relations.Expander.Expand<br/>discovery.BuildSkillMap"| E1[/"model.Catalog + model.SkillMap"/]
    F -.->|"planning.Planner.Plan<br/>для target default"| F1[/"model.TargetPlan"/]
    G -.->|"planning.Planner.Plan<br/>для остальных target, тот же Catalog/SkillMap"| G1[/"[]model.TargetPlan"/]
    H -.->|"filesystem.Store.Apply"| H1[/"файлы на диске"/]
```

**Важное отличие от буквального прочтения шагов 5–7**: в текущей реализации
нет физической директории «temp skills». Всё от обнаружения до преобразований
каждой цели остаётся в памяти как байты `model.Skill.Files`; `planning.Planner.Plan`
вызывается один раз на каждую цель и независимо пересчитывает выходные байты
из одних и тех же входных данных и общего `SkillMap`. Единственная операция,
которая реально касается диска (кроме исходного clone/download), — это
`PlanWriter.Apply` на шаге 8. Такое решение сознательное: оно проще и быстрее
физического копирования, и весь путь уже покрыт сценариями — см. обсуждение
в [go-cli-migration.md](go-cli-migration.md#правила-зависимостей).

## Контракты SourceMap и SkillMap

```mermaid
classDiagram
    class SourceMap {
        -map~string,Repository~ repos
        -[]string order
        +Put(repo Repository)
        +Get(id string) Repository
        +Repositories() []Repository
    }
    class SkillMap {
        -map~string,SkillEntry~ entries
        -map~string,skillPath~ paths
        +Entry(name string) SkillEntry
        +Owner(sourceKey, path string) SkillEntry, dest, ok
        +Destinations() map~string,string~
    }
    class SkillEntry {
        +string Name
        +string SourceKey
        +string Root
        +string Main
        +bool Flat
        +string Dest
    }
    class Catalog {
        +[]Skill Skills
        +string Conflict
    }
    SkillMap --> SkillEntry : entries
    Catalog --> SkillMap : discovery.BuildSkillMap(cat)
    SourceMap --> Repository : repos
```

| Реестр | Кто пишет | Кто читает | Когда строится |
| --- | --- | --- | --- |
| `SourceManager` (`repository`) | `main` создаёт; `Acquire` кеширует по `SourceKey` | `SyncService.Run` (через порт `SourceProvider`) | один раз на процесс; переживает несколько `Run`, если бы они были |
| `SourceMap` | `SyncService.Run` (цикл acquire) | `OutputLayout` (содержимое внешних вложений) | один раз за запуск, по мере получения каждого source |
| `SkillMap` | `discovery.BuildSkillMap` | `OutputLayout` (назначение выходных путей, включая fallback через `OwnsPath`/`RelativePath` для путей вне инвентаря) | один раз за запуск, сразу после `RelationExpander.Expand` |
| `Catalog` | `discovery.Add` (в цикле acquire), `RelationExpander.Expand` (добавляет связанные скилы) | `discovery.BuildSkillMap`, `SyncPlanner.Plan` | растёт по мере обнаружения и раскрытия связей |

`SkillMap.Owner` сначала ищет точное совпадение среди проинвентаризированных
файлов, а при отсутствии — резолвит через `OwnsPath`/`RelativePath` по
`Root`/`Main`/`Flat` записи, так что путь внутри выбранного directory skill
разрешается даже без явного файла в инвентаре (совместимость с Python,
задокументированная в [compatibility.md](compatibility.md)).
