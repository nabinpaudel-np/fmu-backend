# NL search: frontend integration review

A review of the natural-language search frontend in
`FE-findmeuniversity`, branch `fix/nlsearch-program-slug`.

**Verdict: the endpoint is integrated correctly and the branch fixes a real
bug, but the integration is incomplete.** The programme catalogue sends fields
the frontend has no type for, and the programme cards render as empty. Two
smaller issues sit underneath that.

Everything below was checked against the running backend on
`POST /api/v1/nl-search`, not against the documentation.

> **Update:** items 1, 2 and 5 have been fixed on the frontend branch
> `fix/nlsearch-program-slug` (commit `f9195b6`). Item 3 is worked around in
> the frontend and should be fixed in the backend. Item 4 is a deployment
> setting. The findings are kept below as they were written.

---

## What is correct

These were verified and need no work.

| Area | Status |
|---|---|
| Route | Frontend posts to `/api/v1/nl-search`; backend registers the same path. |
| URL construction | `API_BASE_URL` already strips a trailing `/api/v1`, so `${API_BASE_URL}/api/v1/nl-search` resolves correctly. Same pattern the bulk-upload clients use. |
| `page_size` | Frontend caps at 50; backend's `maxPageSize` is 50. |
| Query length | Frontend caps at 500; backend's `maxQueryLength` is 500. |
| Envelope | Backend's `{success, data}` / `{success, error, errors}` matches the frontend's `NlSearchEnvelope`. |
| Status codes | 400, 422, 429, 502, 503 all map to the states the frontend renders. |
| `Retry-After` | Backend sets it on 429; the frontend reads it and counts down. |
| CSP | `connect-src` lists all three known API origins, so an env edit that only reaches the bundle does not get blocked. |
| Universities, colleges, scholarships links | All three use a real `slug` column and resolve on their detail routes. |
| Rate limit | 10/min with a burst of 5, per user when signed in, per IP otherwise. |

---

## Bug 1 — the programme catalogue sends fields the frontend has no type for

**This is the main gap.** The backend returns three fields for programmes
that the frontend's `NlSearchItem` does not declare.

The backend selects them (`internal/nlsearch/repository.go`):

```go
case "programs":
    return []string{
        a + "id",
        a + "title",
        a + "title AS slug",
        ...
        "COALESCE(" + a + "description, '') AS description",
        "COALESCE(" + a + "excerpt, '') AS excerpt",
        "COALESCE(" + a + "career_options, '') AS career_options",
```

The live payload confirms it:

```json
{
  "entity": "programs",
  "slug": "Data Science",
  "title": "Data Science",
  "description": "The Bachelor of Data Science is a four year undergraduate programme...",
  "excerpt": "A four year degree in statistics, data engineering, visualisation and machine learning...",
  "career_options": "Data scientist · Data analyst · Data engineer · Machine learning engineer..."
}
```

`src/features/nlsearch/types/nlsearch.types.ts` has none of `description`,
`excerpt` or `career_options`. TypeScript will not error at the call site
because the payload is typed as `NlSearchEnvelope` and cast to
`NlSearchResponse` — the extra fields arrive at runtime and are simply
invisible to the type.

**Fix.** Add them to `NlSearchItem`:

```ts
  /** Only the programme catalogue sends these. */
  description?: string | null;
  excerpt?: string | null;
  /** Career paths, separated by " · " in the payload. */
  career_options?: string | null;
```

---

## Bug 2 — programme cards render as empty

`pickFigures` in `NlSearchResultCard.tsx` handles scholarships in one branch
and everything else in another. Programmes fall into the second branch, which
looks for `tuition_min`, `acceptance_rate`, `employment_rate` and
`founded_year`. A programme has none of those, so `figures` comes back empty
and the card falls through to:

```tsx
<span className="text-meta font-medium text-muted-foreground">
  No figures published yet
</span>
```

That is wrong. A programme is not missing figures — it has a degree level, a
description and a set of careers. The same is true of `pickAttributes`,
which shows institution type and campus setting, neither of which a
programme has, so that line is empty too.

The result is a programme card with a title, an initials box, and the text
"No figures published yet".

**Fix.** Give programmes their own branch in both functions. The useful
figures are the careers:

```ts
if (item.entity === "programs") {
  const careers = item.career_options
    ?.split("·")
    .map((c) => c.trim())
    .filter(Boolean)
    .slice(0, 3);

  const out: string[] = [];
  if (careers?.length) {
    figures.push({ value: String(careers.length), label: "Career paths" });
  }
  return figures;
}
```

and in `pickAttributes`, surface `excerpt` (or the first line of
`description`) and the first few career names, which is the information a
student is actually comparing.

---

## Bug 3 — the programme slug fix is right, but it is a workaround

The branch's uncommitted change is correct and should be kept:

```diff
-  const href = `/${item.entity}/${item.slug}`;
+  const href = `/${item.entity}/${item.entity === "programs" ? slugify(item.slug) : item.slug}`;
```

The reason it is needed: the backend has no slug column for programmes, so
it sends the title:

```go
// A programme has no slug and no publication state, so the row is
// identified by its title alone. The frontend links to a programme by
// title, so that is what the slug carries here.
a + "title AS slug",
```

That gives `"Data Science"`, `"Artificial Intelligence"`, `"BCA"`, `"CS"`. The
detail route resolves a slug back to an id by slugifying each title:

```ts
const match = all.find((p) => slugify(p.title) === slugOrId);
```

