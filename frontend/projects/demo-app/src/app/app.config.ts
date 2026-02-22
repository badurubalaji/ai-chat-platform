import { ApplicationConfig, provideZoneChangeDetection } from '@angular/core';
import { provideRouter } from '@angular/router';
import { routes } from './app.routes';
import { provideAiChat } from 'mdp-ai-chat';
import { of } from 'rxjs';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { provideHttpClient } from '@angular/common/http';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(routes),
    provideHttpClient(),
    provideAnimationsAsync(),
    provideAiChat({
      apiBaseUrl: 'http://localhost:8080',
      authTokenFn: () => of('mock-jwt-token'),
      appName: 'Demo App',
      appDescription: 'A demo application for the AI Chat library',
      sidebarWidth: 400
    })
  ]
};
