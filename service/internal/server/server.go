package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jmoiron/sqlx"

	coreauth "github.com/berjistech/berjis-ecosystem/shared/coreauth"
)

type Options struct {
	AllowedOrigins      string
	CoreAPIBase         string
	DB                  *sqlx.DB
	AdminUsers          string
	DesertCooldownHours int
	CraftMinutes        int
	AdminGetSeed        bool
}

func New(opts Options) *fiber.App {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins:     opts.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Authorization,Content-Type,Accept",
		AllowCredentials: true,
	}))

	app.Get("/v1/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"success": true}) })

	httpClientAuth := &http.Client{Timeout: 8 * time.Second}
	var authVerifier *coreauth.Verifier
	if base := strings.TrimSpace(opts.CoreAPIBase); base != "" {
		if v, err := coreauth.NewVerifier(coreauth.Config{
			CoreAPIBase: base,
			HTTPClient:  httpClientAuth,
		}); err != nil {
			fmt.Printf("warn: coreauth verifier init failed: %v\n", err)
		} else {
			authVerifier = v
		}
	}

	isAdmin := func(uid string) bool {
		if opts.AdminUsers == "" {
			return false
		}
		for _, p := range strings.Split(opts.AdminUsers, ",") {
			if strings.TrimSpace(p) == uid {
				return true
			}
		}
		return false
	}

	// Auth verify helper, reusing main api
	getUserID := func(c *fiber.Ctx) (string, error) {
		token := ""
		if authz := c.Get("Authorization"); strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			token = strings.TrimSpace(authz[7:])
		}
		if token == "" {
			token = strings.TrimSpace(c.Cookies("access", ""))
		}
		if authVerifier != nil && token != "" {
			if claims, err := authVerifier.Verify(token); err == nil {
				return claims.UUID, nil
			} else if errors.Is(err, coreauth.ErrTokenInvalid) || errors.Is(err, coreauth.ErrTokenExpired) || errors.Is(err, coreauth.ErrTokenMissing) {
				return "", fiber.ErrUnauthorized
			}
		}
		if strings.TrimSpace(opts.CoreAPIBase) == "" {
			return "", fiber.ErrUnauthorized
		}
		req, _ := http.NewRequest("GET", strings.TrimRight(opts.CoreAPIBase, "/")+"/v1/auth/verify", nil)
		if v := c.Get("Authorization"); v != "" {
			req.Header.Set("Authorization", v)
		}
		if v := c.Get("Cookie"); v != "" {
			req.Header.Set("Cookie", v)
		}
		resp, err := httpClientAuth.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		var raw map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return "", err
		}
		data, _ := raw["data"].(map[string]any)
		if data == nil {
			return "", fiber.ErrUnauthorized
		}
		valid, _ := data["valid"].(bool)
		if !valid {
			return "", fiber.ErrUnauthorized
		}
		if uidAny, ok := data["uid"]; ok {
			switch v := uidAny.(type) {
			case float64:
				return fmt.Sprintf("%0.0f", v), nil
			case string:
				return v, nil
			default:
				return "", fiber.ErrUnauthorized
			}
		}
		if uidStr, ok := data["userId"].(string); ok && uidStr != "" {
			return uidStr, nil
		}
		return "", fiber.ErrUnauthorized
	}

	// Preview endpoints for MVP discovery
	app.Get("/v1/countries", func(c *fiber.Ctx) error {
		if opts.DB == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "database not connected"})
		}
		rows := []map[string]any{}
		if err := opts.DB.Select(&rows, `SELECT id, name, status, code, center_lat, center_lng FROM countries ORDER BY name LIMIT 300`); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		if len(rows) == 0 {
			// Lazy-seed a minimal set for dev if empty
			_, _ = opts.DB.Exec(`ALTER TABLE countries
  ADD COLUMN IF NOT EXISTS code TEXT UNIQUE,
  ADD COLUMN IF NOT EXISTS center_lat DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS center_lng DOUBLE PRECISION;`)
			_, _ = opts.DB.Exec(`INSERT INTO countries (name, code, status, center_lat, center_lng) VALUES
  ('United States', 'US', 'active', 39.8, -98.6),
  ('Canada', 'CA', 'active', 62.0, -96.8),
  ('United Kingdom', 'GB', 'active', 55.0, -2.7),
  ('Germany', 'DE', 'active', 51.2, 10.5),
  ('France', 'FR', 'active', 46.2, 2.2),
  ('Japan', 'JP', 'active', 36.2, 138.3),
  ('China', 'CN', 'active', 35.9, 104.2),
  ('India', 'IN', 'active', 20.6, 78.9),
  ('Brazil', 'BR', 'active', -14.2, -51.9),
  ('Australia', 'AU', 'active', -25.3, 133.8),
  ('South Africa', 'ZA', 'active', -30.6, 22.9),
  ('Kenya', 'KE', 'active', 0.02, 37.9),
  ('Nigeria', 'NG', 'active', 9.1, 8.7),
  ('Mexico', 'MX', 'active', 23.6, -102.6),
  ('Russia', 'RU', 'active', 61.5, 105.3)
ON CONFLICT (code) DO NOTHING;`)
			rows = []map[string]any{}
			_ = opts.DB.Select(&rows, `SELECT id, name, status, code, center_lat, center_lng FROM countries ORDER BY name LIMIT 300`)
		}
		return c.JSON(fiber.Map{"success": true, "data": rows})
	})

	app.Get("/v1/countries/geo", func(c *fiber.Ctx) error {
		if opts.DB == nil {
			return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"type": "FeatureCollection", "features": []any{}}})
		}
		type row struct {
			ID   string          `db:"id"`
			Name string          `db:"name"`
			Geo  json.RawMessage `db:"geo"`
		}
		rows := []row{}
		_ = opts.DB.Select(&rows, `SELECT c.id, c.name, g.geo FROM country_geoms g JOIN countries c ON c.id=g.country_id`)
		features := make([]any, 0, len(rows))
		for _, r := range rows {
			var tmp map[string]any
			if err := json.Unmarshal(r.Geo, &tmp); err == nil {
				if t, _ := tmp["type"].(string); t == "Feature" {
					if props, ok := tmp["properties"].(map[string]any); ok {
						props["id"] = r.ID
						props["name"] = r.Name
					} else {
						tmp["properties"] = map[string]any{"id": r.ID, "name": r.Name}
					}
					features = append(features, tmp)
					continue
				}
			}
			features = append(features, map[string]any{"type": "Feature", "properties": map[string]any{"id": r.ID, "name": r.Name}, "geometry": json.RawMessage(r.Geo)})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"type": "FeatureCollection", "features": features}})
	})

	app.Post("/v1/players/join", func(c *fiber.Ctx) error {
		if opts.DB == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false})
		}
		uid, err := getUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false})
		}
		var in struct {
			CountryID string `json:"countryId"`
		}
		if err := c.BodyParser(&in); err != nil || in.CountryID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "countryId required"})
		}
		// Upsert simple profile
		var id string
		q := `INSERT INTO player_profiles (user_id, country_id) VALUES ($1,$2)
              ON CONFLICT (user_id) DO UPDATE SET country_id=EXCLUDED.country_id, updated_at=now()
              RETURNING user_id`
		if err := opts.DB.Get(&id, q, uid, in.CountryID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		// Ensure personal loot exists
		if err := ensureLoot(opts.DB, "player", uid); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"userId": id, "countryId": in.CountryID}})
	})

	// Desert/switch with 24h cooldown
	app.Post("/v1/players/desert", func(c *fiber.Ctx) error {
		if opts.DB == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false})
		}
		uid, err := getUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false})
		}
		var in struct {
			CountryID string `json:"countryId"`
		}
		if err := c.BodyParser(&in); err != nil || in.CountryID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "countryId required"})
		}
		type prof struct {
			CountryID    *string    `db:"country_id"`
			LastDesertAt *time.Time `db:"last_desert_at"`
		}
		var p prof
		if err := opts.DB.Get(&p, `SELECT country_id, last_desert_at FROM player_profiles WHERE user_id=$1`, uid); err != nil {
			if err == sql.ErrNoRows {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "join a country first"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		hours := opts.DesertCooldownHours
		if hours <= 0 {
			hours = 24
		}
		cooldown := time.Duration(hours) * time.Hour
		// Allow customization via env exposed in AdminUsers string for now not ideal; better to pass separate field
		// Keep 24h until we thread cfg through Options in future update
		if p.LastDesertAt != nil && time.Since(*p.LastDesertAt) < cooldown {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "desert cooldown active"})
		}
		if _, err := opts.DB.Exec(`UPDATE player_profiles SET country_id=$1, last_desert_at=now(), updated_at=now() WHERE user_id=$2`, in.CountryID, uid); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// Loot endpoints
	app.Get("/v1/players/me/loot", func(c *fiber.Ctx) error {
		if opts.DB == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false})
		}
		uid, err := getUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false})
		}
		if err := ensureLoot(opts.DB, "player", uid); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		var resources json.RawMessage
		if err := opts.DB.Get(&resources, `SELECT COALESCE(resources,'{}'::jsonb) FROM loots WHERE owner_type='player' AND owner_id=$1`, uid); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"resources": json.RawMessage(resources)}})
	})

	app.Post("/v1/loot/contribute", func(c *fiber.Ctx) error {
		if opts.DB == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false})
		}
		uid, err := getUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false})
		}
		var in struct {
			Resources map[string]int64 `json:"resources"`
		}
		if err := c.BodyParser(&in); err != nil || len(in.Resources) == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "resources required"})
		}
		// Resolve player's country
		var countryID string
		if err := opts.DB.Get(&countryID, `SELECT country_id FROM player_profiles WHERE user_id=$1`, uid); err != nil || countryID == "" {
			return c.Status(409).JSON(fiber.Map{"success": false, "message": "no country"})
		}
		// Ensure both loots exist
		if err := ensureLoot(opts.DB, "player", uid); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		if err := ensureLoot(opts.DB, "country", countryID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		// Load current player resources
		var cur map[string]int64
		if err := getResources(opts.DB, "player", uid, &cur); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		// Check and subtract
		for k, v := range in.Resources {
			if v < 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid amount"})
			}
			if cur[k] < v {
				return c.Status(409).JSON(fiber.Map{"success": false, "message": "insufficient resources"})
			}
			cur[k] -= v
		}
		if err := setResources(opts.DB, "player", uid, cur); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		// Add to country loot
		var base map[string]int64
		_ = getResources(opts.DB, "country", countryID, &base)
		for k, v := range in.Resources {
			base[k] += v
		}
		if err := setResources(opts.DB, "country", countryID, base); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// Admin seed resources (dev convenience). Only for ADMIN_USER_IDS.
	app.Post("/v1/admin/seed/resources", func(c *fiber.Ctx) error {
		if opts.DB == nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		uid, err := getUserID(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"success": false})
		}
		if !isAdmin(uid) {
			return c.Status(403).JSON(fiber.Map{"success": false})
		}
		var in struct {
			UserID    string           `json:"userId"`
			Resources map[string]int64 `json:"resources"`
		}
		if err := c.BodyParser(&in); err != nil || len(in.Resources) == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false})
		}
		target := in.UserID
		if target == "" {
			target = uid
		}
		if err := ensureLoot(opts.DB, "player", target); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		var cur map[string]int64
		if err := getResources(opts.DB, "player", target, &cur); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		for k, v := range in.Resources {
			cur[k] += v
		}
		if err := setResources(opts.DB, "player", target, cur); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// Optional GET variant for dev convenience (query params), guarded by AdminGetSeed flag
	app.Get("/v1/admin/seed/resources", func(c *fiber.Ctx) error {
		if !opts.AdminGetSeed {
			return c.Status(405).JSON(fiber.Map{"success": false, "message": "GET not enabled"})
		}
		if opts.DB == nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		uid, err := getUserID(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"success": false})
		}
		if !isAdmin(uid) {
			return c.Status(403).JSON(fiber.Map{"success": false})
		}
		target := c.Query("userId")
		if target == "" {
			target = uid
		}
		res := map[string]int64{}
		for _, k := range []string{"metal", "fuel", "intel", "jeep", "tank", "radar"} {
			if v := c.Query(k); v != "" {
				if n, err := strconv.ParseInt(v, 10, 64); err == nil {
					res[k] = n
				}
			}
		}
		if len(res) == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "no resources"})
		}
		if err := ensureLoot(opts.DB, "player", target); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		var cur map[string]int64
		if err := getResources(opts.DB, "player", target, &cur); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		for k, v := range res {
			cur[k] += v
		}
		if err := setResources(opts.DB, "player", target, cur); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// Minimal HTML form (dev) to seed via POST
	app.Get("/admin/seed", func(c *fiber.Ctx) error {
		if !opts.AdminGetSeed {
			return c.Status(404).SendString("not found")
		}
		return c.Type("html").SendString(`<!doctype html>
<html><head><meta charset="utf-8"><title>Seed Resources</title></head>
<body>
  <h1>Seed Resources</h1>
  <form method="post" action="/v1/admin/seed/resources">
    <label>User ID (optional): <input name="userId" /></label><br/>
    <label>Metal: <input name="resources.metal" type="number" value="1000"/></label><br/>
    <label>Fuel: <input name="resources.fuel" type="number" value="500"/></label><br/>
    <label>Intel: <input name="resources.intel" type="number" value="200"/></label><br/>
    <button type="submit">Seed</button>
  </form>
</body></html>`)
	})

	// Craft endpoints
	app.Post("/v1/craft/orders", func(c *fiber.Ctx) error {
		if opts.DB == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false})
		}
		uid, err := getUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false})
		}
		var in struct {
			RecipeID string `json:"recipeId"`
		}
		if err := c.BodyParser(&in); err != nil || in.RecipeID == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "recipeId required"})
		}
		// Simple costs for MVP
		cost := recipeCost(in.RecipeID)
		if cost == nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "unknown recipe"})
		}
		if err := ensureLoot(opts.DB, "player", uid); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		var cur map[string]int64
		if err := getResources(opts.DB, "player", uid, &cur); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		for k, v := range cost {
			if cur[k] < v {
				return c.Status(409).JSON(fiber.Map{"success": false, "message": "insufficient resources"})
			}
			cur[k] -= v
		}
		if err := setResources(opts.DB, "player", uid, cur); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false})
		}
		mins := opts.CraftMinutes
		if mins <= 0 {
			mins = 10
		}
		completes := time.Now().Add(time.Duration(mins) * time.Minute)
		var orderID string
		if err := opts.DB.Get(&orderID, `INSERT INTO craft_orders (player_id, recipe_id, completes_at) VALUES ($1,$2,$3) RETURNING id`, uid, in.RecipeID, completes); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"id": orderID, "completesAt": completes}})
	})

	app.Get("/v1/craft/orders", func(c *fiber.Ctx) error {
		if opts.DB == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false})
		}
		uid, err := getUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false})
		}
		rows := []map[string]any{}
		if err := opts.DB.Select(&rows, `SELECT id, recipe_id, started_at, completes_at, status FROM craft_orders WHERE player_id=$1 ORDER BY started_at DESC`, uid); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "data": rows})
	})

	return app
}

