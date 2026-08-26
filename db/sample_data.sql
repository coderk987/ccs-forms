BEGIN;

-- USERS
INSERT INTO users (gmail, name, role) VALUES
    ('alice@example.com', 'Alice Sharma', 'admin'),
    ('bob@example.com', 'Bob Singh', 'user'),
    ('charlie@example.com', 'Charlie Kumar', 'user');

-- DRAFT_FORMS
INSERT INTO draft_forms (title, description, author_id)
SELECT
    'Developer Experience Survey',
    'A short survey about developer tools and experience.',
    id
FROM users
WHERE gmail = 'alice@example.com';

INSERT INTO draft_forms (title, description, author_id)
SELECT
    'Workshop Registration',
    'Registration form for an upcoming technical workshop.',
    id
FROM users
WHERE gmail = 'bob@example.com';

-- SECTIONS
INSERT INTO sections (title, description, form_id, position)
SELECT
    'Personal Information',
    'Basic information about the respondent.',
    id,
    1
FROM draft_forms
WHERE title = 'Developer Experience Survey';

INSERT INTO sections (title, description, form_id, position)
SELECT
    'Development Experience',
    'Questions about programming and development.',
    id,
    2
FROM draft_forms
WHERE title = 'Developer Experience Survey';

INSERT INTO sections (title, description, form_id, position)
SELECT
    'Workshop Preferences',
    'Choose your preferred workshop options.',
    id,
    1
FROM draft_forms
WHERE title = 'Workshop Registration';

-- QUESTIONS
INSERT INTO questions (title, type, validation, section_id, position)
SELECT
    'What is your name?',
    'text',
    '{"required": true, "maxLength": 100}'::jsonb,
    id,
    1
FROM sections
WHERE title = 'Personal Information';

INSERT INTO questions (title, type, validation, section_id, position)
SELECT
    'What is your primary programming language?',
    'text',
    '{"required": true, "maxLength": 50}'::jsonb,
    id,
    2
FROM sections
WHERE title = 'Personal Information';

INSERT INTO questions (title, type, validation, section_id, position)
SELECT
    'Which technologies do you use?',
    'checkbox',
    '{"required": true, "minSelections": 1, "maxSelections": 4}'::jsonb,
    id,
    1
FROM sections
WHERE title = 'Development Experience';

INSERT INTO questions (title, type, validation, section_id, position)
SELECT
    'How long have you been programming?',
    'mcq',
    '{"required": true}'::jsonb,
    id,
    2
FROM sections
WHERE title = 'Development Experience';

INSERT INTO questions (title, type, validation, section_id, position)
SELECT
    'Which workshop track do you prefer?',
    'mcq',
    '{"required": true}'::jsonb,
    id,
    1
FROM sections
WHERE title = 'Workshop Preferences';

INSERT INTO questions (title, type, validation, section_id, position)
SELECT
    'What do you want to learn?',
    'text',
    '{"required": false, "maxLength": 500}'::jsonb,
    id,
    2
FROM sections
WHERE title = 'Workshop Preferences';

-- CHECKBOX_OPTIONS
INSERT INTO checkbox_options (title, question_id)
SELECT 'Go', id
FROM questions
WHERE title = 'Which technologies do you use?';

INSERT INTO checkbox_options (title, question_id)
SELECT 'TypeScript', id
FROM questions
WHERE title = 'Which technologies do you use?';

INSERT INTO checkbox_options (title, question_id)
SELECT 'PostgreSQL', id
FROM questions
WHERE title = 'Which technologies do you use?';

INSERT INTO checkbox_options (title, question_id)
SELECT 'Docker', id
FROM questions
WHERE title = 'Which technologies do you use?';

-- MCQ_OPTIONS
INSERT INTO mcq_options (title, question_id, unlock_section_id)
SELECT 'Less than 1 year', q.id, NULL
FROM questions q
WHERE q.title = 'How long have you been programming?';

INSERT INTO mcq_options (title, question_id, unlock_section_id)
SELECT '1-3 years', q.id, NULL
FROM questions q
WHERE q.title = 'How long have you been programming?';

INSERT INTO mcq_options (title, question_id, unlock_section_id)
SELECT '3-5 years', q.id, NULL
FROM questions q
WHERE q.title = 'How long have you been programming?';

INSERT INTO mcq_options (title, question_id, unlock_section_id)
SELECT '5+ years', q.id, NULL
FROM questions q
WHERE q.title = 'How long have you been programming?';

INSERT INTO mcq_options (title, question_id, unlock_section_id)
SELECT 'Backend Development', q.id, NULL
FROM questions q
WHERE q.title = 'Which workshop track do you prefer?';

INSERT INTO mcq_options (title, question_id, unlock_section_id)
SELECT 'Frontend Development', q.id, NULL
FROM questions q
WHERE q.title = 'Which workshop track do you prefer?';

INSERT INTO mcq_options (title, question_id, unlock_section_id)
SELECT 'Systems & Networking', q.id, NULL
FROM questions q
WHERE q.title = 'Which workshop track do you prefer?';

-- PUBLISHED_FORMS
INSERT INTO published_forms (title, description, author_id, deadline)
SELECT
    'Developer Experience Survey',
    'A published developer experience survey.',
    id,
    NOW() + INTERVAL '14 days'
FROM users
WHERE gmail = 'alice@example.com';

INSERT INTO published_forms (title, description, author_id, deadline)
SELECT
    'Workshop Registration',
    'Registration for the technical workshop.',
    id,
    NOW() + INTERVAL '7 days'
FROM users
WHERE gmail = 'bob@example.com';

-- VIEW_PERMISSIONS
INSERT INTO view_permissions (user_id, form_id)
SELECT u.id, p.id
FROM users u
CROSS JOIN published_forms p
WHERE u.gmail = 'charlie@example.com'
  AND p.title = 'Developer Experience Survey';

INSERT INTO view_permissions (user_id, form_id)
SELECT u.id, p.id
FROM users u
CROSS JOIN published_forms p
WHERE u.gmail = 'alice@example.com'
  AND p.title = 'Workshop Registration';

-- RESPONSES
INSERT INTO responses (user_id, form_id)
SELECT u.id, p.id
FROM users u
CROSS JOIN published_forms p
WHERE u.gmail = 'charlie@example.com'
  AND p.title = 'Developer Experience Survey';

-- ANSWERS
-- These are sample answers against the draft question definitions.
-- In the real publish flow, questions would belong to the published form's
-- cloned structure before responses are accepted.
INSERT INTO answers (response_id, question_id, payload)
SELECT
    r.id,
    q.id,
    '{"value": "Charlie Kumar"}'::jsonb
FROM responses r
JOIN published_forms p ON p.id = r.form_id
JOIN questions q ON q.title = 'What is your name?'
WHERE p.title = 'Developer Experience Survey'
  AND r.user_id = (SELECT id FROM users WHERE gmail = 'charlie@example.com');

INSERT INTO answers (response_id, question_id, payload)
SELECT
    r.id,
    q.id,
    '{"value": "Go"}'::jsonb
FROM responses r
JOIN published_forms p ON p.id = r.form_id
JOIN questions q ON q.title = 'What is your primary programming language?'
WHERE p.title = 'Developer Experience Survey'
  AND r.user_id = (SELECT id FROM users WHERE gmail = 'charlie@example.com');

COMMIT;