package validator

import (
	"context"
	"errors"
	"fmt"
)

// Validator checks a target of type T -- e.g. a loaded SkillCatalog or a
// resolved configuration -- and returns every problem I it finds, not only
// the first.
type Validator[T, I any] interface {
	// Name identifies the validator in other validators' DependsOn.
	Name() string
	// DependsOn lists the validators that must run before this one.
	DependsOn() []Dependency
	Validate(ctx context.Context, target T) []I
}

// Dependency is one validator that must run before the validator declaring
// it. A required one must be registered, and before it; an optional one may
// be missing, but if registered it must come before it.
type Dependency struct {
	Name       string
	IsRequired bool
}

// Manager runs its validators in the order they were registered and
// collects all their problems. One Manager serves every kind of target, so
// checking a configuration and checking skills follow the same rules.
type Manager[T, I any] struct {
	validators []Validator[T, I]
}

// NewManager registers validators in the given order. It does not reorder
// them -- a person does -- but refuses an order that breaks a DependsOn:
// a name registered twice, a required dependency not registered, or any
// registered dependency placed after the validator depending on it. Every
// such problem is reported at once.
//
// Примеры (skill-name-validator зависит от link-validator, IsRequired=false):
//   - [link-validator, skill-name-validator] -> ok
//   - [skill-name-validator]                 -> ok: зависимость не обязательна
//   - [skill-name-validator, link-validator] -> ошибка: link-validator должен
//     быть раньше skill-name-validator
//   - с IsRequired=true и [skill-name-validator] -> ошибка: link-validator
//     обязателен, но не зарегистрирован
func NewManager[T, I any](validators ...Validator[T, I]) (*Manager[T, I], error) {
	position := map[string]int{}
	var errs []error
	for i, v := range validators {
		if _, ok := position[v.Name()]; ok {
			errs = append(errs, fmt.Errorf("validator %q is registered twice", v.Name()))
			continue
		}
		position[v.Name()] = i
	}
	for i, v := range validators {
		for _, dep := range v.DependsOn() {
			j, ok := position[dep.Name]
			switch {
			case !ok && dep.IsRequired:
				errs = append(errs, fmt.Errorf("validator %q requires %q, which is not registered", v.Name(), dep.Name))
			case ok && j > i:
				errs = append(errs, fmt.Errorf("validator %q must be registered before %q, which depends on it", dep.Name, v.Name()))
			}
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &Manager[T, I]{validators: validators}, nil
}

// Validate runs every validator in order -- each one even when an earlier
// one found problems -- and returns all their problems together.
func (m *Manager[T, I]) Validate(ctx context.Context, target T) []I {
	var problems []I
	for _, v := range m.validators {
		problems = append(problems, v.Validate(ctx, target)...)
	}
	return problems
}
