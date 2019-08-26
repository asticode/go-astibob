package speaker

func (s *Speaker) Init() error { return nil }

func (s *Speaker) Close() error { return nil }

func (s *Speaker) Say(i string) error {
	return s.execute("say", i)
}
