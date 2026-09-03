# CCS Forms API

This document describes the HTTP API currently implemented by the backend.

## Server

The local server runs at:

```text
http://0.0.0.0:8080
```

The listen address is `0.0.0.0`; the port is read from `PORT` and defaults to
`8080` when it is not set. For example:

```env
PORT=8080
```

When the backend is exposed through a tunnel such as ngrok, the base URL is the
tunnel's HTTPS URL instead, for example `https://<subdomain>.ngrok-free.app`.

## CORS

Browsers may call the API only from an origin on the allowlist. Set
`CORS_ORIGINS` to a comma-separated list of exact origins (scheme, host and
port), for example:

```text
CORS_ORIGINS=http://localhost:5173,https://ccs-forms.vercel.app
```

If `CORS_ORIGINS` is unset and `APP_ENV` is `dev`, `http://localhost:3000` and
`http://localhost:5173` (and their `127.0.0.1` forms) are allowed. Credentials
are not allowed: the API authenticates with bearer tokens, not cookies.

There is no API version prefix. All IDs are positive integers. Dates and times
are ISO-8601/RFC3339 values, for example `2026-09-02T18:30:00Z`.

## Authentication

Protected routes require:

```http
Authorization: Bearer <jwt>
```

JWTs are returned by signup, email login, and Google OAuth. They expire after
24 hours. The server signs and verifies them with HS256 and requires the
`JWT_SECRET` environment variable to contain at least 32 characters.

The current email signup/login flow is passwordless and is suitable only for
the prototype. Google OAuth is the identity-verified authentication flow.

Common authentication errors:

| Status | Meaning |
|---|---|
| 400 | Malformed request or invalid form ID |
| 401 | Missing, malformed, expired, or invalid JWT |
| 403 | Authenticated but not allowed to perform the operation |
| 404 | Resource does not exist or is not visible to the caller |
| 500 | Server/configuration/database failure |

Error responses use this shape:

```json
{
  "error": "Human-readable error message"
}
```

## Authentication endpoints

### `POST /auth/signup`

Creates a user and returns a JWT.

Request:

```json
{
  "email": "person@example.com",
  "name": "Person Name"
}
```

Response: `201 Created`

```json
{
  "token": "<jwt>"
}
```

The email and name must not be empty. Email is trimmed and lowercased. A
duplicate email returns `409 Conflict`.

### `POST /auth/login`

Looks up an existing user by email and returns a JWT.

Request:

```json
{
  "email": "person@example.com"
}
```

Response: `202 Accepted`

```json
{
  "token": "<jwt>"
}
```

The email is trimmed and lowercased. An unknown email returns `404 Not Found`.

### `GET /auth/google/login`

Starts Google OAuth. The frontend should navigate the browser to this URL; it
is a redirect endpoint, not an AJAX endpoint. The backend sets a short-lived,
HttpOnly, SameSite=Lax state cookie and redirects to Google.

Google OAuth requires:

```text
OAUTH_CLIENT_ID
OAUTH_CLIENT_SECRET
OAUTH_REDIRECT_URL
FRONTEND_URL             # optional, see the callback below
FRONTEND_CALLBACK_PATH   # optional, defaults to /auth/callback
```

The redirect URL must exactly match the URL registered with Google. If it is
not configured, the development fallback is:

```text
http://0.0.0.0:8080/auth/google/callback
```

### `GET /auth/google/callback`

Google redirects here after authorization. The backend validates the OAuth
state, exchanges the authorization code, requires a verified Google email,
creates or updates the local user, and then either redirects or returns JSON.

If `FRONTEND_URL` is set, the browser is redirected to
`<FRONTEND_URL><FRONTEND_CALLBACK_PATH>` with the result in the URL fragment:

```text
https://app.example.com/auth/callback#token=<jwt>&email=person%40example.com
```

The fragment is never sent to a server, so the token stays out of access logs
and `Referer` headers. The frontend page at that path should read
`window.location.hash`, store the token, and clear the hash.

If `FRONTEND_URL` is not set, the endpoint responds with JSON instead:

```json
{
  "token": "<jwt>",
  "email": "person@example.com"
}
```

Either way, the frontend should store the token according to its security
policy and send it as a Bearer token on protected API calls.

## Draft forms

All draft endpoints require authentication and operate only on the current
user's drafts.

### `POST /draft_forms/`

Creates draft metadata.

Request:

```json
{
  "title": "Customer survey",
  "description": "Tell us what you think"
}
```

`title` is required. `description` may be omitted, but clients should send a
string because some existing draft-loading paths expect a non-null value.

