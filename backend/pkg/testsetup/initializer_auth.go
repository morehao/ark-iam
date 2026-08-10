package testsetup

type authappInitializer struct {
	*baseAppInitializer
}

func newAuthappInitializer() (Initializer, error) {
	base, err := newBaseAppInitializer(AppNameAuth)
	if err != nil {
		return nil, err
	}

	return &authappInitializer{
		baseAppInitializer: base,
	}, nil
}

func (i *authappInitializer) Initialize() error {
	return i.Init()
}

func (i *authappInitializer) Close() error {
	return i.baseAppInitializer.Close()
}
