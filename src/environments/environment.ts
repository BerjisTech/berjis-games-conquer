export const environment = {
  production: false,
  apiBase: 'http://conquer-api.berjis.test',
  coreAuthBase: (typeof window !== 'undefined' && (window as any).__BERJIS_API__) || 'https://api.berjis.tech',
  mapboxToken: 'pk.eyJ1IjoibG9yZHNvbWJvIiwiYSI6ImNsNWJjOGdoejA2NXQzanNlNGpqb2Y5d3EifQ.YyGLTwRRa6ZqeNK7c93Rig', // TODO: set MAPBOX token for local dev
  mapboxStyle: 'mapbox://styles/mapbox/streets-v12'
};
