package testsetup

type iamappInitializer struct {
	*baseAppInitializer
}

func newIamappInitializer() (Initializer, error) {
	base, err := newBaseAppInitializer(AppNameIam)
	if err != nil {
		return nil, err
	}

	return &iamappInitializer{
		baseAppInitializer: base,
	}, nil
}

func (i *iamappInitializer) Initialize() error {
	return i.Init()
}

func (i *iamappInitializer) Close() error {
	return i.baseAppInitializer.Close()
}
