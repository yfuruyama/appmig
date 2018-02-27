
```
$ gaemigrate --project=test --service=foo --version=v2 --rate=1,5,10,30,50,75,100 --interval=60

CURRENT: v1
[--------------->    ] 70%

TARGET:  v2
[------>             ] 30%

next: gcloud --project=test app services set-traffic foo --splits v2=.9,v1=.1 --split-by ip
next: gcloud --project=test app services set-traffic foo --splits v2=.8,v1=.2 --split-by ip
next: gcloud --project=test app services set-traffic foo --splits v2=.7,v1=.3 --split-by ip
...
```