// Helpers
func ensureLoot(db *sqlx.DB, ownerType, ownerID string) error {
	_, err := db.Exec(`INSERT INTO loots (owner_type, owner_id, resources) VALUES ($1,$2,'{}'::jsonb)
        ON CONFLICT (id) DO NOTHING`, ownerType, ownerID)
	if err == nil {
		// Try upsert by unique composite if exists; ensure row anyway
		_, _ = db.Exec(`INSERT INTO loots (id, owner_type, owner_id, resources)
            SELECT uuid_generate_v4(), $1, $2, '{}'::jsonb
            WHERE NOT EXISTS (SELECT 1 FROM loots WHERE owner_type=$1 AND owner_id=$2)`, ownerType, ownerID)
	}
	return nil
}

func getResources(db *sqlx.DB, ownerType, ownerID string, out *map[string]int64) error {
	var raw json.RawMessage
	if err := db.Get(&raw, `SELECT COALESCE(resources,'{}'::jsonb) FROM loots WHERE owner_type=$1 AND owner_id=$2`, ownerType, ownerID); err != nil {
		if err == sql.ErrNoRows {
			*out = map[string]int64{}
			return nil
		}
		return err
	}
	m := map[string]int64{}
	_ = json.Unmarshal(raw, &m)
	*out = m
	return nil
}

func setResources(db *sqlx.DB, ownerType, ownerID string, m map[string]int64) error {
	b, _ := json.Marshal(m)
	_, err := db.Exec(`UPDATE loots SET resources=$1, updated_at=now() WHERE owner_type=$2 AND owner_id=$3`, b, ownerType, ownerID)
	return err
}

func recipeCost(id string) map[string]int64 {
	switch id {
	case "jeep":
		return map[string]int64{"metal": 100, "fuel": 50}
	case "tank":
		return map[string]int64{"metal": 500, "fuel": 200}
	case "radar":
		return map[string]int64{"metal": 200, "intel": 150}
	default:
		return nil
	}
}
