package objectstore

import "fmt"

func TemplateDocxKey(tenantID, templateID string, versionNum int) string {
	return fmt.Sprintf("tenants/%s/templates/%s/v%d.docx", tenantID, templateID, versionNum)
}

func TemplateSchemaKey(tenantID, templateID string, versionNum int) string {
	return fmt.Sprintf("tenants/%s/templates/%s/v%d.schema.json", tenantID, templateID, versionNum)
}
