appmig
===
App Engine step-by-step traffic migration tool

```
$ appmig --project=myproject --service=default --version=v2 --rate=10,50,100 --interval=5

Checking current serving version... : v1(100%)
Migration: project = myproject, service = default, from = v1, to = v2
Do you want to continue? [Y/n]
Migrating from v1(90%) to v2(10%)... DONE
Waiting...
Migrating from v1(50%) to v2(50%)... DONE
Waiting...
Migrating from v1(0%) to v2(100%)... DONE
Finish migration!
```
