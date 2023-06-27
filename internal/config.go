package internal

type Config struct {
	Storj struct {
		AccessGrant string `mapstructure:"access_grant"`
	} `mapstructure:"storj"`

	Video struct {
		Quality string `mapstructure:"quality"`
	} `mapstructure:"video"`
}
