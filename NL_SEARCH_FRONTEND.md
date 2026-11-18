# Natural-Language Search — Frontend Integration Guide

**Status: experimental.** A new search endpoint that answers a free-form question
about the catalogue instead of matching a typed keyword. This is **separate** from
the existing catalogue search and is not yet part of the product.

This document is the whole contract. `API.md` is unchanged and does not mention
this endpoint.

- **Base URL (dev):** `http://localhost:3000`
- **Endpoint:** `POST /api/v1/nl-search`
- **Content type:** `application/json`
- **Auth:** none required. A signed-in session cookie is used, if present, only
  to give the caller their own rate-limit budget.

### Server configuration

The endpoint needs a language model. Without one it reports itself as
unavailable and the rest of the API keeps working.

| Variable | Required | Notes |
|---|---|---|
| `GEMINI_API_KEYS` | no | The keys, comma separated. This one on its own is enough. |
| `GEMINI_API_KEY` | no | A single key. This one on its own is enough too. |
| `GEMINI_MODEL` | no | Defaults to `gemini-3.5-flash-lite`. |
| `ALLOWED_ORIGINS` | yes | The frontend's origin, so the browser accepts the reply. |

Either key variable on its own works — use whichever is easier:

```bash
GEMINI_API_KEYS=key1,key2,key3
```

```bash
GEMINI_API_KEY=key1
```

Setting both is allowed: the single key is tried first and the list follows.

**The extra keys must come from a different account.** The provider counts
the day's allowance per project, and every key belongs to a project, so a
second key from the same project shares the first one's quota and adds
nothing. A key from a different account has its own allowance, and the
endpoint falls over to it when the first is spent.

When a key is out of allowance the endpoint remembers that and skips it for
the rest of the day's window rather than paying a failed request on every
search.

---

## 1. Rules for this integration

Read these before writing any code. They are not suggestions.

### 1.1 Put it on its own hidden route

Create **one new route** for this feature and nothing else. Suggested path:

```
/nlsearch
```

**Do not surface it anywhere.** Specifically, do **not**:

- add it to the navbar, header, footer, or any nav/menu/sidebar component
- add a link to it from any existing page
- add it to the sitemap, the routing constants file, or any route-manifest
  that gets rendered into navigation
- add a search icon, a "smart search" entry, or a keyboard shortcut for it
- mention it in a feature-flag list, an announcement, or a changelog

The route exists so it can be reached by **typing the URL directly**. It is
deliberately not discoverable from the UI while it is being tested.

If the frontend uses a catch-all or a dynamic route table, make sure this path
is still reachable by direct navigation.

### 1.2 Do not touch the existing search

This feature is **entirely additive**. The following must be left exactly as
they are:

| Do not change | Why |
|---|---|
| `GET /api/v1/universities/search` | The live product search |
| `GET /api/v1/colleges/search` | The live product search |
| `GET /api/v1/scholarships/search` | The live product search |
| Any existing search page, component, hook, filter UI, or state store | The live product search |
| Any existing search API client or service function | Other pages depend on them |

Do not refactor the existing search into a shared base to "avoid duplication"
with the new page. Duplication is the correct choice here: the existing search
is in production and this endpoint is experimental. If you want a shared
search-input component, add a new one rather than changing the existing one.

**The only new code** is: one route, one page, one API client function for
`POST /api/v1/nl-search`, and the components that page needs.

### 1.3 Keep it self-contained

Everything this page needs should live under its own route folder. Don't add
NL-search state to a global store that other pages read, and don't add
NL-search types to a shared API types file that existing code imports — a
name collision there breaks the live site.

---

## 2. The endpoint

### Request

```http
POST /api/v1/nl-search
Content-Type: application/json
```

```json
{
  "query": "universities in Kathmandu Nepal that offer scholarships",
  "page": 1,
  "page_size": 20
}
```

| Field | Type | Required | Default | Notes |
|---|---|---|---|---|
| `query` | string | yes | — | The user's question, in their own words. Typos and broken English are expected and handled. Max **500 characters**, counted in characters (not bytes). Must not be blank. |
| `page` | int | no | `1` | 1-indexed. Values below 1 fall back to 1. |
| `page_size` | int | no | `20` | Max **50**. Values above 50 are capped. |

> `page_size` maxes at 50 here, not 100 like the catalogue endpoints. This is
> deliberate — each request costs a model call on the backend.

