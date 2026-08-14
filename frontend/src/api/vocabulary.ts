export interface VocabularyImportSourceEntry {
  lemma: string
  hiragana: string
  meaning: string
}

interface VocabularyImportEntry {
  lemma: string
  hiragana?: string
  meaning?: string
}

export interface VocabularyImportPayload {
  language: string
  entries: VocabularyImportEntry[]
}

export function buildVocabularyImport(
  entries: VocabularyImportSourceEntry[],
  language = "ja",
): VocabularyImportPayload {
  return {
    language,
    entries: entries.map((entry) => {
      const lemma = entry.lemma.trim()
      const hiragana = entry.hiragana.trim()
      const meaning = entry.meaning.trim()
      return {
        lemma,
        ...(hiragana ? { hiragana } : {}),
        ...(meaning ? { meaning } : {}),
      }
    }),
  }
}
