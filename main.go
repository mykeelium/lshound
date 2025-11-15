package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	lshound_files "github.com/mykeelium/lshound/files"
	lshound_groups "github.com/mykeelium/lshound/groups"
	model "github.com/mykeelium/lshound/model"
	lshound_users "github.com/mykeelium/lshound/users"
	"github.com/mykeelium/lshound/writer"
)

var (
	startPath      string
	doJSON         bool
	checkACL       bool
	followSymlink  bool
	maxDepth       int
	outputToStdOut bool
	fileChannel    chan model.FileInfoRecord
	outputName     string
	wg             sync.WaitGroup
)

func fromStdin() {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		path := strings.TrimSpace(line)
		if path != "" {
			info, statErr := os.Lstat(path)
			if statErr != nil {
				emit()(fileChannel, doJSON, model.FileInfoRecord{Path: path, Err: statErr.Error()})
			} else {
				rec := lshound_files.ProcessPath(path, info, checkACL)
				emit()(fileChannel, doJSON, rec)
			}
		}
		if err == io.EOF {
			close(fileChannel)
			break
		}
	}
}

func emit() model.Emit {
	if outputToStdOut {
		return writer.EmitStdOut
	} else {
		return writer.EmitChannel
	}
}

func init() {
	flag.StringVar(&startPath, "path", ".", "starting path")
	flag.BoolVar(&doJSON, "json", true, "output to json")
	flag.BoolVar(&checkACL, "acl", false, "check for POSIX ACLs using getfacl (if available)")
	flag.BoolVar(&followSymlink, "follow-symlink", false, "follow symlink when stat'ing files")
	flag.IntVar(&maxDepth, "max-depth", -1, "max recursive depth relative to start (-1 = unlimited)")
	flag.BoolVar(&outputToStdOut, "stdout", false, "Output to standard out")
	flag.StringVar(&outputName, "output", "output", "output file name")
}

func main() {
	flag.Parse()
	stat, _ := os.Stdin.Stat()
	if !outputToStdOut {
		fileChannel = make(chan model.FileInfoRecord)
	}

	users, userErr := lshound_users.GetAllUsers()
	if userErr != nil {
		log.Fatal(userErr)
	}

	groups, groupErr := lshound_groups.GetAllGroups()
	if groupErr != nil {
		log.Fatal(groupErr)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		fromStdin()
	} else {
		if startPath == "" {
			startPath = "."
		}

		wg.Add(1)
		go walk(startPath, maxDepth, followSymlink, checkACL, !doJSON, fileChannel)
	}

	wg.Add(1)
	go runOutput(users, groups, fileChannel)
	wg.Wait()

	fmt.Println("Graph created and output!")
}

func runOutput(users []model.User, groups []model.Group, fileChannel chan model.FileInfoRecord) {
	graph := writer.CreateGraph(users, groups, fileChannel)
	graphJSON, _ := json.MarshalIndent(graph, "", "  ")
	f, _ := os.Create(fmt.Sprintf("%v.json", outputName))
	defer f.Close()
	f.Write(graphJSON)
	wg.Done()
}

func walk(startPath string, maxDepth int, followSymlink bool, checkACL bool, doJSON bool, fileChannel chan model.FileInfoRecord) {
	if err := lshound_files.Walk(startPath, maxDepth, followSymlink, checkACL, !doJSON, fileChannel, emit()); err != nil {
		fmt.Fprintln(os.Stderr, "walk error: ", err)
		wg.Done()
		os.Exit(1)
	}
	wg.Done()
}
