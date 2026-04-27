import { useEffect, useState } from "react"
import Layout from "../components/Layout"
import Pagination from "../components/Pagination"
import { apiClient } from "../api/client"
import { getApiErrorMessage } from "../api/errors"
import { getCached, setCached } from "../api/cache"

interface DictionaryEntry {
  kanji: string
  hiragana: string
  meaning: string
  jlpt_level: number | null
}

const PAGE_SIZE = 15

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
  const cached = getCached<DictionaryEntry[]>("dictionary")
  const [entries, setEntries] = useState<DictionaryEntry[]>(cached ?? [])
  const [isLoading, setIsLoading] = useState(!cached)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [jlptFilter, setJlptFilter] = useState<string>("all")
  const [currentPage, setCurrentPage] = useState(1)

  useEffect(() => {
    const load = async () => {
      if (!getCached("dictionary")) setIsLoading(true)
      setError(null)
      try {
        const response = await apiClient.get("/dictionary", { params: { language: "ja" } })
        const data: DictionaryEntry[] = response.data.entries || []
        setCached("dictionary", data)
        setEntries(data)
      } catch (err: unknown) {
        setError(getApiErrorMessage(err, "Failed to load dictionary"))
      } finally {
        setIsLoading(false)
      }
    }
    load()
  }, [])

  const filteredEntries = entries.filter((entry) => {
    if (jlptFilter !== "all") {
      const wantedLevel = Number(jlptFilter)
      if (entry.jlpt_level !== wantedLevel) return false
    }
    if (!search) return true
    const needle = search.toLowerCase()
    return (
      entry.kanji.includes(search) ||
      entry.hiragana.includes(search) ||
      (entry.meaning || "").toLowerCase().includes(needle)
    )
  })

  const totalPages = Math.max(1, Math.ceil(filteredEntries.length / PAGE_SIZE))
  const safePage = Math.min(currentPage, totalPages)
  const paginated = filteredEntries.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE)

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
                  {paginated.map((entry, idx) => {
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
                        <td className="px-4 py-3"></td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>

              {filteredEntries.length === 0 && (
                <div className="mt-6 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600">
                  No dictionary entries found.
                </div>
              )}

              <Pagination
                currentPage={safePage}
                totalPages={totalPages}
                totalCount={filteredEntries.length}
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
