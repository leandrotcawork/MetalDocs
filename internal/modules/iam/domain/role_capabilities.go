package domain

const RoleCapabilitiesVersion = 1

var RoleCapabilities = map[Role][]Capability{
	RoleViewer: {
		CapDocumentView,
		CapTemplateView,
	},
	RoleEditor: {
		CapDocumentView,
		CapDocumentCreate,
		CapDocumentEdit,
		CapTemplateView,
	},
	RoleReviewer: {
		CapDocumentView,
		CapDocumentEdit,
		CapWorkflowReview,
		CapTemplateView,
	},
	RoleApprover: {
		CapDocumentView,
		CapWorkflowApprove,
		CapTemplateView,
		CapTemplatePublish,
	},
}
