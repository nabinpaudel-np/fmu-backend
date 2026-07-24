package mail

import "errors"

// ErrNotConfigured: required MAIL_USERNAME / MAIL_PASSWORD env vars are missing.
// Returned from New so callers can decide whether to start without email.
var ErrNotConfigured = errors.New("mail: smtp username and password are required")

// ErrSendFailed wraps any failure inside SendClaimApproved — render, dial, or
// message send. The underlying error is wrapped so callers can inspect it.
var ErrSendFailed = errors.New("mail: send failed")
