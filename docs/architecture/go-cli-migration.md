# Перенос ai-skill-manager на Go

Статус: согласованная декомпозиция реализована. Рабочие точки входа: `bin/aism` и `bin/ai-skill-manager`.

Основание: Python CLI в `deprecated/ai-skill-manager`, версия пакета 1.8.1,
и локальный skill `plateau-http-service`. Входной адаптер плато заменяется
на `internal/command`. Программа остаётся CLI с командой `sync`.

## Что должен получить пользователь

- Исполняемые файлы `ai-skill-manager` и `aism` с общей реализацией.
- Синхронизацию local/GitHub-источников из YAML/JSON или CLI-параметров.
- Поддержку Agent, HumanDir, HumanFlat, фильтров тегов, связанных скилов,
  внешних файлов, переписывания ссылок, нескольких целей и Claude-адаптера.
- Dry-run, инкрементальное копирование, force и удаление управляемых orphan-скилов.
- Документацию, по которой можно перейти от команды к ответственному модулю,
  его зависимостям, сценариям и результатам тестирования.

## Структура

Структура реализованного проекта; навигация по файлам и сценариям — в [индексе](../features/sync.md).

```text
cmd/ai-skill-manager/main.go       сборка зависимостей, сигналы, код завершения
internal/
  command/                       аргументы, вызов use case, вывод результата
  config/                        YAML/JSON, значения по умолчанию, overrides
  domain/
    model/                       Skill, Source, Target, Link, Plan, Result, ошибки
    interfaces/                  узкие порты внешних операций
    services/
      sync.go                    порядок выполнения синхронизации
      discovery/                 распознавание форматов, каталог, файлы скила
      tags/                      выражения фильтров тегов
      links/                     извлечение, исключения, разрешение ссылок
      relations/                 очередь связанных скилов
      planning/                  состав выходных файлов и изменений цели
      transform/                 переписывание ссылок, свойства Claude
      sourcing/                  кеш и время жизни полученных источников
  infrastructure/
    filesystem/                  состояние и запись цели
    repository/                  local, Git clone, GitHub archive fallback
    document/                    кодек YAML frontmatter
  logging/                       настройка slog
  profiling/                     жизненный цикл Go CPU profile
  version/                       версия сборки
docs/
  api/reference.md               команды, флаги, ошибки и примеры
  architecture/                  карта модулей и правила зависимостей
  features/                      возможности → единицы → тесты
  testing.md                     запуск, изоляция, покрытие и мутации
report-template/index.html        стартовая страница отчёта
tools/                           нормализация отчётов и генерация public/
```

В каждом тестируемом пакете: `doc.go` с ответственностью пакета,
`features/*.feature` со сценариями, `test/runner_test.go` с одним
`TestFeatures`, `test/*_steps_test.go` по понятиям, `TESTS.md` с матрицей
«сценарий → функция → проверка». Названия подкаталогов `services` обозначают
самостоятельные предметные области; внутри каждого файла — одна ответственность.

## Правила зависимостей

`main` создаёт адаптеры и сервисы явно. `command` вызывает конкретный
`SyncService`. Доменные сервисы зависят от моделей, чистых функций и узких
портов, объявленных в `domain/interfaces`. Инфраструктура реализует эти порты.
Доменные пакеты не импортируют `command`, `config`, инфраструктурные пакеты,
`os/exec` или HTTP-клиент.

Интерфейсы выделяются по потребностям вызывающего кода: чтение дерева,
чтение содержимого, получение репозитория, декодирование документа, чтение
состояния цели, применение подготовленного плана. Универсальный интерфейс
«вся файловая система» не требуется. Чистым функциям интерфейс не добавляется.

Ошибки предметной области имеют различимые типы/коды и контекст источника,
скила, файла и ссылки. CLI определяет их представление и exit code.
Глобальное изменяемое состояние не используется для настроек, очередей или кешей.

`Catalog` (набор выбранных скилов и политика конфликта) объявлен в `domain/model`,
а не в `domain/services/discovery`: `domain/interfaces` должен ссылаться на его тип
в сигнатурах портов `SourceSelector`/`RelationExpander`/`SyncPlanner`, а `discovery`
уже зависит от `interfaces` (через `DocumentCodec`) — оставление `Catalog` в `discovery`
создало бы цикл `interfaces → discovery → interfaces`. Поведение (`Add`, `Owner`)
остаётся в `discovery` как свободные функции, принимающие `*model.Catalog`.

