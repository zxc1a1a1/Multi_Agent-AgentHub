#!/usr/bin/env node
// Playwright entrypoint for Orchestrator screenshot/PDF artifacts.
// CommonJS keeps NODE_PATH usable for local verification and images.
const { chromium } = require('playwright')

function parseArgs(argv) {
  const args = { format: 'png', fullPage: false }
  for (let i = 0; i < argv.length; i += 1) {
    const item = argv[i]
    if (item === '--url') args.url = argv[++i]
    else if (item === '--html-file') args.htmlFile = argv[++i]
    else if (item === '--output') args.output = argv[++i]
    else if (item === '--format') args.format = argv[++i]
    else if (item === '--full-page') args.fullPage = true
  }
  return args
}

async function main() {
  const args = parseArgs(process.argv.slice(2))
  if ((!args.url === !args.htmlFile) || !args.output || !['png', 'pdf'].includes(args.format)) {
    console.error('usage: render.cjs (--url <url> | --html-file <path>) --output <path> --format png|pdf [--full-page]')
    process.exit(2)
  }
  const browser = await chromium.launch({
    channel: process.env.PLAYWRIGHT_CHANNEL || undefined,
    headless: true,
  })
  try {
    const page = await browser.newPage({ viewport: { width: 1280, height: 720 } })
    if (args.htmlFile) {
      // Markdown export is self-contained: block every external request so a
      // rendered document cannot reach internal or public network targets.
      await page.route('**/*', (route) => {
        const url = route.request().url()
        if (url.startsWith('data:') || url.startsWith('blob:') || url.startsWith('about:')) {
          return route.continue()
        }
        return route.abort()
      })
      const fs = require('fs')
      await page.setContent(fs.readFileSync(args.htmlFile, 'utf8'), { waitUntil: 'load' })
    } else {
      await page.goto(args.url, { waitUntil: 'networkidle', timeout: 30000 })
    }
    if (args.format === 'pdf') {
      await page.pdf({ path: args.output, format: 'A4', printBackground: true })
    } else {
      await page.screenshot({ path: args.output, fullPage: args.fullPage })
    }
  } finally {
    await browser.close()
  }
}

main().catch((err) => {
  console.error(err && err.message ? err.message : String(err))
  process.exit(1)
})
