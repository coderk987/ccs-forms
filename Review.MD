# Codebase Review

Review scope: Go source, routers, controllers, middleware, database setup/sample SQL, and application startup. `go test ./...` compiles successfully. `go vet ./...` fails on `db/pg.go:21` because the result of `fmt.Errorf` is discarded.

## Critical Findings

### `controllers/responses/deleteResponse.go`

- **Critical, line 19:** The delete query filters on `author_id`, but the `responses` table has no `author_id` column. Every delete request reaches PostgreSQL with an undefined-column error and returns HTTP 500. The ownership predicate should use the response respondent (`user_id`) or join `published_forms` when form-author deletion is intended.

### `controllers/forms/updatePublishedForm.go`

- **Critical, lines 41-46:** The handler is named and routed for published forms but executes `UPDATE draft_forms` while passing a `deadline` column that does not exist in `draft_forms`. The query fails for every valid request, and scanning `RETURNING *` into `PublishedForm` would also not match the draft table.

### `controllers/forms/editDraftForm.go`

- **Critical, lines 76-199:** The handler performs a sequence of independent database writes without a transaction. A later failure leaves the draft partially modified even though the endpoint returns an error.
- **High, lines 82-123:** A newly added section has no generated ID assigned back to the in-memory request. Questions under an `ADD` section are inserted using the client-provided section ID, normally zero, so adding a section with questions fails its foreign-key constraint.
- **High, lines 128-132:** A changed question is updated by question ID only. A user who knows another form's question ID can change that question despite the earlier ownership check applying only to the draft itself.
- **High, lines 177-185:** Option updates test `question.Status == "CHANGE"` instead of `option.Status == "CHANGE"`. Changed options under an unchanged question are ignored; options under a changed question can be updated regardless of their own status.
- **Medium, lines 76-110 and 112-197:** Unknown section, question, or option status values are silently accepted. The request can report success while requested changes were not applied.

### `db/setupDb.go`

- **Critical, lines 20-39:** Startup always executes `sample_data.sql` after `setup.sql`. The sample inserts are not idempotent, so the second application start fails on duplicate users and may fail on other unique constraints. The startup path then continues serving after setup errors.
- **High, lines 20-24:** `CREATE TABLE IF NOT EXISTS` does not migrate existing tables. Existing databases will not receive the newly added `answers.structure` column or the `(user_id, form_id)` uniqueness constraint, causing response inserts or duplicate protection to fail after an upgrade.

### `middleware/auth.go`

- **High, lines 38-43:** JWT parsing does not restrict the signing algorithm. The key function accepts any algorithm presented by the token; algorithm confusion or an unexpected signing method can therefore bypass the intended HS256-only policy.
- **High, lines 40-41:** An empty `JWT_SECRET` is accepted as the signing key. In a misconfigured deployment, the application can issue and accept tokens signed with an empty secret.

## High Findings

### `controllers/responses/getPublishedFormResponses.go`

- **High, lines 23-29:** The endpoint only has authentication middleware and does not verify that the caller is the published form author or has a view permission. Any authenticated user can enumerate responses for any form ID.

### `controllers/responder/createResponse.go`

- **High, lines 281-310:** Duplicate checkbox option IDs are not rejected during validation. The second insert violates the composite primary key and returns HTTP 500 instead of a client validation error.
- **High, lines 284-287:** Answers reference mutable rows in `questions`, while the form is validated from the published JSON snapshot. If a draft question is deleted after publication, a valid snapshot answer fails the foreign-key constraint; if a question is changed, response persistence still points to the mutable draft row rather than an immutable published question.
- **Medium, lines 136-139:** The `answers.structure` value stores a per-answer `{question_id,payload}` object, not the request JSON as described in `UPDATE.md`. Confirm the intended audit structure before treating this as complete.

### `controllers/forms/draftFormsHelper.go`

- **High, lines 63-65 and 92-96:** `sections.description` is nullable in the schema but is scanned into a non-nullable `string`. Drafts with a null section description fail to load or publish.
- **High, lines 44-61:** The same nullable mismatch exists for `draft_forms.description`, which is scanned into `DraftFormResponse.Description string`.
- **Medium, lines 89-91, 106-108, and 126-128:** Queries do not specify ordering. The generated form can return sections, questions, and options in nondeterministic order despite each table having a position column.

### `controllers/forms/getPublishedForms.go`

- **High, lines 14-20 and 26-36:** `published_forms.description` and `published_forms.deadline` are nullable, but the response struct uses `string` and `time.Time`. A published form with either value NULL causes row scanning to fail and produces HTTP 500.
- **Low, lines 31 and 38:** Error messages incorrectly say “draft forms” for a published-form endpoint, making failures misleading to clients and operators.

### `controllers/forms/publishDraftForm.go`

- **High, lines 35-66:** Publishing is not transactional. The form is assembled through multiple reads and then inserted separately; concurrent edits can produce a published snapshot that does not correspond to one consistent draft state.
- **Medium, lines 62-66:** The draft remains in `draft_forms` after publication. If the intended state is draft-to-published conversion, there is no state transition or protection against editing the source draft after publication.
- **Medium, lines 68-71:** The database error is returned directly in the JSON response, which can expose SQL details.

