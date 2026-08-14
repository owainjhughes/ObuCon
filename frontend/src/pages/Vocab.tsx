import React, { useEffect, useState } from "react"
import Layout from "../components/Layout"
import Pagination from "../components/Pagination"
import { apiClient } from "../api/client"
import { getApiErrorMessage } from "../api/errors"
import {
  getAnkiDeckNames,
  exportVocabToAnki,
  importVocabFromAnki,
  previewAnkiDeck,
  type AnkiDeckPreview,
} from "../api/ankiConnect"
import { getCached, setCached } from "../api/cache"
import { fetchVocabulary, importVocabulary, type VocabularyEntry } from "../api/vocabulary"

const jlptBadge: Record<number, string> = {
  1: 'bg-blue-700 text-white',
  2: 'bg-indigo-600 text-white',
  3: 'bg-purple-600 text-white',
  4: 'bg-orange-500 text-white',
  5: 'bg-green-600 text-white',
}

function JlptBadge({ level }: { level: number | null | undefined }) {
  if (level == null) return <span className="text-gray-400">—</span>
  const cls = jlptBadge[level] ?? 'bg-gray-500 text-white'
  return (
    <span className={`inline-block rounded px-1.5 py-0.5 text-xs font-bold ${cls}`}>
      N{level}
    </span>
  )
}

function ConjugationBadge() {
  return (
    <span className="inline-block rounded bg-sky-600 px-1.5 py-0.5 text-xs font-bold text-white">
      Conj.
    </span>
  )
}

