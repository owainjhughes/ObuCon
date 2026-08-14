export interface DictionaryEntry {
  kanji: string
  hiragana: string
  meaning: string
  jlpt_level: number | null
}

interface DictionaryPagination {
  page: number
  page_size: number
  total: number
  total_pages: number
}

export interface DictionaryResponse {
  entries?: DictionaryEntry[]
  pagination?: DictionaryPagination
}

export interface DictionaryPageOptions {
  query: string
  jlpt: number | null
  page: number
  pageSize: number
}

export interface DictionaryPage {
  entries: DictionaryEntry[]
  page: number
  pageSize: number
  total: number
  totalPages: number
}

export function parseDictionaryPage(
  response: DictionaryResponse,
  options: DictionaryPageOptions,
): DictionaryPage {
  const entries = response.entries ?? []
  if (response.pagination) {
    return {
      entries,
      page: response.pagination.page,
      pageSize: response.pagination.page_size,
      total: response.pagination.total,
      totalPages: Math.max(1, response.pagination.total_pages),
    }
  }

  const needle = options.query.trim().toLowerCase()
  const filtered = entries.filter((entry) => {
    if (options.jlpt !== null && entry.jlpt_level !== options.jlpt) return false
    if (needle === "") return true
    return (
      entry.kanji.toLowerCase().includes(needle) ||
      entry.hiragana.toLowerCase().includes(needle) ||
      entry.meaning.toLowerCase().includes(needle)
    )
  })
  const totalPages = Math.max(1, Math.ceil(filtered.length / options.pageSize))
  const page = Math.min(options.page, totalPages)
  const start = (page - 1) * options.pageSize

  return {
    entries: filtered.slice(start, start + options.pageSize),
    page,
    pageSize: options.pageSize,
    total: filtered.length,
    totalPages,
  }
}
