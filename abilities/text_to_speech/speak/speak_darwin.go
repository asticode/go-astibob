package speak

func (s *Speaker) Initialize() error { return nil }

func (s *Speaker) Close() error { return nil }

func (s *Speaker) Say(i string) error {
	return s.execute("say", i)
}
