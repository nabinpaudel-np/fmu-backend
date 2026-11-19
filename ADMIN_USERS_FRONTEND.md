# Admin User Management — Frontend Integration Guide

Admins need a way to see who has an account, what role they hold, and to fix
things when they go wrong: delete an account, or send the user a password
reset link. This document is the whole contract for that dashboard.

- **Base URL (dev):** `http://localhost:3000`
- **Prefix:** `/api/v1/admin/users`
- **Content type:** `application/json`
- **Auth:** the admin session cookie. Every endpoint here requires the
  `admin` role — a `student` or `representative` gets `403`.

This is **additive**. Nothing in the existing API changed; the endpoints are
new. `API.md` documents them too, but this document is the one written for
the frontend.

---

## 1. Rules for this integration

Read these before writing any code. They are not suggestions.

### 1.1 Put it behind the admin check

Every endpoint in this guide returns `403` to a non-admin. The frontend must
not rely on that alone — gate the whole page on the role the frontend
already has.

`GET /api/v1/auth/me` returns the current user's `role`. Use it:

```ts
const me = await api.get<User>("/auth/me");
const isAdmin = me.data?.role === "admin";
```

Render the admin section only when `isAdmin` is true. A `student` or
`representative` should never see a link to it.

### 1.2 Do not touch the existing user-facing code

This is a new admin area. It needs new API client functions, new types, and
new components. Do **not** add admin-only fields to the shared user types
that existing pages import — the live site's profile and favourites pages
read those, and a name collision there breaks them.

The one exception is `role` itself, which already exists on the `/auth/me`
response.

### 1.3 Keep the password out of it

The server never returns a password, not even a hash. There is no endpoint
that returns one and no endpoint that accepts one. Do not add a field for it
and do not ask the backend for one — the admin never sees or sets the
password.

### 1.4 Do not build a "set password" form

There is deliberately no way for an admin to choose somebody's password. The
admin triggers a reset; the **account holder** follows the emailed link and
chooses their own password. A "new password" input on this screen would be
asking for something the backend will not accept.

---

## 2. The endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/admin/users` | List users; filter by `role` and `search` |
| `GET` | `/api/v1/admin/users/{id}` | Get a single user |
| `PATCH` | `/api/v1/admin/users/{id}/role` | Change a user's role |
| `POST` | `/api/v1/admin/users/{id}/reset-password` | Email the user a reset link |
| `DELETE` | `/api/v1/admin/users/{id}` | Delete a user |

### GET `/api/v1/admin/users`

Lists accounts, **newest first**.

| Query param | Type | Notes |
|---|---|---|
| `role` | string | `admin`, `student` or `representative`. Omit for every role. |
| `search` | string | Matched case-insensitively against the name **and** the email. |
| `page` | int | 1-indexed. Default `1`. |
| `page_size` | int | Default `20`, max `100`. |

```http
GET /api/v1/admin/users?role=representative&search=ada&page=1&page_size=20
```

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b",
        "full_name": "Ada Lovelace",
        "email": "ada@example.com",
        "avatar": "https://cdn.example.com/avatars/ada.png",
        "role": "student",
        "account_type": "password",
        "email_verified": true,
        "created_at": "2026-06-28T10:55:59Z",
        "updated_at": "2026-06-28T10:55:59Z"
      },
      {
        "id": "a1b2c3d4-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
        "full_name": "Grace Hopper",
        "email": "grace@example.com",
        "role": "representative",
        "account_type": "password",
        "email_verified": true,
        "representative_university_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
        "created_at": "2026-07-01T09:12:00Z",
        "updated_at": "2026-07-01T09:12:00Z"
      }
    ],
    "meta": { "page": 1, "page_size": 20, "total": 2, "total_pages": 1 }
  }
}
```

#### The user item

| Field | Type | Notes |
|---|---|---|
| `id` | string | UUID. Use this for every other call. |
| `full_name` | string | |
| `email` | string | |
| `avatar` | string? | Absolute URL. May be absent. |
| `role` | string | `admin`, `student` or `representative`. |
| `account_type` | string | `password` or `oauth`. See below. |
| `email_verified` | bool | Always present. |
| `representative_university_id` | string? | Only for a `representative` bound to a university. |
| `representative_college_id` | string? | Only for a `representative` bound to a college. |
| `representative_university_name` | string? | The name of that university. See `new_update_rep_admin_user.md`. |
| `created_at` | string | ISO 8601 timestamp. |
| `updated_at` | string | ISO 8601 timestamp. |

**There is no `password` field.** Not in the list, not in the detail
endpoint, not anywhere. If your type has one, delete it.

#### `account_type` — read this before showing a reset button

| Value | Meaning | Can it be reset? |
|---|---|---|
| `password` | The account has a password. | **Yes.** |
| `oauth` | The account came in through Google and has **no password**. | **No.** |

This is the one piece of information that changes what the UI should offer.
For an `oauth` account the reset endpoint answers `409`, because a reset link
would send the user to a "choose a new password" page for an account that
can only ever sign in with Google.

**Hide or disable the reset action for `account_type: "oauth"` accounts** and
show "Signs in with Google" instead. That is a better experience than
letting the admin click a button that is guaranteed to fail.

The two scope ids are mutually exclusive: a representative is bound to exactly
one university **or** one college, never both and never neither. Both fields
are simply absent when they do not apply.

### GET `/api/v1/admin/users/{id}`

Returns one user, same shape as a list row.

```http
GET /api/v1/admin/users/d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b
```

Use it when you need one row's full detail. The list already carries every
field, so you rarely need this — but it exists for a detail drawer or a
refresh after a change.

### PATCH `/api/v1/admin/users/{id}/role`

Changes a user's role. **No body fields beyond `role` are required.**

```http
PATCH /api/v1/admin/users/d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b/role
Content-Type: application/json

