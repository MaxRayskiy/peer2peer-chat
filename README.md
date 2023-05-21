[![distribute](https://github.com/MaxRayskiy/peer2peer-chat/actions/workflows/distribute.yml/badge.svg?branch=master)](https://github.com/MaxRayskiy/peer2peer-chat/actions/workflows/distribute.yml?branch=master) 

### A tiny peer-to-peer chat app that can be used for non-secret messages.

### Todo:
add encryption

run with
```
docker run -it -p 8888:8888  -p1234:1234 --network=host  maxrayskiy/chat
```

to generate docs run
```
godoc -http=:6060
```
