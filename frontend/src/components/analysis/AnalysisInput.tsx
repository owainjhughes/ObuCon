import React, { useState } from 'react'
import { apiClient } from '../../api/client'
import AnalysisOutput from './AnalysisOutput'

type AnalysisMode = 'text' | 'file' | 'link'

interface Token {
  surface: string
  lemma: string
  pos: string
  is_known: boolean
  grade_level?: number | null
}

interface AnalysisResult {
  tokens: Token[]
  total_tokens: number
  missing: string[]
}

export default function AnalysisInput() {
  const [mode, setMode] = useState<AnalysisMode>('text')
  const [text, setText] = useState('')
  const [language] = useState('ja')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<AnalysisResult | null>(null)

  const handleAnalyse = async () => {
    setError('')
    setResult(null)
    setIsLoading(true)

    try {
      const response = await apiClient.post('/analyze', {
        text,
        language,
      })
      setResult(response.data)
    } catch (err: any) {
      const message = err?.response?.data?.error || 'Analysis failed'
      setError(message)
    } finally {
      setIsLoading(false)
    }
  }

  const resetAnalysis = () => {
    setResult(null)
    setError('')
  }

  if (result) {
    return (
      <AnalysisOutput
        tokens={result.tokens}
        missing={result.missing}
        onReset={resetAnalysis}
      />
    )
  }

  return (
    <section className="px-4 py-10">
      <div className="mx-auto rounded-xl border border-gray-200 bg-white p-6 shadow-sm max-w-3xl">
        <h2 className="text-lg font-semibold text-gray-900">Analysis</h2>
        <p className="mt-1 text-sm text-gray-600">Choose a mode to provide your input.</p>

        <div className="mt-4 flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => setMode('text')}
            className={`rounded-full border px-4 py-1.5 text-sm ${
              mode === 'text'
                ? 'border-[#55F] bg-[#55F] text-white'
                : 'border-gray-300 text-gray-700 hover:border-[#55F]'
            }`}
          >
            Text
          </button>
          <button
            type="button"
            onClick={() => setMode('file')}
            className={`rounded-full border px-4 py-1.5 text-sm ${
              mode === 'file'
                ? 'border-[#55F] bg-[#55F] text-white'
                : 'border-gray-300 text-gray-700 hover:border-[#55F]'
            }`}
          >
            File
          </button>
          <button
            type="button"
            onClick={() => setMode('link')}
            className={`rounded-full border px-4 py-1.5 text-sm ${
              mode === 'link'
                ? 'border-[#55F] bg-[#55F] text-white'
                : 'border-gray-300 text-gray-700 hover:border-[#55F]'
            }`}
          >
            Link
          </button>
        </div>

        <div className="mt-6">
          {mode === 'text' && (
            <label className="block">
              <span className="text-sm font-medium text-gray-700">Paste text</span>
              <textarea
                rows={8}
                value={text}
                onChange={(e) => setText(e.target.value)}
                className="mt-2 w-full rounded-lg border border-gray-300 p-3 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
                placeholder="Enter the text you want to analyze"
              />
            </label>
          )}

          {mode === 'file' && (
            <label className="block">
              <span className="text-sm font-medium text-gray-700">Upload a file</span>
              <input
                type="file"
                className="mt-2 block w-full text-sm text-gray-700 file:mr-4 file:rounded-md file:border-0 file:bg-[#55F] file:px-4 file:py-2 file:text-sm file:font-semibold file:text-white hover:file:bg-[#44E]"
              />
            </label>
          )}

          {mode === 'link' && (
            <label className="block">
              <span className="text-sm font-medium text-gray-700">Paste a link</span>
              <input
                type="url"
                className="mt-2 w-full rounded-lg border border-gray-300 p-3 text-sm text-gray-900 focus:border-[#55F] focus:outline-none"
                placeholder="https://example.com"
              />
            </label>
          )}
          <div className="mt-4 flex justify-end">
            <button
              onClick={handleAnalyse}
              disabled={isLoading || !text.trim()}
              className="rounded-full border px-4 py-1.5 text-sm border-[#55F] bg-[#55F] text-white disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isLoading ? 'Analysing...' : 'Analyse'}
            </button>
          </div>

          {error && (
            <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          )}
        </div>
      </div>
    </section>
  )
}
