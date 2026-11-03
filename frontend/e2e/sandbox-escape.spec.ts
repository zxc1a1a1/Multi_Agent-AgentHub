import { expect, test } from '@playwright/test'

// R3-02: the exact sandbox/CSP policy used by SandboxedHtml. The child document
// tries to read the parent, call fetch and load an image; every attempt must be
// blocked and no request may leave the browser.
const CSP =
  "default-src 'none'; img-src data: blob:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'none'; frame-src 'none'; form-action 'none'; base-uri 'none'"

const CHILD_HTML = `<!doctype html><html><head>
<meta http-equiv="Content-Security-Policy" content="${CSP}">
</head><body>
<script>
(async function () {
  const out = { probe: true, parentAccess: 'unknown', fetch: 'unknown', img: 'unknown' };
  try { parent.document.title = 'pwn'; out.parentAccess = 'allowed'; }
  catch (e) { out.parentAccess = 'blocked'; }
  try { await fetch('http://169.254.169.254/latest/meta-data/'); out.fetch = 'allowed'; }
  catch (e) { out.fetch = 'blocked'; }
  await new Promise(function (resolve) {
    const img = new Image();
    img.onload = function () { out.img = 'allowed'; resolve(); };
    img.onerror = function () { out.img = 'blocked'; resolve(); };
    img.src = 'http://127.0.0.1:9/probe.png';
    setTimeout(resolve, 1000);
  });
  parent.postMessage(out, '*');
})();
</script></body></html>`

test('sandboxed HTML cannot read the parent or reach the network', async ({ page }) => {
  const leaked: string[] = []
  page.on('request', (request) => {
    const url = request.url()
    if (url.includes('169.254.169.254') || url.includes('127.0.0.1:9') || url.includes('internal.example')) {
      leaked.push(url)
    }
  })

  await page.setContent('<!doctype html><html><body><iframe id="probe-frame" sandbox="allow-scripts" referrerpolicy="no-referrer"></iframe></body></html>')
  await page.evaluate(() => {
    ;(window as unknown as { __probe: Promise<Record<string, unknown>> }).__probe = new Promise((resolve) => {
      window.addEventListener('message', (event) => {
        const data = event.data as Record<string, unknown> | null
        if (data && data.probe === true) resolve(data)
      })
    })
  })
  await page.evaluate((html) => {
    const frame = document.getElementById('probe-frame') as HTMLIFrameElement | null
    if (!frame) throw new Error('probe frame missing')
    frame.srcdoc = html
  }, CHILD_HTML)

  const result = await page.evaluate(
    () => (window as unknown as { __probe: Promise<Record<string, unknown>> }).__probe,
  )
  expect(result.parentAccess).toBe('blocked')
  expect(result.fetch).toBe('blocked')
  expect(result.img).toBe('blocked')
  await page.waitForTimeout(500)
  expect(leaked).toEqual([])
})
