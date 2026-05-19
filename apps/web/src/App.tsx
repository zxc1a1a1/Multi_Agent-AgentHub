import { useMemo, useRef, useState } from 'react'
import { Bot, Send, Square } from 'lucide-react'
import { streamAgentEvents } from './api/agui'
import { A2UIRenderer } from './components/A2UIRenderer'
import type { A2UISurface } from './types'

export default function App() {
  const [input, setInput] = useState('帮我规划一下 ADK + A2A + AG-UI + A2UI 的开发任务')
  const [answer, setAnswer] = useState('')
  const [events, setEvents] = useState<string[]>([])
  const [surface, setSurface] = useState<A2UISurface | undefined>()
  const [loading, setLoading] = useState(false)
  const controllerRef = useRef<AbortController | null>(null)

  const canSend = useMemo(() => input.trim().length > 0 && !loading, [input, loading])

  async function handleSend() {
    if (!canSend) return
    setLoading(true)
    setAnswer('')
    setSurface(undefined)
    setEvents([])

    const controller = new AbortController()
    controllerRef.current = controller

    try {
      await streamAgentEvents(
        input,
        (event) => {
          setEvents((prev) => [...prev, event.type])
          if (event.type === 'text_message_content') {
            setAnswer((prev) => prev + (event.delta ?? ''))
          }
          if (event.type === 'a2ui_surface') {
            setSurface(event.surface)
          }
          if (event.type === 'run_finished') {
            setLoading(false)
          }
        },
        controller.signal,
      )
    } catch (error) {
      if (!controller.signal.aborted) {
        setAnswer(`请求失败：${String(error)}`)
      }
      setLoading(false)
    }
  }

  function handleStop() {
    controllerRef.current?.abort()
    setLoading(false)
  }

  return (
    <main className="page">
      <section className="hero">
        <div className="badge"><Bot size={16} /> React + Go Multi-Agent Starter</div>
        <h1>多 Agent 框架控制台</h1>
        <p>Go 后端负责 Agent 编排、AG-UI 事件流、A2A 入口；React 前端负责实时交互和 A2UI 渲染。</p>
      </section>

      <section className="layout">
        <div className="panel chatPanel">
          <label htmlFor="message">用户任务</label>
          <textarea
            id="message"
            value={input}
            onChange={(event) => setInput(event.target.value)}
            rows={5}
          />
          <div className="actions">
            <button onClick={handleSend} disabled={!canSend}>
              <Send size={16} /> 发送给 Agent
            </button>
            <button className="secondary" onClick={handleStop} disabled={!loading}>
              <Square size={16} /> 停止
            </button>
          </div>

          <div className="answer">
            <h2>Agent 回复</h2>
            <pre>{answer || '等待输出...'}</pre>
          </div>
        </div>

        <div className="panel">
          <A2UIRenderer surface={surface} />
          <section className="events">
            <h2>AG-UI 事件</h2>
            <ol>
              {events.map((event, index) => <li key={`${event}-${index}`}>{event}</li>)}
            </ol>
          </section>
        </div>
      </section>
    </main>
  )
}
