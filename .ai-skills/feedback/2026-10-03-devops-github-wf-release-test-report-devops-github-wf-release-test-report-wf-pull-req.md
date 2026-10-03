---
status: draft
kind: bug
skill: devops-github-wf-release-test-report
source:
  type: github
  path: https://github.com/InsonusK/ai-skills.git
  tree: master
commit: defd3f9cdf07a06d2a1b5a9abdd003745d5d31cb
skill_path: skills/devops/workflows/devops-github-wf-release-test-report.skill
created_at: 2026-10-03T16:22:13Z
---

# devops-github-wf-release-test-report: противоречие с wf-pull-request — мутационные тесты на PR

Причина обращения: агент сверял репозиторий со скилами `devops-*` и нашёл, что два скила требуют противоположного.

`devops-github-wf-release-test-report` в нескольких местах утверждает, что мутационные тесты входят в проверку PR:

- `# Core Principle`: «The merge gate — `make unit-test` and the delta-scoped `make mutation-test` — lives in devops-github-wf-pull-request».
- `# Workflow`, шаг 3: «the PR-gate workflow already enforced the threshold before this code reached `master`».
- Правило `Keep the mutation-test job report-only`, Fix: «the PR-gate workflow already enforced the threshold pre-merge».
- `templates/release-test-report.example.md`, комментарий в job `mutation-test`: «devops-github-wf-pull-request's mutation-test job already enforced the threshold».

`devops-github-wf-pull-request` это прямо запрещает: правило MUST `Never gate a PR on mutation testing`, пункт чек-листа «No mutation-testing job exists in this workflow», а в примере `pull-request.example.md` job `mutation-test` отсутствует.

Последствие: агент, читающий только `release-test-report`, считает, что порог мутаций уже проверен до слияния, и либо добавляет `mutation-test` в PR-workflow, либо полагается на проверку, которой нет.

Предлагаемое исправление: в `devops-github-wf-release-test-report` убрать все четыре упоминания проверки порога на PR и написать, что мутационные тесты запускаются только здесь и только как отчёт. Если же верно обратное (delta-прогон на PR нужен), тогда править `devops-github-wf-pull-request` и его пример.
