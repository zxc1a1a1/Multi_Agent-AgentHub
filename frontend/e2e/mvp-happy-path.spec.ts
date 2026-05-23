import { test, expect } from '@playwright/test'
import { setupMocks, buildErrorSSE } from './mocks'

test.describe('MVP E2E Happy Path', () => {
  test.beforeEach(async ({ page }) => {
    await setupMocks(page)
  })

  // ── 1. Open page ───────────────────────────────────────

  test('open page — shows welcome screen when no conversation selected', async ({ page }) => {
    await page.goto('/')

    await expect(page.getByText('Welcome to AgentHub')).toBeVisible()
    await expect(page.getByText('Select or create a conversation')).toBeVisible()
  })

  // ── 2. New conversation ────────────────────────────────

  test('new conversation — creates and activates a conversation', async ({ page }) => {
    await page.goto('/')

    await page.click('[title="New conversation"]')

    // Sidebar shows the new conversation
    await expect(page.getByText('New Conversation')).toBeVisible()
    // Main area shows the chat window ready for input
    await expect(page.getByText('Start a conversation')).toBeVisible()
    await expect(page.getByPlaceholder('Type a message...')).toBeVisible()
  })

  // ── 3. Send prompt — user message visible ──────────────

  test('send prompt — user message appears in chat', async ({ page }) => {
    await page.goto('/')
    await page.click('[title="New conversation"]')

    const input = page.getByPlaceholder('Type a message...')
    await input.fill('Write a Go hello world')
    await page.click('[title="Send message"]')

    // The user's message is immediately rendered in a bubble
    await expect(page.getByText('Write a Go hello world')).toBeVisible()
  })

  // ── 4. Streaming reply ─────────────────────────────────

  test('streaming reply — agent text response appears after send', async ({ page }) => {
    await page.goto('/')
    await page.click('[title="New conversation"]')

    const input = page.getByPlaceholder('Type a message...')
    await input.fill('Write a Go hello world')
    await page.click('[title="Send message"]')

    // The agent's streamed text is rendered
    await expect(page.getByText('Here is your Go hello world program:')).toBeVisible()
  })

  // ── 5. CodePreview renders ─────────────────────────────

  test('code preview — CodePreview shows filename, language, code, and line count', async ({ page }) => {
    await page.goto('/')
    await page.click('[title="New conversation"]')

    const input = page.getByPlaceholder('Type a message...')
    await input.fill('Write a Go hello world')
    await page.click('[title="Send message"]')

    // CodePreview header
    await expect(page.getByText('main.go')).toBeVisible()
    // Language badge and line count
    await expect(page.getByText(/go/i)).toBeVisible()
    await expect(page.getByText(/5 lines/)).toBeVisible()
    // Actual code content
    await expect(page.getByText('package main')).toBeVisible()
    await expect(page.getByText('"hello world"')).toBeVisible()
  })

  // ── 6. Copy code ───────────────────────────────────────

  test('copy code — clicking copy shows "Copied" feedback', async ({ page }) => {
    await page.goto('/')
    await page.click('[title="New conversation"]')

    const input = page.getByPlaceholder('Type a message...')
    await input.fill('Write a Go hello world')
    await page.click('[title="Send message"]')

    await page.click('[title="Copy code"]')
    await expect(page.getByText('Copied')).toBeVisible()
  })

  // ── 7. Refresh — history persists ──────────────────────

  test('refresh page — conversation history loads from API after reload', async ({ page }) => {
    await page.goto('/')

    // Click existing conversation in sidebar (loaded from mock API)
    await page.getByText('New Conversation').click()

    // History messages rendered from GET /api/conversations/:id/messages
    await expect(page.getByText('Write a Go hello world')).toBeVisible()
    await expect(page.getByText('Here is your Go hello world program:')).toBeVisible()
    // CodePreview from persisted artifacts
    await expect(page.getByText('main.go')).toBeVisible()
  })

  // ── 8. Error path — graceful UI on error ───────────────

  test('error handling — RUN_ERROR shows failure indicator without crash', async ({ page }) => {
    // Override the agui/run mock for this single test
    await page.route('**/api/agui/run', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/event-stream' },
        body: buildErrorSSE(),
      })
    })

    await page.goto('/')
    await page.click('[title="New conversation"]')

    const input = page.getByPlaceholder('Type a message...')
    await input.fill('trigger error')
    await page.click('[title="Send message"]')

    // Should show failure indicator, not a blank screen
    await expect(page.getByText('Failed to generate response')).toBeVisible()
  })
})
