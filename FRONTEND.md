# Frontend setup

The backend is live at **`https://forms-backend.ccstiet.com`**. Nothing to
install, no tunnel, no local backend — just run your dev server and call it.

**Run your dev server on port 5174.** The backend allowlists that exact origin
for CORS and redirects there after Google sign-in. Another port will fail.

```bash
npm run dev -- --port 5174
```

## API client

```js
const BASE = "https://forms-backend.ccstiet.com";

export async function api(path, options = {}) {
  const token = localStorage.getItem("token");

  const res = await fetch(BASE + path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token && { Authorization: `Bearer ${token}` }),
      ...options.headers,
    },
  });

  if (res.status === 401) {
    localStorage.removeItem("token");
    window.location.href = "/login";
    return;
  }

  const body = res.status === 204 ? null : await res.json();
  if (!res.ok) throw new Error(body?.error ?? `Request failed (${res.status})`);
  return body;
}
```

Every protected endpoint takes `Authorization: Bearer <jwt>`. Errors always
come back as `{"error": "..."}`.

## Sign in with Google

A full-page navigation, not a `fetch` — this endpoint redirects.

```js
<button onClick={() => { window.location.href = `${BASE}/auth/google/login`; }}>
  Sign in with Google
</button>
```

Your Google account must be added as a test user first. Ask the backend dev —
otherwise Google blocks you with "Access blocked".

## Callback route

Add a route at **`/auth/callback`**. The backend redirects there after sign-in
with the token in the URL *fragment* (fragments are never sent to a server, so
the token stays out of logs).

```js
useEffect(() => {
  const params = new URLSearchParams(window.location.hash.slice(1));
  const token = params.get("token");

  if (!token) return navigate("/login?error=signin_failed");

  localStorage.setItem("token", token);
  localStorage.setItem("email", params.get("email") ?? "");
  window.history.replaceState({}, "", "/auth/callback"); // clear the token from the URL
  navigate("/dashboard");
}, []);
```

If sign-in fails, retry by clicking the sign-in button again — never by
reloading the callback page. The backend consumes its one-time state on every
callback, so a reload always fails even when the original attempt was fine.

## Faster path while waiting on Google

There is a passwordless flow you can build against immediately, no Google
account or test-user approval needed. It returns an identical JWT, and every
protected endpoint behaves the same:

```js
// 201 Created -> { token }
await api("/auth/signup", {
  method: "POST",
  body: JSON.stringify({ email: "you@example.com", name: "Your Name" }),
});

// 202 Accepted -> { token }
await api("/auth/login", { method: "POST", body: JSON.stringify({ email: "you@example.com" }) });
```

## Suggested build order

| Step | Call | Proves |
|---|---|---|
| 1 | `GET /` | Backend reachable, CORS fine |
| 2 | `POST /auth/signup` | Requests hit the database |
| 3 | `GET /draft_forms/` | Bearer token accepted |
| 4 | `POST /draft_forms/` | Writes work |
| 5 | `GET`/`PUT /draft_forms/:id` | The form editor |
| 6 | `GET /auth/google/login` | Real sign-in, once you are a test user |

## The Save button

Save is **`PUT /draft_forms/:id`**. `GET` returns the whole form, you hold it in
memory while editing, and Save sends the complete structure back start to
finish. Never invent or change `id` fields — only content, insertions and
deletions.

Every section, question and option carries a `status`:

| Status | Meaning |
|---|---|
| `ADD` | New object — send `id: 0` |
| `CHANGE` | Existing object was edited — send its real id |
| `DELETE` | Remove it — send its real id |
| `UNCHANGED` | Leave alone |

`GET` does not return statuses, so you set them yourself as the user edits.

**Two rules that will cost you an afternoon otherwise:**

1. **Status cascades down.** A parent marked `UNCHANGED` is skipped entirely,
   children included. Rename one MCQ option and you must mark that option
   `CHANGE`, *and* its question `CHANGE`, *and* its section `CHANGE`. Otherwise
   the save silently does nothing.

2. **Re-fetch after saving anything new.** `ADD` objects get their real ids
   generated server-side, and the response does not return them — it is just
   `{"message": "Saved Properly"}`. Call `GET /draft_forms/:id` again after a
   save that contained any `ADD`, or your next save will send `id: 0` for
   objects that already exist.

Published forms reject this endpoint with `409 Conflict`. Unpublish first.

Full endpoint reference: **DOCS.md**

## Troubleshooting

| Symptom | Fix |
|---|---|
| CORS error in console | You are not on port 5174 |
| `Access blocked` from Google | Your account is not a test user yet |
| Lands on `/auth/callback` with no token | Check `window.location.hash`, not `search` |
| `401` on every call | Token missing or expired (they last 24h) — sign in again |
