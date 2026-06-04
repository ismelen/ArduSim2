package usecase

import (
	"encoding/json"
	"netsim/domain/model"
	"time"
)

func (s *Simulator) SendMessages() {
	s.mu.Lock()
	now := time.Now()
	s.Spatial.PurgeExpiredTransmitters(now)

	// Dispatch pending
	if len(s.PendingMsgs) > 0 {
		pending := s.PendingMsgs
		s.PendingMsgs = make(map[string][]model.Message)

		for recID, msgs := range pending {
			totalLen := 0
			for _, m := range msgs {
				totalLen += len(m.Payload)
			}
			if uav, ok := s.UAVs[recID]; ok {
				uav.BufferUsed -= totalLen
				if uav.BufferUsed < 0 {
					uav.BufferUsed = 0
				}
			}
		}

		go s.dispatchToGateway(pending)
	}

	// Retry delayed
	oldDelayed := s.DelayedMsgs
	s.DelayedMsgs = nil

	for _, delayed := range oldDelayed {
		if now.Before(delayed.RetryAfter) {
			s.DelayedMsgs = append(s.DelayedMsgs, delayed)
		} else {
			// s.mu is locked, but EnqueueBroadcast locks it too. So we can't call EnqueueBroadcast directly while locked.
			// Instead, collect and call after unlock.
		}
	}
	
	var toRetry []model.Message
	for _, delayed := range oldDelayed {
		if !now.Before(delayed.RetryAfter) {
			toRetry = append(toRetry, delayed.Msg)
		}
	}
	s.mu.Unlock()

	for _, msg := range toRetry {
		s.EnqueueBroadcast(msg.SenderID, msg.Payload, msg.Retries+1, time.Now())
	}
}

func (s *Simulator) dispatchToGateway(pending map[string][]model.Message) {
	for recID, msgs := range pending {
		for _, msg := range msgs {
			deliver := map[string]any{
				"topic": "deliver",
				"payload": map[string]any{
					"target_uav_id": recID,
					"sender_id":     msg.SenderID,
					"payload":       msg.Payload,
				},
			}
			data, _ := json.Marshal(deliver)
			s.Sender.Send(data, nil)
		}
	}
}
