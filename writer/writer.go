// Package writer handles the logic for writing data, whether it be open graph objects or intermediate objects
package writer

import (
	"encoding/json"
	"fmt"
	"time"
)

type Emit func(bool, FileInfoRecord)

type FileInfoRecord struct {
	Path       string    `json:"path"`
	Type       string    `json:"type"`
	Mode       string    `json:"mode"`
	ModeOctal  string    `json:"mode_octal"`
	UID        uint32    `json:"uid"`
	GID        uint32    `json:"gid"`
	User       string    `json:"user,omitempty"`
	Group      string    `json:"group,omitempty"`
	Size       int64     `json:"size"`
	INode      uint64    `json:"inode"`
	ModTime    time.Time `json:"mod_time"`
	IsSymlink  bool      `json:"is_symlink"`
	LinkTarget string    `json:"link_target,omitempty"`
	ACL        bool      `json:"acl"`
	Err        string    `json:"err,omitempty"`
}

func EmitStdOut(humanReadable bool, rec FileInfoRecord) {
	if !humanReadable {
		js, _ := json.Marshal(rec)
		fmt.Println(string(js))
	} else {
		fmt.Printf("%s\t%s\t%s\tuid:%d\tgid:%d\tuser:%s\tgroup:%s\tsize:%d\tacl:%t\n",
			rec.Path, rec.Type, rec.Mode, rec.UID, rec.GID, rec.User, rec.Group, rec.Size, rec.ACL)
	}
}
