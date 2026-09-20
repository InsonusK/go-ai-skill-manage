# AI Skill Manager

CLI на Go для синхронизации каталога AI-скилов в один или несколько проектов.
Поддерживает локальные и GitHub-источники, фильтры тегов, связанные скилы,
переписывание ссылок, Claude и инкрементальные обновления.

## Быстрый старт

Нужны Go 1.26+ и Make. Git используется для удалённых источников;
для GitHub предусмотрена загрузка архива при ошибке clone.

```sh
make build
./bin/aism --version
./bin/aism sync --config ai-skills.yaml --dry-run
./bin/aism sync --config ai-skills.yaml
```

`bin/aism` и `bin/ai-skill-manager` — одинаковые исполняемые файлы.
Python для работы нового CLI не нужен.

Минимальная конфигурация:

```yaml
sources:
  - type: local
    path: ./my-skills
target: .agents/skills
settings:
  remove_orphans: true
```

Пути считаются от каталога конфигурации. Без `--config` используется
`ai-skills.yaml` из текущей директории. Для прямого вызова:

```sh
./bin/aism sync --type local --path ./my-skills --target .agents/skills
```

Каталоги скилов, созданные CLI, помечаются `.ai-skills-managed`.
Повторный запуск пропускает неизменившиеся скилы; `--force` обновляет их.
При `remove_orphans: true` удаляются ранее управляемые скилы, исчезнувшие из выбора.
Личные каталоги без маркера защищены от замены и удаления.

## Как читать проект

1. [Справочник CLI и конфигурации](docs/api/reference.md).
2. [Архитектура и поток выполнения](docs/architecture/go-cli-migration.md).
3. [Индекс возможностей, модулей и тестов](docs/features/sync.md).
4. [Запуск тестов, coverage и mutation testing](docs/testing.md).
5. [Совместимость с Python](docs/architecture/compatibility.md).
6. [Инструкция для AI-агентов](docs/skills/go/ai-skill-manager.skill/ai-skill-manager.skill.md).

Точка сборки зависимостей — [main.go](cmd/ai-skill-manager/main.go).
Входной адаптер — [internal/command](internal/command/doc.go).
Бизнес-правила — [internal/domain/services](internal/domain/services/doc.go).
Работа с диском, Git, HTTP и YAML — [internal/infrastructure](docs/features/sync.md#модули).
Каждый тестируемый пакет содержит `features/`, `test/`, `TESTS.md` и `usecases.md`.

## Разработка

```sh
make lint
make unit-test WITH_CODE_COVERAGE=true
make mutation-test
make test-report
```

Сайт отчёта: `public/index.html`. Единая команда: `make test-and-report`.
Тесты используют память, временные каталоги и локальный HTTP-сервер.
Сеть для тестовых сценариев не нужна; первая загрузка Go-зависимостей требует сеть.

Исходная Python-реализация сохранена в git submodule
[deprecated/ai-skill-manager](deprecated/ai-skill-manager).
