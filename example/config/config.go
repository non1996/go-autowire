package config

type Settings struct {
	Message string
	Feature struct {
		Enabled bool
	}
}

type MessageProvider interface {
	Message() string
}

type Greeter interface {
	Greet() string
}

type configuration struct {
	settings Settings
}

type messageProvider struct {
	message string
}

func (p *messageProvider) Message() string {
	return p.message
}
