"""
transform_de.py

Extracts German vocabulary from PDF word lists (A1-C1), enriches with English
definitions from dictionary-de.json, and outputs german_vocabulary_seed.csv.

Sources:
  A1: A1_SD1_Wortliste_02.pdf          (Goethe-Institut Start Deutsch 1)
  A1: A1_word_list.csv                  (supplementary flashcard list)
  A2: Goethe-Zertifikat_A2_Wortliste.pdf
  B1: Goethe-Zertifikat_B1_Wortliste.pdf
  B2: aspekte-neu-b2-lb-kapitelwortschatz.pdf
  C1: aspekte-neu-c1-lb-kapitelwortschatz.pdf

CEFR level mapping: A1=1, A2=2, B1=3, B2=4, C1=5
"""

import csv
import json
import os
import re

import fitz  # PyMuPDF

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DE_DIR = os.path.join(SCRIPT_DIR, "original_data", "de")
OUTPUT_CSV = os.path.join(SCRIPT_DIR, "german_vocabulary_seed.csv")

# German articles and auxiliary forms to strip or use as sentence signals
ARTICLES = {"der", "die", "das", "ein", "eine", "des", "dem", "den", "einem", "einer"}

# Auxiliary verb forms that indicate a conjugation-continuation line (not a headword)
AUX_PREFIXES = (
    "hat ", "haben ", "ist ", "sind ", "war ", "waren ",
    "wird ", "werden ", "wurde ", "wurden ",
    "hatte ", "hatten ", "wäre ", "wären ",
    "habe ", "hast ", "sei ",
)

# Words that signal the start of an example sentence
SENTENCE_STARTERS = (
    "ich ", "er ", "sie ", "es ", "wir ", "ihr ", "man ",
    "hier ", "da ", "wo ", "was ", "wie ", "wann ", "bitte ",
    "kannst ", "können ", "nein,", "ja,",
    "guten ", "gute ", "guter ", "in ", "bei ", "auf ",
    "heute ", "morgen ", "gestern ",
    "zum ", "zur ", "am ", "im ",
    "mein ", "meine ", "dein ", "deine ",
    "dieser ", "diese ", "dieses ", "diesen ",
    "mit ", "für ", "von ", "nach ", "aus ",
    "ab ", "bis ", "um ",
)

# Header/footer strings to skip (lowercased)
HEADERS = {
    "a1": {
        "vs_02_280312", "seite", "inhalt", "vorwort", "themen", "wortschatz:",
        "wortgruppenliste", "alphabetische wortliste", "alphabetischer wortschatz",
        "literatur", "inventare", "b1", "b2", "c1", "c2", "a2", "a1",
    },
    "a2": {
        "goethe-zertifikat a2", "wortliste", "a2_wortliste_03_200616", "inhalt",
        "vorwort", "wortgruppen", "abkürzungen", "anweisungssprache",
        "alphabetischer wortschatz", "b1", "b2", "c1", "c2", "a2", "a1",
    },
    "b1": {
        "zertifikat b1", "wortliste", "vs_03", "inhalt", "vorwort",
        "wortgruppen", "abkürzungen", "anglizismen", "b1", "b2", "c1", "c2", "a2", "a1",
        "goethe", "ösd",
    },
    "b2": {
        "kapitelwortschatz", "aspekte neu b2", "seite", "grammatik",
    },
    "c1": {
        "kapitelwortschatz", "aspekte neu c1", "seite", "grammatik",
    },
}


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def is_header_line(line: str, pdf_type: str) -> bool:
    """Return True if this line is a page header, footer, or section label."""
    stripped = line.strip()
    if not stripped:
        return True
    lower = stripped.lower()

    # Pure page number
    if re.match(r"^\d+$", stripped):
        return True
    # Single uppercase letter divider (A, B, C, ...)
    if re.match(r"^[A-ZÄÖÜ]$", stripped):
        return True
    # Chapter/module headings in Aspekte Neu
    if re.match(r"^(Kapitel|Modul|Auftakt)\b", stripped):
        return True
    # Exercise reference-only lines like "1b" or "3a"
    if re.match(r"^\d+[a-z]?$", stripped):
        return True

    for header in HEADERS.get(pdf_type, set()):
        if lower == header or lower.startswith(header + " ") or lower.startswith(header + "\n"):
            return True

    return False