`Repository` хранит только идентичность источника (`ID`, `Root`, `FS`, `SingleFile`,
`SkipFolders`) — ничего, что зависит от того, *какой* `SourceSpec` его запросил.
`Subpaths`/`Tags`/`Name` — параметры выбора конкретного `SourceSpec` — не хранятся
на `Repository` и передаются в `SourceSelector.Select` явно при каждом вызове. Это
разделение обязательно для `sourcing.Manager` (`domain/services/sourcing`): он кеширует
`*Repository` по `SourceKey` (`Type`+`Path`+`Tree`), так что два `sources:` с одним
и тем же источником, но разными `subpath`, не скачиваются повторно — один и тот же
кешированный `Repository` не может при этом нести значения `Subpaths`/`Tags`,
рассчитанные под первый вызов. `SkipFolders` остаётся на `Repository` (не параметр
`Select`), поскольку это не фильтр выбора, а правило разбора дерева: `Detector.Rooted`
читает его и вызывается также из `Find` при раскрытии связей — уже без доступного
исходного `SourceSpec`.

## Декомпозиция

Для каждой единицы указаны ответственность, роли зависимостей и сценарий вызова.
Модели данных и интерфейсы сами по себе не становятся отдельными сервисами.

- **Arguments** (Function), `command`.
  - responsibility: преобразует аргументы CLI в типизированные параметры запуска.
  - depends_on: нет.
  - usage_scenario: вызывается до загрузки конфигурации; возвращает параметры,
    запрос справки либо ошибку аргументов.
- **ConfigLoader** (Service), `config`.
  - responsibility: декодирует конфигурационный документ в настройки запуска.
  - depends_on: YAML-декодер.
  - usage_scenario: принимает bytes YAML/JSON от command, проверяет типы и применяет defaults.
- **OptionResolver** (Function), `config`.
  - responsibility: вычисляет эффективные настройки запуска по приоритетам входов.
  - depends_on: нет.
  - usage_scenario: получает разобранную конфигурацию, CLI overrides и базовый путь;
    выдаёт полностью определённый запрос синхронизации.
- **SyncCommand** (Command), `command`.
  - responsibility: связывает пользовательский вызов `sync` с use case.
  - depends_on: загрузка настроек, выполнение синхронизации, представление результата.
  - usage_scenario: получает параметры, вызывает сервис и возвращает exit code.
- **ResultFormatter** (Function), `command`.
  - responsibility: представляет результат синхронизации в консоли.
  - depends_on: потоки вывода.
  - usage_scenario: печатает найденные скилы, итог или сгруппированные ошибки;
    тест проверяет текст через буфер.
- **SyncService** (Service/orchestrator), `domain/services`.
  - responsibility: координирует стадии одной синхронизации.
  - depends_on: получение источников, обнаружение, обработка связей, планирование,
    применение плана.
  - usage_scenario: выполняет стадии в заданном порядке; ошибки валидации
    блокируют запись во все цели, dry-run возвращает план без применения. Не
    управляет временем жизни источников — не закрывает и не кеширует их;
    это ответственность `sourcing.Manager`, живущего в вызывающем коде (`main`).
- **LocalSource** (Service), `infrastructure/repository`.
  - responsibility: предоставляет корень локального источника.
  - depends_on: чтение дерева.
  - usage_scenario: определяет, является ли путь источника отдельным плоским
    файлом (`Repository.SingleFile`) или директорией; scan paths (из `subpath:`)
    в `Repository` не сохраняются — их резолвит `SourceSelector.Select` для
    каждого `SourceSpec` отдельно.
- **RepositoryFetcher** (Service), `infrastructure/repository`.
  - responsibility: предоставляет временную копию удалённого репозитория.
  - depends_on: клонирование Git, скачивание архива, временное хранилище.
  - usage_scenario: сначала пробует clone, для GitHub использует archive fallback;
    регистрирует очистку временной директории через `Repository.AddCloser`
    вместо возврата отдельной функции очистки.
