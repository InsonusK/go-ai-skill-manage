# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Registering and looking up repositories](features/model.feature) | `SourceMap.Put, SourceMap.Get, SourceMap.Repositories` | ``repositories in order are "a,b"`` | passed |
| [Re-registering a source key updates it without reordering](features/model.feature) | `SourceMap.Put, SourceMap.Get, SourceMap.Repositories` | ``repositories in order are "a,b"`` | passed |
| [Ownership and relative-path rules](features/ownership.feature) | `OwnsPath, RelativePath` | ``path "<path>" is owned "<owned>" and relative is "<relative>"`` | passed |
| [Build a skill map and resolve ownership](features/skillmap.feature) | `NewSkillMap, SkillMap.Owner` | ``owner of "repo" path "a/guide.md" is skill "a" at "a/guide.md"`` | passed |
| [Colliding destinations across skills are rejected](features/skillmap.feature) | `NewSkillMap` | ``building the skill map fails with "output-collision"`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../docs/testing.md)
