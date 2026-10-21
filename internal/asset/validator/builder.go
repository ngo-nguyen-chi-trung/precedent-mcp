package validator

type Step[T any] func(T) error

type Pipeline[T any] []Step[T]

func (p Pipeline[T]) Run(value T) error {
	for _, step := range p {
		if err := step(value); err != nil {
			return err
		}
	}
	return nil
}

type Builder[T any] struct {
	steps Pipeline[T]
}

func NewBuilder[T any]() *Builder[T] {
	return &Builder[T]{}
}

func (b *Builder[T]) Add(step Step[T]) *Builder[T] {
	b.steps = append(b.steps, step)
	return b
}

func (b *Builder[T]) Build() Pipeline[T] {
	return b.steps
}
