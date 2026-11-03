import skillsSource from './skills?raw'
import { describe, expect, it } from 'vitest'
import {
  UI_COMPONENT_ARTIFACT_ID_PATTERN,
  UI_COMPONENT_EVENT_TYPE,
  UI_COMPONENT_PROPS_MAX_BYTES,
  UI_COMPONENT_SCHEMA_VERSION,
  describeUIComponentRejection,
  getUIComponentDefinition,
  isUIComponentCapability,
  isUIComponentName,
  safeComponentToken,
  uiCapabilityAllowlist,
  uiComponentAllowlist,
  uiComponentRegistry,
  validateUIComponentEvent,
} from './skills'

const PROPS_ENVELOPE_OVERHEAD = new TextEncoder().encode(JSON.stringify({ blob: '' })).length

/** props 载荷按 UTF-8 字节精确构造，用于 64KB 边界测试。 */
function propsWithByteSize(targetBytes: number): Record<string, string> {
  return { blob: 'a'.repeat(targetBytes - PROPS_ENVELOPE_OVERHEAD) }
}

function validEvent(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    type: UI_COMPONENT_EVENT_TYPE,
    component: 'html_sandbox',
    props: { title: 'demo' },
    schemaVersion: 1,
    artifactId: 'art_a1b2c3d4e5f6a1b2c3d4e5f6',
    capabilities: ['read_only'],
    ...overrides,
  }
}

function rejection(event: Record<string, unknown>) {
  const result = validateUIComponentEvent(event)
  if (result.ok) {
    throw new Error('expected validation to fail')
  }
  return result
}

describe('ui.component registry (R3-01)', () => {
  it('freezes exactly the four allowlisted components', () => {
    expect([...uiComponentAllowlist].sort()).toEqual(['chart', 'code_editor', 'html_sandbox', 'json_form'])
    expect(uiComponentRegistry.map((entry) => entry.component).sort()).toEqual([...uiComponentAllowlist].sort())
  })

  it('wires only the reviewed html_sandbox renderer (R3-02)', () => {
    for (const entry of uiComponentRegistry) {
      if (entry.component === 'html_sandbox') {
        expect(entry.renderer).toBe('html_sandbox')
      } else {
        expect(entry.renderer).toBeNull()
      }
      expect(entry.interactive).toBe(false)
      expect(entry.acceptsStreamingProps).toBe(false)
    }
    expect(getUIComponentDefinition('chart')?.component).toBe('chart')
    expect(getUIComponentDefinition('not_registered' as never)).toBeUndefined()
  })

  it('keeps the capability allowlist frozen and rejects unknown names', () => {
    expect([...uiCapabilityAllowlist].sort()).toEqual(['read_only', 'sandboxed_script', 'user_input'])
    expect(isUIComponentCapability('read_only')).toBe(true)
    expect(isUIComponentCapability('network_access')).toBe(false)
    expect(isUIComponentCapability('eval')).toBe(false)
    expect(isUIComponentCapability(undefined)).toBe(false)
  })

  it('exposes allowlist membership helpers', () => {
    expect(isUIComponentName('json_form')).toBe(true)
    expect(isUIComponentName('Chart')).toBe(false)
    expect(isUIComponentName('')).toBe(false)
    expect(isUIComponentName(42)).toBe(false)
  })
})

describe('validateUIComponentEvent — component allowlist', () => {
  it('accepts a fully valid event and keeps props untouched', () => {
    const result = validateUIComponentEvent(validEvent({ props: { items: [1, 'two', null] } }))
    expect(result).toMatchObject({
      ok: true,
      component: 'html_sandbox',
      schemaVersion: 1,
      artifactId: 'art_a1b2c3d4e5f6a1b2c3d4e5f6',
      capabilities: ['read_only'],
      renderer: 'html_sandbox',
    })
    if (!result.ok) throw new Error('unexpected rejection')
    expect(result.props).toEqual({ items: [1, 'two', null] })
  })

  it('accepts every allowlisted component', () => {
    for (const component of uiComponentAllowlist) {
      expect(validateUIComponentEvent(validEvent({ component })).ok).toBe(true)
    }
  })

  it('rejects components outside the allowlist', () => {
    expect(rejection(validEvent({ component: 'dangerous_iframe' })).reason).toBe('component_not_allowed')
    expect(rejection(validEvent({ component: 'eval_script' })).reason).toBe('component_not_allowed')
    expect(rejection(validEvent({ component: 'Chart' })).reason).toBe('component_not_allowed')
    expect(rejection(validEvent({ component: '' })).reason).toBe('component_not_allowed')
    expect(rejection(validEvent({ component: undefined })).reason).toBe('component_not_allowed')
    expect(rejection(validEvent({ component: 7 })).reason).toBe('component_not_allowed')
  })
})

