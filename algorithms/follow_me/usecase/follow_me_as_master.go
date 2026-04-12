package usecase

import (
	"follow_me/domain"
	"follow_me/ports"
	"log"

	"github.com/go-viper/mapstructure/v2"
)

type FollowMeAsMaster struct {
	cfg    domain.Config
	broker ports.CommunicationProvider
	state  domain.State
}

func NewFollowMeAsMaster(cfg domain.Config, broker ports.CommunicationProvider) *FollowMeAsMaster {
	return &FollowMeAsMaster{cfg, broker, domain.IDLE}
}

func (f *FollowMeAsMaster) Run() error {
	return f.startComms()
}

func (f *FollowMeAsMaster) startComms() error {
	if err := f.broker.Connect(
		f.cfg.BrokerIP,
		f.cfg.BrokerPort,
		f.cfg.TelemetryTopic,
		f.cfg.SubscriptionTopic,
	); err != nil {
		return err
	}

	msgChan, err := f.broker.Listen()
	if err != nil {
		return err
	}

	log.Printf("FollowMe Algorithm started as %s", f.cfg.Role)

	for msg := range msgChan {
		switch msg.Topic {
		case f.cfg.TelemetryTopic:
			f.handleTelemetry(msg.Payload)
		case f.cfg.SubscriptionTopic:
			f.handleCommand(msg.Payload)
		}
	}

	return nil
}

func (f *FollowMeAsMaster) handleTelemetry(payload any) {
	var tel domain.Telemetry
	if err := mapstructure.Decode(payload.(map[string]any), &tel); err != nil {
		log.Printf("Error unmarshalling telemetry: %v", err)
		return
	}

	if f.state == domain.IDLE ||
		tel.Position.RelativeAlt < 0.5 {
		return
	}

	f.broker.Publish(
		f.cfg.BroadcastTopic,
		map[string]any{
			"topic":   f.cfg.SubscriptionTopic,
			"payload": domain.FollowMeMessage{}.FromTelemetry(tel),
		},
	)
}

func (f *FollowMeAsMaster) handleCommand(payload any) {
	var cmd domain.CommandMessage
	if err := mapstructure.Decode(payload.(map[string]any), &cmd); err != nil {
		log.Printf("Error unmarshalling command: %v", err)
		return
	}

	log.Printf("Received command: %s", cmd.Command)

	switch cmd.Command {
	case "start", "resume":
		f.state = domain.RUNNING
	case "pause", "stop":
		f.state = domain.IDLE
	}
}
