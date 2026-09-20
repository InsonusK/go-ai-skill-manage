# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Defaults for a local source](features/config.feature) | `Parse, Resolve` | ``the configuration is`` | passed |
| [Named targets and global adapters](features/config.feature) | `Parse, Resolve` | ``the configuration is`` | ✅ covered |
| [Unknown adapters fail](features/config.feature) | `Parse, Resolve` | ``the config error contains "unknown adapter"`` | ✅ covered |
| [Invalid settings fail](features/config.feature) | `Parse, Resolve` | ``the config error contains "dry_run must be a boolean"`` | passed |
| [JSON configuration is accepted](features/config.feature) | `Parse, Resolve` | ``the configuration is`` | ✅ covered |
| [Configuration rejects invalid public values](features/config.feature) | `Parse, Resolve` | ``the config error contains "<error>"`` | ✅ covered |
| [Root and settings targets conflict](features/config.feature) | `Parse` | ``the config error contains "target cannot be defined both"`` | ✅ covered |
| [Legacy settings target and source syntax are preserved](features/config.feature) | `Parse, Resolve` | ``the configuration is`` | ✅ covered |
| [Temporary directory resolves from the configuration directory](features/config.feature) | `Parse, Resolve` | ``the configuration is`` | ✅ covered |
| [Non-string temporary directory fails](features/config.feature) | `Parse` | ``the config error contains "temp_dir must be a string"`` | ✅ covered |

## Function → Test

| Изменённая функция или ветвь | Сценарий |
| --- | --- |
| `Parse`: `settings.temp_dir` absent, string, non-string | [Defaults for a local source; Temporary directory resolves from the configuration directory; Configuration rejects invalid public values](features/config.feature) |
| `Parse`: root `target` absent, string, named mapping, invalid; legacy `settings.target`; both forms | [Named targets and global adapters; JSON configuration is accepted; Configuration rejects invalid public values; Legacy settings target and source syntax are preserved](features/config.feature) |
| `Resolve`: relative and absolute non-empty `TempDir` | [Temporary directory resolves from the configuration directory](features/config.feature) |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../docs/testing.md)
