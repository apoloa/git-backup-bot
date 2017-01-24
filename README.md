#GIT BACKUP BOT

[![codebeat badge](https://codebeat.co/badges/70652f77-798b-4386-9909-118ea4e437c5)](https://codebeat.co/projects/github-com-wholedev-git-backup-bot)

## Dependencies

To compile the program its necessary install:

* libgit v.0.25.1.
* sshlib v.1.6.0 or higher.

The best tool to install this is [HomeBrew](https://github.com/Homebrew/brew/) for mac or [LinuxBrew](http://linuxbrew.sh/)

## Compile

To compile only runs 
```
go build
```

## Run

```
git_backup_bot -config=<PATH_CONFIG_FILE.yaml>
```