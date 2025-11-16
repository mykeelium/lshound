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
	baseCollection bool
	skipACL        bool
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
				fileChannel <- model.FileInfoRecord{Path: path, Err: statErr.Error()}
			} else {
				rec := lshound_files.ProcessPath(path, info, skipACL)
				fileChannel <- rec
			}
		}
		if err == io.EOF {
			break
		}
	}
	close(fileChannel)
	wg.Done()
}

func init() {
	flag.StringVar(&startPath, "path", ".", "starting path")
	flag.BoolVar(&baseCollection, "basecollection", false, "output set to be in mapped to the OpenGraph format by default, use this switch to return the base collection")
	flag.BoolVar(&skipACL, "skip-acl", false, "check for POSIX ACLs using getfacl, set this flag to skip")
	flag.BoolVar(&followSymlink, "follow-symlink", false, "follow symlink when stat'ing files")
	flag.IntVar(&maxDepth, "max-depth", -1, "max recursive depth relative to start (-1 = unlimited)")
	flag.BoolVar(&outputToStdOut, "stdout", false, "Output to standard out")
	flag.StringVar(&outputName, "output", "output", "output file name")
}

func main() {
	flag.Parse()
	stat, _ := os.Stdin.Stat()
	fileChannel = make(chan model.FileInfoRecord)

	users, userErr := lshound_users.GetAllUsers()
	if userErr != nil {
		log.Fatal(userErr)
	}

	groups, groupErr := lshound_groups.GetAllGroups()
	if groupErr != nil {
		log.Fatal(groupErr)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		wg.Add(1)
		go fromStdin()
	} else {
		if startPath == "" {
			startPath = "."
		}

		wg.Add(1)
		go walk(startPath, maxDepth, followSymlink, skipACL, fileChannel)
	}

	wg.Add(1)
	go runOutput(users, groups, fileChannel)
	wg.Wait()

	if !outputToStdOut {
		fmt.Println("Graph created and output!")
	}
}

func runOutput(users []model.User, groups []model.Group, fileChannel chan model.FileInfoRecord) {
	var rec any
	if baseCollection {
		rec = writer.CreateBaseCollection(users, groups, fileChannel)
	} else {
		rec = writer.CreateGraph(users, groups, fileChannel)
	}
	outputJSON, _ := json.MarshalIndent(rec, "", "  ")

	if outputToStdOut {
		fmt.Println(string(outputJSON))
	} else {
		f, _ := os.Create(fmt.Sprintf("%v.json", outputName))
		defer f.Close()
		f.Write(outputJSON)
	}
	wg.Done()
}

func walk(startPath string, maxDepth int, followSymlink bool, skipACL bool, fileChannel chan model.FileInfoRecord) {
	if err := lshound_files.Walk(startPath, maxDepth, followSymlink, skipACL, fileChannel); err != nil {
		fmt.Fprintln(os.Stderr, "walk error: ", err)
		wg.Done()
		os.Exit(1)
	}
	wg.Done()
}
