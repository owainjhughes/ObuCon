import React, { useEffect, useState } from "react"
import Layout from "../components/Layout"
import { apiClient } from "../api/client"
import { getApiErrorMessage } from "../api/errors"
import { getAnkiDeckNames, exportVocabToAnki, importVocabFromAnki } from "../api/ankiConnect"

interface VocabEntry {
  lemma: string
  grade_level?: number | null
  meaning: string
}

export default function Vocab() {
  const [vocab, setVocab] = useState<VocabEntry[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState("")
  const [jlptLevel, setJlptLevel] = useState("N5")
  const [importing, setImporting] = useState(false)
  const [editingLemma, setEditingLemma] = useState<string | null>(null)
  const [editMeaning, setEditMeaning] = useState("")
  const [editGradeLevel, setEditGradeLevel] = useState("5")
  const [savingLemma, setSavingLemma] = useState<string | null>(null)
  const [deletingLemma, setDeletingLemma] = useState<string | null>(null)

  const [isMobile] = useState(() => window.matchMedia("(pointer: coarse)").matches)
  const [ankiExportDeck, setAnkiExportDeck] = useState("GinAPI Japanese")
  const [ankiDecks, setAnkiDecks] = useState<string[]>([])
  const [selectedAnkiDeck, setSelectedAnkiDeck] = useState("")
  const [exportingAnki, setExportingAnki] = useState(false)
  const [importingAnki, setImportingAnki] = useState(false)
  const [loadingAnkiDecks, setLoadingAnkiDecks] = useState(false)
  const [ankiMessage, setAnkiMessage] = useState<{ text: string; type: "success" | "error" } | null>(null)

  const loadVocab = async () => {
    setIsLoading(true)
    setError(null)

    try {
      const response = await apiClient.get("/vocab")
      setVocab(response.data.vocab || [])
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Failed to load vocabulary"))
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    loadVocab()
  }, [])

  const filteredVocab = vocab.filter((entry) =>
    entry.lemma.includes(search) || (entry.meaning || "").toLowerCase().includes(search.toLowerCase())
  )

  const handleImport = async () => {
    setImporting(true)
    setError(null)

    try {
      await apiClient.post("/vocab/bulk", {
        jlpt_level: jlptLevel,
        language: "ja",
      })
      await loadVocab()
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Failed to import vocabulary"))
    } finally {
      setImporting(false)
    }
  }

  const startEditing = (entry: VocabEntry) => {
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

  const removeEntry = async (entry: VocabEntry) => {
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
      const { added, skipped } = await exportVocabToAnki(
        vocab.map((e) => ({ lemma: e.lemma, meaning: e.meaning })),
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
      if (decks.length > 0) setSelectedAnkiDeck(decks[0])
    } catch (err: unknown) {
      setAnkiMessage({ text: err instanceof Error ? err.message : "Could not load Anki decks", type: "error" })
    } finally {
      setLoadingAnkiDecks(false)
    }
  }

  const handleImportFromAnki = async () => {
    if (!selectedAnkiDeck) return
    setImportingAnki(true)
    setAnkiMessage(null)
    try {
      const entries = await importVocabFromAnki(selectedAnkiDeck)
      const results = await Promise.allSettled(
        entries.map((e) => apiClient.post("/vocab/known", { lemma: e.lemma, language: "ja" })),
      )
      const added = results.filter((r) => r.status === "fulfilled").length
      const failed = results.filter((r) => r.status === "rejected").length
      await loadVocab()
      setAnkiMessage({
        text: `Imported ${added} word(s) from Anki.${failed > 0 ? ` ${failed} failed (may already exist).` : ""}`,
        type: "success",
      })
    } catch (err: unknown) {
      setAnkiMessage({ text: err instanceof Error ? err.message : "Import failed", type: "error" })
    } finally {
      setImportingAnki(false)
    }
  }

  return (
    <Layout>
      <section className="mx-auto max-w-4xl px-4 py-10">
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h1 className="text-xl font-semibold text-gray-900">Known Vocabulary</h1>
          <p className="mt-1 text-sm text-gray-600">
            Words you have marked as known. Use this list to review or update your vocabulary.
          </p>

{isMobile ? (
          <div className="mt-6 rounded-xl border border-gray-200 bg-gray-50 p-6">
            <h2 className="text-lg font-semibold text-gray-900">Anki Sync</h2>
            <p className="mt-2 text-sm text-gray-500">
              Anki Sync is not available on mobile. Please use a desktop browser with Anki open.
            </p>
          </div>
        ) : (
          <div className="mt-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
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

          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <div className="rounded-lg border border-gray-100 bg-gray-50 p-4">
              <h3 className="text-sm font-semibold text-gray-800">Export to Anki</h3>
              <p className="mt-1 text-xs text-gray-500">
                Pushes your {vocab.length} known word(s) into an Anki deck as flashcards (Front: word, Back: meaning).
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
                  disabled={exportingAnki || vocab.length === 0}
                  className="rounded-full border border-[#55F] bg-[#55F] px-4 py-2 text-sm font-semibold text-white hover:bg-[#44E] disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {exportingAnki ? "Exporting..." : "Export to Anki"}
                </button>
              </div>
            </div>

            <div className="rounded-lg border border-gray-100 bg-gray-50 p-4">
              <h3 className="text-sm font-semibold text-gray-800">Import from Anki</h3>
              <p className="mt-1 text-xs text-gray-500">
                Pulls the front field of each note from an Anki deck into your known vocabulary list.
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
                      onChange={(e) => setSelectedAnkiDeck(e.target.value)}
                      className="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
                    >
                      {ankiDecks.map((deck) => (
                        <option key={deck} value={deck}>{deck}</option>
                      ))}
                    </select>
                    <button
                      type="button"
                      onClick={handleImportFromAnki}
                      disabled={importingAnki || !selectedAnkiDeck}
                      className="rounded-full border border-[#55F] bg-[#55F] px-4 py-2 text-sm font-semibold text-white hover:bg-[#44E] disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {importingAnki ? "Importing..." : "Import from Anki"}
                    </button>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>
        )}
          <div className="mt-6 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div className="flex flex-1 flex-col gap-2 md:flex-row md:items-center">
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search by word or meaning"
                className="w-full rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm text-gray-900 focus:border-[#55F] focus:outline-none md:max-w-sm"
              />
              <div className="flex items-center gap-2">
                <select
                  value={jlptLevel}
                  onChange={(e) => setJlptLevel(e.target.value)}
                  className="rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
                >
                  <option value="N5">JLPT N5</option>
                  <option value="N4">JLPT N4</option>
                  <option value="N3">JLPT N3</option>
                  <option value="N2">JLPT N2</option>
                  <option value="N1">JLPT N1</option>
                </select>
                <button
                  type="button"
                  onClick={handleImport}
                  disabled={importing}
                  className="rounded-full border border-[#55F] bg-[#55F] px-4 py-2 text-sm font-semibold text-white hover:bg-[#44E] disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {importing ? "Importing..." : "Import JLPT list"}
                </button>
              </div>
            </div>
          </div>

          {isLoading ? (
            <div className="mt-8 text-sm text-gray-600">Loading...</div>
          ) : error ? (
            <div className="mt-8 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          ) : (
            <div className="mt-6 overflow-x-auto">
              <table className="w-full min-w-[520px] table-auto border-collapse">
                <thead>
                  <tr className="bg-gray-50 text-left text-xs font-medium uppercase tracking-wider text-gray-600">
                    <th className="px-4 py-3">Word</th>
                    <th className="px-4 py-3">Meaning</th>
                    <th className="px-4 py-3">Grade</th>
                    <th className="px-4 py-3">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 text-sm text-gray-700">
                  {filteredVocab.map((entry) => (
                    <tr key={entry.lemma}>
                      <td className="px-4 py-3 font-semibold text-gray-900">{entry.lemma}</td>
                      <td className="px-4 py-3">
                        {editingLemma === entry.lemma ? (
                          <input
                            value={editMeaning}
                            onChange={(e) => setEditMeaning(e.target.value)}
                            className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
                          />
                        ) : (
                          entry.meaning || "—"
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {editingLemma === entry.lemma ? (
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
                        ) : (
                          entry.grade_level ?? "—"
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
                              <button
                                type="button"
                                onClick={() => startEditing(entry)}
                                className="rounded-full border border-gray-300 bg-white px-3 py-1 text-xs font-semibold text-gray-700 hover:bg-gray-50"
                              >
                                Edit
                              </button>
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
            </div>
          )}
        </div>
      </section>
    </Layout>
  )
}
