---
feature: sync
depends_on:
  - "[cmd/ai-skill-manager](../../cmd/ai-skill-manager/main.go)"
  - "[internal/command](../../internal/command/arguments.go)"
  - "[internal/config](../../internal/config/config.go)"
  - "[internal/domain/interfaces](../../internal/domain/interfaces/ports.go)"
  - "[internal/domain/model](../../internal/domain/model/model.go)"
  - "[internal/domain/services](../../internal/domain/services/sync.go)"
  - "[internal/domain/services/discovery](../../internal/domain/services/discovery/detector.go)"
  - "[internal/domain/services/links](../../internal/domain/services/links/extract.go)"
  - "[internal/domain/services/planning](../../internal/domain/services/planning/layout.go)"
  - "[internal/domain/services/relations](../../internal/domain/services/relations/expand.go)"
  - "[internal/domain/services/sourcing](../../internal/domain/services/sourcing/manager.go)"
  - "[internal/domain/services/tags](../../internal/domain/services/tags/tags.go)"
  - "[internal/domain/services/transform](../../internal/domain/services/transform/links.go)"
  - "[internal/infrastructure/document](../../internal/infrastructure/document/codec.go)"
  - "[internal/infrastructure/filesystem](../../internal/infrastructure/filesystem/apply.go)"
  - "[internal/infrastructure/repository](../../internal/infrastructure/repository/local.go)"
  - "[internal/logging](../../internal/logging/logger.go)"
  - "[internal/profiling](../../internal/profiling/profile.go)"
  - "[internal/version](../../internal/version/version.go)"
---

# Синхронизация скилов

Одна публичная команда `sync` собирается из перечисленных ниже единиц.
[Архитектура](../architecture/go-cli-migration.md) объясняет порядок вызовов;
[справочник](../api/reference.md) задаёт пользовательский контракт.

## Возможности

| Возможность | Основные единицы |
| --- | --- |
| YAML/JSON и CLI | Arguments, ConfigLoader, OptionResolver, SyncCommand |
| Local / GitHub | LocalSource, RepositoryFetcher, GitCloner, ArchiveFetcher, SourceManager, RepositoryLookup |
| Форматы и фильтры | SkillDetector, SourceSelector, TagExpression, SkillCatalog |
| Связи и вложения | FileLoader, LinkExtractor, LinkExclusion, LinkResolver, RelationExpander |
| Выходные файлы | OutputLayout, LinkRewriter, ClaudeTransformer, FrontmatterCodec |
| Incremental / force / orphans | Fingerprint, SyncPlanner, StateReader, PlanApplier |
| Dry-run и lifecycle | SyncService, ResultFormatter, Logging, Profiling |

## Модули

Ссылка на реализацию ведёт к ответственному файлу. Gherkin-сценарии и
step-определения лежат рядом с пакетом (`features/*.feature`, `test/`).
Общий код тестового запуска — `tools/testsupport`.

