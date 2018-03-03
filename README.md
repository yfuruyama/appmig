# Appmig - App Engine step-by-step traffic migration tool

[![CircleCI](https://circleci.com/gh/addsict/appmig.svg?style=svg)](https://circleci.com/gh/addsict/appmig)

Appmig is a traffic migration tool for App Engine.
`gcloud app versions migrate` is 

## Installation

```
$ go get -u github.com/addsict/appmig
```

### requirements

This tool uses `gcloud` command for manipulating App Engine versions.
If you have not installed it yet, please install it (docs: https://cloud.google.com/sdk/downloads).

## Usage

Please refer `appmig --help` as well.

```
$ appmig --project=mytest --service=default --version=v2 --rate=1,10,25,50,100 --interval=30
...

Checking existence of version v2... : OK
Checking current serving version... : v1(100%)

Migrate traffic: project=mytest, service=default, from=v1, to=v2
Do you want to continue? [Y/n] Y

Migrating from v1(99%) to v2(1%)... : DONE
Waiting 30 seconds...

Migrating from v1(90%) to v2(10%)... : DONE
Waiting 30 seconds...

Migrating from v1(75%) to v2(25%)... : DONE
Waiting 30 seconds...

Migrating from v1(50%) to v2(50%)... : DONE
Waiting 30 seconds...

Migrating from v1(0%) to v2(100%)... : DONE

Finish migration!
```

## How it works

In the tool, `gcloud app services set-traffic --splits` is used.
