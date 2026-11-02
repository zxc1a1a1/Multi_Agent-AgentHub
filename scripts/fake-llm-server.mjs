// Test-only fake Anthropic-compatible /v1/messages endpoint.
//
// It exists so the Docker/compose smoke test (plan R0-04) can drive the whole
// stack without a real provider key. It returns a deterministic plan when the
// prompt looks like an orchestrator planning request, and plain text otherwise.
//
// Usage: node scripts/fake-llm-server.mjs [port]
// Nothing here reads .env or any credential file; the API key presented by the
// client is ignored and never logged.
import http from 'node:http';

const port = Number(process.argv[2] || process.env.FAKE_LLM_PORT || 18081);

function textOf(body) {
  const chunks = [];
  if (typeof body?.system === 'string') chunks.push(body.system);
  for (const message of body?.messages ?? []) {
    const content = message?.content;
    if (typeof content === 'string') chunks.push(content);
    else if (Array.isArray(content)) {
      for (const block of content) {
        if (typeof block?.text === 'string') chunks.push(block.text);
      }
    }
  }
  return chunks.join('\n');
}

function lastUserText(body) {
  const messages = body?.messages ?? [];
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i]?.role !== 'user') continue;
    const content = messages[i].content;
    if (typeof content === 'string') return content;
    if (Array.isArray(content)) {
      const text = content.map((block) => block?.text ?? '').join(' ');
      if (text.trim()) return text;
    }
  }
  return '';
}

function allowedAgentsFrom(prompt) {
  const boundary = prompt.match(/Only select agents from: ([^\n]+?)\./i);
  const listed = (boundary ? boundary[1] : '')
    .split(/[,，\n]/)
    .map((name) => name.trim().toLowerCase())
    .filter((name) => /^[a-z0-9-]+$/.test(name));
  if (listed.length) return [...new Set(listed)];
  const section = prompt.match(/## Allowed Agent Boundary\n([\s\S]{0,400})/i);
  if (section) {
    const names = section[1]
      .split('\n')
      .map((line) => line.replace(/[-*:]/g, '').trim().toLowerCase())
      .filter((name) => /^[a-z0-9-]+$/.test(name));
    if (names.length) return [...new Set(names)];
  }
  return [];
}

// The planner only accepts mode single|parallel|sequential|answer (see
// planner/normalizer.go mapModeToStrategy), so the fake must answer with one of
// those and stay inside the allowed agent boundary from the prompt.
function planningPlan(body) {
  const input = lastUserText(body).replace(/\s+/g, ' ').trim().slice(0, 200) || 'smoke task';
  const escape = (value) => JSON.stringify(value).slice(1, -1);
  const allowed = allowedAgentsFrom(textOf(body));
  const single = /single_chat path/i.test(textOf(body)) || allowed.length <= 1;
  const picked = single ? allowed.slice(0, 1) : allowed.slice(0, 3);
  const mode = single ? 'single' : 'sequential';
  const steps = (picked.length ? picked : ['code-agent']).map((agent, index) => ({
    id: `s${index + 1}`,
    agent_name: agent,
    input: `${escape(input)}${index > 0 ? ' (follow-up)' : ''}`,
    reason: 'fake llm deterministic plan',
    ...(index > 0 ? { depends_on: [`s${index}`] } : {}),
  }));
  return `{"intent":"smoke","mode":${JSON.stringify(mode)},"confidence":0.9,"steps":${JSON.stringify(steps)},"user_visible_summary":"smoke plan from fake llm","warnings":[]}`;
}

const server = http.createServer((req, res) => {
  if (req.method === 'GET' && req.url === '/health') {
    res.writeHead(200, { 'content-type': 'application/json' });
    res.end('{"status":"ok"}');
    return;
  }
  if (req.method !== 'POST' || !req.url.endsWith('/v1/messages')) {
    res.writeHead(404, { 'content-type': 'application/json' });
    res.end('{"type":"error","error":{"type":"not_found_error","message":"unsupported path"}}');
    return;
  }
  let raw = '';
  req.setEncoding('utf8');
  req.on('data', (chunk) => {
    raw += chunk;
    if (raw.length > 4_000_000) req.destroy();
  });
  req.on('end', () => {
    let body = {};
    try {
      body = JSON.parse(raw || '{}');
    } catch {
      res.writeHead(400, { 'content-type': 'application/json' });
      res.end('{"type":"error","error":{"type":"invalid_request_error","message":"body is not json"}}');
      return;
    }
    const prompt = textOf(body);
    const isPlanning = /agent_name|"steps"|execution plan|planner/i.test(prompt);
    const text = isPlanning ? planningPlan(body) : 'fake llm deterministic reply';
    const payload = {
      id: `msg_fake_${Date.now()}`,
      type: 'message',
      role: 'assistant',
      content: [{ type: 'text', text }],
      model: body.model ?? 'fake-model',
      stop_reason: 'end_turn',
      usage: { input_tokens: Math.ceil(raw.length / 4), output_tokens: Math.ceil(text.length / 4) },
    };
    const encoded = JSON.stringify(payload);
    console.log(
      `[fake-llm] planning=${isPlanning} in_bytes=${raw.length} out_bytes=${encoded.length}`,
    );
    res.writeHead(200, {
      'content-type': 'application/json',
      'request-id': payload.id,
    });
    res.end(encoded);
  });
});

server.listen(port, '0.0.0.0', () => {
  console.log(`[fake-llm] listening on 0.0.0.0:${port} (test-only, ignores any api key)`);
});
