import React, { useEffect, useState } from 'react'
import { apiClient } from '../../api/client'
import ReviewSession from './ReviewSession'

interface Token {
  surface: string
  lemma: string
  pos: string
  is_known: boolean
  grade_level?: number | null
  is_katakana?: boolean
  is_roman?: boolean
  is_non_japanese?: boolean
  meaning?: string
}

interface AnalysisOutputProps {
  tokens: Token[]
  missing: string[]
  onReset: () => void
}

function formatPercent(value: number) {
  return `${value.toFixed(1)}%`
}

function getRating(knownPct: number) {
  if (knownPct < 75) return { label: 'Ineffective', color: '#dc2626' }
  if (knownPct < 85) return { label: 'Decent', color: '#d97706' }
  if (knownPct < 95) return { label: 'Effective', color: '#16a34a' }
  return { label: 'Decent', color: '#d97706' }
}

function createPieSegments(counts: Record<string, number>) {
  const total = Object.values(counts).reduce((sum, v) => sum + v, 0)
  const colors: Record<string, string> = {
    'JLPT N1': '#2563EB',
    'JLPT N2': '#4F46E5',
    'JLPT N3': '#9333EA',
    'JLPT N4': '#EA580C',
    'JLPT N5': '#16A34A',
    'Unknown (Native)': '#9CA3AF',
    'Katakana': '#D1D5DB',
  }

  const segments: Array<{ label: string; percent: number; color: string }> = []
  Object.entries(counts).forEach(([label, count]) => {
    if (count <= 0) return
    const percent = total === 0 ? 0 : (count / total) * 100
    segments.push({ label, percent, color: colors[label] ?? '#6B7280' })
  })

  return segments
}

function isAuxiliary(token: Token) {
  return token.pos.includes('助動詞') || token.pos.includes('動詞 非自立')
}

function isNonJapanese(token: { is_non_japanese?: boolean; is_roman?: boolean; pos: string }) {
  if (token.is_non_japanese != null) return token.is_non_japanese
  return Boolean(token.is_roman) || token.pos.includes('記号')
}

interface DisplayToken {
  surface: string
  lemma: string
  pos: string
  is_known: boolean
  grade_level?: number | null
  is_katakana?: boolean
  is_roman?: boolean
  is_non_japanese?: boolean
  meaning?: string
}

function combineTokensForDisplay(tokens: Token[]): DisplayToken[] {
  const combined: DisplayToken[] = []

  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i]
    // If this token is a verb and the next is an auxiliary, combine them for clearer display
    if (token.pos.includes('動詞') && i + 1 < tokens.length && isAuxiliary(tokens[i + 1])) {
      const combinedSurface = token.surface + tokens[i + 1].surface
      const combinedKnown = token.is_known || tokens[i + 1].is_known
      const gradeLevel = token.grade_level ?? tokens[i + 1].grade_level
      const isKatakana = token.is_katakana || tokens[i + 1].is_katakana
      const isRoman = token.is_roman || tokens[i + 1].is_roman
      combined.push({
        surface: combinedSurface,
        lemma: token.lemma,
        pos: token.pos,
        is_known: combinedKnown,
        grade_level: gradeLevel,
        is_katakana: isKatakana,
        is_roman: isRoman,
        is_non_japanese: token.is_non_japanese || tokens[i + 1].is_non_japanese,
        meaning: token.meaning || tokens[i + 1].meaning,
      })
      i += 1
      continue
    }

    combined.push({
      surface: token.surface,
      lemma: token.lemma,
      pos: token.pos,
      is_known: token.is_known,
      grade_level: token.grade_level,
      is_katakana: token.is_katakana,
      is_roman: token.is_roman,
      is_non_japanese: token.is_non_japanese,
      meaning: token.meaning,
    })
  }

  return combined
}

