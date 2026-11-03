#!/usr/bin/env node
/**
 * R3-07 build budget gate.
 * - first-screen entry JS gzip must stay <= 500KB;
 * - heavy surfaces (RJSF, charts, Streamdown, Mermaid, Sandpack) must not be
 *   referenced from index.html as preloads/static imports.
 */
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { gzipSync } from 'node:zlib'

const DIST = fileURLToPath(new URL('../dist/', import.meta.url))
const ENTRY_LIMIT = 500 * 1024
const HEAVY = ['JsonSchemaForm', 'DeclarativeChart', 'streamdown', 'mermaid', 'Sandpack', 'sandpack']

function fail(message) {
  console.error(`build budget: ${message}`)
  process.exit(1)
}

let html
try {
  html = readFileSync(join(DIST, 'index.html'), 'utf8')
} catch {
  fail('dist/index.html not found; run npm run build first')
}

const entryMatches = [...html.matchAll(/<script[^>]+src="([^"]+)"/g)].map((m) => m[1])
if (entryMatches.length === 0) fail('no entry script found in index.html')

let totalGzip = 0
for (const src of entryMatches) {
  const file = join(DIST, src.replace(/^\//, ''))
  const bytes = readFileSync(file)
  const gzip = gzipSync(bytes, { level: 9 }).length
  totalGzip += gzip
  console.log(`entry ${src}: ${bytes.length} B raw / ${gzip} B gzip`)
}
console.log(`entry gzip total: ${totalGzip} B (limit ${ENTRY_LIMIT} B)`)
if (totalGzip > ENTRY_LIMIT) fail(`first-screen gzip ${totalGzip} exceeds ${ENTRY_LIMIT}`)

for (const name of HEAVY) {
  if (html.includes(name)) fail(`heavy chunk "${name}" is referenced from index.html`)
}

for (const file of readdirSync(join(DIST, 'assets'))) {
  const full = join(DIST, 'assets', file)
  if (!statSync(full).isFile()) continue
  const size = statSync(full).size
  if (size > 5 * 1024 * 1024) fail(`asset ${file} exceeds 5MB`)
}

console.log('build budget: PASS')