describe('validateUIComponentEvent — schemaVersion', () => {
  it('requires strict numeric 1', () => {
    expect(UI_COMPONENT_SCHEMA_VERSION).toBe(1)
    expect(validateUIComponentEvent(validEvent({ schemaVersion: 1 })).ok).toBe(true)
    expect(rejection(validEvent({ schemaVersion: 2 })).reason).toBe('schema_version_unsupported')
    expect(rejection(validEvent({ schemaVersion: '1' })).reason).toBe('schema_version_unsupported')
    expect(rejection(validEvent({ schemaVersion: 0 })).reason).toBe('schema_version_unsupported')
    expect(rejection(validEvent({ schemaVersion: undefined })).reason).toBe('schema_version_unsupported')
    expect(rejection(validEvent({ schemaVersion: null })).reason).toBe('schema_version_unsupported')
  })
})

describe('validateUIComponentEvent — props', () => {
  it('requires a plain object', () => {
    expect(validateUIComponentEvent(validEvent({ props: {} })).ok).toBe(true)
    for (const props of [[], null, 'string', 42, true, undefined, () => 'x']) {
      expect(rejection(validEvent({ props: props as unknown })).reason).toBe('props_invalid')
    }
  })

  it('rejects non-plain prototypes such as class instances or boxed objects', () => {
    class Hostile {
      url = 'https://internal.example/evil.js'
    }
    expect(rejection(validEvent({ props: new Hostile() })).reason).toBe('props_invalid')
    expect(rejection(validEvent({ props: new Date() })).reason).toBe('props_invalid')
    expect(validateUIComponentEvent(validEvent({ props: Object.create(null) })).ok).toBe(true)
  })

  it('enforces the 64KB limit on serialized UTF-8 bytes, not characters', () => {
    expect(UI_COMPONENT_PROPS_MAX_BYTES).toBe(65536)
    expect(validateUIComponentEvent(validEvent({ props: propsWithByteSize(65536) })).ok).toBe(true)
    expect(rejection(validEvent({ props: propsWithByteSize(65537) })).reason).toBe('props_too_large')

    // 21842 个中文字符 < 65536，但序列化为 65537 字节：按字符计数会误放行。
    const multibyte = { blob: '中'.repeat(21842) }
    const bytes = new TextEncoder().encode(JSON.stringify(multibyte)).length
    expect(bytes).toBe(65537)
    expect(multibyte.blob.length).toBeLessThan(UI_COMPONENT_PROPS_MAX_BYTES)
    expect(rejection(validEvent({ props: multibyte })).reason).toBe('props_too_large')
  })

  it('never invokes anything carried in props', () => {
    let executed = false
    const withFunction = { run: () => { executed = true } }
    expect(rejection(validEvent({ props: withFunction })).reason).toBe('props_invalid')
    expect(executed).toBe(false)

    const withSymbol = { tag: Symbol('x') }
    expect(rejection(validEvent({ props: withSymbol })).reason).toBe('props_invalid')

    const withBigInt = { n: 10n }
    expect(rejection(validEvent({ props: withBigInt })).reason).toBe('props_invalid')

    let getterCalled = false
    const withGetter = {
      get hook() {
        getterCalled = true
        return 'https://internal.example/evil.js'
      },
    }
    expect(rejection(validEvent({ props: withGetter })).reason).toBe('props_invalid')
    expect(getterCalled).toBe(false)

    const circular: Record<string, unknown> = { name: 'loop' }
    circular.self = circular
    expect(rejection(validEvent({ props: circular })).reason).toBe('props_invalid')

    // toJSON 是函数值：必须在校验阶段被拒，而不是被 JSON.stringify 调用。
    let toJSONCalled = false
    const withToJSON = {
      toJSON: () => {
        toJSONCalled = true
        return { injected: true }
      },
    }
    expect(rejection(validEvent({ props: withToJSON })).reason).toBe('props_invalid')
    expect(toJSONCalled).toBe(false)
  })

  it('bounds nesting depth', () => {
    let deep: Record<string, unknown> = { leaf: 'value' }
    for (let level = 0; level < 70; level++) {
      deep = { child: deep }
    }
    expect(rejection(validEvent({ props: deep })).reason).toBe('props_invalid')

    let ok: Record<string, unknown> = { leaf: 'value' }
    for (let level = 0; level < 40; level++) {
      ok = { child: ok }
    }
    expect(validateUIComponentEvent(validEvent({ props: ok })).ok).toBe(true)
  })

  it('accepts hostile markup as inert data only', () => {
    const inert = { html: '<script>window.eval("alert(1)")</script>', url: 'javascript:alert(1)' }
    const result = validateUIComponentEvent(validEvent({ props: inert }))
    expect(result.ok).toBe(true)
    if (!result.ok) throw new Error('unexpected rejection')
    expect(result.props).toEqual(inert)
  })
})

