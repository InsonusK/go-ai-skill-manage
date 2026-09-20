# Матрица тестов

Все строки ниже имеют исполняемые Gherkin-сценарии; состояние текущего baseline — **passed**.
Точные входы, значения ожиданий и Examples находятся по ссылкам.

| Сценарий | Проверяемая единица | Наблюдаемый результат | Статус |
| --- | --- | --- | --- |
| [Git clone arguments preserve branch and URL boundaries](features/git.feature) | `GitCloner.Clone` | ``git arguments are`` | passed |
| [Default branch is master](features/git.feature) | `GitCloner.Clone` | ``git arguments are`` | passed |
| [Git process executes a real local command](features/git.feature) | `GitProcess.Run` | ``git process error contains ""`` | passed |
| [Git process propagates errors](features/git.feature) | `GitProcess.Run` | ``git process error contains "git:"`` | passed |
| [Git process respects cancellation](features/git.feature) | `GitProcess.Run` | ``git process error contains "context canceled"`` | passed |
| [Local source paths remain relative to source root](features/repository.feature) | `Local.Acquire` | ``acquired file "skills/a.skill.md" equals "content"`` | passed |
| [A single-file local source is remembered on the repository](features/repository.feature) | `Local.Acquire` | ``acquired repository single file equals "a.skill.md"; acquired file "a.skill.md" equals "content"`` | passed |
| [Failed clone falls back to GitHub archive](features/repository.feature) | `Fetcher.Acquire` | ``acquired file "skills/a.skill.md" equals "content"; archive calls equal "1"`` | passed |
| [Successful clone needs no archive](features/repository.feature) | `Fetcher.Acquire` | ``acquired file "skills/a.skill.md" equals "content"; archive calls equal "0"`` | passed |
| [Archive traversal is rejected](features/repository.feature) | `Archive.Fetch` | ``repository error contains "unsafe archive"`` | passed |
| [Archive response is extracted](features/repository.feature) | `Archive.Fetch` | ``acquired file "skills/a.skill.md" equals "content"`` | passed |
| [GitHub workspace uses requested temporary directory](features/repository.feature) | `Fetcher.Acquire` | ``acquired repository root matches "aism-source-*/repo" below temporary directory`` | ✅ covered |
| [Acquiring the same source twice reuses the cached repository](features/manager.feature) | `SourceManager.Acquire` | ``the provider was called "1" times; both acquisitions returned the same repository`` | passed |
| [Acquiring different sources calls the provider for each](features/manager.feature) | `SourceManager.Acquire` | ``the provider was called "2" times`` | passed |
| [Unknown source type is rejected](features/manager.feature) | `SourceManager.Acquire` | ``acquiring fails with "unknown source type"`` | passed |
| [Close runs every acquired repository's Close in reverse order](features/manager.feature) | `SourceManager.Close` | ``repositories were closed in order "b,a"`` | passed |

## Function → Test

| Изменённая функция или ветвь | Сценарий |
| --- | --- |
| `Provider.Acquire`: GitHub options forwarding | [GitHub workspace uses requested temporary directory](features/repository.feature) |
| `Fetcher.Acquire`: configured temp-directory root | [GitHub workspace uses requested temporary directory](features/repository.feature) |

[Условия вызовов](usecases.md) · [Step definitions](test/) · [Общие правила запуска](../../../docs/testing.md)
