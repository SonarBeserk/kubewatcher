package main

import (
	"go/build"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/radovskyb/watcher"
	"github.com/sonarbeserk/kubewatcher/dep"

	"github.com/dixonwille/skywalker"
)

var (
	whitelist = []string{
		".go",
		"Dockerfile",
	}
)

type PathWalkerWorker struct {
	*sync.Mutex
	dockerFiles []string
	files       []string
}

func (ew *PathWalkerWorker) Work(path string) {
	ew.Lock()
	defer ew.Unlock()
	if strings.HasSuffix(strings.ToLower(path), "dockerfile") {
		ew.dockerFiles = append(ew.dockerFiles, path)
	}
	ew.files = append(ew.files, path)
}

func main() {
	deps, err := dep.GetDependencies(true)
	if err != nil {
		log.Fatalf("Failed to get project dependencies: %v", err)
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	gopath = gopath + "/src/"

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Look for files in current dir
	deps = append(deps, strings.Replace(pwd, gopath, "", -1))

	files, dockerFiles, err := getProjectFiles(gopath, deps)
	if err != nil {
		log.Fatalf("Failed to get project files: %v", err)
	}

	w := watcher.New()

	for _, f := range files {
		if err = w.Add(f); err != nil {
			log.Fatalln(err)
		}
	}

	for _, df := range dockerFiles {
		log.Println("Docker file: " + df)
	}

	if len(w.WatchedFiles()) <= 10 {
		for path := range w.WatchedFiles() {
			log.Printf("Watching %s\n", strings.Replace(path, gopath, "", -1))
		}
	} else {
		log.Printf("Watching %d files\n", len(w.WatchedFiles()))
	}

	go handleFileEvents(w)

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

func getProjectFiles(rootPath string, deps []string) (files, dockerFiles []string, err error) {
	files = make([]string, 0)
	dockerFiles = make([]string, 0)

	for _, dep := range deps {
		ew := new(PathWalkerWorker)
		ew.Mutex = new(sync.Mutex)

		sw := skywalker.New(rootPath+dep, ew)
		sw.ExtListType = skywalker.LTWhitelist
		sw.ExtList = whitelist

		err := sw.Walk()
		if err != nil {
			return nil, nil, err
		}

		files = append(files, ew.files...)
		dockerFiles = append(dockerFiles, ew.dockerFiles...)
	}

	return files, dockerFiles, nil
}

func handleFileEvents(w *watcher.Watcher) {
	for {
		select {
		case event := <-w.Event:
			log.Println(event) // Print the event's info.
		case err := <-w.Error:
			log.Fatalln(err)
		case <-w.Closed:
			return
		}
	}
}
