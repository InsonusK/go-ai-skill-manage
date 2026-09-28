# Отзыв о скиле: issue в его источник

Агент, работая по скилу, замечает ошибку (битая ссылка, устаревшая команда)
или видит, чем скил можно улучшить. Команда `feedback` превращает такое
замечание в issue в GitHub-репозитории, откуда скил был синхронизирован.

Issue — публичное действие от имени пользователя, а текст пишет агент по
материалам вашего проекта. Поэтому отправка разделена на три шага и
**отправляет только пользователь**:

1. **Черновик.** Агент (или вы) вызывает `feedback draft`: черновик
   записывается файлом в проект, ничего не отправляется.
2. **Проверка.** Вы читаете файл и при необходимости правите его: убираете
   код и данные проекта, которые не должны уйти в чужой репозиторий.
3. **Отправка.** Вы сами запускаете `feedback send` в терминале: команда
   показывает итоговый issue целиком и отправляет его только после ответа
   `y`. Без терминала (так запускает команды агент) `send` ничего не
   отправляет. Через [MCP](#mcp-сервер-для-агента) агент просит отправку
   сам, но решаете вы: Claude Code показывает issue в диалоге и отправляет
   его, только если вы отметили «Open this issue».

Сначала — [настройка](#настройка): `gh` и MCP-сервер.

## Настройка

Один раз на машину — `gh`, один раз на проект — MCP-сервер. После этого агент
в Claude Code сам пишет черновики и просит отправку, а вы подтверждаете её в
диалоге.

### 1. `aism` и синхронизированные скилы

`aism` должен быть в `PATH` ([установка](../README.md#установка)), а скилы —
синхронизированы: источник скила `feedback` берёт из маркера, который
пишет `sync`.

```sh
aism --version
aism sync
```

Скилы, синхронизированные версией без поля `commit` в маркере, — просто
запустите `sync` ещё раз.

### 2. Установите GitHub CLI

| Система | Команда |
| --- | --- |
| macOS | `brew install gh` |
| Windows | `winget install --id GitHub.cli` |
| Linux | по [официальной инструкции](https://github.com/cli/cli/blob/trunk/docs/install_linux.md) (репозиторий пакетов GitHub; пакеты дистрибутивов бывают старыми) |
| Dev container | фича `"ghcr.io/devcontainers/features/github-cli:1": {}` в `devcontainer.json` (в этом репозитории уже есть) |

```sh
gh --version
```

### 3. Войдите в GitHub

```sh
gh auth login
```

Выберите `GitHub.com`, протокол — любой (он нужен `gh` для `git`, на issue
не влияет), вход — `Login with a web browser`: `gh` покажет одноразовый код
и адрес `https://github.com/login/device`, откройте его в браузере на любой
машине и введите код. Так же это работает в контейнере и по SSH.

```sh
gh auth status
```

```text
github.com
  ✓ Logged in to github.com account <you> (keyring)
  - Token scopes: 'gist', 'read:org', 'repo'
```

`gh` хранит токен в системном хранилище секретов (keyring). Где его нет
(контейнер, сервер без графики), `gh` сохраняет токен в файл
`~/.config/gh/hosts.yml` и предупреждает об этом — не копируйте этот файл
в образы и репозитории. Прав токена `gh` (`repo`) достаточно, чтобы
открыть issue в любом публичном репозитории. Без `gh` — свой токен, см.
[доступ к GitHub](#настройка-доступа-к-github).

### 4. Подключите MCP-сервер к Claude Code

Для всей команды — файл `.mcp.json` в корне проекта, его коммитят:

```json
{
  "mcpServers": {
    "ai-skills": {
      "command": "aism",
      "args": ["mcp"]
    }
  }
}
```

или то же командой:

```sh
claude mcp add --scope project ai-skills -- aism mcp
```

Только для себя, без файла в репозитории:

```sh
claude mcp add ai-skills -- aism mcp
```

- Сервер из `.mcp.json` Claude Code запускает только после вашего
  одобрения: при первом запуске `claude` в проекте он спросит, доверять ли
  папке и этому серверу. До одобрения `claude mcp list` показывает
  `⏸ Pending approval`.
- Корень проекта сервер получает от Claude Code (`CLAUDE_PROJECT_DIR`): от
  него ищется `ai-skills.yaml`, черновики лежат в `.ai-skills/feedback/`
  рядом. Конфиг в другом месте — `"args": ["mcp", "-c", "path/to/ai-skills.yaml"]`
  (путь от корня проекта). Конфиг перечитывается на каждый вызов,
  перезапускать сервер после правки не нужно.
- Нет `aism` в `PATH` — укажите в `command` полный путь.
- Токен сервер ищет так же, как CLI: с `gh` ничего не нужно. Свой токен —
  через `"env": {"GH_TOKEN": "${GH_TOKEN:-}"}`: значение подставляется из
  окружения, в котором запущен `claude`, сам токен в файл не попадает;
  `:-` — чтобы без переменной сервер получил пустое значение и перешёл к
  `gh`, а не строку `${GH_TOKEN}`.

> Не настраивайте [хук `Elicitation`](https://code.claude.com/docs/en/hooks#elicitation),
> который отвечает на диалоги автоматически, для этого сервера: тогда issue
> уйдёт без вашей проверки.

### 5. Проверьте

```sh
claude mcp list           # ai-skills: aism mcp - ✔ Connected
```

В сессии Claude Code команда `/mcp` показывает сервер `ai-skills` с
инструментами `feedback_draft` и `feedback_submit`. Попросите агента,
например: «напиши отзыв о скиле guide: якорь #setup никуда не ведёт».
Агент создаст черновик в `.ai-skills/feedback/` и вызовет
`feedback_submit`; Claude Code покажет диалог с итоговым issue. Для первой
проверки нажмите decline — ничего не уйдёт, а цепочка проверена целиком.

## Как команда находит репозиторий

Агенту не нужно знать, откуда скил. `sync` кладёт в папку каждого скила
маркер `.ai-skills-managed`, где записаны источник, коммит и путь скила в
источнике. `feedback draft --skill guide` ищет папку `guide` в target из
конфигурации (по порядку, первая с маркером) и берёт источник оттуда.

- Скил должен быть синхронизирован. Нет его ни в одном target —
  `skill "guide" is in none of the targets ...: run sync first`.
  Маркер старого формата — тоже запустите `sync`.
- Скил из `local`-источника — ошибка `comes from the local source ...: edit
  it there by hand`: у локальной папки нет трекера, правьте скил в ней.
- Сейчас поддерживаются только источники `github`.

## Черновики

Черновики лежат рядом с конфигурацией:

```
{repo}/
├── ai-skills.yaml
└── .ai-skills/feedback/
    └── 2026-09-27-guide-broken-anchor.md
```

**Черновики коммитятся** вместе с проектом — по ним видно, что и когда
отправлялось. Имя файла: дата, скил и заголовок латиницей. Файл выглядит так:

```markdown
---
status: draft
kind: bug
skill: guide
source:
  type: github
  path: https://github.com/owner/ai-skills
  tree: main
commit: 9a9ca973b37c904ff6e547b911f523177d664072
skill_path: skills/guide
created_at: 2026-09-27T10:00:00Z
---

# Broken anchor

The anchor #setup in SKILL.md leads nowhere.
```

Заголовок issue — строка `# ...`, текст issue — всё, что ниже. Их можно
править до отправки. `status` меняет только команда: `draft` → `sent`
(добавляются `issue_url` и `sent_at`) или `declined`; отправленный или
отклонённый черновик больше не отправляется.

> Черновик попадает в коммит ещё до проверки. Если репозиторий проекта
> публичный, проверьте черновик **до** `git commit`, а не только до
> отправки.

К тексту issue при отправке дописывается блок контекста, чтобы автор скила
мог воспроизвести проблему:

```text
---
Skill: `guide` (`skills/guide` at commit `9a9ca97...`)
Source: github https://github.com/owner/ai-skills (tree `main`)
Sent with ai-skill-manager 0.3.0
```

`kind: bug` получает метку `bug`, `kind: improvement` — `enhancement`.
Метки ставит GitHub, только если у вас есть на это право; иначе issue
создаётся без них.

## Пример

Агент пишет черновик (текст можно передать через stdin — так проще с
переносами строк и кавычками):

```sh
aism feedback draft --skill guide --kind bug --title "Broken anchor" --body-file - <<'EOF'
The anchor #setup in SKILL.md leads nowhere: the section is called "Install".
EOF
```

```text
Feedback draft: /project/.ai-skills/feedback/2026-09-27-guide-broken-anchor.md

Repository: github https://github.com/owner/ai-skills
Labels: bug
Title: Broken anchor

The anchor #setup in SKILL.md leads nowhere: the section is called "Install".

---
Skill: `guide` (`skills/guide` at commit `9a9ca97...`)
...

Nothing is sent yet. Review the draft (edit the file if needed); then the user sends it:
  ai-skill-manager feedback send 2026-09-27-guide-broken-anchor
```

Вы проверяете файл и отправляете:

```sh
aism feedback send 2026-09-27-guide-broken-anchor
```

```text
Repository: github https://github.com/owner/ai-skills
...
Open this issue? It is public if the repository is. [y/N] y
Sent: https://github.com/owner/ai-skills/issues/42
```

Любой ответ, кроме `y`/`yes`, — `Not sent.`, черновик остаётся черновиком.
Передумали отправлять — `aism feedback decline <id>`: файл остаётся со
статусом `declined`. Посмотреть черновик и его статус — `aism feedback show
<id>`. Вместо id можно передать путь к файлу черновика.

Если конфигурация не в `ai-skills.yaml` текущей папки, передайте её тем же
`-c` и в `send`: черновики ищутся рядом с ней.

## MCP-сервер для агента

`aism mcp` — тот же `feedback`, но как инструменты
[MCP](https://modelcontextprotocol.io/) (протокол, по которому агент вызывает
внешние инструменты). Сервер работает через stdio, его запускает сам
агент.

| Инструмент | Что делает |
| --- | --- |
| `feedback_draft` | `skill`, `kind` (`bug` / `improvement`), `title`, `body` → черновик, как `feedback draft`; возвращает id, файл и итоговый issue. Ничего не отправляет |
| `feedback_submit` | `id` → диалог в клиенте с итоговым issue и галочкой **Open this issue** (по умолчанию снята). Issue открывается, только если вы нажали accept **и** отметили галочку. Отказ, отмена, снятая галочка — ничего не отправлено, черновик остаётся |

Черновик, изменённый, пока открыт диалог, не отправляется: отправляется
только то, что вы видели. Инструмента «отправить без вопроса» нет. Клиент,
который не показывает диалоги (MCP elicitation в режиме формы), получает
ответ «Not sent» с просьбой, чтобы вы отправили черновик из терминала
через `aism feedback send <id>`.

Как подключить — [Настройка, шаг 4](#4-подключите-mcp-сервер-к-claude-code).

## Настройка доступа к GitHub

GitHub создаёт issue только через API, а API принимает **токен**, не
SSH-ключ: ключ, по которому клонируются репозитории, здесь не подходит.
Сам `aism` токен не хранит и ищет его в таком порядке (как `gh`):

1. переменная `GH_TOKEN`;
2. переменная `GITHUB_TOKEN`;
3. токен, под которым вы вошли в [GitHub CLI](https://cli.github.com/)
   (`gh auth token`).

### Рекомендуется: GitHub CLI

`gh auth login` — и больше ничего, см. [Настройка, шаги 2–3](#2-установите-github-cli).

### Свой токен (без `gh`, CI)

Создайте токен в GitHub: **Settings → Developer settings → Personal access
tokens**. Какой выбрать — зависит от того, чей репозиторий источника:

| Источник скилов | Токен |
| --- | --- |
| Ваш репозиторий или репозиторий вашей организации | **Fine-grained**: Repository access — только эти репозитории; Permissions → Repository → **Issues: Read and write**. Самый узкий вариант |
| Чужой публичный репозиторий | **Classic** со scope `public_repo`. Fine-grained токен в репозиторий, где вы не участник, писать не может |
| Чужой приватный репозиторий, где вы участник | **Classic** со scope `repo` — это доступ ко всем вашим приватным репозиториям, выдавайте осознанно |

Всегда задавайте срок действия (Expiration) и выпускайте токен заново, когда
он истечёт.

Где хранить токен:

- **Не в `ai-skills.yaml`** и не в других файлах, которые коммитятся.
- В менеджере паролей или keyring ОС, подставляя при запуске, например:

  ```sh
  export GH_TOKEN="$(secret-tool lookup service github-issues)"   # Linux keyring
  export GH_TOKEN="$(op read op://Private/github-issues/token)"  # 1Password CLI
  ```

- В `.envrc` для [direnv](https://direnv.net/) — только если `.envrc` в
  `.gitignore`.

### Codespaces и CI

Там `GITHUB_TOKEN` задан автоматически и выдан только на текущий
репозиторий. Из-за порядка поиска он перекроет `gh`, и GitHub ответит 403
или 404. Задайте свой токен в `GH_TOKEN` (он проверяется первым) или
выполните отправку без этой переменной: `env -u GITHUB_TOKEN aism feedback
send <id>`.

## Ошибки

| Сообщение | Причина и что делать |
| --- | --- |
| `feedback send asks the user to confirm and needs a terminal` | `send` запущен не в терминале (агентом, в пайпе). Запустите команду сами |
| `no GitHub token: install gh and run gh auth login, or set GH_TOKEN` | Нет ни переменной, ни `gh`. См. [настройку](#настройка-доступа-к-github) |
| `no GitHub token from gh (...)` | `gh` установлен, но вход не выполнен: `gh auth login` |
| `HTTP 401: Bad credentials` | Токен неверный или истёк — выпустите новый |
| `HTTP 403` / `HTTP 404: Not Found` | Токен не видит репозиторий или не может писать issue (fine-grained токен для чужого репозитория, `GITHUB_TOKEN` Codespaces/CI) — см. таблицу выше. GitHub отвечает 404, а не 403, на репозиторий, которого токен не видит |
| `HTTP 410: Issues are disabled for this repo` | В репозитории выключены issue — сообщите автору другим способом |
| `feedback ... changed since it was confirmed: show it again` | Файл изменили, пока шла отправка. Запустите `send` ещё раз |
| `feedback ... is already sent` / `declined` | Черновик уже закрыт; создайте новый |
| `issue ... opened, but feedback ... isn't marked sent` | Issue создан, но файл черновика не записался. Не отправляйте повторно — впишите `status: sent` и ссылку вручную |