- **SourceManager** (Service), тип `Manager` в `domain/services/sourcing`.
  - responsibility: кеширует полученные `Repository` по `SourceKey`
    (`Type`+`Path`+`Tree`) и диспетчеризует по `SourceSpec.Type` к
    `LocalSource`/`RepositoryFetcher`.
  - depends_on: `interfaces.SourceProvider` реализации, зарегистрированные по типу
    (никаких прямых зависимостей от `os/exec`/`net/http`/файловой системы —
    только делегирование инфраструктурным реализациям порта).
  - usage_scenario: несколько `sources:` с одинаковым источником, но разными
    `subpath`/`tags`, скачиваются один раз; `Close(ctx)` закрывает каждый полученный
    `Repository` в порядке, обратном получению. Владеет временем жизни источников
    вместо `SyncService.Run` — создаётся и закрывается в `main` (`defer sources.Close(ctx)`).
    Живёт в `domain/services/sourcing`, а не в `infrastructure/repository`: сам он
    не выполняет I/O, только оркестрирует кеш поверх инъецированного порта — как и
    `discovery`/`relations`/`planning` рядом с ним.
- **GitCloner** (Service), `infrastructure/repository`.
  - responsibility: получает выбранную ветку или тег через Git.
  - depends_on: запуск процесса с context.
  - usage_scenario: передаёт аргументы напрямую процессу; возвращает корень clone
    либо типизированную ошибку.
- **ArchiveFetcher** (Service), `infrastructure/repository`.
  - responsibility: получает дерево файлов из архива GitHub.
  - depends_on: HTTP transport, временное хранилище.
  - usage_scenario: скачивает tar.gz, проверяет границы путей при распаковке;
    тестируется с локальным HTTP-сервером и подготовленными архивами.
- **SkillDetector** (Service), `domain/services/discovery`.
  - responsibility: распознаёт скилы в выбранном дереве источника.
  - depends_on: чтение дерева, чтение документа, декодирование frontmatter.
  - usage_scenario: обнаруживает Agent/HumanDir/HumanFlat и сообщает о конфликте
    форматов, вложенных скилах или некорректном имени. Его метод `Select`
    связан с `SyncService` через порт `interfaces.SourceSelector`, а не как
    конкретная структура напрямую.
- **TagExpression** (Function), `domain/services/tags`.
  - responsibility: вычисляет результат выражения фильтра по тегам скила.
  - depends_on: нет.
  - usage_scenario: фильтрует кандидатов по `!`, `&`, `|`, скобкам,
    иерархическим тегам и поддерживаемым исходной реализацией wildcard-правилам.
- **SkillCatalog** (Service), тип `domain/model`, поведение `domain/services/discovery`.
  - responsibility: разрешает коллизии имён в наборе найденных скилов.
  - depends_on: нет внешних зависимостей.
  - usage_scenario: `discovery.Add`/`discovery.Owner` — свободные функции над
    `model.Catalog`; объединяют кандидатов с учётом выбранной политики конфликта,
    повтор того же скила не создаёт дубль.
- **FileInventory** (Service), `domain/services/discovery`.
  - responsibility: определяет собственные файлы скила.
  - depends_on: чтение дерева.
  - usage_scenario: возвращает относительные пути и типы файлов для анализа,
    копирования и вычисления отпечатка содержимого.
- **RepositoryLookup** (Port), `domain/interfaces`, реализация — `Manager.Lookup`
  в `domain/services/sourcing`.
  - responsibility: находит уже полученный `Repository` по его `ID`, без нового
    получения.
  - depends_on: нет внешних зависимостей (линейный проход по уже закешированным
    репозиториям).
  - usage_scenario: `OutputLayout` знает только `repoID` (из `Link.Target`) и
    вызывает `Lookup`, чтобы прочитать содержимое внешнего вложения — того же
    `sourcing.Manager`, что уже используется как `SourceProvider`, без отдельного
    реестра. `SyncService.Lookup` указывает на тот же экземпляр, что и `Sources`.
- **SkillMap** (Function), тип `domain/model`, построение `domain/services/discovery`.
  - responsibility: итоговый реестр «имя скила → источник → исходный путь →
    выходной путь», построенный один раз.
  - depends_on: `Catalog`.
  - usage_scenario: `discovery.BuildSkillMap` вызывается один раз после
    `RelationExpander.Expand`; `OutputLayout` переиспользует один и тот же
    `SkillMap` для каждой цели вместо повторного обхода каталога.
