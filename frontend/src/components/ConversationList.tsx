import { useEffect, useState, useMemo } from 'react'
import { useConversationStore } from '../stores/conversationStore'
import { useAgentStore } from '../stores/agentStore'
import { MessageSquarePlus, MessageSquare, Trash2, Pin, PinOff, Search } from 'lucide-react'

export default function ConversationList() {
  const conversations = useConversationStore((s) => s.conversations)
  const activeId = useConversationStore((s) => s.activeId)
  const load = useConversationStore((s) => s.load)
  const create = useConversationStore((s) => s.create)
  const setActive = useConversationStore((s) => s.setActive)
  const deleteConv = useConversationStore((s) => s.delete)
  const pinConversation = useConversationStore((s) => s.pinConversation)
  const searchQuery = useConversationStore((s) => s.searchQuery)
  const setSearchQuery = useConversationStore((s) => s.setSearchQuery)
  const loadAgents = useAgentStore((s) => s.load)
  const defaultAgentName = useAgentStore((s) => s.defaultAgentName)
  const getAgentDisplayName = useAgentStore((s) => s.getDisplayName)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [pinError, setPinError] = useState<string | null>(null)

  useEffect(() => {
    load()
    loadAgents()
  }, [load, loadAgents])

  // Local frontend filter — no API call per keystroke
  const filteredConversations = useMemo(() => {
    const q = searchQuery.toLowerCase().trim()
    if (!q) return conversations
    return conversations.filter(
      (conv) =>
        conv.title.toLowerCase().includes(q) ||
        conv.agentName.toLowerCase().includes(q),
    )
  }, [conversations, searchQuery])

  const handlePin = async (id: string, pinned: boolean) => {
    setPinError(null)
    try {
      await pinConversation(id, pinned)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to update pin'
      setPinError(message)
    }
  }

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

      {/* Search bar */}
      <div className="px-3 py-2 border-b border-gray-100 bg-white">
        <div className="flex items-center gap-1.5 px-2 py-1 rounded-md bg-gray-100 border border-gray-200">
          <Search className="w-3.5 h-3.5 text-gray-400" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search conversations..."
            className="flex-1 bg-transparent text-xs outline-none placeholder:text-gray-400 text-gray-700"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="text-gray-400 hover:text-gray-600 text-xs"
            >
              Clear
            </button>
          )}
        </div>
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
        {pinError && (
          <div className="p-2 mx-2 mt-2 bg-red-50 border border-red-200 rounded text-xs text-red-600">
            {pinError}
            <button
              className="ml-2 underline"
              onClick={() => setPinError(null)}
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
        {conversations.length > 0 && filteredConversations.length === 0 && (
          <div className="p-4 text-center text-gray-400 text-sm mt-8">
            <p>No conversations match "{searchQuery}"</p>
          </div>
        )}
        {filteredConversations.map((conv) => (
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
                {/* Pin indicator */}
                {conv.pinned && (
                  <Pin className="w-3.5 h-3.5 text-amber-500 flex-shrink-0" />
                )}
                <MessageSquare className="w-4 h-4 text-gray-400 flex-shrink-0" />
                <span className="text-sm text-gray-700 truncate">{conv.title || 'New Conversation'}</span>
              </div>
              <p className="text-xs text-gray-400 mt-0.5 ml-6">
                {getAgentDisplayName(conv.agentName)} &middot;{' '}
                {new Date(conv.updatedAt).toLocaleDateString()}
              </p>
            </button>

            {/* Action buttons — visible on hover */}
            <div className="absolute right-2 top-2 flex items-center gap-0.5">
              {/* Pin toggle */}
              <button
                className="p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-amber-50 transition-all"
                title={conv.pinned ? 'Unpin' : 'Pin'}
                onClick={(e) => {
                  e.stopPropagation()
                  handlePin(conv.id, !conv.pinned)
                }}
              >
                {conv.pinned ? (
                  <PinOff className="w-3.5 h-3.5 text-amber-500" />
                ) : (
                  <Pin className="w-3.5 h-3.5 text-gray-400 hover:text-amber-500" />
                )}
              </button>

              {/* Delete button */}
              {deletingId === conv.id ? (
                <div className="inline-flex gap-0.5 items-center p-1 text-xs text-red-500 bg-white rounded shadow-sm border border-gray-100">
                  Delete?
                  <button
                    className="px-1.5 py-0.5 bg-red-500 text-white rounded text-xs hover:bg-red-600"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDelete(conv.id)
                    }}
                  >
                    Yes
                  </button>
                  <button
                    className="px-1.5 py-0.5 bg-gray-200 text-gray-700 rounded text-xs hover:bg-gray-300"
                    onClick={(e) => {
                      e.stopPropagation()
                      setDeletingId(null)
                    }}
                  >
                    No
                  </button>
                </div>
              ) : (
                <button
                  className="p-1 rounded opacity-0 group-hover:opacity-100 hover:bg-red-50 transition-all"
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
          </div>
        ))}
      </div>
    </div>
  )
}
