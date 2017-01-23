package main

import (
	"git-backup-bot/git"
	"git-backup-bot/config"
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
		// TODO: Move this loop to Gorutines and channels for less usage of memory and CPU.
		for ; ; {
			time.Sleep(1000000)
		}
	} else {
		syncronizer.Run()
	}


}