# Frontend integration over ngrok

The backend and its Postgres database run on **one machine** (the backend
developer's). ngrok publishes that local server at a public HTTPS URL so a
frontend developer on a different machine can call it. Nothing is deployed —
while the tunnel is down, the API is unreachable.

```text
teammate's browser  ──►  https://opponent-malt-cartload.ngrok-free.dev  ──►  localhost:8080  ──►  local Postgres
   (localhost:5174)              ngrok edge                     your machine
```

Read [DOCS.md](DOCS.md) for the endpoint reference. This file is only about
wiring the two sides together.

---

## Part 1 — Backend machine, one-time setup

### 1. Claim a static ngrok domain

A random ngrok URL changes on every restart, which would mean re-editing
Google's OAuth settings and your teammate's config every single day. The free
tier includes one permanent domain.

Go to <https://dashboard.ngrok.com/domains>, create a domain, and note it.
This project's domain is `opponent-malt-cartload.ngrok-free.dev` — the rest of
this file uses it literally, so every command below can be copied as-is.

### 2. Register the callback with Google

In <https://console.cloud.google.com/apis/credentials>, open your OAuth 2.0
Client ID and add to **Authorized redirect URIs**:

```text
https://opponent-malt-cartload.ngrok-free.dev/auth/google/callback
```

Paste that exact string — no angle brackets, no trailing slash, `https` not
`http`, and leave **Authorized JavaScript origins** empty (this is a
server-side code flow; no browser JavaScript ever contacts Google).

It must match `OAUTH_REDIRECT_URL` byte for byte, or Google rejects sign-in
with `redirect_uri_mismatch`. Keep `http://localhost:8080/auth/google/callback`
in the list as a second entry so local testing works without editing Google.

### 3. Point `.env` at the tunnel

Ask your teammate which port their dev server uses (this project uses `5174`), then edit `.env`:

```bash
OAUTH_REDIRECT_URL=https://opponent-malt-cartload.ngrok-free.dev/auth/google/callback
FRONTEND_URL=http://localhost:5174
CORS_ORIGINS=http://localhost:5174,http://localhost:5173
```

`FRONTEND_URL` and `CORS_ORIGINS` stay on `localhost` even though the frontend
is on another machine. CORS and the OAuth redirect are both resolved by *their*
browser, so `localhost` there means their own dev server.

`FRONTEND_URL` holds a single value. If you also run a frontend locally on a
different port, OAuth will redirect to whichever port is configured here.

---

## Part 2 — Backend machine, every session

Three things must be running, in this order:

```bash
# 1. Postgres (whatever you normally use, e.g.)
brew services start postgresql@16

# 2. The API — from the module root, so godotenv finds ./.env
cd "~/Desktop/ccs forms/ccs-forms" && go run .

# 3. The tunnel, in a second terminal
ngrok http 8080 --url=https://opponent-malt-cartload.ngrok-free.dev
```

Confirm the whole path works before telling anyone it is up:

```bash
curl https://opponent-malt-cartload.ngrok-free.dev/
# {"message":"Running Server"}
```

Your machine must stay awake and online for the duration. When you close the
laptop, your teammate's frontend stops working — agree on hours, or move the
backend to a real host once the shape of the API settles.

---

## Part 3 — Hand these three things to the frontend developer

1. The base URL: `https://opponent-malt-cartload.ngrok-free.dev`
2. [DOCS.md](DOCS.md) — the full endpoint reference
3. The port they must run their dev server on (the one in `CORS_ORIGINS`)

### Their `.env`

```bash
VITE_API_URL=https://opponent-malt-cartload.ngrok-free.dev
```

### Their API client

Every request needs the `ngrok-skip-browser-warning` header. Without it,
ngrok's free tier returns an HTML interstitial page instead of JSON, and
`response.json()` fails with a confusing parse error.

```js
const BASE = import.meta.env.VITE_API_URL;

export async function api(path, options = {}) {
  const token = localStorage.getItem("token");

  const res = await fetch(BASE + path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      "ngrok-skip-browser-warning": "true",
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

### Their sign-in button

`/auth/google/login` is a redirect endpoint, not an AJAX one. It must be a
full-page navigation — `fetch` will fail on it.

```js
<button onClick={() => { window.location.href = `${BASE}/auth/google/login`; }}>
  Sign in with Google
</button>
```

### Their callback page

Route this at `/auth/callback`, matching `FRONTEND_CALLBACK_PATH`. The backend
returns the token in the URL **fragment**, which browsers never transmit to a
server — so it stays out of ngrok's logs and any `Referer` header.

```js
useEffect(() => {
  const params = new URLSearchParams(window.location.hash.slice(1));
  const token = params.get("token");

  if (!token) return navigate("/login?error=signin_failed");

  localStorage.setItem("token", token);
  localStorage.setItem("email", params.get("email") ?? "");
  // Clear the token out of the address bar and history.
  window.history.replaceState({}, "", "/auth/callback");
  navigate("/dashboard");
}, []);
```

### First run

The very first time they open the ngrok URL, ngrok's free tier shows a browser
warning page. They click **Visit Site** once per browser session. This affects
top-level navigation only — the header above handles it for `fetch` calls.

---

## Suggested integration order

Build against one endpoint at a time and confirm each before moving on.

| Step | Endpoint | Proves |
|---|---|---|
| 1 | `GET /` | Tunnel is up, CORS is right |
| 2 | `POST /auth/signup` | Requests reach the database |
| 3 | `GET /auth/google/login` | Full OAuth round trip, token stored |
| 4 | `GET /draft_forms/` | Bearer token is accepted |
| 5 | `POST /draft_forms/` | Writes work |
| 6 | `GET`/`PUT /draft_forms/:id` | The form editor, the real work |

On step 6, note the contract in [ROUTES.md](ROUTES.md): `GET` returns the whole
form as JSON, the client holds it in memory, and `PUT` sends the complete
structure back. The client must never invent or alter `id` fields.

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `Failed to fetch`, CORS error in console | Their origin is not allowlisted | Add their exact `http://localhost:PORT` to `CORS_ORIGINS`, restart the server |
| `Unexpected token '<'` from `res.json()` | ngrok interstitial returned HTML | Add the `ngrok-skip-browser-warning` header |
| `redirect_uri_mismatch` from Google | `OAUTH_REDIRECT_URL` differs from the console entry | Make them identical, including scheme and trailing slash |
| `Invalid or expired sign-in state` | Flow crossed origins, callback opened directly, or a retry after a failed callback | Start over at `/auth/google/login` on the ngrok domain — never reload the callback page (see below) |
| Lands on JSON showing a raw token | `FRONTEND_URL` is unset | Set it in `.env` and restart |
| `Authentication is not configured` (500) | `JWT_SECRET` under 32 characters | Lengthen it — every existing token is invalidated |
| `warning: could not load .env` at startup | `go run .` was run from the wrong directory | Run it from the module root, next to `main.go` |
| `ERR_NGROK_*`, tunnel refuses | Domain typo, or a tunnel is already running | Check the domain string; only one agent session at a time on the free tier |

### Verifying the OAuth config without a browser

Whatever Google says, this prints what the server is actually sending. Check the
client ID against the one in Google Console before assuming anything else:

```bash
curl -s -D - -o /dev/null http://localhost:8080/auth/google/login \
  | grep -i '^location:' | python3 -c "
import sys, urllib.parse
q = urllib.parse.parse_qs(urllib.parse.urlparse(sys.stdin.read().strip().split(' ',1)[1]).query)
print('client_id   =', q['client_id'][0])
print('redirect_uri=', q['redirect_uri'][0])
"
```

The client ID and secret are read once at startup, so **restart the server**
after editing them. The redirect URI is read per request.

### Two OAuth traps that look like backend bugs

**`redirect_uri_mismatch` when the URI is obviously correct.** Check the
*project*. A Google account can have several Cloud projects, each with its own
clients, and the Console shows whichever you last opened. If `OAUTH_CLIENT_ID`
in `.env` belongs to a different project than the one you are editing, every
redirect URI you add lands on a client the server never uses. Compare the
client ID from the command above against the one on the Console page.

**The sign-in flow must stay on one origin.** The state cookie is set on the
host that served `/auth/google/login`, and Google returns to whatever host is
in `OAUTH_REDIRECT_URL`. Start the flow at `http://localhost:8080` while the
callback points at the ngrok domain and the cookie is invisible to the
callback, which fails as `Invalid or expired sign-in state`.

The callback also deletes the state cookie *before* validating it, so a state
can never be replayed. A reload or back-button on the callback page therefore
always fails, even when the original attempt was fine. Retry by revisiting
`/auth/google/login`, never by reloading.
