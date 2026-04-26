package resolvers

func RegisterBuiltins(r *Registry) {
	r.Register(DocCodeResolver{})
	r.Register(DocTitleResolver{})
	r.Register(RevisionNumberResolver{})
	r.Register(EffectiveDateResolver{})
	r.Register(ControlledByAreaResolver{})
	r.Register(AuthorResolver{})
	r.Register(ApproversResolver{})
	r.Register(ApprovalDateResolver{})
}
