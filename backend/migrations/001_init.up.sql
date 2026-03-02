-- Reference: https://www.postgresql.org/docs/current/datatype.html

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE known_words (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language VARCHAR(10) NOT NULL,  -- 'ja', 'de', 'ko', etc.
    lemma TEXT NOT NULL,
    grade_level INTEGER,
    status VARCHAR(20) DEFAULT 'unknown',
    metadata JSONB,  -- Flexible language-specific data
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, language, lemma)
);

CREATE TABLE analyses (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    language VARCHAR(10) NOT NULL,
    text_hash CHAR(64),
    coverage_pct DECIMAL(5, 2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE analysis_tokens (
    id SERIAL PRIMARY KEY,
    analysis_id INTEGER NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    surface TEXT NOT NULL,
    lemma TEXT NOT NULL,
    grade_level INTEGER,
    is_known BOOLEAN
);

CREATE TABLE japanese_dictionary (
    id SERIAL PRIMARY KEY,
    kanji VARCHAR(50) NOT NULL,
    hiragana VARCHAR(100) NOT NULL,
    meaning TEXT NOT NULL,
    jlpt_level INTEGER CHECK (jlpt_level >= 1 AND jlpt_level <= 5),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);

CREATE INDEX idx_known_words_user_language ON known_words(user_id, language);
CREATE INDEX idx_known_words_user_id ON known_words(user_id);
CREATE INDEX idx_known_words_lemma ON known_words(lemma);
CREATE INDEX idx_known_words_grade_level ON known_words(grade_level);
CREATE INDEX idx_known_words_metadata_gin ON known_words USING gin (metadata);

CREATE INDEX idx_analyses_user_id ON analyses(user_id);
CREATE INDEX idx_analyses_text_hash ON analyses(text_hash);

CREATE INDEX idx_tokens_analysis_id ON analysis_tokens(analysis_id);
CREATE INDEX idx_tokens_is_known ON analysis_tokens(is_known);

CREATE INDEX idx_japanese_dict_kanji ON japanese_dictionary(kanji);
CREATE INDEX idx_japanese_dict_hiragana ON japanese_dictionary(hiragana);
CREATE INDEX idx_japanese_dict_jlpt ON japanese_dictionary(jlpt_level);
CREATE INDEX idx_japanese_dict_kanji_hiragana ON japanese_dictionary(kanji, hiragana);

-- PostgreSQL trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();