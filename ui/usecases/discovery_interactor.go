package usecases

import (
	"ui/domain"
	"ui/ports"
)

type DiscoveryInteractor struct {
	repo ports.ConfigRepository
}

func NewDiscoveryInteractor(repo ports.ConfigRepository) *DiscoveryInteractor {
	return &DiscoveryInteractor{repo: repo}
}

func (i *DiscoveryInteractor) GetAvailableServices() []domain.ServiceType {
	return i.repo.GetAvailableServices()
}
