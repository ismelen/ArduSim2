package usecase

import (
	"netsim/domain/model"
	"netsim/domain/service"
	"time"
)

func (s *Simulator) EnqueueBroadcast(senderID, payload string, retries uint32, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sender, exists := s.UAVs[senderID]
	if !exists {
		return
	}

	payloadLen := len(payload)
	txNs := service.CalculateTxTimeNs(payloadLen)
	busyUntil := now.Add(time.Duration(txNs))

	if now.Before(sender.BusyUntil) {
		if retries >= s.Config.MaxCsmaRetries {
			s.Logger.Warn("Max retries exceeded for sender", "sender", senderID)
		} else {
			s.DelayedMsgs = append(s.DelayedMsgs, model.DelayedMessage{
				Msg: model.Message{
					SenderID: senderID,
					Payload:  payload,
					Retries:  retries,
				},
				RetryAfter: now.Add(time.Millisecond * time.Duration(1<<retries)),
			})
		}
		return
	}

	if service.HasNearTransmitters(s.Spatial, &sender.Position, s.Config.CsmaRangeM, now) {
		if retries >= s.Config.MaxCsmaRetries {
			s.Logger.Warn("Max retries exceeded (CSMA) for sender", "sender", senderID)
		} else {
			s.DelayedMsgs = append(s.DelayedMsgs, model.DelayedMessage{
				Msg: model.Message{
					SenderID: senderID,
					Payload:  payload,
					Retries:  retries,
				},
				RetryAfter: now.Add(time.Millisecond * time.Duration(1<<retries)),
			})
		}
		return
	}

	s.Spatial.RecordTransmitter(senderID, sender.Position, busyUntil)
	sender.BusyUntil = busyUntil

	receiverIDs := s.Spatial.GetNearbyUAVIDs(sender.ChunkKey, s.Config.ChunkRadius, senderID)
	for _, recID := range receiverIDs {
		s.processReceiver(senderID, recID, &sender.Position, payload, now, busyUntil, txNs)
	}
}

func (s *Simulator) HandlePeerBroadcast(senderID string, senderPos *model.Position, payload string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payloadLen := len(payload)
	txNs := service.CalculateTxTimeNs(payloadLen)
	busyUntil := now.Add(time.Duration(txNs))

	centerChunk := s.Spatial.GetChunkKey(senderPos)
	receiverIDs := s.Spatial.GetNearbyUAVIDs(centerChunk, s.Config.ChunkRadius, "")
	for _, recID := range receiverIDs {
		if recID != senderID {
			s.processReceiver(senderID, recID, senderPos, payload, now, busyUntil, txNs)
		}
	}
}

func (s *Simulator) processReceiver(senderID, receiverID string, senderPos *model.Position, payload string, now, busyUntil time.Time, txNs uint64) {
	receiver, exists := s.UAVs[receiverID]
	if !exists {
		return
	}

	if !service.PassDistanceCheck(senderPos, &receiver.Position, s.Config.MaxRangeM) {
		return
	}

	if now.Before(receiver.BusyUntil) {
		return
	}

	if receiver.BufferUsed+len(payload) > s.Config.BufferSizeBytes {
		return
	}

	msg := model.Message{
		SenderID: senderID,
		Payload:  payload,
		From:     now,
		To:       busyUntil,
		TxNs:     txNs,
		Retries:  0,
	}

	s.PendingMsgs[receiverID] = append(s.PendingMsgs[receiverID], msg)
	receiver.BufferUsed += len(payload)
}
