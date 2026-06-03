package usecases

import "context"

type activeSession struct {
	ctx            context.Context
	composePath    string
	stackName      string
	swarmHost      string
	simulationName string
	uavIDs         []string
	algorithmIDs   map[string]bool
	stoppedIDs     map[string]bool
	loggingEnabled bool
}

func newActiveSession() *activeSession {
	return &activeSession{
		algorithmIDs: make(map[string]bool),
		stoppedIDs:   make(map[string]bool),
	}
}

func (s *activeSession) isRunning() bool {
	return s.composePath != "" || s.stackName != ""
}

func (s *activeSession) isLocal() bool {
	return s.stackName == ""
}

func (s *activeSession) loggerHost() string {
	if s.isLocal() {
		return ""
	}
	// Si el string swarmHost es de la forma "192.168.1.5:2375",
	// deberíamos devolver solo la IP, pero por ahora podemos
	// delegar el parsing al LoggerClient o simplemente devolverlo y que
	// el LoggerClient lo parsee.
	return s.swarmHost
}

func (s *activeSession) markStopped(id string) (allStopped bool) {
	s.stoppedIDs[id] = true
	if len(s.algorithmIDs) == 0 {
		return false
	}
	allStopped = true
	for algoId := range s.algorithmIDs {
		if !s.stoppedIDs[algoId] {
			allStopped = false
			break
		}
	}
	return allStopped
}

func (s *activeSession) clear() {
	s.composePath = ""
	s.stackName = ""
	s.swarmHost = ""
	s.simulationName = ""
	s.uavIDs = nil
	s.algorithmIDs = make(map[string]bool)
	s.stoppedIDs = make(map[string]bool)
	s.loggingEnabled = false
}