describe('validateUIComponentEvent — artifactId', () => {
  it('matches the repository artifact id shape', () => {
    expect(UI_COMPONENT_ARTIFACT_ID_PATTERN).toEqual(/^art_[A-Za-z0-9_-]{1,64}$/)
    // 生成端实测格式：art_ + hex（pkg/runtime/tools/artifact.go、handler_render.go）。
    for (const artifactId of [
      'art_a1b2c3d4e5f6a1b2c3d4e5f6',
      'art_deadbeefdeadbeefdeadbeefdeadbeef',
      'art_01HXYZABCDEF0123456789abcdef',
      'art_1234abcd',
    ]) {
      expect(validateUIComponentEvent(validEvent({ artifactId })).ok).toBe(true)
    }
  })

  it('rejects missing, prefixed-wrong, hostile or oversized ids', () => {
    for (const artifactId of [
      undefined,
      null,
      '',
      'art_',
      'artifact_1',
      'art',
      'art_has space',
      'art_<script>alert(1)</script>',
      'art_../../etc/passwd',
      'art_' + 'a'.repeat(65),
      1234,
    ]) {
      expect(rejection(validEvent({ artifactId: artifactId as unknown })).reason).toBe('artifact_id_invalid')
    }
    expect(validateUIComponentEvent(validEvent({ artifactId: 'art_' + 'a'.repeat(64) })).ok).toBe(true)
  })
})

describe('validateUIComponentEvent — capabilities', () => {
  it('requires a subset of the known capability names', () => {
    expect(validateUIComponentEvent(validEvent({ capabilities: [] })).ok).toBe(true)
    expect(validateUIComponentEvent(validEvent({ capabilities: ['read_only', 'user_input'] })).ok).toBe(true)
    expect(rejection(validEvent({ capabilities: ['network_access'] })).reason).toBe('capabilities_invalid')
    expect(rejection(validEvent({ capabilities: ['file_write'] })).reason).toBe('capabilities_invalid')
    expect(rejection(validEvent({ capabilities: ['read_only', 'eval'] })).reason).toBe('capabilities_invalid')
    expect(rejection(validEvent({ capabilities: 'read_only' })).reason).toBe('capabilities_invalid')
    expect(rejection(validEvent({ capabilities: ['read_only', 2] })).reason).toBe('capabilities_invalid')
    expect(rejection(validEvent({ capabilities: undefined })).reason).toBe('capabilities_invalid')
  })

  it('copies the capability list instead of aliasing the event payload', () => {
    const capabilities = ['read_only']
    const result = validateUIComponentEvent(validEvent({ capabilities }))
    if (!result.ok) throw new Error('unexpected acceptance')
    expect(result.capabilities).toEqual(['read_only'])
    expect(result.capabilities).not.toBe(capabilities)
  })
})

