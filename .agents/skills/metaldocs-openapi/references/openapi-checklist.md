# OpenAPI Checklist

## Before editing
- [ ] Existing endpoint checked to avoid duplication
- [ ] Request/response schema reviewed for existing reusable components

## While editing
- [ ] Path and method defined
- [ ] Request schema defined (if POST/PUT/PATCH)
- [ ] Response schema defined (success + error cases)
- [ ] Error responses follow standard format (code, message, details, trace_id)
- [ ] Security requirement specified for protected endpoints

## Final
- [ ] No breaking change to existing v1 paths
- [ ] Handler response shape matches schema
- [ ] Commit includes OpenAPI change alongside API change