export default function AnalysisOutput({ tokens, missing, onReset }: AnalysisOutputProps) {
  const [localTokens, setLocalTokens] = useState<Token[]>(tokens)
  const [localMissing, setLocalMissing] = useState<string[]>(missing)
  const [addingByLemma, setAddingByLemma] = useState<Record<string, boolean>>({})
  const [addError, setAddError] = useState('')
  const [addingAll, setAddingAll] = useState(false)
  const [showReview, setShowReview] = useState(false)

  useEffect(() => {
    setLocalTokens(tokens)
    setLocalMissing(missing)
    setAddingByLemma({})
    setAddError('')
    setShowReview(false)
  }, [tokens, missing])

  const handleAddKnown = async (lemma: string) => {
    if (addingByLemma[lemma]) return
    setAddingByLemma((prev) => ({ ...prev, [lemma]: true }))
    try {
      const response = await apiClient.post('/vocab/known', { lemma, language: 'ja' })
      const resolvedGrade = response?.data?.grade_level ?? null
      setLocalTokens((prev) =>
        prev.map((token) =>
          token.lemma === lemma || token.surface === lemma
            ? { ...token, is_known: true, grade_level: resolvedGrade ?? token.grade_level ?? null }
            : token
        )
      )
      setLocalMissing((prev) => prev.filter((w) => w !== lemma))
    } catch (err: any) {
      setAddError(err?.response?.data?.error || 'Failed to add word to known list')
    } finally {
      setAddingByLemma((prev) => ({ ...prev, [lemma]: false }))
    }
  }

  const handleAddAll = async () => {
    if (addingAll || localMissing.length === 0) return
    setAddError('')
    setAddingAll(true)
    try {
      await Promise.all(localMissing.map((lemma) => handleAddKnown(lemma)))
    } finally {
      setAddingAll(false)
    }
  }

  const handleWordMarked = (lemma: string) => {
    setLocalTokens((prev) =>
      prev.map((t) =>
        t.lemma === lemma || t.surface === lemma ? { ...t, is_known: true } : t
      )
    )
  }

  const reviewableCount = [...new Set(
    localTokens.filter((t) => !t.is_known && t.grade_level != null).map((t) => t.lemma)
  )].length

  const meaningByLemma = React.useMemo(() => {
    const map: Record<string, string> = {}
    for (const t of localTokens) {
      if (t.meaning && !map[t.lemma]) map[t.lemma] = t.meaning
    }
    return map
  }, [localTokens])

  const displayTokens = combineTokensForDisplay(localTokens)
  
  const scorableTokens = localTokens.filter((t) => !isNonJapanese(t))
  const pieTokens = localTokens.filter((t) => !isNonJapanese(t))
  
  const total = scorableTokens.length
  const knownCount = scorableTokens.filter((t) => t.is_known).length
  const missingCount = localMissing.length
  const knownPct = total === 0 ? 0 : (knownCount / total) * 100
  const rating = getRating(knownPct)

  const jlptCounts = pieTokens.reduce<Record<string, number>>((acc, token) => {
    let category: string
    if (token.grade_level != null) {
      category = `JLPT N${token.grade_level}`
    } else if (token.is_katakana) {
      category = 'Katakana'
    } else {
      category = 'Unknown (Native)'
    }
    acc[category] = (acc[category] ?? 0) + 1
    return acc
  }, {})

  const pieSegments = createPieSegments(jlptCounts)

  return (
    <section className="px-4 py-6">
      <div className="w-full">
        <div className="mb-3 flex items-center justify-between">
          <button
            type="button"
            onClick={onReset}
            className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-800 transition-colors"
          >
            <span aria-hidden="true">&larr;</span>
            New analysis
          </button>
          {reviewableCount > 0 && (
            <button
              type="button"
              onClick={() => setShowReview(true)}
              className="rounded-lg border border-[#55F] bg-[#55F] px-3 py-1.5 text-xs font-semibold text-white hover:bg-[#44E] transition-colors"
            >
              Review {reviewableCount} unknown word{reviewableCount === 1 ? '' : 's'}
            </button>
          )}
        </div>
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h3 className="mb-4 border-b border-gray-100 pb-3 text-base font-semibold text-gray-900">Text</h3>
          <div className="text-base leading-loose">
            {displayTokens.map((token, idx) => {
              let className = `inline font-medium `

              if (isNonJapanese(token)) {
                className += 'text-gray-700'
                return (
                  <span key={idx} className={className}>{token.surface}</span>
                )
              }

              if (token.is_katakana) {
                if (!token.is_known) className += 'bg-rose-50 text-rose-700 border-b border-rose-300'
              } else {
                if (!token.is_known) className += 'bg-rose-100 text-rose-800'
              }

              return (
                <span key={idx} className="group relative inline-block">
                  <span className={className}>{token.surface}</span>
                  <span className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-1 hidden -translate-x-1/2 whitespace-nowrap rounded bg-gray-900 px-2 py-1 text-xs font-normal text-white shadow-lg group-hover:block">
                    {token.meaning?.trim() || 'unknown meaning'}
                  </span>
                </span>
              )
            })}
          </div>
        </div>

        <aside className="lg:col-span-1 self-start rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h3 className="mb-4 border-b border-gray-100 pb-3 text-base font-semibold text-gray-900">Stats</h3>

          <div className="mt-4">
            <div className="flex items-start gap-6">
              <div>
                <div className="text-sm font-semibold text-gray-700">Known words</div>
                <div className="text-2xl font-bold text-gray-900">{knownCount}</div>
                <div className="text-xs text-gray-500">of {total}</div>
              </div>
              <div>
                <div className="text-sm font-semibold text-gray-700">Known %</div>
                <div className="text-3xl font-bold" style={{ color: rating.color }}>
                  {formatPercent(knownPct)}
                </div>
                <div className="text-xs font-semibold text-gray-600">{rating.label}</div>
              </div>
            </div>
          </div>

          <div className="mt-8">
            <h4 className="text-sm font-semibold text-gray-700">JLPT distribution</h4>
            <div className="mt-4 flex gap-6">
              <svg viewBox="0 0 42 42" className="h-24 w-24">
                <circle
                  cx="21"
                  cy="21"
                  r="15.91549430918954"
                  fill="transparent"
                  stroke="#E5E7EB"
                  strokeWidth="6"
                />
                {pieSegments.reduce((acc, segment) => {
                  const offset = acc.offset
                  const dash = `${segment.percent} ${100 - segment.percent}`
                  acc.elements.push(
                    <circle
                      key={segment.label}
                      cx="21"
                      cy="21"
                      r="15.91549430918954"
                      fill="transparent"
                      stroke={segment.color}
                      strokeWidth="6"
                      strokeDasharray={dash}
                      strokeDashoffset={`${100 - offset}`}
                      transform="rotate(-90 21 21)"
                    />
                  )
                  acc.offset += segment.percent
                  return acc
                }, { offset: 0, elements: [] as React.ReactNode[] }).elements}
              </svg>
              <div className="flex-1">
                {pieSegments.map((segment) => (
                  <div key={segment.label} className="flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2 text-gray-600">
                      <span
                        className="h-2.5 w-2.5 rounded-full"
                        style={{ backgroundColor: segment.color }}
                      />
                      {segment.label}
                    </span>
                    <span className="font-semibold text-gray-900">
                      {segment.percent.toFixed(1)}%
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {missingCount > 0 && (
            <div className="mt-8">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-semibold text-gray-700">Missing words</h4>
                <button
                  type="button"
                  onClick={handleAddAll}
                  disabled={addingAll}
                  className="rounded-full border border-[#55F] px-3 py-1 text-xs font-semibold text-[#55F] hover:bg-[#55F] hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {addingAll ? 'Adding...' : 'Add all'}
                </button>
              </div>
              <p className="mt-1 text-xs text-gray-500">
                These words were not found in your known vocabulary or the JLPT dictionary.
              </p>
              <div className="mt-2 flex flex-wrap gap-2">
                {localMissing.slice(0, 30).map((word) => {
                  const meaning = meaningByLemma[word]?.trim() || 'unknown meaning'
                  return (
                    <span key={word} className="group relative inline-block">
                      <button
                        type="button"
                        onClick={() => handleAddKnown(word)}
                        disabled={addingByLemma[word]}
                        className="rounded-full bg-rose-50 px-2.5 py-1 text-xs font-medium text-rose-700 hover:bg-rose-100 hover:text-rose-900 disabled:cursor-not-allowed disabled:opacity-50 transition-colors"
                      >
                        {word}
                      </button>
                      <span className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-1 hidden max-w-xs -translate-x-1/2 whitespace-nowrap rounded bg-gray-900 px-2 py-1 text-xs font-normal text-white shadow-lg group-hover:block">
                        {meaning}
                      </span>
                    </span>
                  )
                })}
                {missingCount > 30 && (
                  <span className="text-xs text-gray-500">+{missingCount - 30} more</span>
                )}
              </div>
              {addError && (
                <div className="mt-2 rounded border border-red-200 bg-red-50 px-2 py-1 text-xs text-red-700">
                  {addError}
                </div>
              )}
            </div>
          )}
        </aside>
        </div>
      </div>
      {showReview && (
        <ReviewSession
          tokens={localTokens}
          language="ja"
          onClose={() => setShowReview(false)}
          onWordMarked={handleWordMarked}
        />
      )}
    </section>
  )
}
