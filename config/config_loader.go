package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type (

	RepositoryConfiguration struct {
		Url string `yaml:"url"`
		Name string `yaml:"name"`
	}

	GitHubConfiguration struct {
		AccessToken string `yaml:"access_token"`
		PassPhase string `yaml:"passphase"`
		PublicKey string `yaml:"public_key"`
		PrivateKey string `yaml:"private_key"`
	}
	MainConfiguration struct {
		Repositories []RepositoryConfiguration `yaml:"repos"`
		GitHub GitHubConfiguration `yaml:"github"`
		Organization string `yaml:"organization"`
		WorkingFolder string `yaml:"working_folder"`
		CronTime string `yaml:"cron_job"`
	}
)

func LoadConfiguration(file string) *MainConfiguration {
	bytesFile, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	config := &MainConfiguration{}
	err = yaml.Unmarshal(bytesFile, config)
	if err != nil {
		panic(err)
	}
	return config
}
