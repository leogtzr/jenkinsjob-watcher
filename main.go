package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/leogtzr/jenkinsjob-watcher/internal/config"
	"github.com/leogtzr/jenkinsjob-watcher/internal/jenkins"
)

var errMissingArgument = errors.New("missing argument")
var errMissingJenkinsEnvVar = errors.New("missing Jenkins environment variable")

const errRunningProgram = 1

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(errRunningProgram)
	}
}

func getCheckTimeInterval(args []string) (time.Duration, error) {
	interval := 20 * time.Second

	if len(args) >= 3 {
		seconds, err := strconv.Atoi(args[2])
		if err != nil {
			return 0, fmt.Errorf("el intervalo debe ser un número entero: %w", err)
		}
		if seconds < 1 {
			return 0, fmt.Errorf("el intervalo debe ser mayor a 0")
		}

		interval = time.Duration(seconds) * time.Second
	}

	return interval, nil
}

func run() error {
	if len(os.Args) < 2 {
		return errMissingArgument
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	jenkinsJobUrl := os.Args[1]
	jenkinsJobUrl = sanitizeUrl(jenkinsJobUrl)
	jobApiURL := buildApiUrl(jenkinsJobUrl)

	interval := time.Duration(cfg.Interval) * time.Second

	// Override
	if len(os.Args) >= 3 {
		interval, err = getCheckTimeInterval(os.Args[:])
		if err != nil {
			return fmt.Errorf("error: getting time interval: %w", err)
		}
	}

	jenkinsWatcherUser, err := mustEnv("JENKINS_WATCHER_USER")
	if err != nil {
		return err
	}

	jenkinsWatcherApiToken, err := mustEnv("JENKINS_WATCHER_API_TOKEN")
	if err != nil {
		return err
	}

	req, err := buildRequest(jobApiURL, jenkinsWatcherUser, jenkinsWatcherApiToken)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	_, _ = fmt.Fprintf(os.Stdout, "Running check every %v\n", interval)

	for {
		//Copy the request (the body is consumed)
		reqCopy := req.Clone(req.Context())
		resp, err := client.Do(reqCopy)
		if err != nil {
			return fmt.Errorf("error calling jenkins: %w", err)
		}

		build, err := getBuildInformation(resp)
		resp.Body.Close()

		if err != nil {
			return err
		}

		_, _ = fmt.Fprintf(os.Stdout, "Build: %s\n", build)

		// If the job is not building, quit.
		if !build.Building {
			// Notify ...
			if cfg.NotifyCommand != "" {
				result := "null"
				if build.Result != nil {
					result = *build.Result
				}

				cmd := exec.Command("sh", "-c", cfg.NotifyCommand)
				cmd.Env = append(os.Environ(),
					"BUILD_NUMBER="+strconv.Itoa(build.Number),
					"BUILD_RESULT="+result,
					"BUILD_URL="+build.URL,
					"BUILD_DISPLAY_NAME="+build.DisplayName,
				)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				if err := cmd.Run(); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "error executing jenkins: %s\n", err)
				}
			}
			_, _ = fmt.Fprintf(os.Stdout, "Job terminado.\n")
			return nil
		}

		time.Sleep(interval)
	}

	return nil
}

func getBuildInformation(resp *http.Response) (jenkins.Build, error) {
	// Equivalente a curl -f (falla en códigos 4xx/5xx)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return jenkins.Build{}, fmt.Errorf("request falló con status %d: %s", resp.StatusCode, string(body))
	}

	// Leer el body (lo que normalmente imprime curl)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return jenkins.Build{}, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	var build jenkins.Build
	if err := json.Unmarshal(body, &build); err != nil {
		return jenkins.Build{}, fmt.Errorf("error parsing JSON: %w", err)
	}

	// Pretty print:
	//b, err := json.MarshalIndent(build, "", "  ")
	//if err != nil {
	//	return jenkins.Build{}, fmt.Errorf("error parsing JSON: %w", err)
	//}

	return build, nil
}

func buildRequest(apiURL, jenkinsUser, jenkinsToken string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}

	// equivalent to a -u user:token
	req.SetBasicAuth(jenkinsUser, jenkinsToken)
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func buildApiUrl(jenkinsJobUrl string) string {
	return fmt.Sprintf("%s/api/json", jenkinsJobUrl)
}

func sanitizeUrl(url string) string {
	if strings.HasSuffix(url, "/") {
		url = strings.TrimSuffix(url, "/")
	}
	if strings.HasSuffix(url, "/console") {
		url = strings.TrimSuffix(url, "/console")
	}

	if strings.HasSuffix(url, "/") {
		url = strings.TrimSuffix(url, "/")
	}

	return url
}

func mustEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", errMissingJenkinsEnvVar
	}

	return val, nil
}
