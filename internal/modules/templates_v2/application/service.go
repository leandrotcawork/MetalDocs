package application

type Service struct {
	repo    Repository
	presign Presigner
	clock   Clock
	uuid    UUIDGen
}

func New(repo Repository, presign Presigner, clock Clock, uuid UUIDGen) *Service {
	return &Service{repo: repo, presign: presign, clock: clock, uuid: uuid}
}
