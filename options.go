package astibob

type ServerOptions struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
	Username string `toml:"username"`
}