### Response

```json
{
  "success": true,
  "data": {
    "query": "universities in Kathmandu Nepal that offer scholarships",
    "interpretation": {
      "entity": "universities",
      "assumptions": "searched the universities catalogue; read \"great scholarship\" as merit scholarships worth a substantial amount",
      "unmapped": ["\"get into\""]
    },
    "items": [
      {
        "entity": "universities",
        "id": "c5f7d664-c16c-4294-af74-fe9aa080a5e4",
        "slug": "kathmandu-university",
        "title": "Kathmandu University",
        "logo": "https://...",
        "cover_image": "https://...",
        "country": "Nepal",
        "state": "Bagmati",
        "city": "Kathmandu",
        "full_location": "Kathmandu, Bagmati, Nepal",
        "institution_type": "Private",
        "campus_setting": "Urban",
        "tuition_min": 400000,
        "tuition_max": 900000,
        "acceptance_rate": 45.5,
        "employment_rate": 88.0,
        "founded_year": 1991,
        "is_popular": true,
        "is_featured": false
      }
    ],
    "meta": {
      "page": 1,
      "page_size": 20,
      "total": 12,
      "total_pages": 1
    }
  }
}
```

### The result item

One shape for all four catalogues, because a question can span them. The
`entity` field says what each row is.

| Field | Type | Notes |
|---|---|---|
| `entity` | string | `"universities"`, `"colleges"`, `"scholarships"` or `"programs"` |
| `id` | string | UUID |
| `slug` | string | Use this to link to the existing detail page. For a programme it is the programme's title, since a programme has no slug of its own. |
| `title` | string | The university's or college's name, the scholarship's title, or the programme's name |
| `logo` | string? | Absolute URL. May be absent. Scholarships have no `cover_image`. |
| `cover_image` | string? | Absolute URL. May be absent. |
| `country`, `state`, `city`, `full_location` | string? | May be absent |
| `institution_type`, `campus_setting` | string? | Universities and colleges. May be absent. |
| `tuition_min`, `tuition_max` | number? | Universities. May be absent. |
| `acceptance_rate`, `employment_rate` | number? | Universities. May be absent. |
| `founded_year` | int? | Universities and colleges. May be absent. |
| `award_amount` | string? | Scholarships. A ready-to-display amount, e.g. `"100,000 - 500,000"`. Free text — render it as-is. May be absent. |
| `award_min`, `award_max` | number? | Scholarships. The same range as numbers. May be absent. |
| `number_of_awards` | int? | Scholarships. May be absent. |
| `min_gpa` | number? | Scholarships. May be absent. |
| `provider_name`, `provider_type` | string? | Scholarships. May be absent. |
| `application_deadline` | string? | Scholarships — ISO 8601 timestamp, may be `null` |
| `description`, `excerpt`, `career_options` | string? | Programmes. May be absent. |
| `is_popular`, `is_featured` | bool | Always present. Always `false` for a programme. |

**Fields not in this list are not in the response.** In particular there is no
`overview`, no `seo_*` — the endpoint returns summary fields only. If your
card needs more, fetch the existing detail endpoint using `slug` for the row
the user clicks.

### The interpretation

`data.interpretation` explains how the question was understood. It is worth
showing — it's the difference between "the search worked" and "the search
understood me".

| Field | Type | Notes |
|---|---|---|
| `entity` | string | Which catalogue was searched. A question that spans more than one catalogue lists them all, separated by commas, e.g. `"universities, colleges"`. |
| `assumptions` | string? | Which catalogue was searched, plus any judgement the search made, e.g. how a vague word was mapped to a field. May be empty. |
| `unmapped` | string[]? | Parts of the question that matched no field. May be absent. |

Suggested treatment: show `assumptions` as a small note above the results
("Read as: …"), and show `unmapped` as a gentle hint ("We ignored: …"). Both
are optional and can be omitted entirely.

---

## 3. Errors

The standard envelope applies: `{ "success": false, "error": "..." }`, with
`errors: [{ "field", "message" }]` for a validation failure.