{ "role": "admin" }
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `role` | string | yes | `admin`, `student` or `representative`. |
| `representative_university_id` | string? | depends | Only for the `representative` role. |
| `representative_college_id` | string? | depends | Only for the `representative` role. |

#### The scope rules — read these, they cause most failures

**A `representative` must be bound to exactly one institution.** Sending
neither, or both, is a `400`. So a role change to `representative` looks
like this:

```json
{
  "role": "representative",
  "representative_university_id": "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10"
}
```

or this:

```json
{
  "role": "representative",
  "representative_college_id": "1b5f2c43-9f68-5d3f-ce74-3f8b9d5e0a21"
}
```

**The other two roles carry no scope.** Changing a user to `admin` or
`student` clears any scope they had:

```json
{ "role": "student" }
```

That is the whole body. Do not send `representative_university_id: null` or
an empty string — the server ignores the scope fields for these roles and
clears the binding itself.

**Why this matters:** a representative who is still bound to a university
keeps the right to edit that university, and the `UNIQUE` constraint on the
column stops any other representative from ever claiming it. Promoting
somebody to `student` and leaving the binding in place would strand the
institution.

**Promoting to `admin` is the point of this endpoint.** An admin can promote
any user to `admin` with a single body, and the change takes effect on that
user's next login — their existing session keeps the old role until the
access token expires or they log in again. Say so in the UI if it matters
("takes effect on next login").

#### Response

The updated user, same shape as a list row. Use it to update the row in place
rather than refetching the whole list.

### POST `/api/v1/admin/users/{id}/reset-password`

Emails the account holder a reset link. **No request body.**

```http
POST /api/v1/admin/users/d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b/reset-password
```

```json
{
  "success": true,
  "data": { "sent": true, "email": "ada@example.com" }
}
```

`email` is the address the link went to. `sent` is `true` when the mail was
handed off successfully.

**The admin never sees or sets the new password.** The user follows the
emailed link and chooses their own — exactly the same flow as the
self-service forgot-password page the site already has. Any earlier unused
reset link for that account is invalidated first, so only the newest one
works.

If the server has no email configured, this answers `503`. The admin cannot
fix that from this screen, so show "ask the user to use the forgot-password
link on the login page" rather than a raw error.

### DELETE `/api/v1/admin/users/{id}`

Removes the account.

```http
DELETE /api/v1/admin/users/d3b07384-d9a2-4e0a-b71e-1c9f3e3e0a1b
```

The rows that reference the account go with it — refresh tokens, favourites
and the claims it created. The user is signed out everywhere.

**Two deletions are refused, both with `409`:**

| Refused | Why |
|---|---|
| Deleting **your own** account | You would lock yourself out mid-session. |
| Deleting the **last admin** | Nobody would be left able to manage the catalogue. |

These are server-side refusals, not client-side rules. You cannot know
whether a user is the last admin from the list, so handle the `409` and
show its message. Do not try to work around it.

---

## 3. Errors

The standard envelope applies: `{ "success": false, "error": "..." }`, with
`errors: [{ "field", "message" }]` for a validation failure.

| Status | Meaning | What to do |
|---|---|---|
| `400` | Unknown role, or a `representative` with no scope or with both scopes | Show the message on the role form |
| `401` | No session, or the session expired | Refresh and retry, then send to login |
| `403` | Signed in but not an admin | Show "not allowed". Hide the whole area |
| `404` | No such user | Show "user not found"; it may have been deleted |
| `409` | A refusal — last admin, self-delete, institution already has a representative, or an OAuth-only account | Show the `error` message as-is; it is written for the admin |
| `503` | Email is not configured on the server | Show "reset is unavailable" |
| `500` | Anything else | Generic error |

