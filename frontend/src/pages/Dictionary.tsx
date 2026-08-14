import { useEffect, useState } from "react"
import Layout from "../components/Layout"
import Pagination from "../components/Pagination"
import { apiClient } from "../api/client"
import { getApiErrorMessage } from "../api/errors"
import { getCached, setCached } from "../api/cache"
import { fetchDictionaryPage, type DictionaryPage } from "../api/dictionary"
import { fetchVocabulary, type VocabularyEntry } from "../api/vocabulary"

const PAGE_SIZE = 15
const SEARCH_DEBOUNCE_MS = 250

const jlptBadge: Record<number, string> = {
  1: 'bg-blue-700 text-white',
  2: 'bg-indigo-600 text-white',
  3: 'bg-purple-600 text-white',
  4: 'bg-orange-500 text-white',
  5: 'bg-green-600 text-white',
}

function JlptBadge({ level }: { level: number | null }) {
  if (level == null) return <span className="text-gray-400">—</span>
  const cls = jlptBadge[level] ?? 'bg-gray-500 text-white'
  return (
    <span className={`inline-block rounded px-1.5 py-0.5 text-xs font-bold ${cls}`}>
      N{level}
    </span>
  )
}

export default function Dictionary() {
  const cachedVocab = getCached<VocabularyEntry[]>("vocab")
  const [page, setPage] = useState<DictionaryPage | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [debouncedSearch, setDebouncedSearch] = useState("")
  const [jlptFilter, setJlptFilter] = useState<string>("all")
  const [currentPage, setCurrentPage] = useState(1)
  const [knownLemmas, setKnownLemmas] = useState<Set<string>>(
    () => new Set((cachedVocab ?? []).map((v) => v.lemma))
  )
  const [addingByLemma, setAddingByLemma] = useState<Record<string, boolean>>({})
  const [addError, setAddError] = useState<string | null>(null)

  useEffect(() => {
    // The known-word badges are cosmetic, so a failed lookup just leaves every row as "Add".
    fetchVocabulary()
      .then((vocab) => {
        setCached("vocab", vocab)
        setKnownLemmas(new Set(vocab.map((v) => v.lemma)))
      })
      .catch(() => undefined)
  }, [])

  useEffect(() => {
    const timeout = window.setTimeout(() => setDebouncedSearch(search), SEARCH_DEBOUNCE_MS)
    return () => window.clearTimeout(timeout)
  }, [search])

  useEffect(() => {
    let active = true
    setIsLoading(true)
    setError(null)
    fetchDictionaryPage({
      query: debouncedSearch,
      jlpt: jlptFilter === "all" ? null : Number(jlptFilter),
      page: currentPage,
      pageSize: PAGE_SIZE,
    })
      .then((result) => {
        if (active) setPage(result)
      })
      .catch((err: unknown) => {
        if (active) setError(getApiErrorMessage(err, "Failed to load dictionary"))
      })
      .finally(() => {
        if (active) setIsLoading(false)
      })

    return () => {
      active = false
    }
  }, [currentPage, jlptFilter, debouncedSearch])

  const entries = page?.entries ?? []

  const handleAddKnown = async (lemma: string) => {
    if (!lemma || addingByLemma[lemma] || knownLemmas.has(lemma)) return
    setAddError(null)
    setAddingByLemma((prev) => ({ ...prev, [lemma]: true }))
    try {
      const response = await apiClient.post("/vocab/known", { lemma, language: "ja" })
      setKnownLemmas((prev) => {
        const next = new Set(prev)
        next.add(lemma)
        return next
      })
      const existing = getCached<VocabularyEntry[]>("vocab") ?? []
      if (!existing.some((v) => v.lemma === lemma)) {
        const matchingDictEntry = entries.find(
          (e) => (e.kanji || e.hiragana) === lemma
        )
        const newEntry: VocabularyEntry = {
          lemma,
          hiragana: matchingDictEntry?.hiragana ?? "",
          grade_level: response?.data?.grade_level ?? matchingDictEntry?.jlpt_level ?? null,
          meaning: matchingDictEntry?.meaning ?? "",
        }
        setCached("vocab", [...existing, newEntry])
      }
    } catch (err: unknown) {
      setAddError(getApiErrorMessage(err, "Failed to add word to known list"))
    } finally {
      setAddingByLemma((prev) => ({ ...prev, [lemma]: false }))
    }
  }

  return (
    <Layout>
      <section className="mx-auto max-w-7xl px-4 py-10">
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h1 className="text-xl font-semibold text-gray-900">Japanese Dictionary</h1>

          <div className="mt-6 flex flex-col gap-2 md:flex-row md:items-center">
            <input
              value={search}
              onChange={(e) => { setSearch(e.target.value); setCurrentPage(1) }}
              placeholder="Search by kanji, hiragana, or meaning"
              className="w-full rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm text-gray-900 focus:border-[#55F] focus:outline-none md:max-w-sm"
            />
            <select
              value={jlptFilter}
              onChange={(e) => { setJlptFilter(e.target.value); setCurrentPage(1) }}
              className="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
            >
              <option value="all">All JLPT levels</option>
              <option value="5">JLPT N5</option>
              <option value="4">JLPT N4</option>
              <option value="3">JLPT N3</option>
              <option value="2">JLPT N2</option>
              <option value="1">JLPT N1</option>
            </select>
          </div>

          {isLoading ? null : error ? (
            <div className="mt-8 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          ) : (
            <div className="mt-6 overflow-x-auto">
              {addError && (
                <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                  {addError}
                </div>
              )}
              <table className="w-full min-w-[600px] table-fixed border-collapse">
                <colgroup>
                  <col className="w-[18%]" />
                  <col className="w-[22%]" />
                  <col className="w-[35%]" />
                  <col className="w-[8%]" />
                  <col className="w-[17%]" />
                </colgroup>
                <thead>
                  <tr className="border-b-2 border-[#55F] bg-gray-50 text-left text-xs font-semibold uppercase tracking-wider text-gray-500">
                    <th className="px-4 py-3">Word</th>
                    <th className="px-4 py-3">Hiragana</th>
                    <th className="px-4 py-3">Meaning</th>
                    <th className="px-4 py-3">JLPT</th>
                    <th className="px-4 py-3"></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 text-sm text-gray-700">
                  {entries.map((entry, idx) => {
                    const word = entry.kanji || entry.hiragana
                    return (
                      <tr key={`${entry.kanji}-${entry.hiragana}-${idx}`} className="hover:bg-gray-50 transition-colors">
                        <td className="px-4 py-3 font-semibold text-gray-900">
                          <div className="truncate" title={word}>{word}</div>
                        </td>
                        <td className="px-4 py-3 text-gray-600">
                          <div className="truncate" title={entry.hiragana || undefined}>{entry.hiragana}</div>
                        </td>
                        <td className="px-4 py-3">
                          <div className="truncate" title={entry.meaning || undefined}>{entry.meaning || "—"}</div>
                        </td>
                        <td className="px-4 py-3"><JlptBadge level={entry.jlpt_level} /></td>
                        <td className="px-4 py-3">
                          {knownLemmas.has(word) ? (
                            <span className="inline-block rounded bg-green-600 px-1.5 py-0.5 text-xs font-bold text-white">
                              ✓ Known
                            </span>
                          ) : (
                            <button
                              type="button"
                              onClick={() => handleAddKnown(word)}
                              disabled={!!addingByLemma[word]}
                              className="rounded-full bg-[#55F] px-3 py-1 text-xs font-medium text-white hover:bg-[#44E] disabled:cursor-not-allowed disabled:opacity-50 transition-colors"
                            >
                              {addingByLemma[word] ? "Adding…" : "Add"}
                            </button>
                          )}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>

              {entries.length === 0 && (
                <div className="mt-6 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600">
                  No dictionary entries found.
                </div>
              )}

              <Pagination
                currentPage={currentPage}
                totalPages={page?.totalPages ?? 1}
                totalCount={page?.total ?? 0}
                noun="words"
                onChange={setCurrentPage}
              />
            </div>
          )}
        </div>
      </section>
    </Layout>
  )
}
