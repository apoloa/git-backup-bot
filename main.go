package main

import (
	"flag"
	"github.com/robfig/cron"
	"time"
	"git-backup-bot/config"
	"git-backup-bot/git"
	"log"
)

func main(){
	configPath := flag.String("config", "config.yaml", "Config Path")
	flag.Parse()
	log.Println("Service Initiated")
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