describe('validateUIComponentEvent — event shape', () => {
  it('rejects non-objects and foreign event types', () => {
    for (const raw of [null, undefined, 'ui.component', 42, [], ['ui.component']]) {
      const result = validateUIComponentEvent(raw)
      expect(result.ok).toBe(false)
      if (result.ok) throw new Error('unexpected acceptance')
      expect(result.reason).toBe('event_shape_invalid')
    }
    expect(rejection({ ...validEvent(), type: 'ui.component.v2' }).reason).toBe('event_shape_invalid')
    expect(rejection({ component: 'chart', schemaVersion: 1 }).reason).toBe('event_shape_invalid')
  })

  it('checks component before every other field', () => {
    // 未知组件即便其余字段合法也必须拒绝。
    expect(rejection(validEvent({ component: 'webshell', schemaVersion: 2, props: 'x' })).reason).toBe(
      'component_not_allowed',
    )
  })
})

describe('fallback display tokens', () => {
  it('sanitizes tokens to a safe display charset', () => {
    expect(safeComponentToken('html_sandbox')).toBe('html_sandbox')
    expect(safeComponentToken('art_a1b2c3d4')).toBe('art_a1b2c3d4')
    expect(safeComponentToken(1)).toBe('1')
    expect(safeComponentToken('<script>alert(1)</script>')).toBe('<redacted>')
    expect(safeComponentToken('has space')).toBe('<redacted>')
    expect(safeComponentToken('a'.repeat(65))).toBe('<redacted>')
    expect(safeComponentToken(undefined)).toBe('<redacted>')
    expect(safeComponentToken(null)).toBe('<redacted>')
    expect(safeComponentToken({ nested: 'object' })).toBe('<redacted>')
  })

  it('describes rejections without echoing props', () => {
    const hostile = validEvent({
      component: 'xss_card',
      artifactId: 'art_<img onerror=alert(1)>',
      props: { secret: 'props-原文-不应出现' },
    })
    const args = describeUIComponentRejection(hostile, 'component_not_allowed')
    expect(args).toEqual({
      reason: 'component_not_allowed',
      // 名称字符集安全时保留原名，便于调试；不安全时整体替换。
      component: 'xss_card',
      artifactId: '<redacted>',
      schemaVersion: '1',
    })
    expect(JSON.stringify(args)).not.toContain('props-原文')
    expect(JSON.stringify(args)).not.toContain('onerror')

    expect(
      describeUIComponentRejection(validEvent({ component: '<script>alert(1)</script>' }), 'component_not_allowed')
        .component,
    ).toBe('<redacted>')
  })

  it('describes non-object events safely', () => {
    expect(describeUIComponentRejection('nope', 'event_shape_invalid')).toEqual({
      reason: 'event_shape_invalid',
      component: '<redacted>',
      artifactId: '<redacted>',
      schemaVersion: '<redacted>',
    })
  })
})

describe('no-eval guarantees', () => {
  const codeOnly = skillsSource.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/[^\n]*/g, '')

  it('registry module contains no executable-code sink', () => {
    expect(codeOnly).toContain('export function validateUIComponentEvent')
    expect(codeOnly).not.toMatch(/\beval\s*\(/)
    expect(codeOnly).not.toMatch(/new\s+Function/)
    expect(codeOnly).not.toMatch(/\bFunction\s*\(/)
    expect(codeOnly).not.toMatch(/\bimport\s*\(/)
    expect(codeOnly).not.toMatch(/\brequire\s*\(/)
    expect(codeOnly).not.toMatch(/dangerouslySetInnerHTML/)
    expect(codeOnly).not.toMatch(/document\.write/)
    expect(codeOnly).not.toMatch(/setTimeout\s*\(\s*['"`]/)
  })

  it('registry module never treats props as executable', () => {
    // 校验只读描述符与 JSON.stringify，不调用事件携带的任何函数。
    expect(codeOnly).not.toMatch(/props\.[A-Za-z_$][\w$]*\s*\(/)
    expect(codeOnly).not.toMatch(/event\.props\s*\(/)
    expect(codeOnly).toContain('hasOnlyJSONValues')
    expect(codeOnly).toContain('getOwnPropertyDescriptor')
  })
})
