# disconter - discovery containers(docker and podman)

## run
```
podman run --rm --net=host -v /run/podman/podman.sock:/var/run/docker.sock ghcr.io/psvmcc/disconter:latest
```

## how it works

### run container with label `disconter.service`
```
podman run --rm -l disconter.service=test -ti centos bash
```

### dig

#### srv
```
$ dig @127.0.0.1 -p 53535 _test._tcp.service.disconter SRV

; <<>> DiG 9.18.24 <<>> @127.0.0.1 -p 53535 _test._tcp.service.disconter SRV
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 33492
;; flags: qr rd; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; QUESTION SECTION:
;_test._tcp.service.disconter.	IN	SRV

;; ANSWER SECTION:
_test._tcp.service.disconter. 0	IN	SRV	1 1 80 nice_golick.container.disconter.

;; ADDITIONAL SECTION:
nice_golick.container.disconter. 0 IN	A	10.88.0.5

;; Query time: 3 msec
;; SERVER: 127.0.0.1#53535(127.0.0.1) (UDP)
;; WHEN: Wed Mar 06 20:32:46 UTC 2024
;; MSG SIZE  rcvd: 172
```
or

```
$ dig @127.0.0.1 -p 53535 test.service.disconter SRV

; <<>> DiG 9.18.24 <<>> @127.0.0.1 -p 53535 test.service.disconter SRV
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 22671
;; flags: qr rd; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; QUESTION SECTION:
;test.service.disconter.		IN	SRV

;; ANSWER SECTION:
test.service.disconter.	0	IN	SRV	1 1 80 nice_golick.container.disconter.

;; ADDITIONAL SECTION:
nice_golick.container.disconter. 0 IN	A	10.88.0.5

;; Query time: 0 msec
;; SERVER: 127.0.0.1#53535(127.0.0.1) (UDP)
;; WHEN: Wed Mar 06 20:33:54 UTC 2024
;; MSG SIZE  rcvd: 160
```


#### a
```
$ dig @127.0.0.1 -p 53535 test.service.disconter A

; <<>> DiG 9.18.24 <<>> @127.0.0.1 -p 53535 test.service.disconter A
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 34103
;; flags: qr rd; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 0
;; WARNING: recursion requested but not available

;; QUESTION SECTION:
;test.service.disconter.		IN	A

;; ANSWER SECTION:
test.service.disconter.	0	IN	A	10.88.0.5

;; Query time: 3 msec
;; SERVER: 127.0.0.1#53535(127.0.0.1) (UDP)
;; WHEN: Wed Mar 06 20:34:34 UTC 2024
;; MSG SIZE  rcvd: 78
```

or container by name `nice_golick`

```
$ dig @127.0.0.1 -p 53535 nice_golick.container.disconter A

; <<>> DiG 9.18.24 <<>> @127.0.0.1 -p 53535 nice_golick.container.disconter A
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 20492
;; flags: qr rd; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 0
;; WARNING: recursion requested but not available

;; QUESTION SECTION:
;nice_golick.container.disconter. IN	A

;; ANSWER SECTION:
nice_golick.container.disconter. 0 IN	A	10.88.0.5

;; Query time: 0 msec
;; SERVER: 127.0.0.1#53535(127.0.0.1) (UDP)
;; WHEN: Wed Mar 06 20:35:50 UTC 2024
;; MSG SIZE  rcvd: 96
```

### metrics

```
$ curl 127.0.0.1:9553/metrics -s |grep disco
disconter_dns_queries_total 7
disconter_dns_queries{type="A"} 3
disconter_dns_queries{type="SRV"} 4
disconter_info{version="v0.0.0",commit="7639f0cf6424a0d4be7a93ada21b78a0c33394fc"} 0
```