def is_sentence_line(line: str) -> bool:
    """Return True if this line looks like an example sentence, not a headword."""
    stripped = line.strip()
    if not stripped:
        return True

    # Ends with sentence-ending punctuation
    if stripped.endswith((".", "!", "?", "…")):
        return True

    lower = stripped.lower()

    # Starts with an auxiliary verb form (conjugation continuation)
    if lower.startswith(AUX_PREFIXES):
        return True

    # Starts with a known sentence-starting particle
    if lower.startswith(SENTENCE_STARTERS):
        return True

    # Very long lines are likely continuation sentences
    if len(stripped.split()) > 6:
        return True

    return False


def clean_headword(text: str) -> str | None:
    """
    Extract the clean base word from a raw dictionary entry fragment.
    Returns None if the fragment should be discarded.
    """
    text = text.strip()
    if not text or len(text) < 2:
        return None

    # Remove trailing commas, punctuation, and stray parentheses from word list notation
    text = text.rstrip(",. )(")
    # Remove leading stray parentheses
    text = text.lstrip("(")

    # Strip parenthetical usage hints: "(sich)", "(D.)", "(etw.)", etc.
    text = re.sub(r"\(.*?\)", "", text).strip()

    # Strip leading article
    parts = text.split()
    if not parts:
        return None
    if parts[0].lower() in ARTICLES:
        parts = parts[1:]
    if not parts:
        return None

    word = parts[0].rstrip(",.")

    # Skip entries that contain digit, special, or unmatched parenthesis characters
    if re.search(r"[0-9@#$%^&*()\[\]{};:\"\\|<>/?]", word):
        return None

    # Skip dash-tailed morphological stubs like "ander-" or "-weise" prefix entries
    if word.startswith("-") or word.endswith("-"):
        return None

    # Skip very short artefacts
    if len(word) < 2:
        return None

    # Skip obvious non-word strings (only hyphens, slashes, etc.)
    if not re.search(r"[A-Za-zÄÖÜäöüß]", word):
        return None

    return word


# ---------------------------------------------------------------------------
# Per-PDF extraction
# ---------------------------------------------------------------------------

def extract_a1_pdf(pdf_path: str) -> list[str]:
    """
    A1 PDF: alphabetical word list starts at page 9 (index 8).
    Layout: headword on its own line, followed by a German example sentence.
    Sub-entries are indented with leading spaces.
    """
    doc = fitz.open(pdf_path)
    words = []

    for page_num in range(8, doc.page_count):
        text = doc[page_num].get_text()
        for line in text.split("\n"):
            if is_header_line(line, "a1"):
                continue
            if is_sentence_line(line):
                continue
            # Strip indentation and take the base form (before first comma)
            base = line.strip().split(",")[0]
            word = clean_headword(base)
            if word:
                words.append(word)

    doc.close()
    return words


def extract_a2_pdf(pdf_path: str) -> list[str]:
    """
    A2 PDF: alphabetical word list starts at page 8 (index 7).
    Layout: each PyMuPDF block contains "headword + conjugation   example sentence".
    The headword is the text before the first large whitespace gap on the first line.
    """
    doc = fitz.open(pdf_path)
    words = []

    for page_num in range(7, doc.page_count):
        blocks = doc[page_num].get_text("blocks")
        for block in blocks:
            text: str = block[4]
            first_line = text.split("\n")[0].strip()

            if is_header_line(first_line, "a2"):
                continue

            # Split on 2+ consecutive spaces to separate headword column from example column
            parts = re.split(r"\s{2,}", first_line, maxsplit=1)
            headword_fragment = parts[0].strip()

            # Skip auxiliary-verb continuation lines (e.g., "hat angeboten")
            if headword_fragment.lower().startswith(AUX_PREFIXES):
                continue
            if is_sentence_line(headword_fragment):
                continue

            # Take only the part before the first comma (strips conjugation forms)
            base = headword_fragment.split(",")[0]
            word = clean_headword(base)
            if word:
                words.append(word)

    doc.close()
    return words


