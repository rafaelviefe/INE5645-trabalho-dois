package saga

import "context"

type Step struct {
	Execute    func(context.Context) error
	Compensate func(context.Context) error
}

type Orchestrator struct {
	steps []Step
}

func New() *Orchestrator {
	return &Orchestrator{}
}

func (o *Orchestrator) AddStep(execute, compensate func(context.Context) error) {
	o.steps = append(o.steps, Step{Execute: execute, Compensate: compensate})
}

func (o *Orchestrator) Execute(ctx context.Context) error {
	var executed int

	for _, step := range o.steps {
		if err := step.Execute(ctx); err != nil {
			for j := executed - 1; j >= 0; j-- {
				if fn := o.steps[j].Compensate; fn != nil {
					fn(ctx)
				}
			}
			return err
		}
		executed++
	}

	return nil
}