export default function Vocab() {
  const cached = getCached<VocabularyEntry[]>("vocab")
  const [vocab, setVocab] = useState<VocabularyEntry[]>(cached ?? [])
  const [isLoading, setIsLoading] = useState(!cached)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [jlptFilter, setJlptFilter] = useState<string>("all")
  const [jlptLevel, setJlptLevel] = useState("N5")
  const [importing, setImporting] = useState(false)
  const [showJlptModal, setShowJlptModal] = useState(false)
  const [jlptIncludeEasier, setJlptIncludeEasier] = useState(false)
  const [jlptMessage, setJlptMessage] = useState<{ text: string; type: "success" | "error" } | null>(null)
  const [editingLemma, setEditingLemma] = useState<string | null>(null)
  const [editMeaning, setEditMeaning] = useState("")
  const [editGradeLevel, setEditGradeLevel] = useState("5")
  const [savingLemma, setSavingLemma] = useState<string | null>(null)
  const [deletingLemma, setDeletingLemma] = useState<string | null>(null)

  const PAGE_SIZE = 15
  const [currentPage, setCurrentPage] = useState(1)

  const [isMobile] = useState(() => window.matchMedia("(pointer: coarse)").matches)
  const [showAnkiModal, setShowAnkiModal] = useState(false)
  const [ankiExportDeck, setAnkiExportDeck] = useState("ObuCon Japanese")
  const [ankiDecks, setAnkiDecks] = useState<string[]>([])
  const [selectedAnkiDeck, setSelectedAnkiDeck] = useState("")
  const [exportingAnki, setExportingAnki] = useState(false)
  const [importingAnki, setImportingAnki] = useState(false)
  const [loadingAnkiDecks, setLoadingAnkiDecks] = useState(false)
  const [ankiMessage, setAnkiMessage] = useState<{ text: string; type: "success" | "error" } | null>(null)
  const [ankiPreview, setAnkiPreview] = useState<AnkiDeckPreview | null>(null)
  const [ankiLemmaField, setAnkiLemmaField] = useState("")
  const [ankiMeaningField, setAnkiMeaningField] = useState("")
  const [ankiHiraganaField, setAnkiHiraganaField] = useState("")
  const [loadingAnkiPreview, setLoadingAnkiPreview] = useState(false)

  const loadVocab = async () => {
    if (!getCached("vocab")) setIsLoading(true)
    setError(null)

    try {
      const data = await fetchVocabulary()
      setCached("vocab", data)
      setVocab(data)
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Failed to load vocabulary"))
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadVocab()
  }, [])

  const filteredVocab = vocab.filter((entry) => {
    if (jlptFilter === "conjugation") {
      if (entry.kind !== "conjugation") return false
    } else if (jlptFilter !== "all") {
      const wantedLevel = Number(jlptFilter)
      if (entry.grade_level !== wantedLevel) return false
    }
    if (!search) return true
    return (
      entry.lemma.includes(search) ||
      (entry.hiragana || "").includes(search) ||
      (entry.meaning || "").toLowerCase().includes(search.toLowerCase())
    )
  })

  const totalPages = Math.max(1, Math.ceil(filteredVocab.length / PAGE_SIZE))
  const paginatedVocab = filteredVocab.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)

  const handleImport = async () => {
    setImporting(true)
    setJlptMessage(null)

    const selectedNum = Number(jlptLevel.replace("N", ""))
    const levelsToImport = jlptIncludeEasier
      ? [5, 4, 3, 2, 1].filter((n) => n >= selectedNum)
      : [selectedNum]
    const levelLabels = levelsToImport.map((n) => `N${n}`)

    try {
      for (const label of levelLabels) {
        await apiClient.post("/vocab/bulk", {
          jlpt_level: label,
          language: "ja",
        })
      }
      await loadVocab()
      setJlptMessage({
        text: `Imported ${levelLabels.join(", ")}.`,
        type: "success",
      })
    } catch (err: unknown) {
      setJlptMessage({
        text: getApiErrorMessage(err, "Failed to import vocabulary"),
        type: "error",
      })
    } finally {
      setImporting(false)
    }
  }

  const closeJlptModal = () => {
    setShowJlptModal(false)
    setJlptMessage(null)
  }

  const startEditing = (entry: VocabularyEntry) => {
    setEditingLemma(entry.lemma)
    setEditMeaning(entry.meaning || "")
    setEditGradeLevel(String(entry.grade_level ?? 5))
    setError(null)
  }

  const cancelEditing = () => {
    setEditingLemma(null)
    setEditMeaning("")
    setEditGradeLevel("5")
  }

  const saveEntry = async () => {
    if (!editingLemma) {
      return
    }

    setSavingLemma(editingLemma)
    setError(null)

    try {
      await apiClient.put("/vocab/known", {
        lemma: editingLemma,
        language: "ja",
        meaning: editMeaning,
        jlpt_level: Number(editGradeLevel),
      })
      cancelEditing()
      await loadVocab()
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Failed to update vocabulary entry"))
    } finally {
      setSavingLemma(null)
    }
  }

  const removeEntry = async (entry: VocabularyEntry) => {
    const confirmed = window.confirm(`Remove ${entry.lemma} from your vocabulary list?`)
    if (!confirmed) {
      return
    }

    setDeletingLemma(entry.lemma)
    setError(null)

    try {
      await apiClient.delete("/vocab/known", {
        data: {
          lemma: entry.lemma,
          language: "ja",
        },
      })
      if (editingLemma === entry.lemma) {
        cancelEditing()
      }
      await loadVocab()
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Failed to remove vocabulary entry"))
    } finally {
      setDeletingLemma(null)
    }
  }

  const handleExportToAnki = async () => {
    setExportingAnki(true)
    setAnkiMessage(null)
    try {
      const exportable = vocab.filter((e) => e.kind !== "conjugation")
      const { added, skipped } = await exportVocabToAnki(
        exportable.map((e) => ({ lemma: e.lemma, meaning: e.meaning })),
        ankiExportDeck,
      )
      setAnkiMessage({ text: `Exported ${added} card(s) to Anki. ${skipped} duplicate(s) skipped.`, type: "success" })
    } catch (err: unknown) {
      setAnkiMessage({ text: err instanceof Error ? err.message : "Export failed", type: "error" })
    } finally {
      setExportingAnki(false)
    }
  }

  const handleLoadAnkiDecks = async () => {
    setLoadingAnkiDecks(true)
    setAnkiMessage(null)
    try {
      const decks = await getAnkiDeckNames()
      setAnkiDecks(decks)
      if (decks.length > 0) {
        setSelectedAnkiDeck(decks[0])
        resetAnkiFieldSelection()
      }
    } catch (err: unknown) {
      setAnkiMessage({ text: err instanceof Error ? err.message : "Could not load Anki decks", type: "error" })
    } finally {
      setLoadingAnkiDecks(false)
    }
  }

  const resetAnkiFieldSelection = () => {
    setAnkiPreview(null)
    setAnkiLemmaField("")
    setAnkiMeaningField("")
    setAnkiHiraganaField("")
  }

  const handleSelectAnkiDeck = (deck: string) => {
    setSelectedAnkiDeck(deck)
    resetAnkiFieldSelection()
  }

  const handleLoadAnkiFields = async () => {
    if (!selectedAnkiDeck) return
    setLoadingAnkiPreview(true)
    setAnkiMessage(null)
    try {
      const preview = await previewAnkiDeck(selectedAnkiDeck)
      setAnkiPreview(preview)
      if (preview.fields.length > 0) {
        setAnkiLemmaField(preview.fields[0])
        setAnkiMeaningField(preview.fields[1] ?? "")
      }
    } catch (err: unknown) {
      setAnkiMessage({ text: err instanceof Error ? err.message : "Could not load deck fields", type: "error" })
    } finally {
      setLoadingAnkiPreview(false)
    }
  }

  const handleImportFromAnki = async () => {
    if (!selectedAnkiDeck || !ankiLemmaField || !ankiMeaningField) return
    setImportingAnki(true)
    setAnkiMessage(null)
    try {
      const entries = await importVocabFromAnki(
        selectedAnkiDeck,
        ankiLemmaField,
        ankiMeaningField,
        ankiHiraganaField || undefined,
      )
      if (entries.length === 0) {
        setAnkiMessage({ text: "No mature cards with a word were found in this deck.", type: "success" })
        return
      }
      const { added, updated, skipped } = await importVocabulary(entries)
      await loadVocab()
      setAnkiMessage({
        text: `Imported ${added} new word(s), updated ${updated}, and skipped ${skipped} unchanged word(s).`,
        type: "success",
      })
    } catch (err: unknown) {
      setAnkiMessage({ text: getApiErrorMessage(err, "Import failed"), type: "error" })
    } finally {
      setImportingAnki(false)
    }
  }

  const closeAnkiModal = () => {
    setShowAnkiModal(false)
    setAnkiMessage(null)
    resetAnkiFieldSelection()
  }

  return (
    <Layout>
      <section className="mx-auto max-w-7xl px-4 py-10">
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h1 className="text-xl font-semibold text-gray-900">Known Vocabulary</h1>

          <div className="mt-6 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div className="flex flex-1 flex-col gap-2 md:flex-row md:items-center">
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
                <option value="conjugation">Conjugations</option>
              </select>
              <button
                type="button"
                onClick={() => setShowJlptModal(true)}
                className="rounded-full border border-[#55F] bg-[#55F] px-4 py-2 text-sm font-semibold text-white hover:bg-[#44E]"
              >
                Import JLPT list
              </button>
            </div>

            {!isMobile && (
              <button
                type="button"
                onClick={() => setShowAnkiModal(true)}
                className="rounded-full border border-[#55F] bg-white px-4 py-2 text-sm font-semibold text-[#55F] hover:bg-[#55F] hover:text-white"
              >
                Anki Sync
              </button>
            )}
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
                    <th className="px-4 py-3">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 text-sm text-gray-700">
                  {paginatedVocab.map((entry) => (
                    <tr key={entry.lemma} className="hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 font-semibold text-gray-900">
                        <div className="truncate" title={entry.lemma}>{entry.lemma}</div>
                      </td>
                      <td className="px-4 py-3 text-gray-600">
                        <div className="truncate" title={entry.hiragana || undefined}>{entry.hiragana || "—"}</div>
                      </td>
                      <td className="px-4 py-3">
                        {editingLemma === entry.lemma ? (
                          <input
                            value={editMeaning}
                            onChange={(e) => setEditMeaning(e.target.value)}
                            className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
                          />
                        ) : (
                          <div className="truncate" title={entry.meaning || undefined}>{entry.meaning || "—"}</div>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {editingLemma === entry.lemma && entry.kind !== "conjugation" ? (
                          <select
                            value={editGradeLevel}
                            onChange={(e) => setEditGradeLevel(e.target.value)}
                            className="rounded-lg border border-gray-300 bg-white px-2 py-1 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
                          >
                            <option value="5">N5</option>
                            <option value="4">N4</option>
                            <option value="3">N3</option>
                            <option value="2">N2</option>
                            <option value="1">N1</option>
                          </select>
                        ) : entry.kind === "conjugation" ? (
                          <ConjugationBadge />
                        ) : (
                          <JlptBadge level={entry.grade_level} />
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          {editingLemma === entry.lemma ? (
                            <>
                              <button
                                type="button"
                                onClick={saveEntry}
                                disabled={savingLemma === entry.lemma}
                                className="rounded-full border border-[#55F] bg-[#55F] px-3 py-1 text-xs font-semibold text-white hover:bg-[#44E] disabled:cursor-not-allowed disabled:opacity-50"
                              >
                                {savingLemma === entry.lemma ? "Saving..." : "Save"}
                              </button>
                              <button
                                type="button"
                                onClick={cancelEditing}
                                className="rounded-full border border-gray-300 bg-white px-3 py-1 text-xs font-semibold text-gray-700 hover:bg-gray-50"
                              >
                                Cancel
                              </button>
                            </>
                          ) : (
                            <>
                              {entry.kind !== "conjugation" && (
                                <button
                                  type="button"
                                  onClick={() => startEditing(entry)}
                                  className="rounded-full border border-gray-300 bg-white px-3 py-1 text-xs font-semibold text-gray-700 hover:bg-gray-50"
                                >
                                  Edit
                                </button>
                              )}
                              <button
                                type="button"
                                onClick={() => removeEntry(entry)}
                                disabled={deletingLemma === entry.lemma}
                                className="rounded-full border border-red-300 bg-white px-3 py-1 text-xs font-semibold text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
                              >
                                {deletingLemma === entry.lemma ? "Removing..." : "Remove"}
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {filteredVocab.length === 0 && (
                <div className="mt-6 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600">
                  No known words found.
                </div>
              )}

              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                totalCount={filteredVocab.length}
                noun="words"
                onChange={setCurrentPage}
              />
            </div>
          )}
        </div>
      </section>

      {showJlptModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={closeJlptModal}
        >
          <div
            className="relative w-full max-w-md rounded-2xl bg-white p-8 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <button
              type="button"
              onClick={closeJlptModal}
              aria-label="Close JLPT import"
              className="absolute right-5 top-5 rounded p-1 text-gray-400 hover:text-gray-700"
            >
              ✕
            </button>

            <h2 className="text-lg font-semibold text-gray-900">Import JLPT vocabulary</h2>
            <p className="mt-1 text-sm text-gray-600">
              Pulls the full official vocabulary list for the chosen JLPT level into your known words.
            </p>

            {jlptMessage && (
              <div
                className={`mt-4 rounded-lg border px-4 py-3 text-sm ${
                  jlptMessage.type === "success"
                    ? "border-green-200 bg-green-50 text-green-700"
                    : "border-red-200 bg-red-50 text-red-700"
                }`}
              >
                {jlptMessage.text}
              </div>
            )}

            <div className="mt-4 flex flex-col gap-3">
              <label className="flex flex-col gap-1 text-xs font-semibold text-gray-700">
                JLPT level
                <select
                  value={jlptLevel}
                  onChange={(e) => setJlptLevel(e.target.value)}
                  className="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-normal text-gray-900 focus:border-[#55F] focus:outline-none"
                >
                  <option value="N5">JLPT N5</option>
                  <option value="N4">JLPT N4</option>
                  <option value="N3">JLPT N3</option>
                  <option value="N2">JLPT N2</option>
                  <option value="N1">JLPT N1</option>
                </select>
              </label>

              <label className="flex items-start gap-2 text-sm text-gray-700">
                <input
                  type="checkbox"
                  checked={jlptIncludeEasier}
                  onChange={(e) => setJlptIncludeEasier(e.target.checked)}
                  className="mt-0.5 h-4 w-4 rounded border-gray-300 text-[#55F] focus:ring-[#55F]"
                />
                <span>Also include easier levels</span>
              </label>

              <div className="mt-2 flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={closeJlptModal}
                  className="rounded-full border border-gray-300 bg-white px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleImport}
                  disabled={importing}
                  className="rounded-full border border-[#55F] bg-[#55F] px-4 py-2 text-sm font-semibold text-white hover:bg-[#44E] disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {importing ? "Importing..." : "Import"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {showAnkiModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={closeAnkiModal}
        >
          <div
            className="relative w-full max-w-xl rounded-2xl bg-white p-8 shadow-xl max-h-[80vh] overflow-y-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <button
              type="button"
              onClick={closeAnkiModal}
              aria-label="Close Anki Sync"
              className="absolute right-5 top-5 rounded p-1 text-gray-400 hover:text-gray-700"
            >
              ✕
            </button>

            <h2 className="text-lg font-semibold text-gray-900">Anki Sync</h2>
            <p className="mt-1 text-sm text-gray-600">
              Export your vocabulary to Anki as flashcards, or import words from an existing Anki deck. Requires Anki to
              be open with the{" "}
              <a
                href="https://ankiweb.net/shared/info/2055492159"
                target="_blank"
                rel="noreferrer"
                className="text-[#55F] underline"
              >
                AnkiConnect
              </a>{" "}
              plugin installed.
            </p>

            {window.location.hostname !== "localhost" && window.location.hostname !== "127.0.0.1" && (
              <p className="mt-2 text-xs text-gray-500">
                First time? In Anki: Tools → Add-ons → AnkiConnect → Config, and add{" "}
                <code className="mx-0.5 rounded bg-gray-100 px-1 py-0.5">"{window.location.origin}"</code>
                {" "}to <code className="rounded bg-gray-100 px-1 py-0.5">webCorsOriginList</code>, then restart Anki.
              </p>
            )}

            {ankiMessage && (
              <div
                className={`mt-4 rounded-lg border px-4 py-3 text-sm ${
                  ankiMessage.type === "success"
                    ? "border-green-200 bg-green-50 text-green-700"
                    : "border-red-200 bg-red-50 text-red-700"
                }`}
              >
                {ankiMessage.text}
              </div>
            )}

            <div className="mt-4 flex flex-col gap-4">
              <div className="rounded-lg border border-gray-100 bg-gray-50 p-4">
                <h3 className="text-sm font-semibold text-gray-800">Export to Anki</h3>
                <p className="mt-1 text-xs text-gray-500">
                  Pushes your {vocab.filter((e) => e.kind !== "conjugation").length} known word(s) into an Anki deck as flashcards (Front: word, Back: meaning).
                </p>
                <div className="mt-3 flex flex-col gap-2">
                  <input
                    value={ankiExportDeck}
                    onChange={(e) => setAnkiExportDeck(e.target.value)}
                    placeholder="Deck name"
                    className="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
                  />
                  <button
                    type="button"
                    onClick={handleExportToAnki}
                    disabled={exportingAnki || vocab.filter((e) => e.kind !== "conjugation").length === 0}
                    className="rounded-full border border-[#55F] bg-[#55F] px-4 py-2 text-sm font-semibold text-white hover:bg-[#44E] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {exportingAnki ? "Exporting..." : "Export to Anki"}
                  </button>
                </div>
              </div>

              <div className="rounded-lg border border-gray-100 bg-gray-50 p-4">
                <h3 className="text-sm font-semibold text-gray-800">Import from Anki</h3>
                <p className="mt-1 text-xs text-gray-500">
                  Only mature cards (interval ≥ 21 days) are imported. Pick which Anki field is the word, meaning, and (optionally) hiragana. Words already in your known list are skipped.
                </p>
                <div className="mt-3 flex flex-col gap-2">
                  {ankiDecks.length === 0 ? (
                    <button
                      type="button"
                      onClick={handleLoadAnkiDecks}
                      disabled={loadingAnkiDecks}
                      className="rounded-full border border-gray-300 bg-white px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {loadingAnkiDecks ? "Loading decks..." : "Load Anki decks"}
                    </button>
                  ) : (
                    <>
                      <select
                        value={selectedAnkiDeck}
                        onChange={(e) => handleSelectAnkiDeck(e.target.value)}
                        className="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
                      >
                        {ankiDecks.map((deck) => (
                          <option key={deck} value={deck}>{deck}</option>
                        ))}
                      </select>

                      {!ankiPreview ? (
                        <button
                          type="button"
                          onClick={handleLoadAnkiFields}
                          disabled={loadingAnkiPreview || !selectedAnkiDeck}
                          className="rounded-full border border-gray-300 bg-white px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          {loadingAnkiPreview ? "Loading fields..." : "Load fields"}
                        </button>
                      ) : ankiPreview.fields.length === 0 ? (
                        <p className="rounded-lg border border-yellow-200 bg-yellow-50 px-3 py-2 text-xs text-yellow-800">
                          This deck has no mature cards (interval ≥ 21 days).
                        </p>
                      ) : (
                        <>
                          <div className="rounded-lg border border-gray-200 bg-white p-3">
                            <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">
                              Sample card ({ankiPreview.noteCount} mature)
                            </p>
                            <dl className="mt-2 grid grid-cols-[max-content_1fr] gap-x-3 gap-y-1 text-xs">
                              {ankiPreview.fields.map((name) => (
                                <React.Fragment key={name}>
                                  <dt className="font-semibold text-gray-700">{name}</dt>
                                  <dd className="truncate text-gray-600" title={ankiPreview.sample[name] || ""}>
                                    {ankiPreview.sample[name] || <span className="text-gray-400">(empty)</span>}
                                  </dd>
                                </React.Fragment>
                              ))}
                            </dl>
                          </div>

                          <label className="flex flex-col gap-1 text-xs font-semibold text-gray-700">
                            Lemma field
                            <select
                              value={ankiLemmaField}
                              onChange={(e) => setAnkiLemmaField(e.target.value)}
                              className="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-normal text-gray-900 focus:border-[#55F] focus:outline-none"
                            >
                              {ankiPreview.fields.map((name) => (
                                <option key={name} value={name}>{name}</option>
                              ))}
                            </select>
                          </label>

                          <label className="flex flex-col gap-1 text-xs font-semibold text-gray-700">
                            Meaning field
                            <select
                              value={ankiMeaningField}
                              onChange={(e) => setAnkiMeaningField(e.target.value)}
                              className="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-normal text-gray-900 focus:border-[#55F] focus:outline-none"
                            >
                              <option value="">Select field…</option>
                              {ankiPreview.fields.map((name) => (
                                <option key={name} value={name}>{name}</option>
                              ))}
                            </select>
                          </label>

                          <label className="flex flex-col gap-1 text-xs font-semibold text-gray-700">
                            Hiragana field <span className="font-normal text-gray-400">(optional)</span>
                            <select
                              value={ankiHiraganaField}
                              onChange={(e) => setAnkiHiraganaField(e.target.value)}
                              className="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-normal text-gray-900 focus:border-[#55F] focus:outline-none"
                            >
                              <option value="">(None)</option>
                              {ankiPreview.fields.map((name) => (
                                <option key={name} value={name}>{name}</option>
                              ))}
                            </select>
                          </label>

                          <button
                            type="button"
                            onClick={handleImportFromAnki}
                            disabled={importingAnki || !ankiLemmaField || !ankiMeaningField}
                            className="rounded-full border border-[#55F] bg-[#55F] px-4 py-2 text-sm font-semibold text-white hover:bg-[#44E] disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            {importingAnki ? "Importing..." : "Import from Anki"}
                          </button>
                        </>
                      )}
                    </>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </Layout>
  )
}
