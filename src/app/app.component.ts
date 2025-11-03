import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink],
  template: `
    <div class="min-h-screen flex flex-col bg-app-gradient bg-app-gradient-animated text-[color:var(--color-text-primary)] dark:text-[color:var(--color-text-dark)]">
      <button type="button" (click)="toggleTheme()"
              class="fixed top-4 right-4 z-50 rounded-full p-3 backdrop-blur shadow-brand bg-white/80 text-blue-900 hover:bg-white dark:bg-slate-800/80 dark:text-[color:var(--color-gold)]"
              aria-label="Toggle theme">
        <span class="material-symbols-outlined align-middle">{{ isDark ? 'light_mode' : 'dark_mode' }}</span>
      </button>
      <div class="hero">
        <h1>Conquer</h1>
        <p>World strategy game. Pick a country, gather resources, craft, and wage war. Auth and persistence handled by the main Berjis API.</p>
        <a routerLink="/" class="btn">Home</a>
      </div>
      <router-outlet />
    </div>
  `
})
export class AppComponent implements OnInit {
  isDark = false;
  ngOnInit(): void {
    const persisted = (localStorage.getItem('theme') || '').toLowerCase();
    const preferDark = persisted === 'dark';
    this.setTheme(preferDark ? 'dark' : 'light');
  }
  toggleTheme() { this.setTheme(this.isDark ? 'light' : 'dark'); }
  private setTheme(mode: 'light' | 'dark') {
    this.isDark = mode === 'dark';
    document.documentElement.classList.toggle('dark', mode === 'dark');
    try { localStorage.setItem('theme', mode); } catch {}
  }
}

