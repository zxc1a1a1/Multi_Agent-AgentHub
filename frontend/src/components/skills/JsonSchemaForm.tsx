import { useMemo, useState } from 'react'
import Form from '@rjsf/core'
import validator from '@rjsf/validator-ajv8'
import * as api from '../../services/api'

/**
 * R3-04: JSON-schema form with a hard limit set. RJSF only renders the
 * sanitized schema; submission still goes through the tool-result endpoint, so
 * the Orchestrator re-validates against the tool's own input schema.
 */

export const SCHEMA_MAX_DEPTH = 5
export const SCHEMA_MAX_FIELDS = 30
export const SCHEMA_MAX_ENUM = 50
export const SCHEMA_MAX_STRING_LENGTH = 4096

const ALLOWED_KEYS = new Set([
  'type',
  'title',
  'description',
  'properties',
  'required',
  'enum',
  'default',
  'const',
  'minimum',
  'maximum',
  'minLength',
  'maxLength',
  'items',
  'additionalProperties',
])

const ALLOWED_TYPES = new Set(['string', 'number', 'integer', 'boolean', 'object', 'array'])

type Schema = Record<string, unknown>

export type SchemaSanitizeResult = { ok: true; schema: Schema } | { ok: false; reason: string }

function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return false
  const proto = Object.getPrototypeOf(value)
  return proto === Object.prototype || proto === null
}

function hasUnsafeString(value: unknown): boolean {
  if (typeof value === 'string') return /^\s*javascript:/i.test(value) || /^\s*data:/i.test(value)
  if (Array.isArray(value)) return value.some(hasUnsafeString)
  if (isPlainObject(value)) return Object.values(value).some(hasUnsafeString)
  return false
}

function sanitizeNode(input: unknown, depth: number, counter: { fields: number }): Schema | string {
  if (!isPlainObject(input)) return 'schema node must be an object'
  if (depth > SCHEMA_MAX_DEPTH) return 'schema depth exceeds limit'
  const out: Schema = {}
  for (const [key, value] of Object.entries(input)) {
    if (key === '__proto__' || key === 'constructor' || key === 'prototype') return `forbidden key ${key}`
    if (!ALLOWED_KEYS.has(key)) return `keyword not allowed: ${key}`
    switch (key) {
      case 'type': {
        if (typeof value !== 'string' || !ALLOWED_TYPES.has(value)) return 'unsupported type'
        out.type = value
        break
      }
      case 'title':
      case 'description': {
        if (typeof value !== 'string' || value.length > 200) return `${key} must be a short string`
        out[key] = value
        break
      }
      case 'minimum':
      case 'maximum': {
        if (typeof value !== 'number' || !Number.isFinite(value)) return `${key} must be finite`
        out[key] = value
        break
      }
      case 'minLength':
      case 'maxLength': {
        if (typeof value !== 'number' || !Number.isInteger(value) || value < 0) return `${key} must be a non-negative integer`
        if (key === 'maxLength' && value > SCHEMA_MAX_STRING_LENGTH) return 'maxLength exceeds limit'
        out[key] = value
        break
      }
      case 'additionalProperties': {
        if (typeof value !== 'boolean') return 'additionalProperties must be boolean'
        out.additionalProperties = value
        break
      }
      case 'enum': {
        if (!Array.isArray(value) || value.length === 0 || value.length > SCHEMA_MAX_ENUM) return 'enum exceeds limit'
        if (!value.every((item) => ['string', 'number', 'boolean'].includes(typeof item))) return 'enum values must be primitives'
        if (hasUnsafeString(value)) return 'enum contains an unsafe value'
        out.enum = value
        break
      }
      case 'default':
      case 'const': {
        if (!['string', 'number', 'boolean'].includes(typeof value)) return `${key} must be a primitive`
        if (hasUnsafeString(value)) return `${key} contains an unsafe value`
        out[key] = value
        break
      }
      case 'required': {
        if (!Array.isArray(value) || !value.every((item) => typeof item === 'string')) return 'required must be string[]'
        out.required = value
        break
      }
      case 'properties': {
        if (!isPlainObject(value)) return 'properties must be an object'
        const properties: Schema = {}
        for (const [name, child] of Object.entries(value)) {
          if (name === '__proto__' || name === 'constructor' || name === 'prototype') return 'forbidden property name'
          counter.fields += 1
          if (counter.fields > SCHEMA_MAX_FIELDS) return 'field count exceeds limit'
          const sanitized = sanitizeNode(child, depth + 1, counter)
          if (typeof sanitized === 'string') return `${name}: ${sanitized}`
          properties[name] = sanitized
        }
        out.properties = properties
        break
      }
      case 'items': {
        const sanitized = sanitizeNode(value, depth + 1, counter)
        if (typeof sanitized === 'string') return `items: ${sanitized}`
        out.items = sanitized
        break
      }
      default:
        return `keyword not allowed: ${key}`
    }
  }
  if (out.type === 'string' && out.maxLength === undefined) {
    out.maxLength = SCHEMA_MAX_STRING_LENGTH
  }
  return out
}

