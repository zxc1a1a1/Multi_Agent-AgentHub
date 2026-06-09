import { useState } from 'react'
import ChatLayout from './components/ChatLayout'
import AgentDirectory from './components/agents/AgentDirectory'
import { MessageSquare, Bot } from 'lucide-react'

type Tab = 'chat' | 'agents'

export default function App() {
  const [tab, setTab] = useState<Tab>('chat')

  return (
    <div className="flex h-screen bg-white">
      {/* Minimal tab navigation sidebar */}
      <nav className="w-14 border-r border-gray-200 flex flex-col items-center py-3 gap-1 bg-gray-50 flex-shrink-0">
        <button
          onClick={() => setTab('chat')}
          className={`p-2 rounded-lg transition-colors ${
            tab === 'chat'
              ? 'bg-indigo-100 text-indigo-600'
              : 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'
          }`}
          title="Chat"
        >
          <MessageSquare className="w-5 h-5" />
        </button>
        <button
          onClick={() => setTab('agents')}
          className={`p-2 rounded-lg transition-colors ${
            tab === 'agents'
              ? 'bg-indigo-100 text-indigo-600'
              : 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'
          }`}
          title="Agents"
        >
          <Bot className="w-5 h-5" />
        </button>
      </nav>

      {/* Main content area */}
      <div className="flex-1 flex min-w-0">
        {tab === 'chat' ? <ChatLayout /> : <AgentDirectory />}
      </div>
    </div>
  )
}
