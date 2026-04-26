package domain

import "errors"

var ErrISOSegregationViolation = errors.New("templates_v2: iso_segregation_violation")
var ErrForbidden = errors.New("templates_v2: forbidden")
var ErrForbiddenRole = errors.New("templates_v2: forbidden_role")
var ErrUploadMissing = errors.New("templates_v2: upload_missing")
var ErrInvalidApprovalConfig = errors.New("templates_v2: invalid_approval_config")
var ErrPlaceholderIDEmpty = errors.New("placeholder id empty")
var ErrDuplicatePlaceholderID = errors.New("duplicate placeholder id")
var ErrPlaceholderNameInvalid = errors.New("placeholder name invalid")
var ErrDuplicatePlaceholderName = errors.New("duplicate placeholder name")
var ErrInvalidConstraint = errors.New("invalid constraint")
var ErrPlaceholderCycle = errors.New("placeholder visibility cycle")
var ErrUnknownResolver = errors.New("unknown resolver")
