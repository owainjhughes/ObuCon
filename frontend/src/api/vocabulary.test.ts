import { apiClient } from "./client"
import { fetchVocabulary, importVocabulary } from "./vocabulary"

jest.mock("./client", () => ({ apiClient: { get: jest.fn(), post: jest.fn() } }))

const get = apiClient.get as jest.Mock
const post = apiClient.post as jest.Mock

test("drops duplicate lemmas from the vocabulary list", async () => {
  const first = { lemma: "食べる", hiragana: "たべる", meaning: "to eat" }
  const second = { lemma: "飲む", hiragana: "のむ", meaning: "to drink" }
  get.mockResolvedValue({ data: { vocab: [first, second, { ...first, meaning: "eat" }] } })

  await expect(fetchVocabulary()).resolves.toEqual([first, second])
})

test("posts one import request for the whole batch", async () => {
  const entries = [{ lemma: "食べる", hiragana: "たべる", meaning: "to eat" }]
  post.mockResolvedValue({ data: { added: 1, updated: 0, skipped: 0 } })

  await expect(importVocabulary(entries)).resolves.toEqual({ added: 1, updated: 0, skipped: 0 })
  expect(post).toHaveBeenCalledWith("/vocab/import", { language: "ja", entries })
})
