import { useEffect, useRef } from 'react'
import { connectSSE, type SSEEvent } from '../lib/api'

export function useSSE(onEvent: (e: SSEEvent) => void) {
  const cb = useRef(onEvent)
  cb.current = onEvent

  useEffect(() => {
    return connectSSE((e) => cb.current(e))
  }, [])
}
