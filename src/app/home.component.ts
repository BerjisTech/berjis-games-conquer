import { Component, AfterViewInit, OnDestroy, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import mapboxgl from 'mapbox-gl';
import { getConfig } from './config';
import { ApiService } from './api.service';
import { CoreAuthSession } from '@berjis/angular-auth';

type Country = { id: string; name: string; status: string; code?: string; center_lat?: number; center_lng?: number };

@Component({
  standalone: true,
  selector: 'app-home',
  imports: [CommonModule],
  template: `
    <section class="hero">
      <h2>Welcome to Conquer</h2>
      <p>Pick a country to join. Map shows seeded countries as markers.</p>
      <div class="map" id="map"></div>
      <div>
        <label>Country:</label>
        <select [value]="selected()" (change)="onSelect($any($event.target).value)">
          <option value="">-- choose --</option>
          <option *ngFor="let c of countries()" [value]="c.id">{{ c.name }}</option>
        </select>
        <button class="btn" [disabled]="!selected()" (click)="join()">Join</button>
      </div>
      <p *ngIf="!hasToken">Mapbox token is not set; set environment.mapboxToken.</p>
    </section>
  `
})
export class HomeComponent implements AfterViewInit, OnDestroy {
  private http = inject(HttpClient);
  private api = inject(ApiService);
  private removeSessionListener = this.api.onSessionChange((session: CoreAuthSession) => {
    this.authed = !!session?.valid;
  });

  countries = signal<Country[]>([]);
  selected = signal<string>('');
  map?: mapboxgl.Map;
  authed = false;

  get hasToken() { return !!getConfig().mapboxToken; }

  async ngAfterViewInit() {
    // Ensure user is authenticated (shared session via landing)
    try {
      const session = await this.api.ensureAuth();
      this.authed = !!session?.valid;
    } catch {
      this.authed = false;
    }

    // Load countries
    this.http.get<{ success: boolean; data: Country[] }>(`${getConfig().apiBase}/v1/countries`).subscribe(r => {
      this.countries.set(r?.data || []);
      this.addMarkers();
    });

    // Init mapbox
    const cfg = getConfig();
    if (!cfg.mapboxToken) return;
    (mapboxgl as any).accessToken = cfg.mapboxToken;
    this.map = new mapboxgl.Map({
      container: 'map',
      style: cfg.mapboxStyle,
      center: [0, 20],
      zoom: 1.3
    });
    this.map.on('load', () => {
      this.addMarkers();
      // Try to load polygon GeoJSON layer if available
      this.http.get<{ success: boolean; data: any }>(`${getConfig().apiBase}/v1/countries/geo`).subscribe(res => {
        const fc = res?.data;
        if (!fc || !fc.features || fc.features.length === 0) return;
        if (this.map!.getSource('countries')) return;
        this.map!.addSource('countries', { type: 'geojson', data: fc });
        this.map!.addLayer({ id: 'countries-fill', type: 'fill', source: 'countries', paint: { 'fill-color': '#3b82f6', 'fill-opacity': 0.15 } });
        this.map!.addLayer({ id: 'countries-line', type: 'line', source: 'countries', paint: { 'line-color': '#1f2937', 'line-width': 1 } });
        this.map!.on('click', 'countries-fill', (e: any) => {
          const f = e.features && e.features[0];
          if (f && f.properties && f.properties.id) {
            this.selected.set(f.properties.id);
          }
        });
        this.map!.on('mouseenter', 'countries-fill', () => this.map!.getCanvas().style.cursor = 'pointer');
        this.map!.on('mouseleave', 'countries-fill', () => this.map!.getCanvas().style.cursor = '');
      });
    });
  }

  ngOnDestroy() {
    this.removeSessionListener?.();
  }

  private addMarkers() {
    if (!this.map) return;
    this.countries().forEach(c => {
      if (c.center_lng != null && c.center_lat != null) {
        new mapboxgl.Marker({ color: '#111827' })
          .setLngLat([c.center_lng!, c.center_lat!])
          .setPopup(new mapboxgl.Popup().setText(c.name))
          .addTo(this.map!);
      }
    });
  }

  onSelect(id: string) { this.selected.set(id); }

  join() {
    if (!this.authed) { alert('Please sign in at berjis.tech first.'); return; }
    const id = this.selected(); if (!id) return;
    this.http.post(`${getConfig().apiBase}/v1/players/join`, { countryId: id }, { withCredentials: true })
      .subscribe(() => alert('Joined country.')); // MVP toast
  }
}
