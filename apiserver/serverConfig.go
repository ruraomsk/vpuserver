package apiserver

var ServerConfig *ServerConf

type ServerConf struct {
	LoggerPath     string `toml:"logger_path"`     //путь до каталога с логами сервера
	StaticPath     string `toml:"static_path"`     //путь до каталога static
	HTMLPath       string `toml:"html_path"`       //путь до каталога free
	PortHTTP       string `toml:"portHTTP"`        // порт http
	PortHTTPS      string `toml:"portHTTPS"`       // порт https
	ServerExchange string `toml:"server_exchange"` //ip  / порт сервера обмена
}

func NewConfig() *ServerConf {
	return &ServerConf{}
}