def extract_b1_pdf(pdf_path: str) -> list[str]:
    """
    B1 PDF: alphabetical word list starts at page 16 (index 15).
    Layout: true two-column — headword blocks at x0 < 200, example blocks at x0 >= 200.
    Headword blocks may span multiple lines (conjugation forms); only the first line matters.
    """
    doc = fitz.open(pdf_path)
    words = []

    for page_num in range(15, doc.page_count):
        blocks = doc[page_num].get_text("blocks")
        for block in blocks:
            x0: float = block[0]
            text: str = block[4]

            # Right-column blocks contain example sentences — skip
            if x0 >= 200:
                continue

            first_line = text.split("\n")[0].strip()

            if is_header_line(first_line, "b1"):
                continue
            if is_sentence_line(first_line):
                continue

            # Strip reflexive pronoun prefix: "(sich) anmelden" → "anmelden"
            first_line = re.sub(r"^\(sich\)\s*", "", first_line)

            # Take only the part before the first comma (strips "kommt an, kam an, …")
            base = first_line.split(",")[0]
            word = clean_headword(base)
            if word:
                words.append(word)

    doc.close()
    return words


def extract_aspekte_pdf(pdf_path: str, pdf_type: str) -> list[str]:
    """
    Aspekte Neu B2 / C1: chapter-based vocabulary lists, all pages.
    Format per line: [exercise_ref] [article] word[, conj forms] [(usage hint)]
    """
    doc = fitz.open(pdf_path)
    words = []

    for page_num in range(doc.page_count):
        text = doc[page_num].get_text()
        for line in text.split("\n"):
            line = line.strip()
            if not line:
                continue
            if is_header_line(line, pdf_type):
                continue

            # Strip leading exercise reference: "1b ", "2a ", "3 "
            line = re.sub(r"^\d+[a-z]?\s+", "", line)
            if not line:
                continue

            if is_sentence_line(line):
                continue

            # Strip parenthetical usage hint from the end before splitting on comma
            # e.g. "angeben, gibt an, gab an, hat angegeben (eine Begründung angeben)"
            line = re.sub(r"\s*\(.*", "", line).strip()

            # Take only the part before the first comma
            base = line.split(",")[0]
            word = clean_headword(base)
            if word:
                words.append(word)

    doc.close()
    return words


# ---------------------------------------------------------------------------
# Definition lookup
# ---------------------------------------------------------------------------