### `controllers/admin/admin.go`

- **Medium, lines 18-35:** Admin status is checked for the caller, but the JWT identity is not revalidated anywhere in the middleware against a live user record. Deleted users with an unexpired token can still reach non-admin handlers; admin handlers happen to fail only when their database check runs.
- **Medium, lines 115-120:** Existing permissions are implemented as an update/upsert but the handler always returns HTTP 201. Repeated grants should normally return 200 or 204, or the API should document the idempotent 201 behavior.

## Medium Findings

### `controllers/auth/loginUser.go`

- **Medium, lines 25-29:** The email is not trimmed or validated, unlike signup. Inputs containing surrounding whitespace cannot log in even when the Gmail is valid.
- **Medium, lines 34-36:** `fmt.Println("Running Login")` writes request activity to stdout and should not remain as application logging in production.
- **Medium, lines 40-48:** Errors are returned directly to clients through `err.Error()`, which can expose database details.

### `controllers/auth/signupUser.go`

- **Medium, lines 29-31:** Email syntax and Gmail-domain requirements are not validated. Any non-empty string accepted by the database becomes a Gmail identity.
- **Low, lines 47-50:** Database errors other than duplicate email are exposed directly through `err.Error()`.

### `controllers/forms/createDraftForm.go`

- **Medium, lines 10-12 and 18-22:** `binding:"required"` on a string rejects a missing or empty title, but whitespace-only titles are accepted. The same applies to descriptions where meaningful content is expected.

### `controllers/forms/getDraftForm.go`

- **Medium, lines 9-19:** The endpoint returns the serialized form as `json.RawMessage` inside `gin.H`. This currently relies on Gin's raw-message handling and is less explicit than decoding into a response type; malformed stored JSON is not surfaced distinctly.

### `controllers/responder/getResponderForm.go`

- **Medium, lines 33-44:** Access is checked against author/view permission, but the endpoint does not expose or calculate closed/deadline state even though response creation enforces the deadline. Clients can receive a form that cannot accept responses without a consistent status signal.

### `controllers/responses/getResponse.go`

- **Medium, lines 17-22 and 64-70:** The endpoint returns `answers` but does not query `checkbox_answers`. The response is incomplete if the relational checkbox selections are considered authoritative rather than the JSON payload.

### `controllers/responses/helpers.go`

- **Low, lines 10-16 and 27-33:** Response ID and published-form ID parsing is duplicated across controller packages, increasing the chance that validation and error behavior diverge.

### `routes/draftForms.go`

- **Medium, lines 24-27:** `GET /published_forms/:id/responses` is mounted under the general authenticated group without an authorization handler of its own. The controller currently confirms the exposure problem noted above.

### `routes/admin.go`

- **Low, lines 14-17:** The endpoint paths and payload contract are not documented in `ROUTES.md`, so the new admin API is not discoverable alongside the rest of the route surface.

### `main.go`

- **High, lines 13-18:** Database setup errors are printed and ignored. The server starts even when schema/sample setup fails, and later handlers can panic or return cascading database errors.
- **Medium, line 33:** The return value from `router.Run` is ignored, so listener startup and shutdown errors are not reported.

## Low Findings

### `db/pg.go`

- **Medium, line 21:** `fmt.Errorf` is called and discarded. This is the `go vet` failure and means development-mode dotenv errors are not logged or returned.
- **High, lines 26-34:** `pgxpool.New` creates a pool configuration but does not prove that the database is reachable. The code logs “Connected to Postgres” without a ping and defers the actual failure to later queries.

### `db/setup.sql`

- **Medium, lines 18-24:** `sections.form_id` has no foreign key to `draft_forms(id)`, allowing orphaned sections and making ownership/data-integrity checks unreliable.
- **Medium, lines 49-55:** Published forms snapshot JSON but retain references to mutable question IDs in the rest of the schema. This is the source of the published-response integrity issue described above.

### `db/sample_data.sql`

- **Medium, lines 4-7 and throughout:** The seed script is not idempotent despite being run on every startup.
- **Medium, lines 191-225 and 263-295:** The published JSON omits `unlock_section_id` for MCQ options even though the database stores it and the responder model supports it. Conditional section behavior is lost from the published snapshot.
- **Low, lines 330-370:** Seed responses insert only two answers for a form containing required checkbox and MCQ questions, so the sample data does not represent a valid submitted response under the response controller's validation rules.

## Files With No Material Finding In This Pass

`routes/auth.go`, `routes/responder.go`, `routes/responses.go`, `controllers/forms/getAllDraftForms.go`, `controllers/forms/deleteDraftForm.go`, `controllers/forms/unpublishForm.go`, `controllers/responder/responder.go`, `controllers/responses/helpers.go` beyond the duplication noted above, and `config/config.go` had no additional material issue beyond findings already grouped above.

## Verification

- `go test ./...`: passed.
- `go vet ./...`: failed because `db/pg.go:21` discards the result of `fmt.Errorf`.
- No live PostgreSQL integration test was available, so SQL execution findings were verified against `db/setup.sql` statically.