- **LinkExtractor** (Function), `domain/services/links`.
  - responsibility: извлекает ссылки с позициями в исходном тексте.
  - depends_on: нет.
  - usage_scenario: получает Markdown; выдаёт Markdown-ссылки, изображения
    и wikilinks с исходными диапазонами текста.
- **LinkExclusion** (Function), `domain/services/links`.
  - responsibility: определяет необходимость проверки найденной ссылки.
  - depends_on: нет.
  - usage_scenario: применяет правила web/anchor, inline code, блоков `example`
    и исключённых директорий.
- **LinkResolver** (Service), `domain/services/links`.
  - responsibility: определяет адресата файловой ссылки.
  - depends_on: проверка пути в источнике, каталог скилов.
  - usage_scenario: разрешает относительные и корневые пути с сохранением anchor;
    отличает собственный файл, другой скил, внешний файл и ошибку.
- **RelationExpander** (Service), `domain/services/relations`.
  - responsibility: вычисляет замыкание зависимостей выбранных скилов.
  - depends_on: поиск скила по пути, каталог, получение ссылок скила.
  - usage_scenario: при `add_relations` добавляет связанные скилы до исчерпания
    очереди; защищает обход от циклов и повторной обработки. Связан с
    `SyncService` через порт `interfaces.RelationExpander`.
- **OutputLayout** (Function), `domain/services/planning`.
  - responsibility: назначает выходные пути файлам синхронизации.
  - depends_on: `Catalog`, `SkillMap`, `RepositoryLookup` (для содержимого внешних вложений).
  - usage_scenario: отображает входные скилы в `{name}/SKILL.md`, назначает пути
    вложениям и внешним файлам, сообщает о коллизиях выходных путей. Читает
    готовые назначения из `SkillMap.Destinations()` вместо повторного обхода
    каталога на каждую цель.
- **LinkRewriter** (Function), `domain/services/transform`.
  - responsibility: заменяет адреса разрешённых ссылок на выходные пути.
  - depends_on: нет.
  - usage_scenario: использует позиции ссылок и layout конкретной цели;
    сохраняет подписи и anchors, не затрагивает исключённые ссылки.
- **ClaudeTransformer** (Function), `domain/services/transform`.
  - responsibility: преобразует свойства документа по правилам Claude-адаптера.
  - depends_on: нет.
  - usage_scenario: обрабатывает `whenToUse`/`when_to_use` и переносит
    дополнительные свойства в Metadata, сохраняя содержимое скила.
- **Fingerprint** (Function), `domain/services/planning`.
  - responsibility: вычисляет отпечаток подготовленного содержимого скила.
  - depends_on: нет.
  - usage_scenario: получает детерминированный список путей и содержимого;
    позволяет обнаружить изменение файла, имени, внешнего вложения или адаптера.
- **SyncPlanner** (Service), `domain/services/planning`.
  - responsibility: определяет набор изменений каждой цели.
  - depends_on: `Catalog`, `SkillMap`, `RepositoryLookup`, чтение состояния цели,
    подготовка выходных документов.
  - usage_scenario: выдаёт операции create/update/skip/remove с причинами;
    учитывает force, managed-маркеры и политику orphan; переписывает ссылки
    только когда `link-adapter` присутствует в объединённом списке адаптеров
    цели. Связан с `SyncService` через порт `interfaces.SyncPlanner`.
- **PlanApplier** (Service), `infrastructure/filesystem`.
  - responsibility: применяет подготовленные изменения к файловой системе.
  - depends_on: файловые операции цели.
  - usage_scenario: записывает готовые файлы и managed-state, удаляет только
    запланированные управляемые orphan-директории; сообщает ошибки записи.
- **FrontmatterCodec** (Service), `infrastructure/document`.
  - responsibility: преобразует YAML-заголовок между текстом и моделью документа.
  - depends_on: YAML-библиотека.
  - usage_scenario: используется обнаружением и подготовкой выходного документа;
    сохраняет тело, неизвестные поля и Unicode.

Технические единицы: настройка `slog`, версия, CPU profiler и инструменты отчётов
имеют собственные небольшие пакеты и сценарии. Файловые адаптеры чтения
реализуют узкие доменные порты; проверки этих адаптеров используют временные
директории. Подмена адаптеров для unit-тестов не требует переменных окружения.

## Совместимость и принятые уточнения

