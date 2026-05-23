import { useConversationStore } from '../stores/conversationStore'
import ConversationList from './ConversationList'
import ChatWindow from './ChatWindow'

export default function ChatLayout() {
  const activeId = useConversationStore((s) => s.activeId)

  return (
    <div className="flex h-screen bg-white">
      {/* Sidebar */}
      <aside className="w-72 border-r border-gray-200 flex-shrink-0 hidden md:flex md:flex-col">
        <ConversationList />
      </aside>

      {/* Main chat area */}
      <main className="flex-1 flex flex-col min-w-0">
        {activeId ? (
          <ChatWindow conversationId={activeId} />
        ) : (
          <div className="flex-1 flex items-center justify-center text-gray-400">
            <div className="text-center">
              <p className="text-lg">Welcome to AgentHub</p>
              <p className="text-sm mt-1">Select or create a conversation to start</p>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
