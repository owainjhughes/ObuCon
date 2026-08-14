import { apiClient } from "./client"

export interface DictionaryEntry {
  kanji: string
  hiragana: string
  meaning: string
  jlpt_level: number | null
}

export interface DictionaryPage {
  entries: DictionaryEntry[]
  total: number
  totalPages: number
}

export interface DictionaryPageOptions {
  query: string
  jlpt: number | null
  page: number
  pageSize: number
}

export async function fetchDictionaryPage(options: DictionaryPageOptions): Promise<DictionaryPage> {
  const response = await apiClient.get("/dictionary", {
    params: {
      language: "ja",
      query: options.query.trim() || undefined,
      jlpt: options.jlpt ?? undefined,
      page: options.page,
      page_size: options.pageSize,
    },
  })
  return {
    entries: response.data.entries,
    total: response.data.total,
    totalPages: response.data.total_pages,
  }
}
