# Admin Users — Representative Institution Name

**One additive change.** The admin users list and the single-user endpoint now
carry the **name** of the university or college a representative is bound to,
alongside the id that was already there.

This is an update to `ADMIN_USERS_FRONTEND.md`. Read that document for the
full contract — this one only covers what changed.

- **Endpoints affected:** `GET /api/v1/admin/users`, `GET /api/v1/admin/users/{id}`,
  `PATCH /api/v1/admin/users/{id}/role`
- **New response fields:** `representative_university_name`, `representative_college_name`
- **Nothing is removed.** Every field that was there before is still there.

---

## 1. What changed

Before, a representative row showed only the institution's **id**:

```json
{
  "id": "cc2eb6e4-7525-4238-8bc5-2d9a628df6a5",
  "full_name": "Prabess",
  "email": "sthxprabess@gmail.com",
  "role": "representative",
  "account_type": "oauth",
  "email_verified": true,
  "representative_university_id": "e8d584d5-1955-49b2-a13a-5a301ba023fe",
  "created_at": "2026-11-19T08:50:07.297147Z",
  "updated_at": "2026-11-19T09:14:06.699906Z"
}
```

Now it carries the **name** as well:

```json
{
  "id": "cc2eb6e4-7525-4238-8bc5-2d9a628df6a5",
  "full_name": "Prabess",
  "email": "sthxprabess@gmail.com",
  "avatar": "https://lh3.googleusercontent.com/a/ACg8ocIMztVGoJaZ-EiXZFNqpzA56Acl-ZySFfRkzQ5GtrL0hG3GDg=s96-c",
  "role": "representative",
  "account_type": "oauth",
  "email_verified": true,
  "representative_university_id": "e8d584d5-1955-49b2-a13a-5a301ba023fe",
  "representative_university_name": "Harvard University",
  "created_at": "2026-11-19T08:50:07.297147Z",
  "updated_at": "2026-11-19T09:14:06.699906Z"
}
```

---

## 2. The new fields

| Field | Type | Notes |
|---|---|---|
| `representative_university_name` | string? | The name of the university the account represents. Absent when there is none. |
| `representative_college_name` | string? | The name of the college the account represents. Absent when there is none. |

They follow exactly the same rules as the ids they sit beside:

- **Only for the `representative` role.** A `student` or an `admin` has
  neither, and the fields are simply absent.
- **Only one of the two is ever set.** A representative is bound to exactly
  one university **or** one college.
- **The id is still there.** The name is an addition, not a replacement — keep
  the id if you link to the institution.

### When the name is absent

The name is missing in two cases, and they look the same to the frontend:

1. The account is not a representative.
2. The account **is** a representative, but the institution it was bound to
   has since been deleted from the catalogue.

The second one is rare but real, so **do not assume a representative always
has a name.** Render a fallback rather than a blank cell.


---

## 3. What to do on the frontend

### 3.1 Add the fields to the type

```ts
// src/features/admin-users/types/admin-users.types.ts

export interface AdminUser {
  id: string;
  full_name: string;
  email: string;
  avatar?: string;
  role: AdminUserRole;
  account_type: "password" | "oauth";
  email_verified: boolean;
  representative_university_id?: string;
  representative_college_id?: string;
  // NEW — the name of the institution the account represents.
  representative_university_name?: string;
  representative_college_name?: string;
  created_at: string;
  updated_at: string;
}
```

Both fields are **optional** — they are absent, not `null`, when they do not
apply. Marking them optional is what stops TypeScript from complaining about
every row that is not a representative.

### 3.2 Show the name in the list

The name is what the row should show. The id is not something an admin
recognises.

```tsx
// inside the row, next to the role badge
{u.role === "representative" && (
  <div className="text-sm">
    <span className="font-medium">
      {u.representative_university_name ?? u.representative_college_name}
    </span>
  </div>
)}
```

Or, if you want a fallback for a representative whose institution was deleted:

