import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink],
  template: `
    <div class="hero">
      <h1>Conquer</h1>
      <p>World strategy game. Pick a country, gather resources, craft, and wage war. Auth and persistence handled by the main Berjis API.</p>
      <a routerLink="/" class="btn">Home</a>
    </div>
    <router-outlet />
  `
})
export class AppComponent {}

