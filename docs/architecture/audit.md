# Архитектурный аудит

Проверка реализации по `architect-validator` и согласованной
[декомпозиции](go-cli-migration.md). База — `plateau-http-service`; входной
HTTP-адаптер заменён командой CLI по запросу пользователя.

## Применимые правила

| Область | Правило и подтверждение |
| --- | --- |
| R: repository | `solution-go-repository-structure`: go.mod, один cmd entry point, VERSION, internal; сборка и vet |
| C: composition | main создаёт один SyncService и все реальные адаптеры; logger до адаптеров; signals → context; profile закрывается до os.Exit |
| I: inbound | `internal/command`: аргументы, загрузка запроса, вызов use case, вывод и exit code; нет обхода файлов скила или правил планирования |
| D: domain | `solution-go-domain-logic`: правила в services; инфраструктура доступна через узкие порты; нет импортов command/config/infrastructure/HTTP/process |
| P: ports | `solution-go-domain-ports`: SourceProvider, DocumentCodec, StateReader, PlanWriter принадлежат домену; fs.FS передаёт ограниченное дерево |
| A: adapters | Файловые, Git/HTTP и YAML детали сосредоточены в infrastructure; тесты используют временные каталоги, fake Runner, локальный HTTP |
| L: logging | `solution-go-app-logging`: единственная настройка slog.SetDefault в logging.Init; debug выключен; logger не хранится в доменном сервисе |
| T: tests | `solution-go-conformance-testing`: godog, один TestFeatures на пакет, features рядом, context у шагов, ожидаемые данные в сценариях |
| Q: assertions | `no-test-theater`: проверяются файлы/байты/состояние/ошибки/вызовы |
| U: tools | Четыре стандартные Make-цели; coverage всегда; нормализованные JSON, native reports, public; исключены tools/gen/test/сторонние примеры |

Архитектурный Gherkin-сценарий разбирает Go AST: проверяет импорты всего
домена и `context.Context` первым аргументом публичных методов сервисов.
Чистые функции и методы моделей не являются сервисными методами.

При аудите исправлены два нарушения: отсутствующий context у методов
Detector/Catalog и прямое поле logger в SyncService. Дополнительно выделены
файлы config по ответственности; в filesystem Snapshot учитывает занятые
имена файлов и symlinks, чтобы ошибка обнаруживалась до применения планов.

## Адаптация HTTP-плато

- HTTP port, health endpoint, server lifecycle и environment-based HTTP config
  не применяются к выбранному пользователем CLI.
- Публичный контракт Python сохраняет YAML/JSON и `--debug`; поэтому нет
  дополнительных обязательных LOG_LEVEL/HTTP_LISTEN_PORT параметров.
- Main использует отмену context вместо server.Shutdown.
- `make diagrams` — генерация документации; тестовые/lifecycle цели плато сохранены.
- Внешний diagram-renderer отсутствовал; локальная реализация читает depends_on
  и реальные импорты, выдаёт Canvas/SVG без ручного рисования.

## Проверенные файлы

Таблица перечисляет все Go-файлы реализации и тестовой инфраструктуры.
Правила относятся к роли файла; интеграционные ограничения и измеренные
пробелы тестов приведены в [testing](../testing.md) и [compatibility](compatibility.md).

