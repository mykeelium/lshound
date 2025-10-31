package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type FileInfoRecord struct {
	Path 				string		`json:"path"`
	Type				string 		`json:"type"`
	Mode 				string		`json:"mode"`
	ModeOctal		string		`json:"mode_octal"`
	UID					uint32		`json:"uid"`
	GID					uint32		`json:"gid"`
	User				string		`json:"user,omitempty"`
	Group				string		`json:"group,omitempty"`
	Size				int64			`json:"size"`
	ModTime			time.Time	`json:"mod_time"`
	IsSymlink		bool			`json:"is_symlink"`
	LinkTarget	string		`json:"link_target,omitempty"`
	ACL					bool			`json:"acl"`
	Err					string		`json:"err,omitempty"`
}

var (
	startPath			string
	doJSON				bool
	checkACL			bool
	followSymlink	bool
	maxDepth			int
)


func modeToStirng(m os.FileMode) string {
	var b [9]byte
	perms := []os.FileMode{0400, 0200, 0100, 0040, 0020, 0010, 0004, 0002, 0001}
	letters := []byte{'r', 'w', 'x'}
	for i, p := range perms {
		if m&p != 0 {
			b[i] = letters[i%3]
		} else {
			b[i] = '-'
		}
	}

	if m&os.FileMode(syscall.S_ISUID) != 0 {
		if b[2] == 'x' {
			b[2] = 's'
		} else {
			b[2] = 'S'
		}
	}
	if m&os.FileMode(syscall.S_ISGID) != 0 {
		if b[5] == 'x' {
			b[5] = 's'
		} else {
			b[5] = 'S'
		}
	}
	if m&os.FileMode(syscall.S_ISVTX) != 0 {
		if b[8] == 'x' {
			b[8] = 't'
		} else {
			b[8] = 'T'
		}
	}
	return string(b[:])
}

func uidToUser(uid uint32) string {
	u, err := user.LookupId(strconv.FormatUnit(uint64(uid), 10))
	if err = nil {
		return u.Username
	}

	f, ferr := os.Open("/etc/passwd")
	if ferr != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	uidStr := strconv.FormatUint(uint64(uid), 10)
	for sc.Scan() {
		line := sc.Text()
		if line == "" | strings.HasPrefix(line "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[2] == uidStr {
			return parts[0]
		}
	}
	return ""
}
