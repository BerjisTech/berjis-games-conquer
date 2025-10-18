package worker

import (
    "database/sql"
    "encoding/json"
    "log"
    "time"

    "github.com/jmoiron/sqlx"
)

type Order struct {
    ID        string    `db:"id"`
    PlayerID  string    `db:"player_id"`
    RecipeID  string    `db:"recipe_id"`
    Status    string    `db:"status"`
    Completes time.Time `db:"completes_at"`
}

func StartCraftWorker(db *sqlx.DB, interval time.Duration, logger *log.Logger) chan struct{} {
    stop := make(chan struct{})
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                if db != nil { _ = processDueOrders(db, logger) }
            case <-stop:
                return
            }
        }
    }()
    return stop
}

func processDueOrders(db *sqlx.DB, logger *log.Logger) error {
    rows := []Order{}
    if err := db.Select(&rows, `SELECT id, player_id, recipe_id, status, completes_at FROM craft_orders WHERE status='queued' AND completes_at <= now() LIMIT 50`); err != nil {
        return err
    }
    for _, o := range rows {
        tx, err := db.Beginx()
        if err != nil { return err }
        // Mark completed
        if _, err := tx.Exec(`UPDATE craft_orders SET status='completed' WHERE id=$1 AND status='queued'`, o.ID); err != nil {
            tx.Rollback(); return err
        }
        // Award 1 unit of crafted asset into player's loot resources as a count
        if err := ensureLoot(tx, "player", o.PlayerID); err != nil { tx.Rollback(); return err }
        var m map[string]int64
        if err := getResources(tx, "player", o.PlayerID, &m); err != nil { tx.Rollback(); return err }
        m[o.RecipeID] = m[o.RecipeID] + 1
        if err := setResources(tx, "player", o.PlayerID, m); err != nil { tx.Rollback(); return err }
        if err := tx.Commit(); err != nil { return err }
    }
    if logger != nil && len(rows) > 0 {
        logger.Printf("craft worker: processed %d orders", len(rows))
    }
    return nil
}

// Local helpers operating within tx
func ensureLoot(db *sqlx.Tx, ownerType, ownerID string) error {
    // Ensure row exists for owner
    var exists bool
    if err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM loots WHERE owner_type=$1 AND owner_id=$2)`, ownerType, ownerID); err != nil && err != sql.ErrNoRows {
        return err
    }
    if !exists {
        _, err := db.Exec(`INSERT INTO loots (id, owner_type, owner_id, resources) VALUES (uuid_generate_v4(), $1, $2, '{}'::jsonb)`, ownerType, ownerID)
        return err
    }
    return nil
}

func getResources(db *sqlx.Tx, ownerType, ownerID string, out *map[string]int64) error {
    var raw json.RawMessage
    if err := db.Get(&raw, `SELECT COALESCE(resources,'{}'::jsonb) FROM loots WHERE owner_type=$1 AND owner_id=$2`, ownerType, ownerID); err != nil {
        if err == sql.ErrNoRows { *out = map[string]int64{}; return nil }
        return err
    }
    m := map[string]int64{}
    _ = json.Unmarshal(raw, &m)
    *out = m
    return nil
}

func setResources(db *sqlx.Tx, ownerType, ownerID string, m map[string]int64) error {
    b, _ := json.Marshal(m)
    _, err := db.Exec(`UPDATE loots SET resources=$1, updated_at=now() WHERE owner_type=$2 AND owner_id=$3`, b, ownerType, ownerID)
    return err
}

