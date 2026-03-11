import React from 'react'

interface Token {
  surface: string
  pos: string
  is_known: boolean
  grade_level?: number | null
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
  if (knownPct < 85) return { label: 'Poor', color: '#dc2626' }
  if (knownPct < 95) return { label: 'Good', color: '#16a34a' }
  return { label: 'Ineffective', color: '#c2410c' }
}

function createPieSegments(counts: Record<string, number>) {
  const total = Object.values(counts).reduce((sum, v) => sum + v, 0)
  const colors: Record<string, string> = {
    'JLPT N1': '#2563EB',
    'JLPT N2': '#4F46E5',
    'JLPT N3': '#9333EA',
    'JLPT N4': '#EA580C',
    'JLPT N5': '#16A34A',
    Unknown: '#6B7280',
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

function combineTokensForDisplay(tokens: Token[]) {
  const combined: Array<{ surface: string; is_known: boolean; grade_level?: number | null }> = []

  for (let i = 0; i < tokens.length; i++) {
    const token = tokens[i]
    // If this token is a verb and the next is an auxiliary, combine them for clearer display
    if (token.pos.includes('動詞') && i + 1 < tokens.length && isAuxiliary(tokens[i + 1])) {
      const combinedSurface = token.surface + tokens[i + 1].surface
      const combinedKnown = token.is_known || tokens[i + 1].is_known
      const gradeLevel = token.grade_level ?? tokens[i + 1].grade_level
      combined.push({ surface: combinedSurface, is_known: combinedKnown, grade_level: gradeLevel })
      i += 1
      continue
    }

    combined.push({ surface: token.surface, is_known: token.is_known, grade_level: token.grade_level })
  }

  return combined
}

export default function AnalysisOutput({ tokens, missing, onReset }: AnalysisOutputProps) {
  const displayTokens = combineTokensForDisplay(tokens)
  const total = tokens.length
  const knownCount = tokens.filter((t) => t.is_known).length
  const missingCount = missing.length
  const knownPct = total === 0 ? 0 : (knownCount / total) * 100
  const rating = getRating(knownPct)

  const jlptCounts = tokens.reduce<Record<string, number>>((acc, token) => {
    const level = token.grade_level != null ? `JLPT N${token.grade_level}` : 'Unknown'
    acc[level] = (acc[level] ?? 0) + 1
    return acc
  }, {})

  const pieSegments = createPieSegments(jlptCounts)

  return (
    <section className="px-4 py-10">
      <div className="mx-auto flex max-w-4xl flex-col gap-6 lg:flex-row">
        <div className="flex-1 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Analysis Output</h2>
              <p className="mt-1 text-sm text-gray-600">
                Highlighted words are based on your known vocabulary.
              </p>
            </div>
            <button
              type="button"
              onClick={onReset}
              className="rounded-full border border-[#55F] bg-white px-4 py-1.5 text-sm font-semibold text-[#55F] hover:bg-[#55F] hover:text-white"
            >
              New analysis
            </button>
          </div>

          <div className="mt-6 rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm leading-relaxed">
            {displayTokens.map((token, idx) => (
              <React.Fragment key={idx}>
                <span
                  className={`inline text-xs font-medium ${
                    token.is_known
                      ? 'bg-emerald-100 text-emerald-800'
                      : 'bg-rose-100 text-rose-800'
                  }`}
                >
                  {token.surface}
                </span>
                {idx < displayTokens.length - 1 && (
                  <span className="mx-1 inline text-xs font-medium text-gray-400">·</span>
                )}
              </React.Fragment>
            ))}
          </div>
        </div>

        <aside className="w-full max-w-sm rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-gray-900">Stats</h3>

          <div className="mt-6">
            <div className="flex items-center gap-3">
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
                      strokeLinecap="round"
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
              <h4 className="text-sm font-semibold text-gray-700">Missing words</h4>
              <p className="mt-1 text-xs text-gray-500">
                These words were not found in your known vocabulary or the JLPT dictionary.
              </p>
              <div className="mt-2 flex flex-wrap gap-2">
                {missing.slice(0, 30).map((word) => (
                  <span
                    key={word}
                    className="rounded bg-rose-50 px-2 py-1 text-xs font-medium text-rose-700"
                  >
                    {word}
                  </span>
                ))}
                {missingCount > 30 && (
                  <span className="text-xs text-gray-500">+{missingCount - 30} more</span>
                )}
              </div>
            </div>
          )}
        </aside>
      </div>
    </section>
  )
}