**Show the `409` message verbatim.** Each one is a sentence written for the
admin — "the last admin account cannot be removed", "you cannot delete your
own account" — and they explain the refusal better than any generic
"something went wrong" would.

**A `409` on the reset endpoint means the account signs in with Google.**
If you have not already hidden the button for `account_type: "oauth"`, this
is the fallback. Treat it as "this cannot be reset", not as an error.

### The exact refusal messages

Copy these if you want to show them in your own words:

| Situation | `error` string |
|---|---|
| Deleting yourself | `you cannot delete your own account` |
| Deleting the last admin | `the last admin account cannot be removed` |
| Demoting the last admin | `the last admin account cannot be removed` |
| Resetting a Google account | `this account signs in with Google and has no password to reset` |
| Institution already claimed | `that university or college already has a representative` |
| Bad role | `role must be one of: admin, student, representative` |
| Representative with no scope | `a representative must be bound to one university or one college` |

---

## 4. A minimal example

```ts
// api/adminUsers.ts  — new file, nothing existing imports this

export interface AdminUser {
  id: string;
  full_name: string;
  email: string;
  avatar?: string;
  role: "admin" | "student" | "representative";
  account_type: "password" | "oauth";
  email_verified: boolean;
  representative_university_id?: string;
  representative_college_id?: string;
  // The name of that institution — see new_update_rep_admin_user.md
  representative_university_name?: string;
  representative_college_name?: string;
  created_at: string;
  updated_at: string;
}

export interface AdminUserPage {
  items: AdminUser[];
  meta: { page: number; page_size: number; total: number; total_pages: number };
}

export interface UpdateRoleBody {
  role: AdminUser["role"];
  representative_university_id?: string;
  representative_college_id?: string;
}

const BASE = "/api/v1/admin/users";

export async function listUsers(params: {
  role?: string;
  search?: string;
  page?: number;
  page_size?: number;
}) {
  const qs = new URLSearchParams();
  if (params.role) qs.set("role", params.role);
  if (params.search) qs.set("search", params.search);
  if (params.page) qs.set("page", String(params.page));
  if (params.page_size) qs.set("page_size", String(params.page_size));

  const res = await fetch(`${BASE}?${qs}`, { credentials: "include" });
  const body = await res.json();
  if (!res.ok) throw new Error(body.error ?? "could not load users");
  return body.data as AdminUserPage;
}

export async function updateUserRole(id: string, body: UpdateRoleBody) {
  const res = await fetch(`${BASE}/${id}/role`, {
    method: "PATCH",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const payload = await res.json();
  if (!res.ok) throw new Error(payload.error ?? "could not change the role");
  return payload.data as AdminUser;
}

export async function sendResetLink(id: string) {
  const res = await fetch(`${BASE}/${id}/reset-password`, {
    method: "POST",
    credentials: "include",
  });
  const payload = await res.json();
  if (!res.ok) throw new Error(payload.error ?? "could not send the reset link");
  return payload.data as { sent: boolean; email: string };
}

export async function deleteUser(id: string) {
  const res = await fetch(`${BASE}/${id}`, {
    method: "DELETE",
    credentials: "include",
  });
  const payload = await res.json();
  if (!res.ok) throw new Error(payload.error ?? "could not delete the user");
}
```

### The page

