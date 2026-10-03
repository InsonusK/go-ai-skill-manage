---
status: sent
kind: bug
skill: devops-github-wf-release-test-report
source:
  type: github
  path: https://github.com/InsonusK/ai-skills.git
  tree: master
commit: 44f349f8843c058d0b38eaf415f976e42d30fb7c
skill_path: skills/devops/workflows/devops-github-wf-release-test-report.skill
created_at: 2026-10-03T15:56:12Z
issue_url: https://github.com/InsonusK/ai-skills/issues/166
sent_at: 2026-10-03T16:04:17Z
---

# devops-github-wf-release-test-report: бейджи в README не добавляются, а пример ссылается на несуществующие файлы

Причина обращения: агент, применяя скил, обратил внимание, что в `SKILL.md` недостаточно явно указано, что нужно добавить бейджи по тестированию в README.

1. **Нет шага «добавить бейджи в README».** Раздел `# Workflow` заканчивается на deploy. Правило `Source badges only from this workflow's output` описывает только, откуда бейджи брать, а пункт чек-листа проверяет их вид, а не наличие. Сниппет лежит в конце `templates/release-test-report.example.md` после YAML. Нужно: отдельное MUST «добавь в README четыре бейджа из примера», шаг 6 в `# Workflow` и пункт чек-листа «бейджи есть в README».

2. **Неверное имя файла бейджа мутаций.** В примере `mutation-badge.json`, а по контракту `solution-conformance-testing` («Public site output») файл называется `<label>-badge.json` с меткой `mutation score`, то есть `mutation-score-badge.json`. Бейдж из примера всегда получает 404.

3. **Неверная ссылка на отчёт мутаций.** В примере `mutation/reports/mutation-report.html`; Go-реализация (`solution-conformance-testing-in-go`, `tools/test_report`) кладёт отчёт в `mutation/index.html`. Путь зависит от стека — либо зафиксировать его в контракте, либо ссылаться на `mutation/`.

4. **Не сказано про включение Pages.** Без Settings → Pages → Source: GitHub Actions job `deploy` падает; это стоит добавить как предусловие.

5. **Ручной запуск, скорее всего, ничего не делает.** При `workflow_dispatch` на master `dorny/paths-filter` сравнивает ветку с ней же, изменений нет, все jobs пропускаются. Не проверял на прогоне. Вероятное исправление: `relevant: ${{ github.event_name == 'workflow_dispatch' || ... }}`.
