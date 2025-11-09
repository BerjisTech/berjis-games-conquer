import 'zone.js';
import { bootstrapApplication } from '@angular/platform-browser';
import { provideHttpClient } from '@angular/common/http';
import { provideRouter, Routes } from '@angular/router';
import { CORE_AUTH_API_BASE, createAuthGuard } from '@berjis/angular-auth';
import { AppComponent } from './app/app.component';
import { getConfig } from './app/config';

const runtimeConfig = getConfig();
const authGuard = createAuthGuard({
  ensureOptions: { maxAgeMs: 1500 }
});

const routes: Routes = [
  { path: '', loadComponent: () => import('./app/home.component').then(m => m.HomeComponent), canActivate: [authGuard] },
  { path: '**', redirectTo: '' }
];

bootstrapApplication(AppComponent, {
  providers: [
    provideHttpClient(),
    provideRouter(routes),
    { provide: CORE_AUTH_API_BASE, useValue: runtimeConfig.coreAuthBase }
  ]
}).catch(err => console.error(err));
