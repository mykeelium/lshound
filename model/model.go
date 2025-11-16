// Package model contains shared structs across lshound
package model

import (
	"os"
	"time"
)

type FileInfoRecord struct {
	Path       string      `json:"path"`
	Type       string      `json:"type"`
	Mode       os.FileMode `json:"mode"`
	ModeString string      `json:"mode_string"`
	ModeOctal  string      `json:"mode_octal"`
	UID        uint32      `json:"uid"`
	GID        uint32      `json:"gid"`
	User       string      `json:"user,omitempty"`
	Group      string      `json:"group,omitempty"`
	Size       int64       `json:"size"`
	INode      uint64      `json:"inode"`
	ModTime    time.Time   `json:"mod_time"`
	IsSymlink  bool        `json:"is_symlink"`
	LinkTarget string      `json:"link_target,omitempty"`
	ACL        bool        `json:"acl"`
	Err        string      `json:"err,omitempty"`
}

type User struct {
	Username string `json:"username"`
	UID      uint32 `json:"uid"`
	GID      uint32 `json:"gid"`
	Home     string `json:"home"`
	Shell    string `json:"shell"`
}

type Group struct {
	Name    string   `json:"name"`
	GID     uint32   `json:"gid"`
	Members []string `json:"members"`
}

type CollectionEnvelope struct {
	Users           []User           `json:"users"`
	Groups          []Group          `json:"groups"`
	FileSystemItems []FileInfoRecord `json:"file_system_items"`
}

type GraphEnvelope struct {
	Graph Graph `json:"graph"`
}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID          string            `json:"id"`
	Kinds       []string          `json:"kinds"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

type Connection struct {
	MatchBy string `json:"match_by"`
	Value   string `json:"value"`
}

type Edge struct {
	Start      Connection        `json:"start"`
	End        Connection        `json:"end"`
	Kind       string            `json:"kind"`
	Properties map[string]string `json:"properties,omitempty"`
}
