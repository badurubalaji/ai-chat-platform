import { test, expect } from '@playwright/test';

test('chat flow with mocked backend', async ({ page }) => {
    // Mock the config endpoint to return a configured provider
    await page.route('**/api/v1/ai/config', async route => {
        if (route.request().method() === 'GET') {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    id: '123',
                    provider: 'openai',
                    model: 'gpt-4',
                    enabled: true,
                    settings: { temperature: 0.7 }
                })
            });
        } else {
            await route.continue();
        }
    });

    // Mock the chat endpoint to return a stream
    await page.route('**/api/v1/ai/chat', async route => {
        const responseBody = `data: {"content": "Hello"}\n\ndata: {"content": " world"}\n\ndata: {"content": "!"}\n\ndata: {}\n\n`;
        await route.fulfill({
            status: 200,
            contentType: 'text/event-stream',
            body: responseBody
        });
    });

    // Mock conversations endpoint
    await page.route('**/api/v1/ai/conversations', async route => {
        if (route.request().method() === 'GET') {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify([])
            });
        } else {
            await route.continue();
        }
    });


    // 1. Navigate to the app
    await page.goto('http://localhost:4200/');

    // 2. Verify title
    await expect(page).toHaveTitle(/AI Chat Platform Demo/);

    // 3. Type a message
    const input = page.locator('input[placeholder="Ask AI..."]'); // Adjust selector based on actual UI
    // Wait for sidebar or chat to load
    await expect(input).toBeVisible();

    await input.fill('Hello AI');

    // 4. Send message
    await page.keyboard.press('Enter');

    // 5. Verify user message appears
    await expect(page.getByText('Hello AI', { exact: true })).toBeVisible();

    // 6. Verify assistant response appears (mocked "Hello world!")
    await expect(page.getByText('Hello world!')).toBeVisible();
});