| Единица | Вид | Ответственность и реализация | Зависимости |
| --- | --- | --- | --- |
| Arguments | Function | [Parse](../../internal/command/arguments.go) — Разбирает аргументы CLI | нет |
| ConfigLoader | Service | [Parse](../../internal/config/config.go) — Декодирует конфигурацию с defaults | YAML decoder |
| OptionResolver | Function | [Resolve](../../internal/config/resolve.go) — Вычисляет эффективный запрос | базовый путь и overrides |
| SyncCommand | Command | [App.Execute](../../internal/command/sync.go) — Связывает команду с синхронизацией | чтение конфига, SyncService, output streams |
| ResultFormatter | Function | [PrintResult](../../internal/command/format.go) — Представляет план в консоли | io.Writer |
| SyncService | Orchestrator | [SyncService.Run](../../internal/domain/services/sync.go) — Координирует стадии одного запуска | SourceCache, RepositoryLookup, SourceSelector, RelationExpander, SyncPlanner, PlanWriter (порты); не управляет временем жизни источников |
| LocalSource | Service | [Local.Acquire](../../internal/infrastructure/repository/local.go) — Открывает локальное дерево, определяет `Repository.SingleFile` | os.Root / fs.FS |
| RepositoryFetcher | Service | [Fetcher.Acquire](../../internal/infrastructure/repository/fetch.go) — Предоставляет временную копию репозитория, регистрирует очистку через `Repository.AddCloser` | Cloner, ArchiveFetcher, временная директория |
| GitCloner | Service | [GitCloner.Clone / GitProcess.Run](../../internal/infrastructure/repository/git.go) — Получает ветку или тег через Git | ProcessRunner |
| ArchiveFetcher | Service | [Archive.Fetch](../../internal/infrastructure/repository/archive.go) — Предоставляет дерево из GitHub tar.gz | HTTPClient, ограниченный файловый корень |
| SourceManager | Service | [Manager.GetOrAdd / Lookup / Close](../../internal/domain/services/sourcing/manager.go) — Кеширует `Repository` по `SourceKey`, диспетчеризует по типу, владеет их временем жизни; не выполняет I/O сама, только делегирует зарегистрированным `SourceProvider` | LocalSource, RepositoryFetcher (по типу, через порт) |
| RepositoryLookup | Port | [Lookup](../../internal/domain/interfaces/ports.go) — Находит уже полученный `Repository` по `ID`; реализован `Manager.Lookup` | нет |
| SkillDetector | Service | [Detector.Discover / Rooted / Find](../../internal/domain/services/discovery/detector.go) — Распознаёт расположение скила | DocumentCodec, fs.FS |
| SourceSelector | Function | [Detector.Select](../../internal/domain/services/discovery/source.go) — Выбирает скилы источника по `SourceSpec`, резолвит scan paths (`scanPaths`) | Detector, TagExpression |
| TagExpression | Function | [Match](../../internal/domain/services/tags/tags.go) — Вычисляет фильтр тегов | нет |
| SkillCatalog | Service | [SkillCatalog.GetOrAdd / .Owner / .Destination](../../internal/domain/model/catalog.go) — Разрешает коллизии скилов, индексирует выходные назначения файлов по мере добавления, находит владельца пути | нет |
| FileLoader | Function | [Tags / LoadFiles](../../internal/domain/services/discovery/inventory.go) — Извлекает теги из frontmatter; догружает содержимое вложенных файлов скила, уже прошедшего отбор | fs.FS |
| LinkExtractor | Function | [Extract](../../internal/domain/services/links/extract.go) — Извлекает диапазоны ссылок | нет |
| LinkExclusion | Function | [Excluded](../../internal/domain/services/links/exclusion.go) — Определяет исключения проверки ссылки | нет |
| LinkResolver | Function | [Resolve](../../internal/domain/services/links/resolve.go) — Определяет путь адресата | fs.FS, knownOwner predicate |
| RelationExpander | Service | [Expander.Expand](../../internal/domain/services/relations/expand.go) — Строит замыкание связанных скилов | Detector, SkillCatalog, LinkResolver |
| OutputLayout | Function | [BuildLayout](../../internal/domain/services/planning/layout.go) — Назначает выходные пути | SkillCatalog, RepositoryLookup (для внешних вложений) |
| LinkRewriter | Function | [Rewrite](../../internal/domain/services/transform/links.go) — Заменяет адреса ссылок по layout | нет |
| ClaudeTransformer | Function | [Claude](../../internal/domain/services/transform/claude.go) — Преобразует свойства Claude | нет |
| Fingerprint | Function | [Fingerprint](../../internal/domain/services/planning/fingerprint.go) — Вычисляет отпечаток результата | готовые OutputFile |
| SyncPlanner | Service | [Planner.Plan](../../internal/domain/services/planning/planner.go) — Определяет операции обновления цели | SkillCatalog, RepositoryLookup, StateReader, DocumentCodec, OutputLayout, преобразования |
| PlanApplier | Service | [Store.Apply](../../internal/infrastructure/filesystem/apply.go) — Применяет готовый план | os.Root |
| StateReader | Service | [Store.Snapshot](../../internal/infrastructure/filesystem/store.go) — Читает управляемое состояние цели | os.Root |
| FrontmatterCodec | Service | [Codec.Decode / Encode](../../internal/infrastructure/document/codec.go) — Преобразует frontmatter в модель | YAML codec |
| Logging | Function | [Init](../../internal/logging/logger.go) — Настраивает process logger | stderr writer, debug option |
| Profiling | Function | [Start](../../internal/profiling/profile.go) — Управляет CPU profile | runtime/pprof, файл |

