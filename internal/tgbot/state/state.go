package state

import "sync"

type Step string

const (
	StepIdle          Step = ""
	StepAddProvider   Step = "add:provider"
	StepAddNumber     Step = "add:number"
	StepAddAccount    Step = "add:account"
	StepAddNickname   Step = "add:nickname"
	StepAddThreshold  Step = "add:threshold"
	StepAddMode       Step = "add:mode"
	StepAddConfirm    Step = "add:confirm"
	StepEditThreshold Step = "edit:threshold"
	StepFeedback      Step = "feedback:text"
)

type MeterDraft struct {
	Provider      string
	MeterNumber   string
	AccountNumber string
	Nickname      string
	Threshold     float64
	NotifyMode    string
	Balance       float64
}

type Conversation struct {
	Step    Step
	Draft   MeterDraft
	MeterID string
}

type Store struct {
	m sync.Map
}

func NewStore() *Store { return &Store{} }

func (s *Store) Get(userID int64) (Conversation, bool) {
	v, ok := s.m.Load(userID)
	if !ok {
		return Conversation{}, false
	}
	return v.(Conversation), true
}

func (s *Store) Set(userID int64, c Conversation) { s.m.Store(userID, c) }

func (s *Store) Clear(userID int64) { s.m.Delete(userID) }
