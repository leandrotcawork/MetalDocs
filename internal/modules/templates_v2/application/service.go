package application

type Service struct {
	repo      Repository
	presign   Presigner
	clock     Clock
	uuid      UUIDGen
	resolvers ResolverRegistryReader
}

func New(repo Repository, presign Presigner, clock Clock, uuid UUIDGen, resolvers ...ResolverRegistryReader) *Service {
	var registry ResolverRegistryReader
	if len(resolvers) > 0 {
		registry = resolvers[0]
	}
	return &Service{repo: repo, presign: presign, clock: clock, uuid: uuid, resolvers: registry}
}
