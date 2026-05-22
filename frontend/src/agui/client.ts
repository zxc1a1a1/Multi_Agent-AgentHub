export function runAgent(
  _request: unknown,
  _onEvent: (_e: unknown) => void,
  _onError?: (_err: Error) => void,
): AbortController {
  return new AbortController()
}
