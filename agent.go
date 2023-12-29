package main

import (
  "bytes"
  "errors"
  "fmt"
  "log"
  "net/http"
  "os/exec"
  "syscall"

  "github.com/firstrow/tcp_server"
  "github.com/pborman/getopt/v2"
)

var patroni_host string
var patroni_port = "8008"
var patroni_healthcheck_endpoint = "primary"
var agent_port string
var pgisready_port string
var pgisready_path string

func main() {

  getopt.FlagLong(&patroni_host, "patroni-host", 'h', "Host of the patroni server").Mandatory()
  getopt.FlagLong(&patroni_port, "patroni-port", 'o', "Port of the patroni REST API server. Default:")
  getopt.FlagLong(&patroni_healthcheck_endpoint, "patroni-healthcheck", 'k', "Health check endpoint to use. Default:")
  getopt.FlagLong(&agent_port, "port", 'p', "port to use for this agent").Mandatory()
  getopt.FlagLong(&pgisready_port, "pgisready-port", 'r', "The port to check using pg_isready").Mandatory()
  getopt.FlagLong(&pgisready_path, "pgisready-path", 'x', "path of where the pg_isready executable resides")
  getopt.Parse()

  server := tcp_server.New(":" + agent_port)

  server.OnNewClient(func(c *tcp_server.Client) {
    fmt.Println("HAProxy connected to health check agent")

    statusCode, err := patroni_primary_status_code()

    if err != nil {
      fmt.Println(err)
      c.Close()
      return
    }

    exitCode, err := check_pgisready(pgisready_port)
    if err != nil {
      fmt.Println(err)
      c.Close()
      return
    }

    if (err == nil) && (statusCode == 200) && (exitCode == 0) {
      c.Send("up\n")
    } else {
      c.Send("down\n")
    }

    c.Close()
  })

  server.Listen()
}

func check_pgisready(port string) (int, error) {

  var outbuf, errbuf bytes.Buffer
  cmd := exec.Command(pgisready_path+"pg_isready", "-h", patroni_host, "-p", port)
  cmd.Stdout = &outbuf
  cmd.Stderr = &errbuf
  err := cmd.Run()
  stdout := outbuf.String()
  stderr := errbuf.String()

  if err != nil {
    if exitError, ok := err.(*exec.ExitError); ok {
      ws := exitError.Sys().(syscall.WaitStatus)
      exitCode := ws.ExitStatus()
      log.Printf("command result, stdout: %v, stderr: %v, exitcode: %d", stdout, stderr, exitCode)
      return exitCode, nil
    } else {
      if stderr == "" {
        stderr = err.Error()
      }
    }
  } else {
    ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
    exitCode := ws.ExitStatus()
    log.Printf("command result, stdout: %v, stderr: %v, exitcode: %d", stdout, stderr, exitCode)
    return exitCode, nil
  }
  log.Printf("command result, stdout: %v, stderr: %v", stdout, stderr)

  return 0, errors.New("Encountered an unexpected error when executing pg_isready")
}

func patroni_primary_status_code() (int, error) {
  req, err := http.NewRequest("GET", "http://"+patroni_host+":"+patroni_port+"/"+patroni_healthcheck_endpoint, nil)
  if err != nil {
    return 0, err
  }

  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    return 0, err
  }

  defer resp.Body.Close()

  fmt.Println("HTTP Response Status:", resp.StatusCode, http.StatusText(resp.StatusCode))

  return resp.StatusCode, nil
}
