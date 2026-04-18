# W5 Destruction Census

Generated: 2026-04-18
Operator: CI (controller session)

This file is the authoritative kill list for Task 6 (`git rm`).
Every entry was reviewed against the spec §DELETED-at-W5 annotations.
Entries marked AMBIGUOUS must be resolved before Task 6 proceeds.

---

## Apps to delete
- apps/ck5-export
- apps/ck5-studio

## Frontend feature trees to delete
- frontend/apps/web/src/features/documents/ck5

## Go files to delete (internal/modules/documents/application)
- internal/modules/documents/application/adapters.go
- internal/modules/documents/application/adapters_test.go
- internal/modules/documents/application/capture_renderer_pin.go
- internal/modules/documents/application/capture_renderer_pin_test.go
- internal/modules/documents/application/service_attachments.go
- internal/modules/documents/application/service_browser_editor.go
- internal/modules/documents/application/service_ck5.go
- internal/modules/documents/application/service_ck5_export_client.go
- internal/modules/documents/application/service_ck5_export_client_test.go
- internal/modules/documents/application/service_ck5_template.go
- internal/modules/documents/application/service_ck5_template_publish.go
- internal/modules/documents/application/service_ck5_template_publish_test.go
- internal/modules/documents/application/service_ck5_template_test.go
- internal/modules/documents/application/service_ck5_test.go
- internal/modules/documents/application/service_collaboration.go
- internal/modules/documents/application/service_content_docx.go
- internal/modules/documents/application/service_content_native.go
- internal/modules/documents/application/service_core.go
- internal/modules/documents/application/service_core_helpers.go
- internal/modules/documents/application/service_document_runtime.go
- internal/modules/documents/application/service_editor_bundle.go
- internal/modules/documents/application/service_etapa_body.go
- internal/modules/documents/application/service_helpers.go
- internal/modules/documents/application/service_policies.go
- internal/modules/documents/application/service_profile_bundle.go
- internal/modules/documents/application/service_registry.go
- internal/modules/documents/application/service_registry_test.go
- internal/modules/documents/application/service_rich_content.go
- internal/modules/documents/application/service_rich_content_test.go
- internal/modules/documents/application/service_runtime_validation.go
- internal/modules/documents/application/service_schema_runtime.go
- internal/modules/documents/application/service_template_lifecycle.go
- internal/modules/documents/application/service_template_lifecycle_test.go
- internal/modules/documents/application/service_templates.go
- internal/modules/documents/application/service_templates_test.go

## Handler files to delete
- internal/modules/documents/delivery/http/create_document_handler.go
- internal/modules/documents/delivery/http/create_document_handler_test.go
- internal/modules/documents/delivery/http/handler.go
- internal/modules/documents/delivery/http/handler_attachments.go
- internal/modules/documents/delivery/http/handler_ck5_content.go
- internal/modules/documents/delivery/http/handler_ck5_content_test.go
- internal/modules/documents/delivery/http/handler_ck5_export.go
- internal/modules/documents/delivery/http/handler_ck5_export_test.go
- internal/modules/documents/delivery/http/handler_ck5_template.go
- internal/modules/documents/delivery/http/handler_ck5_template_publish.go
- internal/modules/documents/delivery/http/handler_ck5_template_publish_test.go
- internal/modules/documents/delivery/http/handler_ck5_template_test.go
- internal/modules/documents/delivery/http/handler_content.go
- internal/modules/documents/delivery/http/handler_mddm_wiring_test.go
- internal/modules/documents/delivery/http/handler_runtime.go
- internal/modules/documents/delivery/http/handler_telemetry_shadow_diff.go
- internal/modules/documents/delivery/http/handler_telemetry_shadow_diff_test.go
- internal/modules/documents/delivery/http/image_handler.go
- internal/modules/documents/delivery/http/image_handler_test.go
- internal/modules/documents/delivery/http/load_handler.go
- internal/modules/documents/delivery/http/load_handler_test.go
- internal/modules/documents/delivery/http/path_helpers.go
- internal/modules/documents/delivery/http/release_handler.go
- internal/modules/documents/delivery/http/release_handler_test.go
- internal/modules/documents/delivery/http/submit_for_approval_handler.go
- internal/modules/documents/delivery/http/template_admin_handler.go
- internal/modules/documents/delivery/http/template_admin_handler_test.go

## Domain files to delete
- internal/modules/documents/domain/collaboration.go
- internal/modules/documents/domain/etapa_body.go
- internal/modules/documents/domain/image_storage.go
- internal/modules/documents/domain/renderer_pin.go
- internal/modules/documents/domain/renderer_pin_test.go
- internal/modules/documents/domain/rich_envelope.go
- internal/modules/documents/domain/schema_runtime.go
- internal/modules/documents/domain/schema_runtime_errors.go
- internal/modules/documents/domain/shadow_diff.go
- internal/modules/documents/domain/template.go
- internal/modules/documents/domain/template_lifecycle.go
- internal/modules/documents/domain/template_test.go

## Docker Compose services to remove
(none — docker-compose.yml already clean of CK5/MDDM services)

## CI workflows to delete
(none — no separate ck5-ci.yml or mddm-ci.yml found)

## Migrations to archive (NOT delete — kept for history)
- migrations/0061_create_mddm_tables.sql
- migrations/0062_create_mddm_triggers.sql
- migrations/0063_seed_mddm_po_template.sql
- migrations/0065_seed_po_mddm_canvas_template.sql
- migrations/0066_switch_po_profile_default_to_mddm.sql
- migrations/0067_update_po_mddm_template_definition.sql
- migrations/0068_add_theme_to_po_mddm_template.sql
- migrations/0070_create_mddm_shadow_diff_events.sql

## AMBIGUOUS — needs pre-destruction investigation
(none — all entries above confirmed as CK5/MDDM/legacy-docgen only)

## Remaining items in internal/modules/documents/ NOT on kill list
The following files are NOT on the kill list; they remain as infrastructure
supporting the legacy v1 API until the module is either fully deleted or
those routes are confirmed dead by Task 5 grep:
- internal/modules/documents/application/service.go
- internal/modules/documents/application/service_add_version_test.go
- internal/modules/documents/application/load_service.go
- internal/modules/documents/application/load_service_test.go
- internal/modules/documents/application/release_service.go
- internal/modules/documents/application/release_service_test.go
- internal/modules/documents/delivery/http/handler_test_helpers_test.go
- internal/modules/documents/infrastructure/ (entire dir)

Task 5 (compile-time dep check) must confirm NONE of these are imported
by internal/modules/documents_v2/ before Task 6 proceeds.
