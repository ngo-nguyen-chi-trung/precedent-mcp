package validator

import "context"

type Step[Req any, Res comparable] func(ctx context.Context, req Req) (Res, error)

type Pipeline[Req any, Res comparable] []Step[Req, Res]

func (p Pipeline[Req, Res]) Run(ctx context.Context, req Req) (Res, error) {
	var zero Res
	for _, step := range p {
		if res, err := step(ctx, req); err != nil || res != zero {
			return res, err
		}
	}
	return zero, nil
}

type Builder[Req any, Res comparable] struct {
	steps Pipeline[Req, Res]
}

func NewBuilder[Req any, Res comparable]() *Builder[Req, Res] {
	return &Builder[Req, Res]{}
}

func (b *Builder[Req, Res]) Add(step Step[Req, Res]) *Builder[Req, Res] {
	b.steps = append(b.steps, step)
	return b
}

func (b *Builder[Req, Res]) Build() Pipeline[Req, Res] {
	return b.steps
}
