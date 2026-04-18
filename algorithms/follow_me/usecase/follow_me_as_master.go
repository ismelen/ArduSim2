package usecase

import (
	"follow_me/domain"
	"follow_me/ports"
	"log"

	"github.com/go-viper/mapstructure/v2"
)

type FollowMeAsMaster struct {
	FollowMeBase
}

func NewFollowMeAsMaster(cfg domain.Config, broker ports.CommunicationProvider) *FollowMeAsMaster {
	return &FollowMeAsMaster{
		FollowMeBase: FollowMeBase{
			Cfg:    cfg,
			Broker: broker,
			State:  domain.IDLE,
		},
	}
}

func (f *FollowMeAsMaster) handleTelemetryTopic(payload any) {
	var tel domain.Telemetry
	if err := mapstructure.Decode(payload.(map[string]any), &tel); err != nil {
		log.Printf("Error unmarshalling telemetry: %v", err)
		return
	}

	if f.State == domain.IDLE ||
		tel.Position.RelativeAlt < 0.5 {
		return
	}

	f.Broker.Publish(
		f.Cfg.BroadcastTopic,
		map[string]any{
			"topic":   f.Cfg.SubscriptionTopic,
			"payload": domain.FollowMeMessage{}.FromTelemetry(tel),
		},
	)
}

func (f *FollowMeAsMaster) onStart() {
	f.State = domain.RUNNING
}

func (f *FollowMeAsMaster) onPause() {
	f.State = domain.IDLE
}

func (f *FollowMeAsMaster) onStop() {
	f.State = domain.IDLE
	f.Broker.Publish(f.Cfg.SuggestionsTopic, domain.Suggestion{
		Endpoint: domain.ActionLand,
	})
}