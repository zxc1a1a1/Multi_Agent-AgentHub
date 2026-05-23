import { Bot } from 'lucide-react'

interface Props {
  name?: string
  isStreaming?: boolean
}

export default function AgentAvatar({ name, isStreaming }: Props) {
  return (
    <div className="relative w-8 h-8 rounded-full bg-indigo-100 flex items-center justify-center flex-shrink-0">
      <Bot className="w-5 h-5 text-indigo-600" />
      {isStreaming && (
        <span className="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-green-400 rounded-full border-2 border-white animate-pulse" />
      )}
    </div>
  )
}
