# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [All options are parsed without executing integrations](features/arguments.feature) | `Parse` | ``parsed options are`` | passed |
| [Malformed arguments fail parsing](features/arguments.feature) | `Parse` | ``argument error contains "<error>"`` | passed |
| [Sync a local directory from configuration](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "0"; stdout contains "Synced 1 skill"; project file "output/a/SKILL.md" contains "Content"`` | passed |
| [Config dry run does not create targets](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "0"; stdout contains "Dry run"; project path "output" exists "false"`` | passed |
| [Direct mode skips config loading](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "0"; project file "output/a/SKILL.md" contains "Content"`` | passed |
| [Usage and failures have stable exit codes](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "<code>"; console contains "<message>"`` | passed |
| [Named targets share dependencies with different adapters](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "0"; exit code is "0"; exit code is "0"; project file "out/a/SKILL.md" contains "[Read](out/b/SKILL.md#part)"; project file "claude/a/SKILL.md" contains "[Read](claude/b/SKILL.md#part)"; project file "claude/a/SKILL.md" contains "## Metadata"; stdout contains "(unchanged)"; stdout contains "(forced)"`` | passed |
| [Tag filter and name override select one skill](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "0"; project file "out/renamed/SKILL.md" contains "name: renamed"; project path "out/b" exists "false"`` | passed |
| [Last source wins a configured name conflict](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "0"; project file "out/same/SKILL.md" contains "Second"`` | passed |
| [Duplicate names block writes by default](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "1"; console contains "duplicate-name"; project path "out" exists "false"`` | passed |
| [Human directory attachments and external folders are copied](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "0"; project file "out/a/notes.txt" contains "Own"; project file "out/files/assets/nested/info.txt" contains "Info"; project file "out/a/SKILL.md" contains "[own](out/a/notes.txt) [ext](out/files/assets)"`` | passed |
| [Missing subpaths produce an empty selection](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "0"; stdout contains "Synced 0 skill"`` | passed |
| [Whitespace GitHub path is a configuration error](features/command.feature) | `App.Execute, request, PrintResult` | ``exit code is "1"; console contains "--path is required"`` | passed |

## Function → Test

| Изменённая функция или ветвь | Сценарий |
| --- | --- |
| `App.request`: configuration with root `target` | [Sync a local directory from configuration; Named targets share dependencies with different adapters](features/command.feature) |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../docs/testing.md)
