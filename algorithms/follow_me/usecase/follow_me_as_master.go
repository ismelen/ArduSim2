package usecase

import (
	"follow_me/domain"
	chronjob "follow_me/infrastructure/chron-job"
	"follow_me/ports"
	"log"
	"time"

	"github.com/go-viper/mapstructure/v2"
)

type FollowMeAsMaster struct {
	FollowMeBase
	lastTelemetry *domain.Telemetry
	chronosned *chronjob.ChronJob
}

func NewFollowMeAsMaster(cfg domain.Config, broker ports.CommunicationProvider) *FollowMeAsMaster {
	fm := &FollowMeAsMaster{
		FollowMeBase: FollowMeBase{
			Cfg:    cfg,
			Broker: broker,
			State:  domain.IDLE,
		},
	}
	fm.handler = fm
	fm.chronosned = chronjob.NewChronJob(fm.sendTelemetry, time.Duration(cfg.SendPeriodMs)*time.Millisecond)

	return fm
}

func (f *FollowMeAsMaster) HandleTelemetryTopic(payload any) {
	var tel domain.Telemetry
	if err := mapstructure.Decode(payload.(map[string]any), &tel); err != nil {
		log.Printf("Error unmarshalling telemetry: %v", err)
		return
	}
	f.lastTelemetry = &tel

	log.Println(tel)

	if f.State == domain.IDLE ||
		tel.Position.RelativeAlt < 0.5 {
		return
	}
}


func (f *FollowMeAsMaster) OnStart() {
	f.State = domain.RUNNING
	f.chronosned.Start()
}

func (f *FollowMeAsMaster) OnPause() {
	f.State = domain.IDLE
	f.chronosned.Pause()
}

func (f *FollowMeAsMaster) OnStop() {
	f.State = domain.IDLE
	f.Broker.Publish(f.Cfg.SuggestionsTopic, domain.Suggestion{
		Endpoint: domain.ActionLand,
	})
	f.chronosned.Stop()
}

func (f *FollowMeAsMaster) sendTelemetry() {
	f.Broker.Publish(
		f.Cfg.BroadcastTopic,
		map[string]any{
			"topic":   f.Cfg.SubscriptionTopic,
			"payload": domain.FollowMeMessage{}.FromTelemetry(*f.lastTelemetry),
		},
	)
}
