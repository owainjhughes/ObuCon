import { buildVocabularyImport } from "./vocabulary"

test("builds one normalized vocabulary import payload", () => {
  expect(
    buildVocabularyImport([
      { lemma: " 食べる ", meaning: " to eat ", hiragana: " たべる " },
      { lemma: "飲む", meaning: "", hiragana: "" },
    ]),
  ).toEqual({
    language: "ja",
    entries: [
      { lemma: "食べる", meaning: "to eat", hiragana: "たべる" },
      { lemma: "飲む" },
    ],
  })
})
