# README #

This README would normally document whatever steps are necessary to get your application up and running.

### What is this repository for? ###

* Quick summary
* Version
* [Learn Markdown](https://bitbucket.org/tutorials/markdowndemo)

### How do I get set up? ###

* Summary of set up
* Configuration
* Dependencies
* Database configuration
* How to run tests
* Deployment instructions

### Contribution guidelines ###

* Writing tests
* Code review
* Other guidelines

### Who do I talk to? ###

* Repo owner or admin
* Other community or team contact

### Configure log channels
Debugging issues by tracking logs

There are 4 log channels including console, file, cloudwatch, logstash
Set LOG_DESTINATIONS variable in .env file

ex: export LOG_DESTINATIONS=file,cloudwatch

## Linting and pre-comit hook

### Go lint
```bash
sudo snap install golangci-lint
```
Config .golangci.yaml file to add or remote lint options

### pre-commit hook
```bash
sudo snap install pre-commit --classic
```
Config .pre-commit-config.yaml file to enable or disable pre-commit hook