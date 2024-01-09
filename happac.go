package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"syscall"

	"github.com/firstrow/tcp_server"
	"github.com/pborman/getopt/v2"
)

type ExecFlags struct {
	PatroniHost                string
	PatroniPort                string
	PatroniHealthcheckEndpoint string
	AgentPort                  string
	PgisreadyPort              string
	PgisreadyPath              string
}

func main() {

	execFlags := ExecFlags{
		PatroniPort:                "8008",
		PatroniHealthcheckEndpoint: "primary",
	}

	getopt.FlagLong(&execFlags.PatroniHost, "patroni-host", 'h', "Host of the patroni server").Mandatory()
	getopt.FlagLong(&execFlags.PatroniPort, "patroni-port", 'o', "Port of the patroni REST API server. Default:")
	getopt.FlagLong(&execFlags.PatroniHealthcheckEndpoint, "patroni-healthcheck", 'k', "Health check endpoint to use. Default:")
	getopt.FlagLong(&execFlags.AgentPort, "port", 'p', "port to use for this agent").Mandatory()
	getopt.FlagLong(&execFlags.PgisreadyPort, "pgisready-port", 'r', "The port to check using pg_isready").Mandatory()
	getopt.FlagLong(&execFlags.PgisreadyPath, "pgisready-path", 'x', "path of where the pg_isready executable resides")
	getopt.Parse()

	server := tcp_server.New(":" + execFlags.AgentPort)

	server.OnNewClient(func(c *tcp_server.Client) {
		fmt.Println("HAProxy connected to health check agent")

		statusCode, err := patroniPrimaryStatusCode(&execFlags)
		if err != nil {
			fmt.Println(err)
			c.Close()
			return
		}

		exitCode, err := checkPgisready(&execFlags)
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

func checkPgisready(execFlags *ExecFlags) (int, error) {

	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(execFlags.PgisreadyPath+"pg_isready", "-h", execFlags.PatroniHost, "-p", execFlags.PgisreadyPort)
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

	return 0, fmt.Errorf("Encountered an unexpected error when executing pg_isready")
}

func patroniPrimaryStatusCode(execFlags *ExecFlags) (int, error) {
	req, err := http.NewRequest("GET", "http://"+execFlags.PatroniHost+":"+execFlags.PatroniPort+"/"+execFlags.PatroniHealthcheckEndpoint, nil)
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
