import { apiClient } from "./client"

export interface VocabularyEntry {
  lemma: string
  hiragana: string
  grade_level?: number | null
  meaning: string
  kind?: string
}

export interface VocabularyImportResult {
  added: number
  updated: number
  skipped: number
}

export async function fetchVocabulary(): Promise<VocabularyEntry[]> {
  const response = await apiClient.get("/vocab")
  const raw: VocabularyEntry[] = response.data.vocab || []
  const seen = new Set<string>()
  return raw.filter((entry) => (seen.has(entry.lemma) ? false : (seen.add(entry.lemma), true)))
}

export async function importVocabulary(
  entries: Array<{ lemma: string; hiragana: string; meaning: string }>,
): Promise<VocabularyImportResult> {
  const response = await apiClient.post("/vocab/import", { language: "ja", entries })
  return response.data
}
