const store: Record<string, unknown> = {}

export function getCached<T>(key: string): T | undefined {
  return store[key] as T | undefined
}

export function setCached<T>(key: string, value: T): void {
  store[key] = value
}