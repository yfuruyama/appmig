# Appmig - App Engine traffic migration tool
[![CircleCI](https://circleci.com/gh/yfuruyama/appmig.svg?style=svg)](https://circleci.com/gh/yfuruyama/appmig)

This tool allows you to migrate an App Engine service's version gradually from one version to another.

## Description

Normally, you can migrate an App Engine service's version by `gcloud app versions migrate` command or from the web console, but the speed of migration is out of control, sometimes it ends very fast.  
With this tool, you can control how fast migration proceeds precisely.

## Installation

```
$ go get -u github.com/yfuruyama/appmig
```

This tool uses [gcloud](https://cloud.google.com/sdk/gcloud/) to manipulate App Engine services.  
If you have not installed it yet, please install it before.

## Usage

```
$ appmig --project=PROJECT --service=SERVICE --version=VERSION --rate=RATE --interval=INTERVAL
```

Please look at `appmig --help` for more details.

## Example

Following example shows migration from `v1` to `v2` happens with steps `1% -> 10% -> 25% -> 50% -> 100%` in 30 seconds interval.

```
$ appmig --project=mytest --service=default --version=v2 --rate=1,10,25,50,100 --interval=30

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

If you are migrating from `v1` to `v2`, this tool iterates `gcloud app services set-traffic --splits=<v1=x%,v2=y%>` to increase the `v2` traffic gradually.   
Each step of the execution increases the ratio of `v2` traffic according to the `--rate` option.

Note that traffic splitting is held by the IP address of the requesting client.
