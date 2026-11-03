import { expect, test } from '@playwright/test'
import { setupMocks } from './mocks'

test.beforeEach(async ({ page }) => {
  await setupMocks(page)
})

test('mobile 375x667 has no horizontal overflow and keeps the composer usable', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 667 })
  await page.goto('/')
  // The sidebar is hidden below md but remains in the DOM.
  await page.getByTitle('New conversation').evaluate((el) => (el as HTMLElement).click())
  await expect(page.getByPlaceholder('Type a message...')).toBeVisible()
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(overflow).toBeLessThanOrEqual(1)
})

test('keyboard can select an agent, focus-visible the composer and send with Enter', async ({ page }) => {
  await page.goto('/')
  await page.click('[title="New conversation"]')

  const trigger = page.getByTestId('agent-picker-trigger')
  await trigger.focus()
  await page.keyboard.press('Enter')
  const search = page.getByPlaceholder('Search agents...')
  await expect(search).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(search).toHaveCount(0)

  const input = page.getByPlaceholder('Type a message...')
  await input.focus()
  await input.fill('keyboard message')
  await page.keyboard.press('Tab')
  const active = await page.evaluate(() => ({
    tag: document.activeElement?.tagName ?? '',
    focusVisible: document.activeElement?.matches(':focus-visible') ?? false,
  }))
  expect(['BUTTON', 'A', 'INPUT', 'TEXTAREA', 'SELECT']).toContain(active.tag)
  expect(active.focusVisible).toBe(true)
  await page.keyboard.press('Enter')
  await expect(page.getByText('keyboard message')).toBeVisible()
})
