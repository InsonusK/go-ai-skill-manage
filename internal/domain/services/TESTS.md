# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Product domain only depends on domain packages and allowed standard libraries](features/architecture.feature) | `domain package import graph` | ``forbidden domain imports are`` | passed |
| [Dry run and validation gate all writes](features/sync.feature) | `SyncService.Run` | ``writer calls equal "<writes>"; sync error contains "<error>"`` | passed |
| [All target plans are validated before any write](features/sync.feature) | `SyncService.Run` | ``writer calls equal "0"; sync error contains "state unavailable"`` | passed |
| [Configured temporary directory reaches source acquisition](features/sync.feature) | `SyncService.Run` | ``source acquisition temp dir equals "/project/.tmp"`` | ✅ covered |
| [A failing skill detector blocks writes without real discovery](features/sync.feature) | `SyncService.Run` | ``writer calls equal "0"; sync error contains "boom"`` | passed |

## Function → Test

| Изменённая функция или ветвь | Сценарий |
| --- | --- |
| `SyncService.Run`: acquisition options receive request `TempDir` | [Configured temporary directory reaches source acquisition](features/sync.feature) |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../docs/testing.md)
