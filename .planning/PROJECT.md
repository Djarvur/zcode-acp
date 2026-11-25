# zcode-acp

## What This Is

Go-обёртка (один статический бинарь), которая выставляет ZCode Desktop наружу как
стандартный **ACP (Agent Client Protocol) v1 сервер поверх stdio**: любой ACP-клиент
может открывать сессии ZCode, отправлять промпты и получать поток ответа
(сообщения, размышления, tool-calls, permission-запросы). Публичная архитектура —
честный ACP-сервер, пригодный для любого клиента. README документирует, среди
прочего, рецепт подключения обёртки к Orca в режиме chat UI.

**v0.20.0 (v1) — shipped и field-proven (2026-11-23):** полный цикл подтверждён на
трёх уровнях — CI (мок app-server, golden-корпус), чистая VM (vm-e2e.sh, 16 проверок),
реальная E2E в Orca на живом аккаунте (coding-plan путь — прогон пользователя;
api-key — гарантированный тестовый уровень + security-аудиты обеих фаз).

**v0.21.0 — shipped (2026-11-23):** на `session/new` фронт отдаёт каталог слэш-команд
и скиллов (`available_commands_update`: ZCode-команды, скиллы, `/model` с
`<selector>`-hint) — автодополнение в композере карточки Orca.

**v0.22.0 — shipped (2026-11-23):** диалоги выбора AskUserQuestion
(`interaction/requestUserInput`) рендерятся в карточке как form-elicitation, а ответ
возвращается в merge-контракт диалекта `{action, content.answers}`; каталог моделей
в `/model` и picker полный (19 записей, provider labels) и сохраняется при
переключениях.

## Current Milestone: v0.24.0 macOS Support

**Goal:** Existing `zcode-acp` functionality runs on universal macOS (arm64/amd64):
the binary builds, discovers and launches ZCode Desktop, passes tests, and completes
an Orca end-to-end run on the current machine without adding new ACP capabilities.

**Target features:**
- Remove Linux-only build/runtime assumptions and produce universal macOS binaries.
- Discover `ZCode.app` on macOS and launch its desktop app-server correctly.
- Preserve api-key and OAuth coding-plan authentication through macOS Keychain.
- Add macOS test/CI coverage and a live Orca acceptance run.

## Core Value

ACP-клиент может провести с ZCode полный ход end-to-end — от `initialize` до
стриминга ответа с tool-calls и permission-запросами — на реальном аккаунте z.ai
пользователя (api-key или OAuth coding plan). **Подтверждено живьём:** карточка Orca
рендерит мысли, текст ответа, tool-calls и vision-чтение файлов; модели аккаунта
переключаются через `/model` и picker; диалоги выбора GSD рендерятся в карточке и
их ответы доходят до workflow.

## Requirements

### Validated

- ✓ Полный чат-цикл ACP v1: initialize, session/new, prompt со стримингом
  (agent_message_chunk + agent_thought_chunk), tool_call события,
  session/request_permission, session/cancel — v0.20.0 (Phases 4–5, field-proven)
- ✓ Оба пути авторизации end-to-end: api-key (zai-api; REAL RUN Phases 5) и OAuth
  individual coding plan (REAL RUN Phase 6 + Orca-прогон пользователя Phase 7)
- ✓ Обнаружение и запуск desktop app-server ZCode из установленного desktop-приложения
  (Phase 3; clean-VM field run Phase 7)
- ✓ Скаффолд проекта по образцу Djarvur/goswitch (строгий линтер, GitHub Actions,
  protected branch) — Phase 2
- ✓ Документация на английском: README, рецепты запуска (включая рецепт Orca),
  протокольные доки — Phase 7 (DOCS-01..03)
- ✓ v1 поддерживает Ubuntu GNOME; код без ограничений ОС — Phase 7 (QA-02/QA-03)
- ✓ ACP Front (Phase 4): минимальный чат-цикл end-to-end — CORE-01..05 complete
- ✓ Back leg (Phase 3): JSONL-кодек диалекта, scriptable fake + golden-корпус,
  discovery-матрица + супервизор, машина ходов (exactly-once, cancel, -32010)
