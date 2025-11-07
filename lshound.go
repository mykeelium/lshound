package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	lshound_files "github.com/MikeX777/lshound/files"
	lshound_groups "github.com/MikeX777/lshound/groups"
	lshound_users "github.com/MikeX777/lshound/users"
	lshound_writer "github.com/MikeX777/lshound/writer"
)

var (
	startPath     string
	doJSON        bool
	checkACL      bool
	followSymlink bool
	maxDepth      int
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
				lshound_writer.EmitStdOut(!doJSON, lshound_writer.FileInfoRecord{Path: path, Err: statErr.Error()})
			} else {
				rec := lshound_files.ProcessPath(path, info, checkACL)
				lshound_writer.EmitStdOut(!doJSON, rec)
			}
		}
		if err == io.EOF {
			break
		}
	}
}

func init() {
	flag.StringVar(&startPath, "path", ".", "starting path")
	flag.BoolVar(&doJSON, "json", true, "output to json")
	flag.BoolVar(&checkACL, "acl", false, "check for POSIX ACLs using getfacl (if available)")
	flag.BoolVar(&followSymlink, "follow-symlink", false, "follow symlink when stat'ing files")
	flag.IntVar(&maxDepth, "max-depth", -1, "max recursive depth relative to start (-1 = unlimited)")
}

func main() {
	flag.Parse()
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		fromStdin()
	} else {
		if startPath == "" {
			startPath = "."
		}
		if err := lshound_files.Walk(startPath, maxDepth, followSymlink, checkACL, !doJSON, lshound_writer.EmitStdOut); err != nil {
			fmt.Fprintln(os.Stderr, "walk error: ", err)
			os.Exit(1)
		}
	}

	_, userErr := lshound_users.GetAllUsers()
	if userErr != nil {
		log.Fatal(userErr)
	}
	_, groupErr := lshound_groups.GetAllGroups()
	if groupErr != nil {
		log.Fatal(groupErr)
	}
}
