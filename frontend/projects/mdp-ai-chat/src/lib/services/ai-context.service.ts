import { Injectable, signal, computed } from '@angular/core';

export interface PageContext {
    page: string;
    [key: string]: unknown;
}

@Injectable({ providedIn: 'root' })
export class AiContextService {
    private readonly _context = signal<PageContext | null>(null);

    /** Current page context as a readonly signal */
    readonly context = this._context.asReadonly();

    /** Whether context is currently set */
    readonly hasContext = computed(() => this._context() !== null);

    /** Current page name */
    readonly currentPage = computed(() => this._context()?.page ?? '');

    /**
     * Set the current page context.
     * Called by consuming apps when the user navigates to a new page.
     */
    setContext(context: PageContext): void {
        this._context.set(context);
    }

    /**
     * Clear the current page context.
     */
    clearContext(): void {
        this._context.set(null);
    }

    /**
     * Get the current context value (for use in API calls).
     */
    getContext(): PageContext | null {
        return this._context();
    }
}
