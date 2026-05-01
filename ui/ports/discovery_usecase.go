package ports

import "ui/domain"

// DiscoveryUseCase defines the business actions for service discovery.
type DiscoveryUseCase interface {
	GetAvailableServices() []domain.ServiceType
}