| Status | Meaning | What to do |
|---|---|---|
| `400` | Empty `query`, `query` over 500 characters, or the body is not valid JSON | Show a validation message on the input |
| `422` | The question could not be turned into a valid search | Show the error and suggest rephrasing |
| `429` | Too many searches | Show "slow down"; respect the `Retry-After` header |
| `502` | The model was reached but could not answer | Show a "try rephrasing" message |
| `503` | The feature is switched off on the server, the model's allowance for the day is spent, the model is busy, or the provider is rate limiting | Show "search is unavailable right now" |
| `500` | Anything else | Generic error |

**Handle `503` properly.** It means one of four things: the backend has no
language model configured, the model provider's daily allowance is spent, the
provider is saturated, or the provider is rate limiting. All four are
server-side states, not a bad question, and none of them is fixed by
rephrasing. The page should show a clear "not available" state rather than an
error toast, and it should not invite the user to rephrase: the question was
fine, the server simply could not ask the model.

A `503` may carry a `Retry-After` header saying how many seconds to wait.
Honour it if you show a retry affordance.

**Handle `429` properly.** The endpoint is rate limited per signed-in user
(or per IP when signed out): 10 requests per minute, with a burst of 5. The
`Retry-After` header says how long to wait in seconds.

---

## 4. A minimal example

```ts
// api/nlsearch.ts  — new file, nothing existing imports this
//
// Use the same base URL your existing API client uses. Do not change that
// shared config — read it, don't edit it.

const BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:3000";

export type NlSearchItem = {
  entity: "universities" | "colleges" | "scholarships" | "programs";
  id: string;
  slug: string;
  title: string;
  logo?: string;
  cover_image?: string;
  country?: string;
  state?: string;
  city?: string;
  full_location?: string;
  institution_type?: string;
  campus_setting?: string;
  tuition_min?: number;
  tuition_max?: number;
  acceptance_rate?: number;
  employment_rate?: number;
  founded_year?: number;
  award_amount?: string;
  award_min?: number;
  award_max?: number;
  number_of_awards?: number;
  min_gpa?: number;
  provider_name?: string;
  provider_type?: string;
  application_deadline?: string | null;
  description?: string;
  excerpt?: string;
  career_options?: string;
  is_popular: boolean;
  is_featured: boolean;
};

export type NlSearchResponse = {
  query: string;
  interpretation: {
    entity: string;
    assumptions?: string;
    unmapped?: string[];
  };
  items: NlSearchItem[];
  meta: { page: number; page_size: number; total: number; total_pages: number };
};

export async function nlSearch(
  query: string,
  page = 1,
  pageSize = 20,
  signal?: AbortSignal,
): Promise<NlSearchResponse> {
  const res = await fetch(`${BASE}/api/v1/nl-search`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, page, page_size: pageSize }),
    signal,
  });

  const body = await res.json();

  if (!res.ok) {
    const detail = body?.errors?.[0]?.message ?? body?.error ?? "Search failed";
    const err = new Error(detail) as Error & { status: number; retryAfter?: number };
    err.status = res.status;
    err.retryAfter = Number(res.headers.get("Retry-After")) || undefined;
    throw err;
  }

  return body.data as NlSearchResponse;
}
```

### Page component sketch

```tsx
// pages/NlSearch.tsx  — new file

export default function NlSearch() {
  const [query, setQuery] = useState("");
  const [data, setData] = useState<NlSearchResponse | null>(null);
  const [error, setError] = useState<{ status: number; message: string } | null>(null);
  const [loading, setLoading] = useState(false);

  async function run(q: string) {
    setLoading(true);
    setError(null);
    try {
      setData(await nlSearch(q));
    } catch (e: any) {
      setError({ status: e.status, message: e.message });
    } finally {
      setLoading(false);
    }
  }

  if (error?.status === 503) {
    return <p>Natural-language search isn’t available right now.</p>;
  }

  return (
    <div>
      <form onSubmit={(e) => { e.preventDefault(); run(query); }}>
        <input value={query} onChange={(e) => setQuery(e.target.value)}
               placeholder="Describe what you're looking for" />
        <button type="submit" disabled={loading || !query.trim()}>Search</button>
      </form>

      {error && error.status !== 503 && <p role="alert">{error.message}</p>}

      {data?.interpretation?.assumptions && (
        <p><small>Read as: {data.interpretation.assumptions}</small></p>
      )}

      <ul>
        {data?.items.map((item) => (
          <li key={`${item.entity}-${item.id}`}>
            {/* link with the existing detail route for that catalogue */}
            <a href={`/${item.entity}/${item.slug}`}>{item.title}</a>
          </li>
        ))}
      </ul>
    </div>
  );
}
```

