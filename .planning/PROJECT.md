# zcode-acp

## What This Is

Go-обёртка (один статический бинарь), которая выставляет ZCode Desktop наружу как
стандартный **ACP (Agent Client Protocol) v1 сервер поверх stdio**: любой ACP-клиент
может открывать сессии ZCode, отправлять промпты и получать поток ответа
(сообщения, размышления, tool-calls, permission-запросы). Публичная архитектура —
честный ACP-сервер, пригодный для любого клиента. README документирует, среди
прочего, рецепт подключения обёртки к Orca в режиме chat UI.

## Core Value

ACP-клиент может провести с ZCode полный ход end-to-end — от `initialize` до
стриминга ответа с tool-calls и permission-запросами — на реальном аккаунте z.ai
пользователя (api-key или OAuth coding plan).

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Полный чат-цикл ACP v1: initialize, session/new, prompt со стримингом
      (agent_message_chunk + agent_thought_chunk), tool_call события,
      session/request_permission, session/cancel
- [ ] Оба пути авторизации end-to-end: api-key (zai-api) и OAuth individual
      coding plan (снапшот реестра, минт PAT, ответы на interaction/*)
- [ ] Обнаружение и запуск desktop app-server ZCode (zcode.cjs через
      ELECTRON_RUN_AS_NODE) из установленного desktop-приложения
- [ ] Скаффолд проекта по образцу Djarvur/goswitch: строгий линтер,
      GitHub Actions, protected branch
- [ ] Документация на английском: README, рецепты запуска (включая рецепт Orca),
      протокольные доки
- [ ] v1 поддерживает Ubuntu GNOME; код без ограничений ОС

### Out of Scope

- Подделка аттестации/подписи ради квот — сознательно не делается; только
  официальные пути (PAT выдаётся z.ai, подписывает официальный бинарь)
- Бэкенд zcode CLI — только desktop app-server в v1
- Окружения кроме Ubuntu GNOME (KDE, macOS, Windows) в v1 — принимаются PR
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
- Исходники ZCode: локальный клон публичного дерева (github.com/zai-org/ZCode), 3.17.x.

**Референсная имплементация:** github.com/william0wang/zcode-acp (TypeScript,
Apache-2.0 → портирование на Go допустимо с атрибуцией) — в ней уже реализованы
и покрыты тестами самые тяжёлые куски: расшифровка keyring (`enc:v2:os:`),
`requestCredentialCipher`, снапшот с точной ревизией, PAT-флоу. Детали и файлы
для портирования: `.planning/research/REFERENCE-ADAPTERS.md`. Ограничение из их
TROUBLESHOOTING: Start Plan headless не работает (нужен браузер) — документируем.
Конкурирующие адаптеры (Rust/Python) coding-plan цепочку не делают.

**Открытые вопросы для фазы исследования:** ✅ RESOLVED Phase 1 (2026-11-19) —
все пять DISC-deliverables доставлены и верифицированы (`docs/protocol/*`,
`docs/research/reference-adapters.md`, эксперименты в `experiments/`):
- Рецепт запуска Orca: omp-карточка, контракт подтверждён дважды (чанки
  установленной сборки + исходники репо @v1.4.262); `--session-dir` пишется
  Go-агентом (E5: slug-файлы Claude-style JSONL) — v1 без loadSession не
  резюмится (клиент -32601), nicety для Фазы 7
- Load-bearing набор вызовов при старте карточки: задокументирован
  (acp-client-dialect.md, 19 методов, +session/close и elicitation/complete)
- Обнаружение установки: матрица AppImage/deb/unpacked + формула revision
  `zcode-builtin:<rev>:<sha256(path.resolve(activeFilePath))>` (install-discovery.md)
- Маппинг выбора модели: `session/setModel` + reasoningLevel обязателен после
  `session/create` (create теряет reasoningLevel — V-EXP); modelDiscovery
  через `models` в ответе session/new (поправка из source cross-verification)

## Constraints

- **Tech stack**: Go; единственный бэкенд — desktop app-server ZCode; CLI —
  единый бинарь `zcode-acp` на cobra (сабкоманды; `omp` реализует контракт
  карточек, требующих preset-агента)
- **Research policy**: всё про ZCode — сначала исходники
  (https://github.com/zai-org/ZCode), потом эксперименты; всё про Orca —
  сначала исходники (https://github.com/stablyai/orca)
- **Platform**: код кроссплатформенный без ограничений, но v1 поддерживает
  только Ubuntu GNOME (зависимость от Secret Service/gnome-keyring при
  расшифровке кредов); остальные окружения — через PR
- **Docs language**: English (README, рецепты, docs/)
- **Repo config**: по образцу https://github.com/Djarvur/goswitch — строгий
  линтер, GitHub Actions, protected branch
- **Git workflow**: фича-ветки от свежего master/master-аналога, напрямую в
  основную ветку не работаем

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Честный ACP-сервер как публичная архитектура; подключение к клиентам, требующим предустановленный агент, — рецепт в README, не архитектурное ограничение | Универсальность (годится для любого ACP-клиента) и честная публичная история | — Pending |
| Единый бинарь `zcode-acp` (cobra) с сабкомандами; `omp`-сабкоманда реализует контракт карточки; в Orca — `agentCmdOverrides: {omp: "zcode-acp omp"}` | Префикс-аргументы в override разрешены — зеркало-имя бинаря не нужно; pflag-allowlist даёт либеральный разбор неизвестных флагов | — Pending |
| Оба пути авторизации в v1 (api-key + OAuth coding plan) | OAuth — реальный сценарий пользователя; api-key — простой fallback; shim v2 доказал оба пути | — Pending |
| Полный чат-цикл в v1 (стриминг, tool_call, permission, cancel) | Карточка в chat UI без инструментов не имеет смысла | — Pending |
| Первая фаза — исследование и документирование, код после | Всё про ZCode/Orca изучается по исходникам до экспериментов | — Pending |
| Бэкенд — только desktop app-server | Явное решение; CLI не поддерживаем в v1 | — Pending |
| done v1 = CI-тесты протокола (мок app-server) + тесты на виртуалке + ручная E2E в Orca | Три уровня проверки: протокол, среда, реальный сценарий | — Pending |
| Документация на английском | Шире аудитория для PR | — Pending |
| Ground-truth hierarchy (Phase 1): installed 3.18.2 bundle = V-SRC первоисточник; GitHub-дерево 3.17.x — подтверждающие цитаты с версией; каждая нагрузочная претензия помечена V-SRC/V-EXP/V-RPT; 3.17↔3.18.2 drift-базлайн — первоклассный раздел | Документы Фаз 3–7 строятся на проверенной правде провода, а не на интерпретациях | ✓ Done (Phase 1) |
| Клиент-программа дважды проверяема (Phase 1, correction): stablyai/orca ПУБЛИЧНЫЙ (MIT) — ранний «404» был временным сетевым сбоем; якоря на чанки установленной сборки остаются истиной для shipped-версии, исходники репо (@v1.4.262) — второй путь | Читатель может сверить по репо; minified-интерпретации заменены чтением исходников | ✓ Done (Phase 1) |
| Три семантические поправки для Фаз 4–6 (Phase 1 source cross-verification): chat-запуск БЕЗ `--session-dir` (он у discovery-процессов — wrapper accept-and-ignore + терпит второй процесс); fullAccess=false только у discovery-сессий; modelDiscovery — модели доступны до сессии через probe-сессию (`models` в ответе session/new, расширение клиента) | Влияют на CLI-контракт (Фаза 2), честность capabilities (Фаза 4), выбор моделей (Фаза 6) | ✓ Done (Phase 1) |
| Экспериментальный конвейер (Phase 1): scrub-first (сырые записи вне репо, scrub-capture.mjs — единственный путь в experiments/captures/), scan-secrets.sh гейтит каждый коммит, check-docs.sh — структурно-provenance гейт; эксперименты никогда не идут из CI | Секреты не попадают в публичный репо; конвейер переиспользуется Фазой 2 для CI | ✓ Done (Phase 1) |

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
*Last updated: 2026-11-19 after Phase 1*