- ✓ OAuth Coding-Plan Path & Model Picker (Phase 6): keyring→entitlement→snapshot→
  read-back→PAT-минт, account-модели в picker, `/model` slash-слой
- ✓ Слэш-каталог в автодополнении карточки Orca (Phase 8, v0.21.0): один
  `available_commands_update` на `session/new` — ZCode-команды, скиллы, `/model`
  (CAT-01..05; VM GUI + живая карточка, 139-item каталог)
- ✓ Ask-the-User Bridge (Phase 9, v0.22.0): `interaction/requestUserInput` → одна
  form-elicitation; accept мапится в `{action, content.answers}` по тексту вопроса,
  multi-select через ", ", decline/cancel verbatim, ошибка поверхности — decline с
  причиной, ход не падает (UIB-01..05; VM GUI + real-machine acceptance)
- ✓ Полный каталог моделей в `/model` и picker (v0.22.0 fix-set): 19 записей,
  provider labels на каждой, каталог сохраняется при switch и между ходами
  (create-snapshot несёт available, rebind — подтверждённый current)
- ✓ Mount-first AppImage backend (Phase 10, v0.23.0): app-server запускается напрямую
  из packed `.AppImage` через управляемый `--appimage-mount` helper — без распаковки и
  дубликата на диске; discovery — единственная pure-read стадия, ранжирующая кандидатов
  по версии (packed 4.0.0 побеждает stale unpacked 3.18.2), unversioned — miss, а не
  last-resort; `--app-server`/env принимает directory/binary/packed с неизменным
  flag-over-env приоритетом (MNT-01..06, DISC-01..04, VER-01..03; real-machine
  mount→session/create acceptance, 30/30 must-haves)

### Active

- macOS support for the existing v0.23.0 feature set — universal arm64/amd64 build,
  ZCode Desktop discovery/launch, Keychain-backed credentials, tests/CI, and live Orca
  acceptance. No new ACP functionality is in scope.

(Вне milestone: пост-v1 пожелания живут в 07-REVIEW-DISPOSITION.md
(IN-a..h, IN-02/03) и deferred-реестре STATE.md. Фазы 8–9 не имеют
VALIDATION.md/SECURITY.md — это advisory-долг, отмеченный при закрытии фаз.)

### Out of Scope

- Подделка аттестации/подписи ради квот — сознательно не делается; только
  официальные пути (PAT выдаётся z.ai, подписывает официальный бинарь)
- Бэкенд zcode CLI — только desktop app-server в v1
- Окружения кроме Ubuntu GNOME и macOS (KDE и Windows) — принимаются PR
- Патчи бандла ZCode (zcode.cjs и др.) — обёртка ничего не модифицирует
- Чтение точного счётчика квоты OAuth — API этого не позволяет

## Context

**Отправная точка исследования:** локальные заметки FINDINGS.md (частные
исследовательские записи, вне этого репозитория) — итоги разбора ZCode 3.18.2
и Orca (установка, протоколы, OAuth-путь, грабли). Проверенные факты оттуда:

