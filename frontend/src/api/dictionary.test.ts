import { apiClient } from "./client"
import { fetchDictionaryPage } from "./dictionary"

jest.mock("./client", () => ({ apiClient: { get: jest.fn() } }))

const get = apiClient.get as jest.Mock

test("sends the filters as query parameters and maps the response", async () => {
  const entry = { kanji: "食べる", hiragana: "たべる", meaning: "to eat", jlpt_level: 5 }
  get.mockResolvedValue({
    data: { entries: [entry], page: 2, page_size: 15, total: 16, total_pages: 2 },
  })

  const page = await fetchDictionaryPage({ query: " 食 ", jlpt: 5, page: 2, pageSize: 15 })

  expect(get).toHaveBeenCalledWith("/dictionary", {
    params: { language: "ja", query: "食", jlpt: 5, page: 2, page_size: 15 },
  })
  expect(page).toEqual({ entries: [entry], total: 16, totalPages: 2 })
})

test("omits an empty search and an unset JLPT filter", async () => {
  get.mockResolvedValue({ data: { entries: [], page: 1, page_size: 15, total: 0, total_pages: 1 } })

  await fetchDictionaryPage({ query: "   ", jlpt: null, page: 1, pageSize: 15 })

  expect(get).toHaveBeenCalledWith("/dictionary", {
    params: { language: "ja", query: undefined, jlpt: undefined, page: 1, page_size: 15 },
  })
})
