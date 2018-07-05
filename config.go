package main

//Auth stores the credentials of a smtp connection
type Auth struct {
	Username string
	Password string
	Host     string
	Port     int
}

//SenderConfig wraps the credentials and the sender address
type SenderConfig struct {
	Auth Auth
	Addr string
}

//EmailConfig config wraps the SenderConfig the list of the receivers of the notifications
type EmailConfig struct {
	Sender    SenderConfig
	Receivers []string
}

//Server stores the host and ip of a server
type Server struct {
	Host    string
	Port    string
	Timeout int
	Status  bool
}

//ServerName returns the host and the port joined by a colon
func (s Server) ServerName() string {
	return s.Host + ":" + s.Port
}

//Config is the main configuration which includes all the others
var Config struct {
	Servers   map[string]Server `yaml:"servers"`
	CheckRate int               `yaml:"checkRate"`
	Email     EmailConfig
}