Источником правил служат код и тесты Python. Публичные команды и конфигурация
сохраняются, а обнаруженные расхождения фиксируются явно.

| Область | Решение для Go |
| --- | --- |
| Команда | `sync`; существующие длинные и короткие флаги |
| Выбор источника | Явный `--config`, затем `--type` + `--path`, затем `ai-skills.yaml` |
| Пути | Относительно конфигурационного файла; прямой CLI-режим — относительно cwd |
| Типы | `local`, `github`; `auto`/`flat`/`directory` как устаревшие local aliases |
| GitHub | Ветка/тег, один или несколько subpaths; clone с archive fallback |
| Targets | Строка или именованные цели, `for_each`, defaults для default/claude |
| Ссылки | Правила разрешения и исключений из Python; адаптация для каждой цели |
| Orphans | Удаление только директорий с `.ai-skills-managed` |
| Старые маркеры | Распознавать маркер Python; иную версию отпечатка считать причиной обновления |
| Ошибки проверки | Собирать ошибки; начинать применение только после успешной проверки всех целей |
| Dry-run | Не изменять цели; получение GitHub-источника может использовать временные файлы |
| Ошибки I/O при применении | Возвращать ошибку; не обещать общую транзакцию между несколькими целями |
| Профилирование | Сохранить `--profile`/`--profile-output`; формат дампа — Go pprof |

В изученном Python-коде `run_sync` не применяет `settings.dry_run`,
`settings.on_conflict`; конфигурационная настройка link validation также
не передаётся в основной pipeline. Поле источника `name` описано в документации,
но не используется `SourceFactory`. В Go реализовано задокументированное
поведение; отличия закреплены сценариями. Для `name` разрешён override только при единственном найденном скиле; неоднозначный override
нескольких скилов считать ошибкой конфигурации.

Новый отпечаток учитывает выходное содержимое и версию преобразования,
включая внешние вложения и изменения отображения ссылок. Это устраняет риск
пропуска обновления из-за хеша только собственных исходных файлов.

## Проверки и связь с документацией

| Группа | Что подтверждают сценарии |
| --- | --- |
| command/config | Флаги, defaults, YAML/JSON, приоритеты, несколько целей, exit codes |
| discovery | Три формата, имена, вложенность, skip folders, дубли, порядок источников |
| tags | Приоритет операторов, отрицание, иерархия, wildcard, синтаксические ошибки |
| links | Markdown/wikilinks/images, anchors, относительные пути, все исключения, ошибки |
| relations | Добавление зависимостей, запрет при выключенной опции, циклы, повторные ссылки |
| planning/transform | SKILL.md layout, внешние файлы, переписывание, Claude, несколько целей |
| incremental/orphans | Повторный запуск, force, изменения вложений, старые маркеры, чужие папки |
| repository | Clone, fallback, HTTP/process failures, cancel, cleanup, archive path traversal |
| filesystem | Реальные файлы во временной директории, ошибки записи, содержимое managed-state |
| orchestration | Ошибки блокируют запись, dry-run не вызывает writer, cleanup при любой ошибке |
| compatibility | Python и Go на общих локальных fixtures; явные проверки согласованных отличий |
| architecture | Направление импортов, отсутствие I/O в чистых функциях, единственная сборка в main |

Каждая функция получает список тест-кейсов по шаблону `workflow-unittest-testplan`
до реализации. Для новых единиц — red/green/refactor. Для сравнения с Python
сначала фиксируется исходный результат на локальных fixtures. Ожидаемые значения
проверяются по содержимому и состоянию, а не только по отсутствию ошибки.

Все выполняемые проверки оформляются как Cucumber/godog-сценарии рядом с пакетом.
Доменные сценарии используют память и подставные узкие порты. Инфраструктурные
сценарии используют временные каталоги, локальный HTTP и контролируемый запуск
процессов; обычный прогон не зависит от доступности GitHub.

Контракт плато: `make unit-test`, `mutation-test`, `test-report`, `test-and-report`;
coverage собирается обязательно, целевой порог — не ниже 80% производственного
кода. Отдельно показываются покрытие бизнес-логики и результат mutation testing.
Отчёты нормализуются в `tmp/result`, исходные отчёты сохраняются в `tmp/report`,
читаемый сайт собирается в `public/`. Проверки сборки: build и vet.