| Файл | Правила | Результат |
| --- | --- | --- |
| [cmd/ai-skill-manager/main.go](../../cmd/ai-skill-manager/main.go) | R, C, L | проверен |
| [internal/command/arguments.go](../../internal/command/arguments.go) | I, R | проверен |
| [internal/command/configuration.go](../../internal/command/configuration.go) | I, R | проверен |
| [internal/command/format.go](../../internal/command/format.go) | I, R | проверен |
| [internal/command/sync.go](../../internal/command/sync.go) | I, R | проверен |
| [internal/command/test/arguments_steps_test.go](../../internal/command/test/arguments_steps_test.go) | T, Q | проверен |
| [internal/command/test/command_steps_test.go](../../internal/command/test/command_steps_test.go) | T, Q | проверен |
| [internal/command/test/runner_test.go](../../internal/command/test/runner_test.go) | T, Q | проверен |
| [internal/config/config.go](../../internal/config/config.go) | R, I | проверен |
| [internal/config/resolve.go](../../internal/config/resolve.go) | R, I | проверен |
| [internal/config/source.go](../../internal/config/source.go) | R, I | проверен |
| [internal/config/targets.go](../../internal/config/targets.go) | R, I | проверен |
| [internal/config/test/config_steps_test.go](../../internal/config/test/config_steps_test.go) | T, Q | проверен |
| [internal/config/test/runner_test.go](../../internal/config/test/runner_test.go) | T, Q | проверен |
| [internal/config/validation.go](../../internal/config/validation.go) | R, I | проверен |
| [internal/config/values.go](../../internal/config/values.go) | R, I | проверен |
| [internal/domain/interfaces/source.go](../../internal/domain/interfaces/source.go) | P | проверен |
| [internal/domain/interfaces/discovery.go](../../internal/domain/interfaces/discovery.go) | P | проверен |
| [internal/domain/interfaces/document.go](../../internal/domain/interfaces/document.go) | P | проверен |
| [internal/domain/interfaces/state.go](../../internal/domain/interfaces/state.go) | P | проверен |
| [internal/domain/model/model.go](../../internal/domain/model/model.go) | D, P | проверен |
| [internal/domain/services/discovery/catalog.go](../../internal/domain/services/discovery/catalog.go) | D, P | проверен |
| [internal/domain/services/discovery/detector.go](../../internal/domain/services/discovery/detector.go) | D, P | проверен |
| [internal/domain/services/discovery/inventory.go](../../internal/domain/services/discovery/inventory.go) | D, P | проверен |
| [internal/domain/services/discovery/source.go](../../internal/domain/services/discovery/source.go) | D, P | проверен |
| [internal/domain/services/discovery/test/discovery_steps_test.go](../../internal/domain/services/discovery/test/discovery_steps_test.go) | T, Q | проверен |
| [internal/domain/services/discovery/test/runner_test.go](../../internal/domain/services/discovery/test/runner_test.go) | T, Q | проверен |
| [internal/domain/services/links/exclusion.go](../../internal/domain/services/links/exclusion.go) | D, P | проверен |
| [internal/domain/services/links/extract.go](../../internal/domain/services/links/extract.go) | D, P | проверен |
| [internal/domain/services/links/resolve.go](../../internal/domain/services/links/resolve.go) | D, P | проверен |
| [internal/domain/services/links/test/links_steps_test.go](../../internal/domain/services/links/test/links_steps_test.go) | T, Q | проверен |
| [internal/domain/services/links/test/runner_test.go](../../internal/domain/services/links/test/runner_test.go) | T, Q | проверен |
| [internal/domain/services/planning/fingerprint.go](../../internal/domain/services/planning/fingerprint.go) | D, P | проверен |
| [internal/domain/services/planning/layout.go](../../internal/domain/services/planning/layout.go) | D, P | проверен |
| [internal/domain/services/planning/planner.go](../../internal/domain/services/planning/planner.go) | D, P | проверен |
| [internal/domain/services/planning/test/planning_steps_test.go](../../internal/domain/services/planning/test/planning_steps_test.go) | T, Q | проверен |
| [internal/domain/services/planning/test/runner_test.go](../../internal/domain/services/planning/test/runner_test.go) | T, Q | проверен |
| [internal/domain/services/relations/expand.go](../../internal/domain/services/relations/expand.go) | D, P | проверен |
| [internal/domain/services/relations/test/relations_steps_test.go](../../internal/domain/services/relations/test/relations_steps_test.go) | T, Q | проверен |
| [internal/domain/services/relations/test/runner_test.go](../../internal/domain/services/relations/test/runner_test.go) | T, Q | проверен |
| [internal/domain/services/sync.go](../../internal/domain/services/sync.go) | D, P | проверен |
| [internal/domain/services/tags/tags.go](../../internal/domain/services/tags/tags.go) | D, P | проверен |
| [internal/domain/services/tags/test/runner_test.go](../../internal/domain/services/tags/test/runner_test.go) | T, Q | проверен |
| [internal/domain/services/tags/test/tags_steps_test.go](../../internal/domain/services/tags/test/tags_steps_test.go) | T, Q | проверен |
| [internal/domain/services/test/architecture_steps_test.go](../../internal/domain/services/test/architecture_steps_test.go) | T, Q | проверен |
| [internal/domain/services/test/runner_test.go](../../internal/domain/services/test/runner_test.go) | T, Q | проверен |
| [internal/domain/services/test/sync_steps_test.go](../../internal/domain/services/test/sync_steps_test.go) | T, Q | проверен |
| [internal/domain/services/transform/claude.go](../../internal/domain/services/transform/claude.go) | D, P | проверен |
| [internal/domain/services/transform/links.go](../../internal/domain/services/transform/links.go) | D, P | проверен |
| [internal/domain/services/transform/test/runner_test.go](../../internal/domain/services/transform/test/runner_test.go) | T, Q | проверен |
| [internal/domain/services/transform/test/transform_steps_test.go](../../internal/domain/services/transform/test/transform_steps_test.go) | T, Q | проверен |
| [internal/infrastructure/document/codec.go](../../internal/infrastructure/document/codec.go) | A, P | проверен |
| [internal/infrastructure/document/test/document_steps_test.go](../../internal/infrastructure/document/test/document_steps_test.go) | T, Q | проверен |
| [internal/infrastructure/document/test/runner_test.go](../../internal/infrastructure/document/test/runner_test.go) | T, Q | проверен |
| [internal/infrastructure/filesystem/apply.go](../../internal/infrastructure/filesystem/apply.go) | A, P | проверен |
| [internal/infrastructure/filesystem/store.go](../../internal/infrastructure/filesystem/store.go) | A, P | проверен |
| [internal/infrastructure/filesystem/test/filesystem_steps_test.go](../../internal/infrastructure/filesystem/test/filesystem_steps_test.go) | T, Q | проверен |
| [internal/infrastructure/filesystem/test/runner_test.go](../../internal/infrastructure/filesystem/test/runner_test.go) | T, Q | проверен |
| [internal/infrastructure/repository/archive.go](../../internal/infrastructure/repository/archive.go) | A, P | проверен |
| [internal/infrastructure/repository/fetch.go](../../internal/infrastructure/repository/fetch.go) | A, P | проверен |
| [internal/infrastructure/repository/git.go](../../internal/infrastructure/repository/git.go) | A, P | проверен |
| [internal/infrastructure/repository/local.go](../../internal/infrastructure/repository/local.go) | A, P | проверен |
| [internal/infrastructure/repository/test/git_steps_test.go](../../internal/infrastructure/repository/test/git_steps_test.go) | T, Q | проверен |
| [internal/infrastructure/repository/test/repository_steps_test.go](../../internal/infrastructure/repository/test/repository_steps_test.go) | T, Q | проверен |
| [internal/infrastructure/repository/test/runner_test.go](../../internal/infrastructure/repository/test/runner_test.go) | T, Q | проверен |
| [internal/logging/logger.go](../../internal/logging/logger.go) | L | проверен |
| [internal/logging/test/logging_steps_test.go](../../internal/logging/test/logging_steps_test.go) | T, Q | проверен |
| [internal/logging/test/runner_test.go](../../internal/logging/test/runner_test.go) | T, Q | проверен |
| [internal/profiling/profile.go](../../internal/profiling/profile.go) | R, C | проверен |
| [internal/profiling/test/profiling_steps_test.go](../../internal/profiling/test/profiling_steps_test.go) | T, Q | проверен |
| [internal/profiling/test/runner_test.go](../../internal/profiling/test/runner_test.go) | T, Q | проверен |
| [internal/version/version.go](../../internal/version/version.go) | R, C | проверен |
| [tools/diagram-renderer/main.go](../../tools/diagram-renderer/main.go) | U | проверен |
| [tools/normalize_mutation/main.go](../../tools/normalize_mutation/main.go) | U | проверен |
| [tools/normalize_unittest/main.go](../../tools/normalize_unittest/main.go) | U | проверен |
| [tools/test_report/main.go](../../tools/test_report/main.go) | U | проверен |
| [tools/testsupport/suite.go](../../tools/testsupport/suite.go) | U | проверен |

## Выполненные проверки

- `make unit-test WITH_CODE_COVERAGE=true`: 133 / 133; coverage 82,7%.
- `make mutation-test`: score 84,2%; 416 killed, 57 survived, 7 timed out, 14 uncovered.
- `make build`, `make lint`, `go test -race ./...`: успешны.
- Cross-build: Windows amd64 и macOS arm64 успешны; Linux amd64 исполнялся.
- Binary smoke: 61 скил / 2 цели, dry-run сохраняет хеши всех файлов цели;
  проверены debug, gzip CPU profile, version 2.0.0 и exit code 2.
- Ссылки документов на файлы, gofmt и `git diff --check`: без ошибок.

## Границы заключения

Проверка структуры не доказывает отсутствие всех ошибок. Mutation report
содержит surviving и uncovered мутации; общий rollback между целями и
координация параллельных писателей не реализованы и не заявлены.
Порог coverage применяется к производственному коду в целом; некоторые
инфраструктурные ветви ошибок покрыты слабее бизнес-правил.

Исполняемый CLI проверен на Linux. Windows/macOS проверяются кросс-компиляцией;
это не является проверкой исполнения на этих ОС.
