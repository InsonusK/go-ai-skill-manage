# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Ownership and relative-path rules](features/ownership.feature) | `OwnsPath, RelativePath` | ``path "<path>" is owned "<owned>" and relative is "<relative>"`` | passed |
| [Build a skill map and resolve ownership](features/skillmap.feature) | `NewSkillMap, SkillMap.Owner` | ``owner of "repo" path "a/guide.md" is skill "a" at "a/guide.md"`` | passed |
| [Colliding destinations across skills are rejected](features/skillmap.feature) | `NewSkillMap` | ``building the skill map fails with "output-collision"`` | passed |
| [Re-adding the same skill is a no-op](features/catalog.feature) | `Catalog.Add, Catalog.Owner` | ``catalog error contains ""; catalog skills are "a"`` | passed |
| [Duplicate name from a different source errors by default](features/catalog.feature) | `Catalog.Add` | ``catalog error contains "duplicate-name"`` | passed |
| [last_wins replaces the earlier skill with the same name](features/catalog.feature) | `Catalog.Add` | ``catalog skill "a" belongs to repo "other"`` | passed |
| [Owner finds the skill owning a path inside its root](features/catalog.feature) | `Catalog.Owner` | ``catalog owner of "repo" path "a/guide.md" is "a"`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../docs/testing.md)