```tsx
// pages/admin/Users.tsx  — new folder, nothing existing imports this

import { useEffect, useState } from "react";
import {
  listUsers,
  updateUserRole,
  sendResetLink,
  deleteUser,
  type AdminUser,
} from "../../api/adminUsers";

export default function AdminUsers() {
  const [items, setItems] = useState<AdminUser[]>([]);
  const [role, setRole] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function load() {
    try {
      const data = await listUsers({ role, search, page, page_size: 20 });
      setItems(data.items);
      setTotal(data.meta.total);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "could not load users");
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [role, page]);

  async function onRoleChange(user: AdminUser, next: AdminUser["role"]) {
    try {
      const body: Parameters<typeof updateUserRole>[1] = { role: next };
      // A representative needs exactly one scope, so ask for it.
      if (next === "representative") {
        const scope = window.prompt(
          "Which university or college id does this representative manage?"
        );
        if (!scope) return;
        body.representative_university_id = scope;
      }
      const updated = await updateUserRole(user.id, body);
      setItems((list) => list.map((u) => (u.id === updated.id ? updated : u)));
      setNotice(`Role updated to ${updated.role}. Takes effect on their next login.`);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "could not change the role");
    }
  }

  async function onReset(user: AdminUser) {
    try {
      const res = await sendResetLink(user.id);
      setNotice(`Reset link sent to ${res.email}.`);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "could not send the reset link");
    }
  }

  async function onDelete(user: AdminUser) {
    if (!window.confirm(`Delete ${user.email}? This cannot be undone.`)) return;
    try {
      await deleteUser(user.id);
      setItems((list) => list.filter((u) => u.id !== user.id));
      setNotice(`${user.email} deleted.`);
      setError("");
    } catch (e) {
      // The server refuses some deletions with a message written for the
      // admin — show it as-is.
      setError(e instanceof Error ? e.message : "could not delete the user");
    }
  }

  return (
    <div>
      <h1>Users</h1>

      {/* Filters */}
      <select value={role} onChange={(e) => { setPage(1); setRole(e.target.value); }}>
        <option value="">All roles</option>
        <option value="admin">Admin</option>
        <option value="student">Student</option>
        <option value="representative">Representative</option>
      </select>

      <input
        value={search}
        placeholder="Search name or email"
        onChange={(e) => setSearch(e.target.value)}
        onKeyDown={(e) => { if (e.key === "Enter") { setPage(1); load(); } }}
      />

      {error && <p role="alert">{error}</p>}
      {notice && <p role="status">{notice}</p>}

      <table>
        <thead>
          <tr>
            <th>Name</th><th>Email</th><th>Role</th><th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {items.map((u) => (
            <tr key={u.id}>
              <td>{u.full_name}</td>
              <td>{u.email}</td>
              <td>{u.role}</td>
              <td>
                <button onClick={() => onRoleChange(u, "admin")}>Make admin</button>
                <button onClick={() => onRoleChange(u, "student")}>Make student</button>

                {/* A Google account has no password to reset. */}
                {u.account_type === "password" ? (
                  <button onClick={() => onReset(u)}>Send reset link</button>
                ) : (
                  <span>Signs in with Google</span>
                )}

                <button onClick={() => onDelete(u)}>Delete</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <p>{total} users</p>
    </div>
  );
}
```

### The route

```tsx
// in your router — add the path, do NOT add it to any nav array
<Route path="/admin/users" element={<AdminUsers />} />
```

Gate it on the role, and keep it **out** of the navbar. The existing admin
area (the claims dashboard at `/api/v1/admin/claims`) is the natural place
for a link to this — if you add one, add it there and nowhere else.

---

## 5. Behaviour worth knowing

**The list is newest first.** Accounts are ordered by `created_at`
descending, so a newly registered student is at the top. There is no sort
parameter.

**Search matches the name and the email.** It is a substring match, so
`ada` finds `ada@example.com` and `Ada Lovelace`. It is not fuzzy — it will
not find `Adelaide` from `ada`, and it will not match on the id.

**The role filter is exact.** `?role=admin` returns only admins. It is not a
"contains" match.

**`account_type` is derived from whether the account has a password**, not
from the `oauth_provider` column. An account that registered with a password
and later signed in with Google is still `password` — it still has a
password, so it can still be reset.

**The role change takes effect on the user's next login.** Their existing
session keeps the old role until the access token expires or they log in
again. This is worth saying in the UI, because an admin who promotes
somebody and then immediately tests with that user's existing cookie will see
the old role and think it did not work.

**The role change is immediate in the database.** Only the *effective* role
of an existing session lags. The user list shows the new role at once.

**Changing away from `representative` clears the binding.** The user stops
being able to edit that university or college, and the institution becomes
available for somebody else to claim.

**One representative per institution.** The database enforces it — a second
representative bound to the same university or college gets `409`. The
frontend can offer a university/college picker, but it should expect the
conflict and show the message.

**The reset link goes to the account's own email address.** The admin cannot
redirect it. If the address is wrong, the link goes to the wrong place —
the admin has to fix the address another way.

**Deleting a user is permanent.** Their favourites, their claims and their
sessions go with them. There is no undo and no trash. Confirm before
calling it.

**The `admin` role is the only thing that unlocks this area.** A
`representative` can edit its own university but has no access to any
`/api/v1/admin/*` route.

---

## 6. Checklist

- [ ] Admin area reachable only when `GET /api/v1/auth/me` says `role === "admin"`
- [ ] **Not** added to the navbar, footer, any menu, or the sitemap
- [ ] New API client file for the five admin user endpoints
- [ ] New page and components in their own folder
- [ ] Types defined locally; **no** admin-only field added to the shared user types
- [ ] The user type has **no** `password` field
- [ ] Role filter and search both wired to the list request
- [ ] Pagination driven by `meta.total` / `meta.page`
- [ ] The reset action is **hidden or disabled** for `account_type: "oauth"`
- [ ] **No** "set password" or "new password" input anywhere
- [ ] `409` messages shown **verbatim** — they explain the refusal
- [ ] Delete is confirmed, and the list updates after a successful delete
- [ ] Role change to `representative` collects exactly one scope id
- [ ] A note that a role change takes effect on the user's next login
- [ ] **No existing user, profile, favourites or auth code was modified**
