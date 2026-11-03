import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import JsonSchemaForm, {
  SCHEMA_MAX_ENUM,
  SCHEMA_MAX_FIELDS,
  sanitizeFormSchema,
} from './JsonSchemaForm'
import { sendToolResult } from '../../services/api'

vi.mock('../../services/api', () => ({
  sendToolResult: vi.fn(),
}))

const mockedSend = vi.mocked(sendToolResult)

const validSchema = {
  type: 'object',
  title: 'Sample',
  required: ['name'],
  properties: {
    name: { type: 'string', title: 'Name' },
    count: { type: 'integer', title: 'Count', minimum: 0 },
    agree: { type: 'boolean', title: 'Agree' },
    color: { type: 'string', title: 'Color', enum: ['red', 'blue'] },
  },
}

function deepSchema(depth: number): Record<string, unknown> {
  let node: Record<string, unknown> = { type: 'string' }
  for (let i = 0; i < depth; i += 1) {
    node = { type: 'object', properties: { child: node } }
  }
  return node
}

beforeEach(() => {
  mockedSend.mockReset()
  mockedSend.mockResolvedValue(undefined)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('sanitizeFormSchema', () => {
  it('accepts a small object schema and bounds strings', () => {
    const result = sanitizeFormSchema(validSchema)
    expect(result.ok).toBe(true)
    if (!result.ok) return
    const props = result.schema.properties as Record<string, Record<string, unknown>>
    expect(props.name.maxLength).toBeGreaterThan(0)
  })

  it('rejects $ref, __proto__, javascript defaults and data-url formats', () => {
    const ref = sanitizeFormSchema({ type: 'object', properties: { a: { $ref: '#/x' } } })
    expect(ref.ok).toBe(false)
    const proto = sanitizeFormSchema(
      JSON.parse('{"type":"object","properties":{"__proto__":{"type":"string"}}}'),
    )
    expect(proto.ok).toBe(false)
    const jsDefault = sanitizeFormSchema(
      JSON.parse('{"type":"object","properties":{"a":{"type":"string","default":"javascript:alert(1)"}}}'),
    )
    expect(jsDefault.ok).toBe(false)
    const dataUrl = sanitizeFormSchema({
      type: 'object',
      properties: { a: { type: 'string', format: 'data-url' } },
    })
    expect(dataUrl.ok).toBe(false)
  })

  it('enforces depth, field and enum limits', () => {
    expect(sanitizeFormSchema(deepSchema(6)).ok).toBe(false)
    const fields: Record<string, unknown> = {}
    for (let i = 0; i <= SCHEMA_MAX_FIELDS; i += 1) fields[`f${i}`] = { type: 'string' }
    expect(sanitizeFormSchema({ type: 'object', properties: fields }).ok).toBe(false)
    const values = Array.from({ length: SCHEMA_MAX_ENUM + 1 }, (_, i) => `v${i}`)
    expect(
      sanitizeFormSchema({ type: 'object', properties: { a: { type: 'string', enum: values } } }).ok,
    ).toBe(false)
  })

  it('rejects a non-object root', () => {
    expect(sanitizeFormSchema({ type: 'string' }).ok).toBe(false)
    expect(sanitizeFormSchema(null).ok).toBe(false)
  })
})

describe('JsonSchemaForm', () => {
  it('renders fields and submits values through the tool-result endpoint', async () => {
    const user = userEvent.setup()
    render(<JsonSchemaForm schema={validSchema} runId="run_r304" taskId="task_r304" toolCallId="call_r304" />)
    await user.type(screen.getByLabelText(/Name/), 'Ada')
    await user.click(screen.getByRole('button', { name: 'Submit' }))
    await waitFor(() => expect(mockedSend).toHaveBeenCalledTimes(1))
    expect(mockedSend).toHaveBeenCalledWith(
      expect.objectContaining({
        runId: 'run_r304',
        taskId: 'task_r304',
        toolCallId: 'call_r304',
        status: 'success',
        data: expect.objectContaining({ name: 'Ada' }),
      }),
    )
  })

  it('cancels without requiring valid fields', async () => {
    const user = userEvent.setup()
    render(<JsonSchemaForm schema={validSchema} runId="run_r304" toolCallId="call_r304" />)
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    await waitFor(() => expect(mockedSend).toHaveBeenCalledWith(expect.objectContaining({ status: 'cancelled' })))
  })

  it('shows a reason instead of a form for a malicious schema', () => {
    const { container } = render(
      <JsonSchemaForm
        schema={JSON.parse('{"type":"object","properties":{"a":{"type":"string","default":"javascript:alert(1)"}}}')}
        runId="run_r304"
        toolCallId="call_r304"
      />,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('Unsupported form schema')
    expect(container.querySelector('input')).toBeNull()
  })

  it('does not submit without a toolCallId', async () => {
    const user = userEvent.setup()
    render(<JsonSchemaForm schema={validSchema} runId="run_r304" />)
    await user.click(screen.getByRole('button', { name: 'Submit' }))
    expect(mockedSend).not.toHaveBeenCalled()
  })
})
