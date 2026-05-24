import { useMessageStore } from '../stores/messageStore'

/**
 * Hook that exposes the send message functionality.
 * Components use this to trigger message sending and monitor streaming state.
 */
export function useSendMessage() {
  const sendMessage = useMessageStore((s) => s.sendMessage)
  const isStreaming = useMessageStore((s) => s.isStreaming)
  const stopStreaming = useMessageStore((s) => s.stopStreaming)

  return { sendMessage, isStreaming, stopStreaming }
}
