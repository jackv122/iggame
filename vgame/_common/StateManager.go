package com

import "time"

type StateManager struct {
	States         []GameState
	StateDurs      []float64
	StateTime      float64
	CurrState      GameState
	StateStartTime int64
	StateIndex     int

	onEnterState func(state GameState)
	onExitState  func(state GameState)
}

func (g *StateManager) Init(gameStates []GameState, stateTimes []float64, onEnterState func(state GameState), onExitState func(state GameState)) *StateManager {
	g.States = gameStates
	g.StateDurs = stateTimes
	g.onEnterState = onEnterState
	g.onExitState = onExitState
	return g
}

// state machine -------------------------------------------

func (g *StateManager) ResetState() {
	g.CurrState = GAME_STATE_STARTING
	g.StateIndex = 0
	g.StateTime = 0
}

func (g *StateManager) SetState(currState GameState, stateTime float64) bool {
	stateInd := -1
	for i, state := range g.States {
		if state == currState {
			stateInd = i
			break
		}
	}
	if stateInd == -1 {
		return false
	}
	g.CurrState = currState
	g.StateIndex = stateInd
	g.StateTime = stateTime
	return true
}

func (g *StateManager) Start() {
	g.onEnterState(g.CurrState)
}

func (g *StateManager) NextState() {
	g.onExitState(g.CurrState)
	g.StateIndex++
	if g.StateIndex >= len(g.States) {
		g.StateIndex = 0
	}
	g.CurrState = g.States[g.StateIndex]
	g.StateTime = 0
	g.StateStartTime = time.Now().UnixMilli()
	g.onEnterState(g.CurrState)
}

func (g *StateManager) StateUpdate(dt float64) {
	dur := g.StateDurs[g.StateIndex]
	g.StateTime += dt
	if dur == 0 {
		return
	}
	if g.StateTime >= dur {
		g.NextState()
	}
}
