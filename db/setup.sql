BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    gmail TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user'
        CHECK (role IN ('user', 'creator', 'admin'))
);

CREATE TABLE IF NOT EXISTS draft_forms (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sections (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    form_id BIGINT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0)
);

CREATE TABLE IF NOT EXISTS questions (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    type TEXT NOT NULL
        CHECK (type IN ('text', 'checkbox', 'mcq')),
    validation JSONB NOT NULL DEFAULT '{}'::jsonb,
    section_id BIGINT NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0)
);

CREATE TABLE IF NOT EXISTS checkbox_options (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    question_id BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS mcq_options (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    question_id BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    unlock_section_id BIGINT NULL REFERENCES sections(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS published_forms (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deadline TIMESTAMPTZ,
    structure JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS view_permissions (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    form_id BIGINT NOT NULL REFERENCES published_forms(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, form_id)
);

CREATE TABLE IF NOT EXISTS responses (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    form_id BIGINT NOT NULL REFERENCES published_forms(id) ON DELETE CASCADE,
    "timestamp" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS answers (
    id BIGSERIAL PRIMARY KEY,
    response_id BIGINT NOT NULL REFERENCES responses(id) ON DELETE CASCADE,
    question_id BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    UNIQUE (response_id, question_id)
);

CREATE TABLE IF NOT EXISTS checkbox_answers (
    checkbox_option_id BIGINT NOT NULL REFERENCES checkbox_options(id) ON DELETE CASCADE,
    answer_id BIGINT NOT NULL REFERENCES answers(id) ON DELETE CASCADE,
    PRIMARY KEY (checkbox_option_id, answer_id)
);

-- Useful indexes for the relationships that will be queried frequently.
CREATE INDEX IF NOT EXISTS idx_draft_forms_author_id
    ON draft_forms(author_id);

CREATE INDEX IF NOT EXISTS idx_sections_form_id
    ON sections(form_id);

CREATE INDEX IF NOT EXISTS idx_questions_section_id
    ON questions(section_id);

CREATE INDEX IF NOT EXISTS idx_checkbox_options_question_id
    ON checkbox_options(question_id);

CREATE INDEX IF NOT EXISTS idx_mcq_options_question_id
    ON mcq_options(question_id);

CREATE INDEX IF NOT EXISTS idx_published_forms_author_id
    ON published_forms(author_id);

CREATE INDEX IF NOT EXISTS idx_responses_form_id
    ON responses(form_id);

CREATE INDEX IF NOT EXISTS idx_responses_user_id
    ON responses(user_id);

CREATE INDEX IF NOT EXISTS idx_answers_response_id
    ON answers(response_id);

CREATE INDEX IF NOT EXISTS idx_answers_question_id
    ON answers(question_id);

CREATE INDEX IF NOT EXISTS idx_checkbox_answers_answer_id
    ON checkbox_answers(answer_id);

COMMIT;
