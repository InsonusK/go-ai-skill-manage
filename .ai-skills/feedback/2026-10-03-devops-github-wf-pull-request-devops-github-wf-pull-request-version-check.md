---
status: draft
kind: bug
skill: devops-github-wf-pull-request
source:
  type: github
  path: https://github.com/InsonusK/ai-skills.git
  tree: master
commit: defd3f9cdf07a06d2a1b5a9abdd003745d5d31cb
skill_path: skills/devops/workflows/devops-github-wf-pull-request.skill
created_at: 2026-10-03T16:22:13Z
---

# devops-github-wf-pull-request: в примере version-check не падает и нет установки стека

Причина обращения: агент сверял `.github/workflows/pull-request.yml` проекта с примером скила; проекту пришлось отступить от примера в двух местах, чтобы workflow делал то, что описано в тексте скила.

1. **`version-check` в примере никогда не падает.** `# Workflow`, шаг 2: «it fails unless the PR's version is strictly greater than `master`'s». В `templates/pull-request.example.md` job `version-check` только вызывает `./.github/actions/check-version`. Этот action (например, `devops-github-action-check-version-in-go`) лишь выставляет выход `bumped` и завершается успешно. В итоге job зелёный и при неподнятой версии, `report` его пропускает, PR в master сливается без поднятия версии. Нужно: добавить в пример шаг после action —

   ```yaml
   - name: Fail unless the version was bumped
     if: steps.version.outputs.bumped != 'true'
     run: |
       echo "Version ${{ steps.version.outputs.current }} must be greater than on master" >&2
       exit 1
   ```

   и пункт чек-листа «`version-check` падает при `bumped != 'true'`».

2. **В job `unit-test` нет шага установки стека.** В примере сразу `run: make unit-test`. В `release-test-report.example.md` для этого есть закомментированная заготовка `# - name: Set up {stack}`, здесь её нет. На `ubuntu-latest` тесты идут на предустановленной версии инструментов, а не на версии проекта (для Go — не на версии из `go.mod`), и без кэша зависимостей, хотя SHOULD скила просит кэшировать через `actions/setup-*`. Нужно: добавить в пример такую же заготовку `Set up {stack}` и упомянуть её в тексте («единственный шаг, который меняется между стеками»).