В [README](../../README.md) дан маршрут чтения: быстрый старт → справочник CLI
→ архитектура → индекс возможностей → сценарии → отчёты. Индекс каждой возможности
связывает единицы с файлами реализации и тестами. Диаграммы генерируются
локальным `tools/diagram-renderer` из `depends_on` и фактических Go imports (`make diagrams`).

## Реализация правил плато

| Требование | Применение в CLI |
| --- | --- |
| Inbound adapter | `internal/command` вместо HTTP, по запросу пользователя |
| Composition root | `cmd/ai-skill-manager/main.go`, один SyncService, явные зависимости |
| Config | YAML/JSON и флаги Python-контракта вместо HTTP environment settings |
| Logging | slog, info по умолчанию, debug через флаг; настройка до создания адаптеров |
| Signals | SIGINT/SIGTERM отменяют context операций Git/HTTP/применения |
| Domain ports | SourceProvider, DocumentCodec, StateReader, PlanWriter, SourceSelector, RelationExpander, SyncPlanner; fs.FS для bounded source tree |
| Pure transformations | tags/transform не выполняют файловые или сетевые операции |
| Conformance testing | godog рядом с пакетами, coverage >=80%, mutation и public reports |
| Runtime | Одноразовый CLI; сервер, health endpoint, HTTP shutdown и порты не требуются |

Имена интерфейсов задают роли. `main` связывает их с `sourcing.Manager` (сам
диспетчеризующий на Local/Fetcher), Codec и Store.
Интерфейсы не добавляются чистым функциям только ради подмены.
`Detector.Select` отдельно выполняет выбор по subpaths, tags и name;
это выделенная часть ответственности SkillDetector из согласованной схемы.

### Контракты данных

- `Repository.FS` ограничен корнем источника; внутренние пути используют `/`.
- `Skill.Key` и `Link.Target` включают идентификатор репозитория, поэтому
  одинаковые относительные пути разных источников не смешиваются.
- `SkillMap` строится один раз за запуск, сразу после `RelationExpander.Expand`,
  и переиспользуется без изменений для каждой цели; `OutputLayout` расширяет свою
  копию `SkillMap.Destinations()` путями внешних вложений конкретной цели, не
  трогая общий `SkillMap`.
- `Skill.Files` содержит snapshot входных байтов; план хранит готовые выходные
  байты. PlanApplier не вычисляет бизнес-правила и не читает исходный каталог.
- `StateReader` сообщает наличие, ownership, hash/version и наличие SKILL.md.
- `Repository` сам владеет своей очисткой (`Close`, накапливает `closers` через
  `AddCloser`) вместо того, чтобы `SourceProvider.Acquire` возвращал отдельную
  функцию очистки. `sourcing.Manager` кеширует `*Repository` по `SourceKey` и
  закрывает каждый в `Close(ctx)`, в порядке, обратном получению; владеет этим
  временем жизни вызывающий код (`main`), не `SyncService.Run`. Тот же экземпляр
  реализует `RepositoryLookup.Lookup` (поиск по `Repository.ID` вместо повторного
  `Acquire` по `SourceKey`) — отдельного реестра вроде `SourceMap` для этого
  больше не требуется.
- Сначала строятся все планы; затем выполняется применение. Общей транзакции
  между целями нет. Замена каждого скила использует staging и backup rename.

## Результаты и ограничения

- Go: 164 проходящих Gherkin-сценария; покрытие 84,5% производственных statements
  (после выделения `SkillMap`/`domain/model`, `sourcing.Manager`, и замены
  `SourceMap` на порт `RepositoryLookup` — реализован только `sourcing.Manager`,
  без отдельного реестра).
- Python baseline: 386 passed; одинаковый исходный каталог даёт совпадающие
  выходные пути и семантическое содержимое 546 файлов в двух целях.
- Полный mutation-прогон и race detector выполнены; подробности в [testing](../testing.md).
- [Совместимость и осознанные отличия](compatibility.md).
- [Проверка архитектуры по файлам](audit.md).
- [Диаграммы потока данных sync](diagrams.md) — Mermaid-схемы порядка вызовов
  и контрактов RepositoryLookup/SkillMap, дополняющие автогенерируемый граф пакетов.
- Генератор диаграмм включён в репозиторий, внешняя установка не требуется.
- Пользовательский `ai-skills.yaml` и исходная Python-реализация сохранены.