Response: `201 Created`

```json
{
  "form": {
    "id": 10,
    "title": "Customer survey",
    "description": "Tell us what you think"
  }
}
```

### `GET /draft_forms/`

Returns the current user's drafts, newest first.

Response: `200 OK`

```json
{
  "forms": [
    {
      "id": 10,
      "title": "Customer survey",
      "description": "Tell us what you think"
    }
  ]
}
```

### `GET /draft_forms/:id`

Returns the complete draft structure. The current implementation serializes
the top-level draft fields as `ID`, `Title`, `Description`, and `Sections`.
Nested section/question fields use lowercase JSON names.

Example shape:

```json
{
  "form": {
    "ID": 10,
    "Title": "Customer survey",
    "Description": "Tell us what you think",
    "Sections": [
      {
        "id": 20,
        "title": "About you",
        "description": "Basic information",
        "position": 0,
        "questions": [
          {
            "id": 30,
            "title": "Your name",
            "type": "text",
            "validation": {"required": true},
            "position": 0,
            "section_id": 20,
            "options": []
          }
        ]
      }
    ]
  }
}
```

Question types are `text`, `mcq`, and `checkbox`. Option objects contain
`id`, `title`, and `question_id`.

### `PATCH /draft_forms/:id`

Updates draft metadata. At least one property is required. Omitted properties
are unchanged.

```json
{
  "title": "Updated title",
  "description": "Updated description"
}
```

Response: `200 OK` with `{ "form": { ... } }`.

### `PUT /draft_forms/:id`

Reconciles sections, questions, and options. Send the complete structure being
edited. Each object must include a `status`:

| Status | Meaning |
|---|---|
| `ADD` | Insert a new object; send `id: 0` or omit the meaningful value |
| `CHANGE` | Update an existing object; send its ID |
| `DELETE` | Delete an existing object; send its ID |
| `UNCHANGED` | Leave the existing object unchanged |

Example:

```json
{
  "sections": [
    {
      "id": 0,
      "title": "Preferences",
      "description": "Choose your preferences",
      "position": 0,
      "status": "ADD",
      "questions": [
        {
          "id": 0,
          "title": "Choose a color",
          "type": "mcq",
          "validation": {"required": true},
          "position": 0,
          "status": "ADD",
          "options": [
            {"id": 0, "title": "Red", "status": "ADD"},
            {"id": 0, "title": "Blue", "status": "ADD"}
          ]
        }
      ]
    }
  ]
}
```

Response: `200 OK`

```json
{
  "message": "Saved Properly"
}
```

After an `ADD`, fetch the draft again to obtain server-generated IDs.
Published forms cannot be edited through this endpoint.

### `DELETE /draft_forms/:id`

Deletes the current user's draft. Response: `204 No Content`.

### `POST /draft_forms/:id/publish`

Publishes the current draft with a response deadline.

Request:

```json
{
  "deadline": "2026-12-31T23:59:59Z"
}
```

The deadline is required. The published form keeps the same numeric ID as the
draft. Response: `200 OK`.

## Published forms

### `GET /published_forms/`

Returns published forms owned by the current user. The current implementation
returns published-form fields using Go's default capitalization:

```json
{
  "forms": [
    {
      "ID": 10,
      "Title": "Customer survey",
      "Description": "Tell us what you think",
      "AuthorID": 1,
      "Deadline": "2026-12-31T23:59:59Z",
      "Structure": { "ID": 10, "Sections": [] }
    }
  ]
}
```

For respondent-facing form data, use `GET /form/:id`, whose structure uses
lowercase field names.

### `PATCH /published_forms/:id`

Updates published metadata. At least one field is required; omitted fields are
unchanged.

```json
{
  "title": "New title",
  "description": "New description",
  "deadline": "2026-12-31T23:59:59Z"
}
```

Response: `200 OK` with `{ "form": { ... } }`.

### `DELETE /published_forms/:id`

Unpublishes the form by deleting its published record. Response: `204 No
Content`.

## Respondent form and submissions

These routes require authentication. Fetching a form requires the caller to
be the author or have view permission. Submitting a response currently only
requires an authenticated user and a valid published form ID; the submission
handler does not currently enforce view permission.

### `GET /form/:id`

Returns the published form snapshot:

```json
{
  "form": {
    "id": 10,
    "title": "Customer survey",
    "description": "Tell us what you think",
    "deadline": "2026-12-31T23:59:59Z",
    "closed": false,
    "sections": [
      {
        "id": 20,
        "title": "Preferences",
        "description": "Choose your preferences",
        "questions": [
          {
            "id": 30,
            "title": "Choose a color",
            "type": "mcq",
            "validation": {"required": true},
            "options": [
              {"id": 40, "title": "Red"}
            ]
          }
        ]
      }
    ]
  }
}
```