so the URL has to be `slugify(title)` — `data-science`, `bca`, `cs`. Without
the fix, every programme link is a 404.

**But the fix belongs in the backend.** `slug` means one thing everywhere
else in this API, and here it means another. Two options:

1. **Keep the title in `slug` and slugify on the frontend.** This is what the
   branch does. It works, but every consumer of the endpoint has to know
   that programmes are the odd one out.

2. **Send a real slug.** Apply the same `slugify` rule in the backend and
   select `slugify(title) AS slug`. The frontend then needs no special case
   at all, and `href` goes back to one line.

Option 2 is better. The frontend already has `slugify` in `@/lib/slug`, so
the backend needs the same rule, and the two must agree — the risk with
option 1 is that the frontend's `slugify` and the route's `slugify` could
drift apart and break the link again. Whichever option is chosen, the
frontend's `slugify` and the route's `slugify` must stay the same function.

Note that the backend's comment says "The frontend links to a programme by
title, so that is what the slug carries here." That comment describes the
bug, not the intent.

---

## Bug 4 — the frontend `.env` points at a backend that is not serving

```
# .env
# NEXT_PUBLIC_API_URL=https://fmu-backend-dev-nabin-paudel.deploymysite.com/api/v1
NEXT_PUBLIC_API_URL=http://localhost:8080
```

The dev host is commented out. That is fine for local work against the
Docker backend, but it must be the other way round before the frontend is
deployed, because `NEXT_PUBLIC_*` is inlined at build time.

The CSP does not have this problem — `next.config.ts` lists all three known
origins:

```ts
const KNOWN_API_ORIGINS = [
  "https://fmu-backend-nabin-paudel.deploymysite.com",
  "https://fmu-backend-dev-nabin-paudel.deploymysite.com",
  "http://localhost:8080",
];
```

so an env edit that only reaches the client bundle is still allowed by the
policy. That is a deliberate and good decision, and it is worth keeping the
comment that explains it.

---

## Bug 5 — `NL_SEARCH_MAX_PAGE_SIZE` is dead code

`nlsearch.api.ts` exports it:

```ts
export const NL_SEARCH_MAX_PAGE_SIZE = 50;
```

Nothing imports it. `NlSearchPage.tsx` defines its own `PAGE_SIZE = 20` and
never reads the export. Either use it or delete it — an exported constant
that claims to be the limit while the page hard-codes a different number is
misleading.

---

## Unnecessary work in the frontend

The brief asked whether the frontend does anything it does not need to.

### Duplicated helpers — deliberate, and should stay

`NlSearchResultCard.tsx` says so explicitly:

```ts
/** Duplicated from the catalogue listing card on purpose: the live search must
 *  not be refactored to share code with an experimental endpoint. */
function compactMoney(value: number): string {
```

This is the right call for an experimental endpoint. Refactoring the live
listing card to share code with NL search would put the live product at risk
for no benefit. Leave it.

### The bespoke `fetch` in `nlsearch.api.ts` — necessary

```ts
/**
 * This does its own fetch rather than going through the shared `api` helper
 * because the rate-limit response carries a `Retry-After` header, and the
 * shared helper unwraps the envelope without exposing response headers.
 */
```

Also correct. The shared `api.post` helper does not surface response
headers, and the 429 countdown needs `Retry-After`. Leave it.

### The one thing that is genuinely unnecessary: a hand-rolled `fetch` in a
TanStack Query codebase

The rest of the app uses TanStack Query — `fetchAllPrograms` is a `queryFn`
in client components, for instance. NL search uses `useState` and `useEffect`
with a manual `AbortController` and a hand-managed loading flag.

This is defensible for a one-off experimental page, and the abort handling
is correct. But it means:

- no caching between pages or back-navigation
- no automatic refetch
- the loading state is three separate `useState` calls that can drift

It is not a bug and it does not need changing now. If NL search is ever
promoted out of experimental, this is the piece to revisit — the `api` client
would need a small change to expose response headers, and then the page
could use `useQuery` like everything else.

### The `/nlsearch` route is deliberately unlinked

```ts
/**
 * Experimental. The route is reachable by typing the URL and is deliberately
 * not linked from anywhere, so it is excluded from indexing as well.
 */
export const metadata: Metadata = {
  robots: { index: false, follow: false },
};
```

Nothing links to it and it is marked `noindex`. That is the intended state
for an experimental feature and should not change until the feature is ready
to launch.

---

## Summary

| # | Problem | Where | Severity |
|---|---|---|---|
| 1 | `description`, `excerpt`, `career_options` missing from `NlSearchItem` | `nlsearch.types.ts` | High — data is sent and thrown away |
| 2 | Programme cards show "No figures published yet" | `NlSearchResultCard.tsx` | High — every programme card looks broken |
| 3 | Programme slug is a title, not a slug | backend `repository.go` | Medium — the branch works around it |
| 4 | `.env` points at the local backend | `.env` | Medium — blocks a correct deployment |
| 5 | `NL_SEARCH_MAX_PAGE_SIZE` is exported but unused | `nlsearch.api.ts` | Low — misleading |

Items 1 and 2 are the same underlying gap: programmes were added to the
backend without the frontend catching up. They should be fixed together.

Item 3 is worth fixing in the backend rather than working around in the
frontend, but the branch's workaround is correct and should be committed as
it stands if the backend fix is not happening now.

Item 5 is a one-line deletion.
