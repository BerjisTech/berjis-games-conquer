Conquer — Game Design and Implementation Plan

Summary
- World-map, persistent strategy game built with Angular (frontend) and Go (service), integrated with main `api` for auth, identity, billing, and shared policies.
- Players join, pick a country, gather resources, craft assets (vehicles, weapons, infrastructure), and coordinate to conquer other countries.
- Loss on failed attack moves attacker’s holdings to defender’s central loot; defeated country can be conquered. Rankings per country reflect real-world military structures.

Tech Stack
- Frontend: Angular + Mapbox GL JS (default). Google Maps JS is a fallback but requires billing and lacks vector tile flexibility.
- Backend: Go service (`service/`) with HTTP + WebSocket. Postgres for persistence, Redis for caches/locks, optional NATS for events.
- Auth: Delegated to root `api` via service-to-service token validation and user session verification from cookies/headers.

Why Mapbox
- Vector tiles and layers for country polygons; performant client-side styling and interaction.
- Easier theming and custom overlays (assets, battles) compared to Google’s data layer.
- Reasonable free tier for development; can abstract provider behind a thin map adapter.

Core Mechanics
- Join/Start: Player selects a country; can desert to switch countries (with cooldown/penalties) if unranked or with governance approval.
- Resources: Players farm resources (materials/fuel/intelligence) via tasks, jobs, and territory yields.
- Crafting: Convert resources to assets: vehicles, weapons, infrastructure. Tiered tech trees; time + resource costs.
- Combat: Countries attack other countries. If attackers lose, their personal and contributed assets/loot transfer to defender’s central loot; attacker country marked conquered if criteria met.
- Rankings: Rank within a country based on contribution, leadership, and activity. Roles map to real-world military titles per country.
- Persistence: Late joiners face the evolved world state (no resets). Seasons/timeboxed events can soft-reset or rebalance if desired.

High-Level Architecture
- Service (`service/`):
  - REST: country listing, player profile, inventory, crafting, orders, attacks, contributions, rankings.
  - WebSocket: realtime updates for country state, battles, map overlays, chat/coordination.
  - Jobs: background workers for crafting timers, battle resolution, territory yields.
  - Storage: Postgres (entities + events), Redis (counters/locks/queues), optional NATS (broadcast events to edge consumers).
- Frontend (Angular):
  - Map view (Mapbox) with country polygons and overlays for assets/battles.
  - Panels: country dashboard, player inventory, crafting, missions, attack planner, rankings, chat.
  - Realtime store synced via WebSocket; fall back to polling.

Data Model (conceptual)
- User (from main `api`)
- PlayerProfile { userId, countryId, rankId, contributionScore, joinedAt }
- Country { id, name, status: active|conquered, centralLootId, territories[] }
- Loot { id, ownerType: player|country, resources, assets[] }
- Asset { id, type: vehicle|weapon|infra, tier, hp, upkeep }
- CraftOrder { id, playerId, recipeId, startedAt, completesAt, status }
- Attack { id, attackerCountryId, defenderCountryId, objective, startedAt, status, outcome, logs }
- BattleEvent { id, attackId, tick, details }
- Rank { id, countryId, title, weight, minScore }

Key Systems
- Contribution and Ranking: aggregate player contributions (resources, command success, defense). Determine rank titles per-country via configuration.
- Combat Resolution: deterministic or seeded RNG with visibility/intel modifiers; transparent logs; anticheat checks.
- Territory Control: country status flips to conquered when control/war goals met.
- Economy: resource sinks (upkeep, repairs) to prevent runaway inflation.

APIs (preview)
- GET /countries; GET /countries/:id
- POST /players/join { countryId }
- POST /players/desert { countryId }
- GET /players/me; GET /players/me/loot
- POST /craft/orders; GET /craft/orders
- POST /attacks { defenderCountryId, objective }
- GET /attacks/:id; WS /ws for state/battle feeds

Security
- All requests authenticated via main `api` session/JWT. Service-to-service validation for internal calls.
- Server-side authoritative state; all actions validated with idempotency keys and optimistic locks.
- Rate limiting and per-action quotas; audit logs for governance actions.

MVP Scope
- Country selection, basic resources and crafting, simple vehicle/weapon types.
- Attack/defense with clear win/lose rules and loot transfer.
- Map overlays for country status and active battles.
- Rankings based on contribution score; basic chat.

Milestones
- M1: Domain model + migrations; auth integration; country map display.
- M2: Loot/resources + crafting loop; inventory UI.
- M3: Attacks + battle resolution + realtime updates.
- M4: Rankings and governance; desert/switch rules.
- M5: Hardening, telemetry, balancing, and season controls.

Operational Notes
- Compose services: `games/conquer/service` with its own Dockerfile, exposed behind `edge` like other apps.
- Environment: MAPBOX token, DB/Redis/NATS creds from compose/env files.

Next Steps
- Finalize provider choice (Mapbox default) and secure tokens.
- Lock MVP ruleset; design initial recipes and rank mappings per country.
- Scaffold `service/` and Angular app following existing patterns.