```tsx
const institutionOf = (u: AdminUser): string | null => {
  if (u.role !== "representative") return null;
  return u.representative_university_name || u.representative_college_name || null;
};

// in the row
{institutionOf(u) ? (
  <span className="text-sm text-gray-600">{institutionOf(u)}</span>
) : u.role === "representative" ? (
  <span className="text-sm text-amber-600">Institution removed</span>
) : null}
```

### 3.3 No extra request is needed

The name comes **with** the list row and with the detail row. Do not fetch the
university or college separately to show the name — that is an extra request
per row for data you already have.

If you want to link the name to the institution, use the **id** you already
have for the link and the **name** for the text:

```tsx
{u.representative_university_id && (
  <a href={`/universities/${u.representative_university_id}`}>
    {u.representative_university_name ?? u.representative_university_id}
  </a>

---

## 4. The role-change response changed too

`PATCH /api/v1/admin/users/{id}/role` now returns the **same shape** as a
list row, including the two new fields.

Before, a role change returned a row without the name. Now:

```json
{
  "id": "cc2eb6e4-7525-4238-8bc5-2d9a628df6a5",
  "full_name": "Prabess",
  "email": "sthxprabess@gmail.com",
  "role": "representative",
  "account_type": "oauth",
  "email_verified": true,
  "representative_university_id": "e8d584d5-1955-49b2-a13a-5a301ba023fe",
  "representative_university_name": "Harvard University",
  "created_at": "2026-11-19T08:50:07.297147Z",
  "updated_at": "2026-11-19T09:14:06.699906Z"
}
```

**This means you can replace the row in place** — the row the role-change
returns carries the name, so the list does not need a refetch after a role
change. That was already how the frontend worked; it now stays correct.

### Two behaviours to be aware of

**A role change that clears the binding drops the name too.** Changing a
representative to `student` or `admin` clears the institution, and the
response then has neither the id nor the name:

```json
{
  "id": "cc2eb6e4-7525-4238-8bc5-2d9a628df6a5",
  "full_name": "Prabess",
  "email": "sthxprabess@gmail.com",
  "role": "student",
  "account_type": "oauth",
  "email_verified": true,
  "created_at": "2026-11-19T08:50:07.297147Z",
  "updated_at": "2026-11-19T09:14:00.719205Z"
}
```

So a role change that leaves the `representative` role **removes** the
institution from the row. If your UI keeps the old name around because it
merges the response over the existing row, it will be left showing a
stale name. **Replace the row, do not merge it field by field.**

**The name follows the id.** If the id changes, the name changes with it, in
the same response. There is no separate step.

---

## 5. A bug that was fixed along the way

While making this change the backend found and fixed a real bug in the
role-change endpoint. **It was returning `500` on any role change that
cleared the institution** — changing a representative to `student` or
`admin`.

The cause was in the SQL: a cleared binding was written as an invalid UUID
(all zeros) instead of a real `NULL`, and the foreign key on
`representative_university_id` rejected it.

**What this means for you:** the endpoint works now, but if you had a
role-change dialog that was failing silently for a non-representative role,
that was this bug. **Retest the role change** as part of picking this up:

- representative → student (clears the institution) — should succeed
- student → representative (with a scope) — should succeed
- student → admin — should succeed

Any of those returning `500` means the backend has not been redeployed with
the fix.

---

## 6. Checklist

- [ ] `representative_university_name` added to the `AdminUser` type as **optional**
- [ ] `representative_college_name` added to the `AdminUser` type as **optional**
- [ ] The name is shown for a representative — the id is not what an admin recognises
- [ ] A fallback is rendered when a representative has no name (institution deleted)
- [ ] **No extra request** to look the institution up — the name is already in the row
- [ ] The role-change response **replaces** the row in the list; it is not merged field by field
- [ ] The role change was retested, including one that clears the institution
- [ ] **No existing user, profile, favourites or auth code was modified**

)}
```

Note the `?? u.representative_university_id` fallback — it keeps the link
usable even in the rare case the name is missing.
