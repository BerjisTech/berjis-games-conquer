import { environment as buildEnv } from '../environments/environment';

export type RuntimeConfig = {
  apiBase: string;
  coreAuthBase: string;
  mapboxToken: string;
  mapboxStyle: string;
};

export function getConfig(): RuntimeConfig {
  const win: any = typeof window !== 'undefined' ? (window as any) : {};
  const env = (win.__ENV || {}) as Partial<RuntimeConfig>;
  return {
    apiBase: env.apiBase || buildEnv.apiBase,
    coreAuthBase: env.coreAuthBase || buildEnv.coreAuthBase,
    mapboxToken: env.mapboxToken || buildEnv.mapboxToken,
    mapboxStyle: env.mapboxStyle || buildEnv.mapboxStyle,
  };
}
