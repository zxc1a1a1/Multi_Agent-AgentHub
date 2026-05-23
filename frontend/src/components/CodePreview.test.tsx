import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi, afterEach } from 'vitest'
import CodePreview from './CodePreview'

describe('CodePreview (MVP P0 minimum tests)', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  test('正常渲染 code preview（文件名、代码、语言）', () => {
    render(
      <CodePreview
        block={{
          code: 'console.log("hello")',
          language: 'javascript',
          filename: 'main.js',
        }}
      />
    )

    expect(screen.getByText('main.js')).toBeInTheDocument()
    expect(screen.getByText(/javascript/i)).toBeInTheDocument()
    expect(screen.getByText((_, el) => !!el && el.tagName === 'CODE' && el.textContent === 'console.log("hello")')).toBeInTheDocument()
  })

  test('空 code 时不崩溃并显示 placeholder', () => {
    render(
      <CodePreview
        block={{
          code: '',
          language: 'go',
          filename: 'empty.go',
        }}
      />
    )

    expect(screen.getByText('empty.go')).toBeInTheDocument()
    expect(screen.getByText('No code to preview')).toBeInTheDocument()
  })

  test('点击复制按钮会调用 clipboard.writeText 并传入原始 code', async () => {
    const user = userEvent.setup()
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })

    render(
      <CodePreview
        block={{
          code: 'package main',
          language: 'go',
          filename: 'main.go',
        }}
      />
    )

    await user.click(screen.getByRole('button', { name: /copy/i }))

    expect(writeText).toHaveBeenCalledTimes(1)
    expect(writeText).toHaveBeenCalledWith('package main')
    expect(screen.getByText('Copied')).toBeInTheDocument()
  })

  test('clipboard 失败时走 execCommand fallback 且不抛异常', async () => {
    const user = userEvent.setup()
    const writeText = vi.fn().mockRejectedValue(new Error('clipboard denied'))
    vi.stubGlobal('navigator', { clipboard: { writeText } })

    const execMock = vi.fn().mockReturnValue(true)
    Object.defineProperty(document, 'execCommand', {
      value: execMock,
      configurable: true,
      writable: true,
    })

    render(
      <CodePreview
        block={{
          code: 'print("fallback")',
          language: 'python',
          filename: 'main.py',
        }}
      />
    )

    await user.click(screen.getByRole('button', { name: /copy/i }))

    expect(writeText).toHaveBeenCalledTimes(1)
    expect(execMock).toHaveBeenCalledWith('copy')
  })

  test('可疑内容仅作为代码展示，不执行脚本', () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})

    render(
      <CodePreview
        block={{
          code: '<img src=x onerror=alert(1)><script>alert(1)</script>',
          language: 'html',
          filename: 'danger.html',
        }}
      />
    )

    expect(screen.getByText('danger.html')).toBeInTheDocument()
    expect(screen.getByText((_, el) => !!el && el.tagName === 'CODE' && (el.textContent ?? '').includes('onerror=alert(1)'))).toBeInTheDocument()
    expect(alertSpy).not.toHaveBeenCalled()
    expect(document.querySelector('script')).toBeNull()
  })

  test('filename/language 缺省时组件可兼容渲染', () => {
    render(
      <CodePreview
        block={{
          code: 'x = 1',
          language: '',
          filename: '',
        }}
      />
    )

    expect(screen.getByText('untitled')).toBeInTheDocument()
    expect(screen.getByText(/text/i)).toBeInTheDocument()
    expect(screen.getByText('x = 1')).toBeInTheDocument()
  })

  test('code 缺失时组件不崩溃并显示 placeholder', () => {
    render(
      <CodePreview
        block={{
          code: undefined as unknown as string,
          language: 'ts',
          filename: 'broken.ts',
        }}
      />
    )

    expect(screen.getByText('broken.ts')).toBeInTheDocument()
    expect(screen.getByText('No code to preview')).toBeInTheDocument()
  })
})