- **app-server** (`ELECTRON_RUN_AS_NODE=1 zcode.cjs app-server --stdio`):
  JSONL по stdio, **не** JSON-RPC; методы `session/create|send|messages|setModel`,
  `provider/updateAccountConfig`, `provider/refreshRegistry`; сервер→хост:
  `state.updated`, `interaction/requestCredentialCipher`,
  `interaction/requestProviderRuntimeHeaders`, `interaction/requestPermission`.
  Работает в host-managed режиме — хост обязан отвечать на interaction/*.
- **OAuth coding plan**: снапшот реестра принимается только с ревизией
  `zcode-builtin:<rev>:<sha256(пути builtin-файла)>`; PAT минтится через
  api.z.ai (getCustomerInfo → api_keys → access_tokens); агенту отдаётся
  `{headersApplied:true, requestAuth:{apiKey, apiKeyId}}`; подпись V4 агент
  делает сам после PAT-handshake — обёртке подделывать нечего.
- **session/create теряет `options.reasoningLevel`** для thinking-моделей —
  нужно `session/setModel` после создания сессии.
- **`headersApplied:false`** для account-провайдера = гарантированный провал хода.
- **Orca**: ACP v1 client (JSON-RPC 2.0 over stdio) лежит в
  `resources/app.asar.unpacked/out/main/chunks/`; ACP-агенты захардкожены:
  `omp` (`omp acp`, min 17.0.5, env `PI_NO_PTY=1`), `opencode` (min 2.0.0),
  `opencode2` (min 2.0.14); детект по имени в PATH + версия.
- Исходники ZCode: локальный клон публичного дерева (github.com/zai-org/ZCode), 3.17.x;
  wire-truth первоисточник — установленный бандл 3.18.2 (grep по app.asar).

**Референсная имплементация:** github.com/william0wang/zcode-acp (TypeScript,
Apache-2.0 → портирование на Go допустимо с атрибуцией) — `docs/research/reference-adapters.md`.
Конкурирующие адаптеры (Rust/Python) coding-plan цепочку не делают.

**Текущее состояние (v0.23.0):** 10 фаз / 66 планов; 12 Go-пакетов, ~63K строк Go
(с тестами), `-race` зелёные, whole-tree lint strict 0 issues, CI (build/vet/lint/
test/tidy + govulncheck + secret-scan) зелёный. После v0.22.0 доставлены
mount-first AppImage backend (Phase 10: app-server запускается из packed `.AppImage`
без распаковки; discovery ранжирует по версии; real-machine acceptance
mount→session/create прошёл), OMP print mode для текстовых задач Orca
(quick `261124-154`) и очередь `-32003 session_busy` для конкурентных prompt'ов
(quick `261124-1ou`). Известный долг: lint-подавления с wire-обоснованиями
(nolint), 2 признанных deferred-пункта фазы 6, advisories 07-REVIEW-DISPOSITION.md
и 7 Info-замечаний Phase 10 (IN-01..07); фазы 8–9 без VALIDATION.md/SECURITY.md
(advisory).

## Constraints

- **Tech stack**: Go; единственный бэкенд — desktop app-server ZCode; CLI —
  единый бинарь `zcode-acp` на cobra (сабкоманды; `omp` реализует контракт
  карточек, требующих preset-агента)
- **Research policy**: всё про ZCode — сначала исходники
  (https://github.com/zai-org/ZCode), потом эксперименты; всё про Orca —
  сначала исходники (https://github.com/stablyai/orca)
- **Platform**: v1 supports Ubuntu GNOME; v0.24.0 adds universal macOS
  (arm64/amd64) for the same existing feature set. KDE and Windows remain out of
  scope; the wrapper does not add a ZCode CLI backend.
- **Docs language**: English (README, рецепты, docs/)
- **Repo config**: по образцу https://github.com/Djarvur/goswitch — строгий
  линтер, GitHub Actions, protected branch
- **Git workflow**: фича-ветки от свежего master/master-аналога, напрямую в
  основную ветку не работаем

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Честный ACP-сервер как публичная архитектура; подключение к клиентам, требующим предустановленный агент, — рецепт в README, не архитектурное ограничение | Универсальность (годится для любого ACP-клиента) и честная публичная история | ✓ Good — Orca подключается по рецепту, архитектура не менялась |
| Единый бинарь `zcode-acp` (cobra) с сабкомандами; `omp`-сабкоманда реализует контракт карточки; в Orca — `agentCmdOverrides: {omp: "zcode-acp omp"}` | Префикс-аргументы в override разрешены — зеркало-имя бинаря не нужно | ✓ Good — field-proven в Orca (card → `omp` shim → `zcode-acp omp`) |
| Оба пути авторизации в v1 (api-key + OAuth coding plan) | OAuth — реальный сценарий пользователя; api-key — простой fallback | ✓ Good — оба доведены до реальных прогонов (путь 1 — тестовый уровень + решение пользователя) |
| Полный чат-цикл в v1 (стриминг, tool_call, permission, cancel) | Карточка в chat UI без инструментов не имеет смысла | ✓ Good — tool-calls, permission-запросы и vision отрендерились в живой Orca |
| Первая фаза — исследование и документирование, код после | Всё про ZCode/Orca изучается по исходникам до экспериментов | ✓ Good — wire-truth документов фаз 3–7 строилась на provenance-правде |
| Бэкенд — только desktop app-server | Явное решение; CLI не поддерживаем в v1 | ✓ Good — без изменений |
| done v1 = CI-тесты протокола (мок app-server) + тесты на виртуалке + ручная E2E в Orca | Три уровня проверки: протокол, среда, реальный сценарий | ✓ Good — все три уровня пройдены (QA-01/02/03 verified) |
| Документация на английском | Шире аудитория для PR | ✓ Good — README + docs/provenance.md + recipes shipped (DOCS-01..03) |
| Ground-truth hierarchy (Phase 1): installed 3.18.2 bundle = V-SRC первоисточник; 3.17↔3.18.2 drift-базлайн — первоклассный раздел | Документы Фаз 3–7 строятся на проверенной правде провода | ✓ Good — оправдался многократно (credentialCipher, turn-сигналы, MCP-auth decline) |
| Перезапуск верификации при фазовых границах — только по необходимости (решение пользователя 2026-11-22); при закрытии v0.20.0 стейлы фаз 1–6 приняты override-решением пользователя | Перезапуск 6 верификаторов не добавляет truth: исходники эволюционировали по проекту, не испортились | ✓ Done (2026-11-23, known override в MILESTONES.md) |
| Слэш-каталог отдаётся на `session/new` одним `available_commands_update`, а не отдельным методом/по запросу | Клиент сканирует каталог при discovery в пределах 9 s; каталог — session-level nicety, не turn-update | ✓ Good — 139-item каталог в живой карточке (v0.21.0) |
| `requestUserInput` бриджится в form-elicitation, ответ мапится в `{answers}` по тексту вопроса; deadline политики снят для bridged-ask | Orca нативно рендерит form-elicitation; merge-контракт диалекта ждёт answers по question text; человеческий диалог занимает минуты и не должен авто-отклоняться | ✓ Good — VM GUI + real-machine acceptance (v0.22.0) |
| Полный каталог моделей собирается merge: create-snapshot несёт available, rebind — подтверждённый current | Backend сужает `settings.model.available` до текущего провайдера; без merge picker сжимается после switch | ✓ Good — 19 записей live-verified, сохраняется при переключении |
| Mount helper env — allowlist by construction (PATH/HOME/TMPDIR); mount-строка принимается целиком только как absolute non-symlink `.mount_*` под private 0700 base; teardown: app-server → helper → mount-gone → base | Deny-list несостоятелен: ветка `APPIMAGE_EXTRACT_AND_RUN` в runtime предшествует mount-ветке; field-split пути ломает пробелы; порядок teardown — единственный без сирот | ✓ Good — Phase 10; fake-runtime e2e + real 4.0.0 mount, 0 survivors, sha256 unchanged |
| Discovery — одна pure-read стадия с версионным ранжированием; unversioned packed/tree = miss, не last-resort; плохой новый кандидат не блокирует годный старый | Stale unpacked не должен пинить версию, но miss не должен превращаться в победителя; per-candidate miss-строки сохраняют диагностику | ✓ Good — packed 4.0.0 beats stale 3.18.2; tie-break documented and tested |
| Packed override абсолютизируется (никогда PATH-resolve); volatile `/tmp/.mount_*` отвергается уже на probe; `desktopVersion` предпочитает `Install.Version`, basename — второй источник | Иначе `.AppImage` уйдёт в GUI-ветку, /tmp-мусор станет spawn root, а PAT-клиент отправит 0.0.0 | ✓ Good — Phase 10; real mount reports version=4.0.0 |
| Real-machine acceptance ограничен mount→session/create (D-14); live prompt на 4.0.0 — отдельный milestone; extraction proof аддитивный (pre-existing extracts разрешены и сохраняются) | Protocol drift 4.0.0 — отдельная работа; blanket «parent clean» дал бы ложный fail на хосте с 3.18.2 extract | ✓ Good — real run PASS 5.45s, 0 added entries, 1 pre-existing retained |
| v0.24.0 is a platform port, not a feature expansion | The user explicitly ruled out new functionality; success means existing ACP behavior works on macOS | — Pending |
| macOS target is universal arm64/amd64 | Go can produce both binaries; live E2E runs on the current Apple Silicon machine | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-11-25 after starting v0.24.0 macOS Support*
