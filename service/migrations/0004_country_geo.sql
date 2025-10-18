CREATE TABLE IF NOT EXISTS country_geoms (
  country_id UUID PRIMARY KEY REFERENCES countries(id) ON DELETE CASCADE,
  geo JSONB NOT NULL -- GeoJSON Feature or Polygon/MultiPolygon geometry
);

CREATE INDEX IF NOT EXISTS idx_country_geoms_gin ON country_geoms USING GIN (geo);

