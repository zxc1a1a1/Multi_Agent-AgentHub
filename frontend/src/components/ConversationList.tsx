import { useEffect } from 'react'
import { useConversationStore } from '../stores/conversationStore'
import { MessageSquarePlus, MessageSquare } from 'lucide-react'

export default function ConversationList() {
  const conversations = useConversationStore((s) => s.conversations)
  const activeId = useConversationStore((s) => s.activeId)
  const load = useConversationStore((s) => s.load)
  const create = useConversationStore((s) => s.create)
  const setActive = useConversationStore((s) => s.setActive)

  useEffect(() => {
    load()
  }, [load])

  const handleNew = () => {
    create('code-agent')
  }

  return (
    <div className="flex flex-col h-full bg-gray-50">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 flex items-center justify-between bg-white">
        <h2 className="font-semibold text-gray-800 text-sm">AgentHub</h2>
        <button
          onClick={handleNew}
          className="p-2 rounded-lg hover:bg-gray-100 transition-colors"
          title="New conversation"
        >
          <MessageSquarePlus className="w-5 h-5 text-gray-600" />
        </button>
      </div>

      {/* Conversation list */}
      <div className="flex-1 overflow-y-auto">
        {conversations.length === 0 && (
          <div className="p-4 text-center text-gray-400 text-sm mt-8">
            <p>No conversations yet</p>
            <p className="mt-1">Click + to start one</p>
          </div>
        )}
        {conversations.map((conv) => (
          <button
            key={conv.id}
            onClick={() => setActive(conv.id)}
            className={`w-full text-left px-4 py-3 border-b border-gray-100 hover:bg-white transition-colors ${
              activeId === conv.id ? 'bg-white border-l-2 border-l-indigo-500' : ''
            }`}
          >
            <div className="flex items-center gap-2">
              <MessageSquare className="w-4 h-4 text-gray-400 flex-shrink-0" />
              <span className="text-sm text-gray-700 truncate">{conv.title}</span>
            </div>
            <p className="text-xs text-gray-400 mt-0.5 ml-6">
              {conv.agentName} &middot; {new Date(conv.updatedAt).toLocaleDateString()}
            </p>
          </button>
        ))}
      </div>
    </div>
  )
}
