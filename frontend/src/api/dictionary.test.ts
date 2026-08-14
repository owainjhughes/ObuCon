import { parseDictionaryPage, type DictionaryEntry } from "./dictionary"

const entries: DictionaryEntry[] = [
  { kanji: "食べる", hiragana: "たべる", meaning: "to eat", jlpt_level: 5 },
  { kanji: "飲む", hiragana: "のむ", meaning: "to drink", jlpt_level: 5 },
  { kanji: "経験", hiragana: "けいけん", meaning: "experience", jlpt_level: 3 },
]

test("uses server pagination without filtering the returned page again", () => {
  const result = parseDictionaryPage(
    {
      entries: [entries[1]],
      pagination: { page: 2, page_size: 1, total: 3, total_pages: 3 },
    },
    { query: "does not match", jlpt: 1, page: 2, pageSize: 1 },
  )

  expect(result).toEqual({
    entries: [entries[1]],
    page: 2,
    pageSize: 1,
    total: 3,
    totalPages: 3,
  })
})

test("filters and slices a legacy unpaginated response", () => {
  const result = parseDictionaryPage(
    { entries },
    { query: "to", jlpt: 5, page: 2, pageSize: 1 },
  )

  expect(result).toEqual({
    entries: [entries[1]],
    page: 2,
    pageSize: 1,
    total: 2,
    totalPages: 2,
  })
})
