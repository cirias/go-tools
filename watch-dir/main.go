package main

import (
	"flag"
	"fmt"
	"gopkg.in/fsnotify.v1"
	"log"
	"os"
	"os/exec"
	"strings"
)

var dir string
var queueSize uint
var cmdTemplate string

func init() {
	flag.StringVar(&cmdTemplate, "cmd", "sleep 10 && echo \"$FILENAME\"", "the command run to each file")
	flag.UintVar(&queueSize, "queue-size", 5, "the max workers number")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <dir>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	dir = flag.Arg(0)
}

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln(err)
	}
	defer watcher.Close()

	workingQueue := make(chan struct{}, queueSize)
	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("Gotcha new file:", event.Name)
					go work(workingQueue, event.Name)
				}
			case err := <-watcher.Errors:
				log.Println("Error:", err)
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatalln(err)
	}
	<-done
}

func work(wc chan struct{}, file string) {
	wc <- struct{}{}
	defer func() {
		<-wc
	}()

	command := strings.Replace(cmdTemplate, "$FILENAME", file, -1)
	log.Println("Excute command:", command)
	cmd := exec.Command("sh", "-c", command)
	//cmd.Stdin = os.Stdin
	//cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Println(err)
	}
}