### Модели, порты, composition root

- [model](../../internal/domain/model/model.go): Request → Repository/Skill/Link → TargetPlan → Result; ошибки Issue/Issues. Также [SkillCatalog](../../internal/domain/model/catalog.go) (растёт через `GetOrAdd`, индексирует выходные назначения) и примитивы [OwnsPath/RelativePath](../../internal/domain/model/ownership.go).
- [interfaces](../../internal/domain/interfaces/ports.go): SourceProvider, SourceCache, RepositoryLookup, DocumentCodec, StateReader, PlanWriter, SourceSelector, RelationExpander, SyncPlanner. Получатель порта определяет необходимую роль.
- [main](../../cmd/ai-skill-manager/main.go): реальные адаптеры, constructor injection, сигналы, profiler, exit code; создаёт `sourcing.Manager` и владеет его временем жизни (`defer sources.Close(ctx)`).
- [version](../../internal/version/version.go): build-time значение из VERSION; без отдельной бизнес-логики.

## Использование единиц

1. Arguments и SyncCommand получают вызов пользователя. ConfigLoader и OptionResolver создают Request.
2. `main` создаёт один `sourcing.Manager` на весь процесс и связывает его с SyncService
   сразу как SourceCache и как RepositoryLookup (`Sources`/`Lookup` — один и тот
   же экземпляр). SyncService получает источники через него (Manager отдаёт кеш по
   `SourceKey`, реально получая источник только на первый запрос — сам делегируя
   LocalSource/RepositoryFetcher, а не выполняя I/O напрямую), вызывает SourceSelector
   и добавляет найденные скилы в SkillCatalog через `GetOrAdd` (после FileLoader.LoadFiles
   догружает содержимое вложенных файлов — только для скилов, прошедших отбор).
3. RelationExpander обходит Markdown; LinkResolver находит владельца или внешнее
   вложение; вновь найденные связанные скилы также проходят через `GetOrAdd`.
4. SyncPlanner строит все TargetPlan, читая StateReader и используя выходные
   назначения, уже проиндексированные SkillCatalog по мере роста; для
   содержимого внешних вложений вызывает RepositoryLookup.Lookup по `Repository.ID`
   вместо отдельного реестра. OutputLayout и преобразования готовят окончательные
   байты; переписывание ссылок применяется только когда `link-adapter` присутствует
   в адаптерах цели.
5. Dry-run возвращает планы. Обычный запуск передаёт планы PlanApplier.
6. ResultFormatter выводит результат. `main` закрывает `sourcing.Manager` (`defer
   sources.Close(ctx)`) после `Execute`, независимо от результата — SyncService.Run
   больше не управляет временем жизни источников само.

## Диаграмма

![Зависимости Go-пакетов](diagrams/sync.svg)

[JSON Canvas](diagrams/sync.canvas) и SVG генерируются командой `make diagrams`
через локальный [diagram-renderer](../../tools/diagram-renderer/main.go).
`depends_on` задаёт состав, Go imports — связи; стрелка означает «импортирует».
Диаграмма показывает зависимости типов/пакетов, порядок выполнения приведён выше.
Порядок вызовов и контракты `RepositoryLookup`/`SkillCatalog` — в
[диаграммах потока данных](../architecture/diagrams.md) (Mermaid, вручную).