Register the route, and only the route:

```tsx
// in your router — add the path, do NOT add it to any nav array
<Route path="/nlsearch" element={<NlSearch />} />
```

Adjust the link target to whatever the existing detail pages use. The point is
that the row links into the **existing** pages — this feature adds a way to
find rows, it does not add new detail pages.

---

## 5. Behaviour worth knowing

**Typo-tolerant.** The question is read semantically, so "kathamndu" and
"universty in kathmandu" both find Kathmandu University. Don't pre-normalise
or validate the user's spelling.

**The search may interpret.** "affordable", "good reputation" and "near the
city centre" are qualities, not fields. The backend maps them to the closest
field and threshold and tells you what it did in `interpretation.assumptions`.
That's the intended behaviour — show the assumption rather than hiding it.

A money quality — **affordable, cheap, budget, inexpensive, low cost** — is a
**ranking**, not a cut-off. It returns every university ordered by tuition with
the cheapest first. The tuition in the catalogue runs from 0 to 9,000,000 and is
in each row's own currency, so a fixed threshold would drop every university
whose fees sit above it. If the question states its own number ("universities
under 10000"), that number is used instead.

A quality with a natural line is a **filter** as well as a ranking: selective →
acceptance rate under 30; prestigious → under 20; old / established → founded
before 1950. Rows outside that line are not that thing at all.

**An ordering word on its own — "cheapest", "least expensive" — is not a
quality.** It returns the whole catalogue ordered by that field, with the
extreme first. "Cheapest universities" is a sort; "affordable universities" is
the same ordering under a friendlier name.

Only universities carry tuition. On a college or a programme, a money quality
has nothing to rank by and is left out of the answer.

**Part of the question may not be searchable.** Anything with no matching
field is listed in `interpretation.unmapped` and the rest of the question is
still searched.

**Only published rows come back.** Drafts are filtered out on the server and
can never appear, however the question is phrased.

**One catalogue per search, but related catalogues are searched too.** A
question about universities can filter on the scholarships those universities
offer, and the results are still university rows. A row's `entity` always
says what it is.

**The catalogue is chosen by the server, not by the question's grammar.** The
server picks the catalogue from the words in the question — the first of
university, college or scholarship it recognises wins, and a question that
names none is about universities. `interpretation.assumptions` says which
catalogue was searched, so show it if the user needs to know why the results
look like what they do.

**Results are ranked by relevance** when the question mentions something in
passing ("universities like Kathmandu University"), otherwise by name.

**The descriptive part of the question is matched against the prose.** A
question like "universities known for machine learning research" or
"programmes for becoming a data analyst" is matched against each row's
description — the university's overview and excerpt, the college's overview
and excerpt, the programme's description, excerpt and career options. The
words do not have to appear in a particular column, and a row whose prose
does not mention them is not returned.

This means the results are not limited to what the structured filters cover.
A question can match on wording that exists nowhere in the catalogue's
facets. Questions that mix both — "engineering colleges in Kathmandu with
strong placement support" — filter on the facts and search the prose, and
both have to hold.

**The endpoint is slower than a normal search.** It makes one model call before
it queries the database — the catalogue is chosen by the server itself, and
the model only turns the question into filters. Expect roughly one to four
seconds. A question that has been asked recently is answered from a short-lived
server-side cache with no model call at all, and comes back faster. Show a
loading state, and debounce or use a submit button rather than firing on every
keystroke.

---

## 6. Checklist

- [ ] New route `/nlsearch` (or similar) added to the router
- [ ] **Not** added to the navbar, footer, any menu, or the sitemap
- [ ] **No** link to it from any existing page
- [ ] New API client function for `POST /api/v1/nl-search` in its own file
- [ ] New page and its components in their own folder
- [ ] Types defined locally; nothing added to shared API types or stores
- [ ] `503` renders a "not available" state, not an error toast
- [ ] `429` renders a "slow down" state and honours `Retry-After`
- [ ] `assumptions` is shown (or deliberately omitted — your call, but not
      shown as if it were a search result)
- [ ] Loading state shown; the endpoint takes a second or two
- [ ] Each row links to the **existing** detail page for its `entity` and `slug`
- [ ] **No existing search endpoint, component, page, hook or store was modified**
