package main

import (
	"go-backup-bot/git"
	"go-backup-bot/config"
	"github.com/robfig/cron"
	"time"
	"flag"
)

func main(){
	configPath := flag.String("config_file", "config.yaml", "Config Path")
	configuration := config.LoadConfiguration(*configPath)
	syncronizer := git.NewGitSyncronizer(*configuration)
	if len(configuration.CronTime) > 0  {
		c := cron.New()
		c.AddJob(configuration.CronTime, syncronizer)
		go c.Start()
		for ; ; {
			time.Sleep(1000000)
		}
	} else {
		syncronizer.Run()
	}


}