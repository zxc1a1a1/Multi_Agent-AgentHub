import { useEffect, useState } from 'react'
import { useConversationStore } from '../stores/conversationStore'
import { useAgentStore } from '../stores/agentStore'
import { MessageSquarePlus, MessageSquare, Trash2 } from 'lucide-react'

export default function ConversationList() {
  const conversations = useConversationStore((s) => s.conversations)
  const activeId = useConversationStore((s) => s.activeId)
  const load = useConversationStore((s) => s.load)
  const create = useConversationStore((s) => s.create)
  const setActive = useConversationStore((s) => s.setActive)
  const deleteConv = useConversationStore((s) => s.delete)
  const loadAgents = useAgentStore((s) => s.load)
  const defaultAgentName = useAgentStore((s) => s.defaultAgentName)
  const getAgentDisplayName = useAgentStore((s) => s.getDisplayName)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  useEffect(() => {
    load()
    loadAgents()
  }, [load, loadAgents])

  const handleNew = () => {
    create(defaultAgentName())
  }

  const handleDelete = async (id: string) => {
    setDeleteError(null)
    try {
      await deleteConv(id)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete conversation'
      setDeleteError(message)
    } finally {
      setDeletingId(null)
    }
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
        {deleteError && (
          <div className="p-2 mx-2 mt-2 bg-red-50 border border-red-200 rounded text-xs text-red-600">
            {deleteError}
            <button
              className="ml-2 underline"
              onClick={() => setDeleteError(null)}
            >
              Dismiss
            </button>
          </div>
        )}
        {conversations.length === 0 && (
          <div className="p-4 text-center text-gray-400 text-sm mt-8">
            <p>No conversations yet</p>
            <p className="mt-1">Click + to start one</p>
          </div>
        )}
        {conversations.map((conv) => (
          <div
            key={conv.id}
            className={`group relative border-b border-gray-100 ${
              activeId === conv.id ? 'bg-white border-l-2 border-l-indigo-500' : ''
            }`}
          >
            <button
              onClick={() => setActive(conv.id)}
              className="w-full text-left px-4 py-3 hover:bg-white transition-colors"
            >
              <div className="flex items-center gap-2">
                <MessageSquare className="w-4 h-4 text-gray-400 flex-shrink-0" />
                <span className="text-sm text-gray-700 truncate">{conv.title || 'New Conversation'}</span>
              </div>
              <p className="text-xs text-gray-400 mt-0.5 ml-6">
                {getAgentDisplayName(conv.agentName)} &middot;{' '}
                {new Date(conv.updatedAt).toLocaleDateString()}
              </p>
            </button>
            {/* Delete button — visible on hover */}
            {deletingId === conv.id ? (
              <div className="absolute right-2 top-2 p-1 text-xs text-red-500">
                确认删除？
                <button
                  className="ml-1 px-1.5 py-0.5 bg-red-500 text-white rounded text-xs hover:bg-red-600"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleDelete(conv.id)
                  }}
                >
                  删除
                </button>
                <button
                  className="ml-1 px-1.5 py-0.5 bg-gray-300 text-gray-700 rounded text-xs hover:bg-gray-400"
                  onClick={(e) => {
                    e.stopPropagation()
                    setDeletingId(null)
                  }}
                >
                  取消
                </button>
              </div>
            ) : (
              <button
                className="absolute right-2 top-3 p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-red-50 transition-all"
                title="Delete conversation"
                onClick={(e) => {
                  e.stopPropagation()
                  setDeletingId(conv.id)
                }}
              >
                <Trash2 className="w-3.5 h-3.5 text-gray-400 hover:text-red-500" />
              </button>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
