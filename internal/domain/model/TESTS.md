# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Ownership and relative-path rules](features/ownership.feature) | `OwnsPath, RelativePath` | ``path "<path>" is owned "<owned>" and relative is "<relative>"`` | passed |
| [Re-adding the same skill is a no-op](features/catalog.feature) | `SkillCatalog.GetOrAdd, SkillCatalog.Owner` | ``catalog error contains ""; catalog skills are "a"`` | passed |
| [Duplicate name from a different source errors by default](features/catalog.feature) | `SkillCatalog.GetOrAdd` | ``catalog error contains "duplicate-name"`` | passed |
| [last_wins replaces the earlier skill with the same name](features/catalog.feature) | `SkillCatalog.GetOrAdd` | ``catalog skill "a" belongs to repo "other"`` | passed |
| [Owner finds the skill owning a path inside its root](features/catalog.feature) | `SkillCatalog.Owner` | ``catalog owner of "repo" path "a/guide.md" is "a"`` | passed |
| [GetOrAdd indexes a skill's own and nested file destinations](features/catalog.feature) | `SkillCatalog.GetOrAdd, SkillCatalog.Destination` | ``catalog destination of "repo" path "a/notes.md" is "guide" at "guide/notes.md"`` | passed |
| [last_wins re-indexes destinations onto the newer skill's source](features/catalog.feature) | `SkillCatalog.GetOrAdd, SkillCatalog.Destination` | ``catalog destination of "other" path "a/SKILL.md" is "a" at "a/SKILL.md"`` | passed |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../docs/testing.md)