def load_dictionary(dict_path: str) -> dict[str, str]:
    """Load dictionary-de.json into a lowercase-word → first-English-definition map."""
    lookup: dict[str, str] = {}
    with open(dict_path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
                word = obj.get("", "")
                defs = obj.get("d", [])
                if word and defs:
                    key = word.lower()
                    if key not in lookup:
                        lookup[key] = defs[0]
            except (json.JSONDecodeError, AttributeError):
                pass
    return lookup


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    print("Loading dictionary-de.json …")
    lookup = load_dictionary(os.path.join(DE_DIR, "dictionary-de.json"))
    print(f"  {len(lookup):,} entries with English definitions")

    # entries: word_lower → (display_word, definition, cefr_level)
    entries: dict[str, tuple[str, str, int]] = {}

    def add_words(word_list: list[str], cefr_level: int) -> None:
        for word in word_list:
            key = word.lower()
            # Keep the lowest (easiest) level if the word appears in multiple sources
            if key in entries and entries[key][2] <= cefr_level:
                continue
            defn = lookup.get(key, "")
            entries[key] = (word, defn, cefr_level)

    # A1 supplementary CSV (has paired German → English already)
    # Each Question field may contain "headword, plural\nexample sentence" across multiple lines.
    # Take only the first line and extract the base word before the first comma.
    print("Processing A1 CSV …")
    a1_csv_path = os.path.join(DE_DIR, "A1_word_list.csv")
    with open(a1_csv_path, encoding="utf-8") as f:
        reader = csv.DictReader(f)
        count = 0
        for row in reader:
            question = (row.get("Question") or "").strip()
            answer = (row.get("Answer") or "").strip()
            if not question:
                continue
            # Take only the first line (before any embedded newline)
            headword_line = question.split("\n")[0].strip()
            # Strip plural/grammar notation after the first comma
            base = headword_line.split(",")[0].strip()
            word = clean_headword(base)
            if not word:
                continue
            # Take the first line of the answer as the English definition
            defn_csv = answer.split("\n")[0].strip()
            key = word.lower()
            # Prefer dict definition when available; fall back to CSV answer
            defn = lookup.get(key, "") or defn_csv
            if key not in entries:
                entries[key] = (word, defn, 1)
            count += 1
    print(f"  {count} entries")

    # A1 PDF
    print("Processing A1 PDF …")
    a1_words = extract_a1_pdf(os.path.join(DE_DIR, "A1_SD1_Wortliste_02.pdf"))
    print(f"  {len(a1_words)} raw entries extracted")
    add_words(a1_words, 1)

    # A2 PDF
    print("Processing A2 PDF …")
    a2_words = extract_a2_pdf(os.path.join(DE_DIR, "Goethe-Zertifikat_A2_Wortliste.pdf"))
    print(f"  {len(a2_words)} raw entries extracted")
    add_words(a2_words, 2)

    # B1 PDF
    print("Processing B1 PDF …")
    b1_words = extract_b1_pdf(os.path.join(DE_DIR, "Goethe-Zertifikat_B1_Wortliste.pdf"))
    print(f"  {len(b1_words)} raw entries extracted")
    add_words(b1_words, 3)

    # B2 PDF
    print("Processing B2 PDF …")
    b2_words = extract_aspekte_pdf(
        os.path.join(DE_DIR, "aspekte-neu-b2-lb-kapitelwortschatz.pdf"), "b2"
    )
    print(f"  {len(b2_words)} raw entries extracted")
    add_words(b2_words, 4)

    # C1 PDF
    print("Processing C1 PDF …")
    c1_words = extract_aspekte_pdf(
        os.path.join(DE_DIR, "aspekte-neu-c1-lb-kapitelwortschatz.pdf"), "c1"
    )
    print(f"  {len(c1_words)} raw entries extracted")
    add_words(c1_words, 5)

    # Sort by CEFR level then alphabetically
    all_entries = sorted(entries.values(), key=lambda x: (x[2], x[0].lower()))

    # Summary
    level_labels = {1: "A1", 2: "A2", 3: "B1", 4: "B2", 5: "C1"}
    level_counts: dict[int, int] = {}
    def_count = sum(1 for _, d, _ in all_entries if d)
    for _, _, lvl in all_entries:
        level_counts[lvl] = level_counts.get(lvl, 0) + 1

    print(f"\nTotal unique words: {len(all_entries):,}")
    print(f"Words with English definition: {def_count:,} ({def_count/len(all_entries)*100:.1f}%)")
    for lvl, count in sorted(level_counts.items()):
        print(f"  {level_labels[lvl]}: {count}")

    # Write CSV
    with open(OUTPUT_CSV, "w", encoding="utf-8", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "definition", "cefr_level"])
        for word, definition, cefr_level in all_entries:
            writer.writerow([word, definition, cefr_level])

    print(f"\nWritten to {OUTPUT_CSV}")

    # Print a sample from each level for manual inspection
    print("\nSample words per level:")
    for lvl in sorted(level_counts):
        sample = [w for w, _, l in all_entries if l == lvl][:8]
        print(f"  {level_labels[lvl]}: {', '.join(sample)}")


if __name__ == "__main__":
    main()
