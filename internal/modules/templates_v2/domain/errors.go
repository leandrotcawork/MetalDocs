package domain

import "errors"

var ErrISOSegregationViolation = errors.New("templates_v2: iso_segregation_violation")
var ErrForbiddenRole = errors.New("templates_v2: forbidden_role")
