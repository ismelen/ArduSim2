package usecase

import (
	"fmt"
	"follow_me/domain"
	"follow_me/ports"
	"log"

	"github.com/go-viper/mapstructure/v2"
)

type FollowMeAsSlave struct {
	cfg            domain.Config
	broker         ports.CommunicationProvider
	state          domain.State
	tel            *domain.Telemetry
	alreadyTakeOff bool
}

func NewFollowMeAsSlave(
	cfg domain.Config,
	broker ports.CommunicationProvider,
) *FollowMeAsSlave {
	return &FollowMeAsSlave{
		cfg:    cfg,
		broker: broker,
		state:  domain.IDLE,
	}
}

func (f *FollowMeAsSlave) Run() error {
	return f.startComms()
}

func (f *FollowMeAsSlave) startComms() error {
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
			if err := f.handleCommand(msg.Payload); err == nil {
				continue
			}
			f.handleMasterTelemetry(msg.Payload)
		}
	}

	return nil
}

func (f *FollowMeAsSlave) handleTelemetry(payload any) {
	if f.state == domain.IDLE {
		return
	}

	var tel domain.Telemetry
	if err := mapstructure.Decode(payload.(map[string]any), &tel); err != nil {
		log.Printf("Error unmarshalling telemetry: %v", err)
		return
	}

	f.tel = &tel

	if f.state == domain.TAKEOFF && f.tel.Position.RelativeAlt > f.cfg.SlavesTakeoffAltitude {
		f.state = domain.RUNNING
	}
}

func (f *FollowMeAsSlave) handleCommand(payload any) error {
	var cmd domain.CommandMessage
	if err := mapstructure.Decode(payload.(map[string]any), &cmd); err != nil || cmd.Command == "" {
		if err != nil {
			return err
		}

		return fmt.Errorf("void command")
	}

	log.Printf("Received command: %s", cmd.Command)

	switch cmd.Command {
	case "start", "resume":
		f.state = domain.RUNNING
	case "pause", "stop":
		f.state = domain.IDLE
	}

	return nil
}

func (f *FollowMeAsSlave) handleMasterTelemetry(payload any) error {
	if f.state != domain.RUNNING {
		return nil
	}

	var masterTel domain.FollowMeMessage
	if err := mapstructure.Decode(payload.(map[string]any), &masterTel); err != nil {
		return err
	}

	if masterTel.Timestamp == 0 {
		return fmt.Errorf("void master telemetry")
	}

	isMasterFlying := masterTel.RelativeAlt > 0.5
	imFlying := f.tel.Position.RelativeAlt > 0.5

	if !isMasterFlying {
		if imFlying {
			f.broker.Publish(
				f.cfg.SuggestionsTopic,
				domain.Suggestion{
					Endpoint: domain.ActionLand,
				},
			)
			f.alreadyTakeOff = false
		}
		return nil
	}

	if !imFlying && !f.alreadyTakeOff {
		f.takeOff()
		return nil
	}

	f.broker.Publish(
		f.cfg.SuggestionsTopic,
		domain.Suggestion{
			Endpoint:  domain.ActionMoveTo,
			Latitude:  masterTel.Lat,
			Longitude: masterTel.Lon,
			Altitude:  masterTel.Alt,
		},
	)

	return nil
}

func (f *FollowMeAsSlave) takeOff() {
	f.alreadyTakeOff = true
	f.state = domain.TAKEOFF

	f.broker.Publish(
		f.cfg.SuggestionsTopic,
		domain.Suggestion{
			Endpoint: domain.ActionArm,
		},
	)

	f.broker.Publish(
		f.cfg.SuggestionsTopic,
		domain.Suggestion{
			Endpoint:   domain.ActionSetFlightmode,
			Flightmode: "GUIDED",
		},
	)

	log.Printf("Taking off to %.2f", f.cfg.SlavesTakeoffAltitude)

	f.broker.Publish(
		f.cfg.SuggestionsTopic,
		domain.Suggestion{
			Endpoint: domain.ActionTakeoff,
			Altitude: f.cfg.SlavesTakeoffAltitude,
		},
	)
}
