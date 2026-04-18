package usecase

import (
	"fmt"
	"follow_me/domain"
	"follow_me/ports"
	"log"

	"github.com/go-viper/mapstructure/v2"
)

type FollowMeBase struct {
	Cfg    domain.Config
	Broker ports.CommunicationProvider
	State  domain.State
}

func (f *FollowMeBase) Run() error {
	return f.StartComms()
}

func (f *FollowMeBase) StartComms() error {
	if err := f.Broker.Connect(
		f.Cfg.BrokerIP,
		f.Cfg.BrokerPort,
		f.Cfg.TelemetryTopic,
		f.Cfg.SubscriptionTopic,
	); err != nil {
		return err
	}

	msgChan, err := f.Broker.Listen()
	if err != nil {
		return err
	}

	log.Printf("FollowMe Algorithm started as %s", f.Cfg.Role)

	for msg := range msgChan {
		switch msg.Topic {
		case f.Cfg.TelemetryTopic:
			f.HandleTelemetryTopic(msg.Payload)
		case f.Cfg.SubscriptionTopic:
			f.HandleSubscriptionTopic(msg.Payload)
		}
	}

	return nil
}

func (f *FollowMeBase) HandleCommand(payload any) error {
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
		f.OnStart()
	case "pause":
		f.OnPause()
	case "stop":
		f.OnStop()
	}

	return nil
}

func (f *FollowMeBase) OnStart() {}
func (f *FollowMeBase) OnPause() {}
func (f *FollowMeBase) OnStop()  {}

func (f *FollowMeBase) HandleTelemetryTopic(payload any) {}
func (f *FollowMeBase) HandleSubscriptionTopic(payload any) {
	f.HandleCommand(payload)
}
