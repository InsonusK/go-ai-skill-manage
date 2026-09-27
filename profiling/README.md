# Профилирование `aism sync`

Всё запускается из этой папки (`make -C profiling <цель>`), профиль — ещё и
из корня (`make profile`):

```sh
make validate            # проверить скилы, увидеть дерево проблем
make sync                # проверить и записать в out/<mode>/
make profile             # cold + warm sync с профилями (run.sh)
make report              # CPU и память последнего профиля; RUN=warm — второго прогона
make web                 # CPU-профиль в веб-интерфейсе pprof (порт 8080)
make python              # старый Python-CLI на том же конфиге, dry run
make clean               # удалить out/
```

`MODE=local` (по умолчанию) — локальная копия `ai-skills` (клонируется один
раз в `sources/`); `MODE=github` — репозиторий скачивается на каждом прогоне.

Скрипт собирает `bin/aism` и дважды запускает `sync` с `--profile`:
**cold** — в пустые target (все скилы `create`), **warm** — поверх
записанного cold-прогоном (все скилы `update`). Для каждого прогона в
`profiling/out/<mode>/` остаются:

| Файл | Что это |
| --- | --- |
| `<run>.cpu.prof` | CPU-профиль: `go tool pprof -http=:8080 profiling/out/local/cold.cpu.prof` |
| `<run>.mem.prof` | Профиль живой памяти (heap) в конце прогона |
| `<run>.log` | stdout и stderr `aism`, строка `msg=memory` — итоги по памяти |

В консоль выводятся время, `Synced N skill(s)`, итоги по памяти
(`total_alloc_mb` — всего выделено, `sys_mb` — взято у ОС, близко к пику,
`num_gc` — сборок мусора) и 15 самых затратных функций по CPU.

| Файл | Для чего |
| --- | --- |
| `ai-skill.yaml` | режим `github`: источник — репозиторий на GitHub, target — `profiling/.agents`, `profiling/.claude` |
| `ai-skill.local.yaml` | режим `local`: те же `subpath` из `profiling/sources/ai-skills` (клонируется один раз), target — `profiling/out/local/` |

Списки `subpath` в двух файлах держите одинаковыми. Чтобы обновить локальную
копию, удалите `profiling/sources/`. `profiling/out/` и `profiling/sources/`
в git не попадают.
