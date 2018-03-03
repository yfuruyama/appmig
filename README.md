# Appmig - App Engine traffic migration tool [![CircleCI](https://circleci.com/gh/addsict/appmig.svg?style=svg)](https://circleci.com/gh/addsict/appmig)

This tool allows you to migrate an App Engine service gradually from one version to another.

Normally, you can migrate a service by `gcloud app versions migrate` command, but the speed of migration is out of control, sometimes it ends very fast.  
With this tool, you can control how fast migration proceeds precisely.

## Installation

```
$ go get -u github.com/addsict/appmig
```

This tool uses `gcloud` for manipulating App Engine services.  
If you have not installed it yet, please install [it](https://cloud.google.com/sdk/downloads) before.

## Usage

Please `appmig --help` for more details.

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
