# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Created skills receive state and skipped copies remain untouched](features/filesystem.feature) | `Store.Apply, Store.Snapshot` | ``target file "sample/SKILL.md" equals "body"; target file "sample/SKILL.md" equals "body"; state for "sample" is managed with hash "hash"`` | passed |
| [Update replaces obsolete managed files](features/filesystem.feature) | `Store.Apply, Store.Snapshot` | ``target file "sample/SKILL.md" equals "new"; target path "sample/obsolete.txt" exists "false"`` | passed |
| [Removal only deletes managed directories](features/filesystem.feature) | `Store.Apply, Store.Snapshot` | ``target path "sample" exists "false"`` | passed |
| [Unmanaged directories cannot be removed](features/filesystem.feature) | `Store.Apply, Store.Snapshot` | ``filesystem error contains "unmanaged"; target path "manual" exists "true"`` | passed |
| [Escaping output paths are rejected before writing](features/filesystem.feature) | `Store.Apply, Store.Snapshot` | ``filesystem error contains "unsafe"`` | passed |
| [Existing files cannot be planned as new skills](features/filesystem.feature) | `Store.Snapshot` | ``snapshot marks "sample" as existing unmanaged entry`` | passed |
| [Symlinks cannot be planned as new skills](features/filesystem.feature) | `Store.Snapshot` | ``snapshot marks "sample" as existing unmanaged entry`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../docs/testing.md)
