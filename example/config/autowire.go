package config

import "github.com/non1996/go-autowire/autowire"

func init() {
	autowire.Register(
		autowire.Component[configuration]().
			Configuration().
			Properties(autowire.Property[configuration]("example", func(configuration *configuration) Settings {
				return configuration.settings
			})).
			Beans(autowire.Bean[configuration, MessageProvider]("messageProvider", func(configuration *configuration) MessageProvider {
				return &messageProvider{message: configuration.settings.Message}
			})).
			PostConstruct(func(configuration *configuration) error {
				configuration.settings.Message = "configuration and bean are ready"
				configuration.settings.Feature.Enabled = true
				return nil
			}).
			Register(),
	)
}
