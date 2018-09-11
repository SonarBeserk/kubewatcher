package dep

import (
	"bufio"
	"os/exec"
	"strings"
)

func callStringListCommand(cmdStr ...string) ([]string, error) {
	deps := make([]string, 0)

	cmd := exec.Command(cmdStr[0], cmdStr[1:]...)
	w, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	out := bufio.NewScanner(w)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	for out.Scan() {
		deps = append(deps, out.Text())
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	return deps, nil
}

func GetDependencies(includeVendored bool) ([]string, error) {
	stdDeps, err := callStringListCommand(`go`, `list`, `std`)
	if err != nil {
		return nil, err
	}

	allDeps, err := callStringListCommand(`go`, `list`, `-f`, `{{ join .Imports "\n" }}`)
	if err != nil {
		return nil, err
	}

	deps := make([]string, 0)

	for _, dep := range allDeps {
		standard := false
		for _, std := range stdDeps {
			if dep == std {
				standard = true
				break
			}

		}

		if !includeVendored && strings.Contains(dep, "vendor") {
			continue
		}

		if !standard {
			deps = append(deps, dep)
		}
	}

	return deps, nil
}
