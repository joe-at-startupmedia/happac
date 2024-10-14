services {
  id = "patroni-primary"
  name = "patroni-primary"
  tags = [
    "primary"
  ]
  address = "0.0.0.0"
  port = 5000
  checks = [
    {
      args = ["/etc/consul.d/scripts/happac_healthcheck.bash", "0.0.0.0", "5555"]
      interval = "5s"
      timeout = "20s"
    }
  ]
}

services {
  id = "patroni-replica"
  name = "patroni-replica"
  tags = [
    "replica"
  ]
  address = "0.0.0.0"
  port = 5001
  checks = [
    {
      args = ["/etc/consul.d/scripts/patroni_healthcheck.bash", "0.0.0.0", "5001"]
      interval = "5s"
      timeout = "20s"
    }
  ]
}