export function sanitizeFormSchema(input: unknown): SchemaSanitizeResult {
  const sanitized = sanitizeNode(input, 0, { fields: 0 })
  if (typeof sanitized === 'string') return { ok: false, reason: sanitized }
  if (sanitized.type !== 'object') return { ok: false, reason: 'root schema must be an object' }
  if (!isPlainObject(sanitized.properties) || Object.keys(sanitized.properties).length === 0) {
    return { ok: false, reason: 'root schema needs properties' }
  }
  return { ok: true, schema: sanitized }
}

type Props = {
  schema: unknown
  title?: string
  runId?: string
  taskId?: string
  toolCallId?: string
  disabled?: boolean
}

export default function JsonSchemaForm({ schema, title, runId, taskId, toolCallId, disabled }: Props) {
  const cleaned = useMemo(() => sanitizeFormSchema(schema), [schema])
  const [formData, setFormData] = useState<Record<string, unknown>>({})
  const [submitting, setSubmitting] = useState(false)
  const [done, setDone] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  if (!cleaned.ok) {
    return (
      <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800" role="alert">
        <div className="font-medium">Unsupported form schema</div>
        <div className="mt-1 text-xs">reason: {cleaned.reason}</div>
      </div>
    )
  }

  const submit = async (cancelled: boolean) => {
    if (!runId || !toolCallId || submitting || done) return
    setSubmitting(true)
    setError(null)
    try {
      await api.sendToolResult({
        runId,
        taskId,
        toolCallId,
        status: cancelled ? 'cancelled' : 'success',
        contentType: 'application/json',
        data: cancelled ? { cancelled: true } : formData,
      })
      setDone(cancelled ? 'cancelled' : 'submitted')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit form')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mt-2 rounded-lg border border-indigo-200 bg-white p-3">
      <h4 className="mb-2 text-sm font-semibold text-gray-800">{title || 'Input required'}</h4>
      <Form
        schema={cleaned.schema}
        validator={validator}
        formData={formData}
        disabled={disabled || submitting || done !== null}
        noHtml5Validate
        showErrorList={false}
        omitExtraData
        onChange={(event: { formData?: Record<string, unknown> }) => setFormData(event.formData ?? {})}
        onSubmit={() => {
          void submit(false)
        }}
      >
        {error && <p className="mt-2 text-xs text-red-600">{error}</p>}
        {done ? (
          <p className="mt-2 text-xs text-green-600">{done}</p>
        ) : (
          <div className="mt-3 flex gap-2">
            <button
              type="submit"
              disabled={submitting || !runId || !toolCallId}
              className="rounded bg-indigo-600 px-2 py-1 text-xs text-white disabled:opacity-50"
            >
              Submit
            </button>
            <button
              type="button"
              disabled={submitting || !runId || !toolCallId}
              onClick={() => {
                void submit(true)
              }}
              className="rounded border border-gray-300 bg-white px-2 py-1 text-xs text-gray-600 disabled:opacity-50"
            >
              Cancel
            </button>
          </div>
        )}
      </Form>
    </div>
  )
}
