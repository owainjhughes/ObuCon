import React, { useCallback, useEffect, useRef, useState } from 'react'
import { apiClient } from '../../api/client'
import { getApiErrorMessage } from '../../api/errors'

interface Token {
  surface: string
  lemma: string
  pos: string
  is_known: boolean
  grade_level?: number | null
}

interface ReviewWord {
  lemma: string
  hiragana: string
  meaning: string
  jlpt_level: number | null
}

interface ReviewSessionProps {
  tokens: Token[]
  language: string
  onClose: () => void
  onWordMarked: (lemma: string) => void
}

const JLPT_BADGE: Record<number, string> = {
  1: 'bg-blue-100 text-blue-800',
  2: 'bg-indigo-100 text-indigo-800',
  3: 'bg-purple-100 text-purple-800',
  4: 'bg-orange-100 text-orange-800',
  5: 'bg-green-100 text-green-800',
}

export default function ReviewSession({ tokens, language, onClose, onWordMarked }: ReviewSessionProps) {
  const [words, setWords] = useState<ReviewWord[]>([])
  const [index, setIndex] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [marking, setMarking] = useState(false)
  const [markedCount, setMarkedCount] = useState(0)
  const [done, setDone] = useState(false)
  const [slideState, setSlideState] = useState<'idle' | 'out' | 'in'>('idle')

  const initialTokens = useRef(tokens)

  useEffect(() => {
    const reviewableLemmas = [
      ...new Set(
        initialTokens.current
          .filter((t) => !t.is_known && t.grade_level != null)
          .map((t) => t.lemma)
      ),
    ]

    if (reviewableLemmas.length === 0) {
      setIsLoading(false)
      setDone(true)
      return
    }

    const fetchWords = async () => {
      try {
        const params = new URLSearchParams({
          lemmas: reviewableLemmas.join(','),
          language,
        })
        const response = await apiClient.get(`/review/words?${params}`)
        setWords(response.data.words ?? [])
      } catch (err: unknown) {
        setError(getApiErrorMessage(err, 'Failed to load review words'))
      } finally {
        setIsLoading(false)
      }
    }

    fetchWords()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [language])

  const advance = useCallback((nextIndex: number, endReview: boolean) => {
    setSlideState('out')
    setTimeout(() => {
      if (endReview) {
        setDone(true)
        return
      }
      setIndex(nextIndex)
      setSlideState('in')
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          setSlideState('idle')
        })
      })
    }, 200)
  }, [])

  const handleMarkKnown = async () => {
    const word = words[index]
    setMarking(true)
    setError(null)
    try {
      await apiClient.post('/vocab/known', { lemma: word.lemma, language })
      onWordMarked(word.lemma)
      setMarkedCount((prev) => prev + 1)
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, 'Failed to mark word as known'))
    } finally {
      setMarking(false)
      advance(index + 1, index + 1 >= words.length)
    }
  }

  const handleSkip = () => {
    advance(index + 1, index + 1 >= words.length)
  }

  const currentWord = words[index]

  const cardClass =
    slideState === 'out'
      ? 'transition-[transform,opacity] duration-200 ease-in -translate-x-16 opacity-0'
      : slideState === 'in'
      ? 'transition-none translate-x-16 opacity-0'
      : 'transition-[transform,opacity] duration-200 ease-out translate-x-0 opacity-100'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="relative w-full max-w-xl rounded-2xl bg-white p-10 shadow-xl">
        <button
          type="button"
          onClick={onClose}
          className="absolute right-5 top-5 rounded p-1 text-gray-400 hover:text-gray-700"
          aria-label="Close review"
        >
          ✕
        </button>

        <p className="mb-6 text-xs font-semibold uppercase tracking-wide text-gray-400">
          Word Review
        </p>

        {isLoading && (
          <div className="flex items-center justify-center py-12 text-sm text-gray-400">
            Loading...
          </div>
        )}

        {!isLoading && error && !done && (
          <div className="rounded border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        )}

        {!isLoading && done && (
          <div className="py-8 text-center">
            <div className="mb-4 text-5xl">✓</div>
            <h2 className="text-xl font-bold text-gray-900">Review complete</h2>
            <p className="mt-2 text-sm text-gray-500">
              {markedCount === 0
                ? 'No words were marked as known.'
                : `${markedCount} word${markedCount === 1 ? '' : 's'} added to your vocabulary.`}
            </p>
            <button
              type="button"
              onClick={onClose}
              className="mt-6 rounded-full bg-[#55F] px-6 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
            >
              Done
            </button>
          </div>
        )}

        {!isLoading && !done && currentWord && (
          <>
            <div className="mb-5 flex items-center gap-4">
              <span className="text-xs text-gray-400">
                {index + 1} / {words.length}
              </span>
              <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-gray-100">
                <div
                  className="h-1.5 rounded-full bg-[#55F] transition-all duration-300"
                  style={{ width: `${((index + 1) / words.length) * 100}%` }}
                />
              </div>
            </div>

            <div className={`rounded-xl border border-gray-100 bg-gray-50 p-8 text-center overflow-hidden ${cardClass}`}>
              <div className="mb-2 text-7xl font-bold text-gray-900">
                {currentWord.lemma}
              </div>
              {currentWord.hiragana && currentWord.hiragana !== currentWord.lemma && (
                <div className="mb-4 text-xl text-gray-500">{currentWord.hiragana}</div>
              )}
              {currentWord.jlpt_level != null && (
                <span
                  className={`inline-block rounded-full px-3 py-0.5 text-sm font-semibold ${
                    JLPT_BADGE[currentWord.jlpt_level] ?? 'bg-gray-100 text-gray-700'
                  }`}
                >
                  JLPT N{currentWord.jlpt_level}
                </span>
              )}
              <div className="mt-5 text-lg text-gray-700">
                {currentWord.meaning || (
                  <span className="italic text-gray-400">No meaning available</span>
                )}
              </div>
            </div>

            {error && (
              <div className="mt-3 rounded border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">
                {error}
              </div>
            )}

            <div className="mt-7 flex gap-3">
              <button
                type="button"
                onClick={handleSkip}
                disabled={marking || slideState !== 'idle'}
                className="flex-1 rounded-full border border-gray-300 py-3 text-sm font-semibold text-gray-600 hover:bg-gray-50 disabled:opacity-50"
              >
                Skip
              </button>
              <button
                type="button"
                onClick={handleMarkKnown}
                disabled={marking || slideState !== 'idle'}
                className="flex-1 rounded-full bg-[#55F] py-3 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {marking ? 'Saving...' : 'Mark as known'}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
