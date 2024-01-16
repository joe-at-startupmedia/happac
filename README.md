# happac: (HA)Proxy (P)atroni (P)GBouncer (A)gent (C)hecker

## Purpose
In cases where you have Patroni, PgBouncer and HAProxy installed on the same machine, the HAProxy healthchecks require the capability of detecting:
1. which server is the Patroni primary
2. if the PGBouncer daemon is running on the system
3. if PGBouncer is properly routing requests from HAProxy to Patroni.

## Installation
```bash
make build
sudo make install
sudo wget https://raw.githubusercontent.com/joe-at-startupmedia/happac/master/systemd/happac.service -P /usr/lib/systemd/system/
sudo wget https://raw.githubusercontent.com/joe-at-startupmedia/happac/master/systemd/happac.env -P /etc/haproxy/
#Modify /etc/haproxy/happac.env variable specifc to your needs
sudo systemctl enable happac
sudo systemctl start happac
```

### Using Release Installer

```bash
wget -O - https://raw.githubusercontent.com/joe-at-startupmedia/happac/master/release-installer.bash | bash -s 0.0.1
```

## Debugging
```bash
journalctl -u happac -f &
nc 0.0.0.0 5555 --recv-only
#You can also use curl for communicating with TCP connection but you might have to specify the 0.9 flag
curl -iL --http0.9 "http://0.0.0.0:5555"
```

### CMD Arguments
```
happac --help
unknown option: --help
Usage: happac [-h value] [-k value] [-o value] [-p value] [-r value] [-x value] [parameters ...]
 -h, --patroni-host=value
                   Host of the patroni server (required)
 -k, --patroni-healthcheck=value
                   Health check endpoint to use. Default: [primary]
 -o, --patroni-port=value
                   Port of the patroni REST API server. Default: [8008]
 -p, --port=value  port to use for this agent Default: [5555]
 -r, --pgisready-port=value
                   The port to check using pg_isready (required)
 -x, --pgisready-path=value
                   path of where the pg_isready executable resides
```


## HAProxy configuration options

### Scenario A
This is the method used without utilizing happac agent checks. If PGBouncer goes down on the patroni primary, the primary node connections in HAProxy would still be marked as available. Communication between haproxy and patroni would be disrupted resulting in failing read/write queries on port 5000.
```
listen master
        bind 10.132.200.201:5000
        option httpchk OPTIONS /primary
        http-check expect status 200
        default-server inter 3s fastinter 1s fall 3 rise 4 on-marked-down shutdown-sessions
        server patroni-1-pgbouncer 10.132.200.201:6432 maxconn 100 check port 8008
        server patroni-2-pgbouncer 10.132.200.202:6432 maxconn 100 check port 8008
        server patroni-3-pgbouncer 10.132.200.203:6432 maxconn 100 check port 8008
```

### Scenario B
In this method, happac agent checks are utilized in place of direct patroni `httpchk` healthchecks. Unlike Scenario A, when PGBouncer goes down on the patroni primary, all primary node connections in HAProxy would become unavalilable. Communication between haproxy and patroni would still be disrupted resulting in failing read/write queries on port 5000. 
```
listen master
        bind 10.132.200.201:5000
        default-server inter 3s fastinter 1s fall 3 rise 4 on-marked-down shutdown-sessions
        server patroni-1-pgbouncer 10.132.200.201:6432 maxconn 100 check agent-check agent-addr 10.132.200.201 agent-port 5555 agent-inter 5s
        server patroni-2-pgbouncer 10.132.200.202:6432 maxconn 100 check agent-check agent-addr 10.132.200.202 agent-port 5555 agent-inter 5s
        server patroni-3-pgbouncer 10.132.200.203:6432 maxconn 100 check agent-check agent-addr 10.132.200.203 agent-port 5555 agent-inter 5s
```

### Scenario C
In this method happac agent checks are utilized along with backup routing by leveraging `use_backend` conditionals. Unlike Scenario A and B, when PGBouncer goes down on the patroni primary, HAProxy picks the next eligible backend to serve requests. In the primary_patroni backend we establish connections directly from HAProxy to Patroni effectively bypassing the downed PGBouncer.

```
listen master
        bind 10.132.204.203:5000
        acl is_pgb_alive nbsrv(master_pgbouncer) -m int eq 1
        acl is_happac_alive nbsrv(happac) -m int eq 3
        use_backend master_pgbouncer if is_pgb_alive is_happac_alive
        use_backend master_patroni

backend happac
        server patroni-1-happac 10.132.200.201:5555 check
        server patroni-2-happac 10.132.200.202:5555 check
        server patroni-3-happac 10.132.200.203:5555 check

backend master_pgbouncer
        option tcplog
        default-server inter 3s fastinter 1s fall 3 rise 4 on-marked-down shutdown-sessions
        server patroni-1-pgbouncer 10.132.200.201:6432 maxconn 100 check agent-check agent-addr 10.132.200.201 agent-port 5555 agent-inter 5s
        server patroni-2-pgbouncer 10.132.200.202:6432 maxconn 100 check agent-check agent-addr 10.132.200.202 agent-port 5555 agent-inter 5s
        server patroni-3-pgbouncer 10.132.200.203:6432 maxconn 100 check agent-check agent-addr 10.132.200.203 agent-port 5555 agent-inter 5s

backend master_patroni
        option httpchk OPTIONS /primary
        http-check expect status 200
        default-server inter 3s fastinter 1s fall 3 rise 4 on-marked-down shutdown-sessions
        server patroni-1-pgbouncer 10.132.200.201:5432 maxconn 100 check port 8008
        server patroni-2-pgbouncer 10.132.200.202:5432 maxconn 100 check port 8008
        server patroni-3-pgbouncer 10.132.200.203:5432 maxconn 100 check port 8008
```

## Practicality
The argument can be made that the configuration overhead outweighs the benefit of accounting for such a statistically insignificant scenario: the pgbouncer service becoming unavailable on a master node. 
