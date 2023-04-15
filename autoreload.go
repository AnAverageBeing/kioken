package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"net/http"

	"github.com/go-playground/webhooks/v6/github"
)

const (
	path = "/webhook"
)

func main() {
	str, ok := os.LookupEnv("GITHUB_WEBHOOK_SECRET")
	if !ok {
		log.Fatalln("env var GITHUB_WEBHOOK_SECRET is not set")
	}

	hook, _ := github.New(github.Options.Secret(str))

	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			w.Write([]byte("ACCESS DENIED"))
			return
		}

		_, err := hook.Parse(r, github.ReleaseEvent, github.PullRequestEvent)
		if err != nil {
			w.Write([]byte("Error: " + err.Error()))
			return
		}

		restart()
	})

	buildCmd := exec.Command("go", "build", "-o", "kioken", "cmd/kioken/kioken.go")
	if err := buildCmd.Start(); err != nil {
		log.Fatalf("error building application: %v", err)

	}
	// start a new instance of the application
	startCmd := exec.Command("./kioken")
	if err := startCmd.Start(); err != nil {
		log.Fatalf("error starting application: %v", err)
	}

	http.ListenAndServe(":5000", nil)
}

func restart() error {
	fmt.Println("Restarting kioken server...")
	// find the process ID of the running application
	pidCmd := exec.Command("pidof", "kioken")
	pidOutput, err := pidCmd.Output()
	if err != nil {
		return fmt.Errorf("error getting pid: %v", err)
	}

	// kill the existing process
	if len(pidOutput) > 0 {
		pid := strings.TrimSpace(string(pidOutput))
		killCmd := exec.Command("kill", "-9", pid)
		if err := killCmd.Run(); err != nil {
			return fmt.Errorf("error killing process: %v", err)
		}
	}

	buildCmd := exec.Command("go", "build", "-o", "kioken", "cmd/kioken/kioken.go")
	if err := buildCmd.Start(); err != nil {
		return fmt.Errorf("error building application: %v", err)

	}
	// start a new instance of the application
	startCmd := exec.Command("./kioken")
	if err := startCmd.Start(); err != nil {
		return fmt.Errorf("error starting application: %v", err)
	}

	fmt.Println("Restarted successfully")
	return nil
}
