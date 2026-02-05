/*
Copyright © 2025 Gonzalo Alvarez

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package workflow

import (
	"context"
	"fmt"

	"github.com/qmuntal/stateless"
)

type Workflow struct {
	sm       *stateless.StateMachine
	triggers []Trigger
}

func New(initialState State) *Workflow {
	return &Workflow{
		sm:       stateless.NewStateMachine(initialState),
		triggers: make([]Trigger, 0),
	}
}

func (w *Workflow) Configure(state State) *stateless.StateConfiguration {
	return w.sm.Configure(state)
}

func (w *Workflow) AddTrigger(t Trigger) {
	w.triggers = append(w.triggers, t)
}

func (w *Workflow) Run(ctx context.Context) error {
	for _, trigger := range w.triggers {
		if err := w.sm.FireCtx(ctx, trigger); err != nil {
			return fmt.Errorf("step %s failed: %w", trigger, err)
		}
	}
	return nil
}

func (w *Workflow) State() State {
	s, _ := w.sm.State(context.Background())
	return s.(State)
}

func (w *Workflow) CanFire(trigger Trigger) bool {
	can, _ := w.sm.CanFire(trigger)
	return can
}
