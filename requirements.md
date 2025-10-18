Conquer — Requirements and Roadmap

Goals
- Persistent, world-map strategy experience leveraging existing Angular + Go patterns in the ecosystem.
- Reuse main `api` for auth, identity, roles, billing, and shared infra.

Functional Requirements
- Join/Desert: Player selects or switches countries with rules and cooldowns.
- Resources: Earn resources via time-based yields, jobs, and missions tied to territories.
- Crafting: Queue-based crafting of vehicles, weapons, and infrastructure with tiered recipes.
- Combat: Plan attacks; resolve battles; update map and transfer loot on defeat.
- Rankings: Contribution-based hierarchy per country reflecting real-world military structures.
- Realtime: WebSocket updates for battles, territory changes, and team comms.

Non-Functional Requirements
- Scale: Handle thousands of concurrent connections and active battles.
- Fairness: Server authoritative state; anti-cheat; transparent battle logs.
- Observability: Metrics, logs, traces; admin panel for moderation and balancing.

System Design
- Frontend (Angular)
  - Map: Mapbox GL JS with country polygons and overlays for assets/attacks.
  - State: NgRx or signal-based store; WS for pushes; REST for commands.
  - Modules: Map, Country, Player, Crafting, Combat, Rankings, Chat.
- Backend (Go `service/`)
  - HTTP (REST) for commands/queries; WebSocket for subscriptions.
  - Workers for crafting timers and battle ticks; cron for yields.
  - Storage: Postgres (entities, events); Redis (queues/locks/counters); optional NATS for broadcast.
  - Integrations: main `api` for auth/user; edge for routing.

Data Entities
- Country(id, name, status, centralLootId)
- Territory(id, countryId, polygonRef, yieldProfile)
- PlayerProfile(userId, countryId, rankId, contributionScore)
- Loot(id, ownerType, resources, assets)
- Asset(id, type, tier, hp, upkeep, ownerLootId)
- CraftOrder(id, playerId, recipeId, startedAt, completesAt, status)
- Attack(id, attackerCountryId, defenderCountryId, startedAt, status, outcome)
- BattleEvent(id, attackId, tick, details)
- Rank(id, countryId, title, weight, minScore)

API Sketch
- Auth: validate via main `api` session/JWT on every request.
- Countries: GET /countries; GET /countries/:id; GET /countries/:id/rankings
- Players: GET /players/me; POST /players/join; POST /players/desert
- Loot: GET /players/me/loot; POST /loot/contribute; GET /countries/:id/loot
- Craft: POST /craft/orders; GET /craft/orders; GET /recipes
- Combat: POST /attacks; GET /attacks/:id; WS /ws (topics: country, attacks, player)

Game Rules (MVP)
- Desert Cooldown: 24h; loss of uncommitted personal queue on desert.
- Loot Transfer: On failed attack, attacker personal + contributed assets transfer to defender central loot.
- Conquer Condition: Defender retains control after final battle and attacker fails predefined objectives.
- Ranks: Computed daily from contribution; manual promotions allowed for top roles with quorum.

Map Data
- Country polygons via public dataset (e.g., Natural Earth) preprocessed to vector tiles.
- Store references/ids in DB; render layers in client; highlight active regions.

Security and Anti-Cheat
- All actions validated server-side with idempotency keys and optimistic concurrency.
- Rate-limits per action/user; anomaly detection on resource gains and battle actions.
- Audit logs for governance (promotions, war declarations).

DevOps
- Dockerfile for `service/`; compose integration mirroring other apps.
- ENV: MAPBOX_TOKEN, DATABASE_URL, REDIS_URL, NATS_URL, API_BASE, WS_BASE.
- Migrations in `service/migrations`; seed initial countries and ranks.

Milestones & Deliverables
- M1: Schema + migrations + seed; minimal country list API; Angular map scaffold.
- M2: Player join/desert; loot model; crafting queues; basic UI.
- M3: Combat engine (MVP), attack planner, WS updates; overlays.
- M4: Rankings/governance; contributions; country roles; admin tools.
- M5: Balancing, telemetry, scaling tests; season management.

Open Questions
- Season model and soft resets cadence.
- Country-specific rank trees customization UX.
- Monetization and cosmetics (if any) and their impact on fairness.

