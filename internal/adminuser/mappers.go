package adminuser

import (
	"github.com/jackc/pgx/v5/pgtype"

	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/user"
)

// toListItem turns a stored row into the shape the dashboard shows.
//
// The password hash is never carried across — not even to be dropped later.
// AccountType tells the dashboard how the account signs in.
func toListItem(row sqlc.ListUsersRow) UserListItem {
	item := UserListItem{
		ID:            row.ID,
		FullName:      row.FullName,
		Email:         row.Email,
		Role:          row.Role,
		AccountType:   accountType(row.Password == nil),
		EmailVerified: row.EmailVerified,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}

	if row.Avatar != nil {
		item.Avatar = *row.Avatar
	}
	if row.RepresentativeUniversityID.Valid {
		item.RepresentativeUniversityID = formatUUID(row.RepresentativeUniversityID.Bytes)
	}
	if row.RepresentativeCollegeID.Valid {
		item.RepresentativeCollegeID = formatUUID(row.RepresentativeCollegeID.Bytes)
	}
	if row.RepresentativeUniversityName != nil {
		item.RepresentativeUniversityName = *row.RepresentativeUniversityName
	}
	if row.RepresentativeCollegeName != nil {
		item.RepresentativeCollegeName = *row.RepresentativeCollegeName
	}
	return item
}

// toListItemFromRoleUpdate turns the row a role change returns into the same
// shape the list uses, so the dashboard can replace the row it holds without
// refetching the whole list.
func toListItemFromRoleUpdate(row sqlc.UpdateUserRoleRow) UserListItem {
	item := UserListItem{
		ID:            row.ID,
		FullName:      row.FullName,
		Email:         row.Email,
		Role:          row.Role,
		AccountType:   accountType(row.Password == nil),
		EmailVerified: row.EmailVerified,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}

	if row.Avatar != nil {
		item.Avatar = *row.Avatar
	}
	if row.RepresentativeUniversityID.Valid {
		item.RepresentativeUniversityID = formatUUID(row.RepresentativeUniversityID.Bytes)
	}
	if row.RepresentativeCollegeID.Valid {
		item.RepresentativeCollegeID = formatUUID(row.RepresentativeCollegeID.Bytes)
	}
	if row.RepresentativeUniversityName != "" {
		item.RepresentativeUniversityName = row.RepresentativeUniversityName
	}
	if row.RepresentativeCollegeName != "" {
		item.RepresentativeCollegeName = row.RepresentativeCollegeName
	}
	return item
}

// toListItemFromDetail turns the row the detail endpoint reads into the same
// shape the list uses.
func toListItemFromDetail(row sqlc.GetAdminUserByIDRow) UserListItem {
	item := UserListItem{
		ID:            row.ID,
		FullName:      row.FullName,
		Email:         row.Email,
		Role:          row.Role,
		AccountType:   accountType(row.Password == nil),
		EmailVerified: row.EmailVerified,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}

	if row.Avatar != nil {
		item.Avatar = *row.Avatar
	}
	if row.RepresentativeUniversityID.Valid {
		item.RepresentativeUniversityID = formatUUID(row.RepresentativeUniversityID.Bytes)
	}
	if row.RepresentativeCollegeID.Valid {
		item.RepresentativeCollegeID = formatUUID(row.RepresentativeCollegeID.Bytes)
	}
	if row.RepresentativeUniversityName != nil {
		item.RepresentativeUniversityName = *row.RepresentativeUniversityName
	}
	if row.RepresentativeCollegeName != nil {
		item.RepresentativeCollegeName = *row.RepresentativeCollegeName
	}
	return item
}

// toListItemFromDomain turns a domain user into the shape the dashboard
// shows.
func toListItemFromDomain(u *user.User) UserListItem {
	item := UserListItem{
		ID:            u.ID,
		FullName:      u.FullName,
		Email:         u.Email,
		Role:          u.Role,
		AccountType:   accountType(u.Password == nil),
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}

	if u.Avatar != nil {
		item.Avatar = *u.Avatar
	}
	if u.RepresentativeUniversityID != nil {
		item.RepresentativeUniversityID = *u.RepresentativeUniversityID
	}
	if u.RepresentativeCollegeID != nil {
		item.RepresentativeCollegeID = *u.RepresentativeCollegeID
	}
	return item
}

// accountType names how an account signs in. An account with no password
// came in through Google, so it has no password to reset.
func accountType(noPassword bool) string {
	if noPassword {
		return "oauth"
	}
	return "password"
}

// uuidBytes reads a UUID string into its 16 bytes. An unreadable id becomes
// an invalid UUID, which the database rejects — the caller gets a clear
// error rather than a silent no-op.
func uuidBytes(s string) [16]byte {
	var out [16]byte
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return out
	}
	return u.Bytes
}

// formatUUID renders a 16-byte UUID in canonical 8-4-4-4-12 form.
func formatUUID(b [16]byte) string {
	hex := make([]byte, 36)
	const digits = "0123456789abcdef"
	pos := 0
	for i, by := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			hex[pos] = '-'
			pos++
		}
		hex[pos] = digits[by>>4]
		hex[pos+1] = digits[by&0x0f]
		pos += 2
	}
	return string(hex)
}