The frontend should use the IDs returned in this response. Do not generate or
modify question or option IDs.

### `POST /form/:id/response`

Submits one response. A user may submit only once per form.

Request:

```json
{
  "answers": {
    "31": "Free-form text",
    "30": [40],
    "32": [50, 51]
  }
}
```

The object keys are question IDs represented as strings.

Answer formats:

| Question type | JSON value |
|---|---|
| `text` | String, for example `"Alice"` |
| `mcq` | Array containing exactly one option ID, for example `[40]` |
| `checkbox` | Array of zero or more option IDs, for example `[50, 51]` |

Required questions must be present. Option IDs must belong to their question.
The response also enforces configured text validation such as required, min,
max, and regex rules.

Response: `201 Created`

```json
{
  "response": {
    "id": 100,
    "form_id": 10,
    "timestamp": "2026-09-02T12:00:00Z",
    "structure": {
      "30": [40],
      "31": "Free-form text",
      "32": [50, 51]
    }
  }
}
```

The `structure` field is the submitted answer snapshot and is the source used
by CSV export.

## Responses

### `GET /published_forms/:id/responses`

Returns response summaries for a published form. The current handler requires
authentication but does not independently enforce author/view permission, so
frontend code should not treat a successful response as proof that the caller
is authorized to view every form's responses.

```json
{
  "responses": [
    {
      "id": 100,
      "user_id": 2,
      "form_id": 10,
      "timestamp": "2026-09-02T12:00:00Z"
    }
  ]
}
```

### `GET /responses/:id`

Returns a response and its normalized answer rows when the caller owns the
published form or has view permission.

```json
{
  "response": {
    "id": 100,
    "user_id": 2,
    "form_id": 10,
    "timestamp": "2026-09-02T12:00:00Z",
    "structure": {
      "30": [40]
    }
  },
  "answers": [
    {
      "id": 200,
      "question_id": 30,
      "payload": [40]
    }
  ]
}
```

### `DELETE /responses/:id`

Deletes a response when the caller is the respondent who submitted it.
Response: `204 No Content`.

## Analytics and CSV

Both endpoints require authentication and are intended for the published form
author.

### `GET /published_forms/:id/analytics`

Returns counts for MCQ and checkbox questions:

```json
{
  "form_id": 10,
  "title": "Customer survey",
  "total_responses": 12,
  "questions": [
    {
      "question_id": 30,
      "title": "Choose a color",
      "type": "mcq",
      "options": [
        {"option_id": 40, "label": "Red", "count": 8},
        {"option_id": 41, "label": "Blue", "count": 4}
      ]
    }
  ]
}
```

Text questions are currently not included in this response.

### `GET /published_forms/:id/export`

Downloads a UTF-8 CSV file with a BOM for spreadsheet compatibility. Columns
are:

```text
Response ID, Submitted at, Respondent, <question 1>, <question 2>, ...
```

Text values are exported directly. MCQ and checkbox option IDs are converted
to option labels; multiple checkbox labels are separated with `; `.

The exporter reads values from the response `structure` snapshot, so it is not
dependent on mutable normalized answer rows.

## Admin endpoints

These endpoints require authentication and the caller must currently have the
`admin` role.

### `PATCH /admin/users/admin`

Promotes another user by email.

Request:

```json
{
  "gmail": "person@example.com"
}
```

Response: `200 OK`

```json
{
  "user_id": 2,
  "gmail": "person@example.com",
  "role": "admin"
}
```

### `POST /admin/forms/:id/view-permissions`

Grants a user permission to view a published form. The admin must also be the
author of that form.

Request:

```json
{
  "gmail": "viewer@example.com"
}
```

Response: `201 Created`

```json
{
  "permission": {
    "user_id": 2,
    "form_id": 10
  }
}
```

## Frontend integration notes

- Always use IDs returned by the API for forms, sections, questions, and
  options.
- Send MCQ and checkbox answers as arrays of numeric option IDs.
- Use the published form from `/form/:id` when rendering a form for a
  respondent.
- Treat `204 No Content` responses as successful responses with no JSON body.
- Handle `401` by refreshing/re-authenticating and retrying only when safe.
- Handle `409` for duplicate signup or duplicate form submission.
- Do not expose JWTs in URLs or log them in browser consoles.
- Google OAuth login must be started as a browser navigation so its redirect
  and state cookie work correctly.
