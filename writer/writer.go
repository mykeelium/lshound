// Package writer handles the logic for writing data, whether it be open graph objects or intermediate objects
package writer

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	model "github.com/mykeelium/lshound/model"
)

func EmitStdOut(_ chan<- model.FileInfoRecord, writeJSON bool, rec model.FileInfoRecord) {
	if writeJSON {
		js, _ := json.MarshalIndent(rec, "", "  ")
		fmt.Println(string(js))
	} else {
		fmt.Printf("%s\t%s\t%s\tuid:%d\tgid:%d\tuser:%s\tgroup:%s\tsize:%d\tacl:%t\n",
			rec.Path, rec.Type, rec.Mode, rec.UID, rec.GID, rec.User, rec.Group, rec.Size, rec.ACL)
	}
}

func EmitChannel(fileChannel chan<- model.FileInfoRecord, isJSON bool, record model.FileInfoRecord) {
	fileChannel <- record
}

func CreateGraph(users []model.User, groups []model.Group, fileChannel chan model.FileInfoRecord) model.GraphEnvelope {
	nodes := []model.Node{}
	edges := []model.Edge{}
	for _, group := range groups {
		nodes = append(nodes, model.Node{
			ID:    fmt.Sprintf("gid-%d", group.GID),
			Kinds: []string{"Group"},
			Title: group.Name,
			Properties: map[string]string{
				"name": group.Name,
			},
		})
		for _, member := range group.Members {
			for _, user := range users {
				if member == user.Username {
					edges = append(edges, model.Edge{
						Kind: "InGroup",
						Start: model.Connection{
							Value:   fmt.Sprintf("uid-%d", user.UID),
							MatchBy: "id",
						},
						End: model.Connection{
							Value:   fmt.Sprintf("gid-%d", group.GID),
							MatchBy: "id",
						},
					})
				}
			}
		}
	}
	for _, user := range users {
		nodes = append(nodes, model.Node{
			ID:    fmt.Sprintf("uid-%d", user.UID),
			Kinds: []string{"User"},
			Title: user.Username,
			Properties: map[string]string{
				"name":  user.Username,
				"shell": user.Shell,
				"home":  user.Home,
				"gid":   fmt.Sprintf("%d", user.GID),
			},
		})

		edges = append(edges, model.Edge{
			Kind: "InGroup",
			Start: model.Connection{
				Value:   fmt.Sprintf("uid-%d", user.UID),
				MatchBy: "id",
			},
			End: model.Connection{
				Value:   fmt.Sprintf("gid-%d", user.GID),
				MatchBy: "id",
			},
		})
	}

	nodes = append(nodes, model.Node{
		ID:          "uid-other",
		Kinds:       []string{"User"},
		Title:       "Other",
		Description: "This is used for all other users that are not the owner or in the group for a specific file",
	})

	for file := range fileChannel {
		nodes = append(nodes, model.Node{
			ID:    fmt.Sprintf("inode-%d", file.INode),
			Title: file.Path,
			Kinds: []string{file.Type},
			Properties: map[string]string{
				"name":        file.Path,
				"type":        file.Type,
				"mode_string": file.ModeString,
				"mode_octal":  file.ModeOctal,
				"uid":         fmt.Sprintf("uid-%d", file.UID),
				"owner":       file.User,
				"gid":         fmt.Sprintf("gid-%d", file.GID),
				"group":       file.Group,
				"is_sym_link": strconv.FormatBool(file.IsSymlink),
				"link_target": file.LinkTarget,
				"size":        fmt.Sprintf("%d", file.Size),
			},
		})

		edges = append(edges, model.Edge{
			Kind: "Owns",
			Start: model.Connection{
				Value:   fmt.Sprintf("uid-%d", file.UID),
				MatchBy: "id",
			},
			End: model.Connection{
				Value:   fmt.Sprintf("inode-%d", file.INode),
				MatchBy: "id",
			},
		})

		edges = append(edges, model.Edge{
			Kind: "Owns",
			Start: model.Connection{
				Value:   fmt.Sprintf("gid-%d", file.GID),
				MatchBy: "id",
			},
			End: model.Connection{
				Value:   fmt.Sprintf("inode-%d", file.INode),
				MatchBy: "id",
			},
		})

		if ownerCanExecute(file.Mode) {
			edges = append(edges, model.Edge{
				Kind: "CanExecute",
				Start: model.Connection{
					Value:   fmt.Sprintf("uid-%d", file.UID),
					MatchBy: "id",
				},
				End: model.Connection{
					Value:   fmt.Sprintf("inode-%d", file.INode),
					MatchBy: "id",
				},
			})
		}

		if ownerCanWrite(file.Mode) {
			edges = append(edges, model.Edge{
				Kind: "CanWrite",
				Start: model.Connection{
					Value:   fmt.Sprintf("uid-%d", file.UID),
					MatchBy: "id",
				},
				End: model.Connection{
					Value:   fmt.Sprintf("inode-%d", file.INode),
					MatchBy: "id",
				},
			})
		}

		if ownerCanRead(file.Mode) {
			edges = append(edges, model.Edge{
				Kind: "CanRead",
				Start: model.Connection{
					Value:   fmt.Sprintf("uid-%d", file.UID),
					MatchBy: "id",
				},
				End: model.Connection{
					Value:   fmt.Sprintf("inode-%d", file.INode),
					MatchBy: "id",
				},
			})
		}

		if groupCanExecute(file.Mode) {
			edges = append(edges, model.Edge{
				Kind: "CanExecute",
				Start: model.Connection{
					Value:   fmt.Sprintf("gid-%d", file.GID),
					MatchBy: "id",
				},
				End: model.Connection{
					Value:   fmt.Sprintf("inode-%d", file.INode),
					MatchBy: "id",
				},
			})
		}

		if groupCanWrite(file.Mode) {
			edges = append(edges, model.Edge{
				Kind: "CanWrite",
				Start: model.Connection{
					Value:   fmt.Sprintf("gid-%d", file.GID),
					MatchBy: "id",
				},
				End: model.Connection{
					Value:   fmt.Sprintf("inode-%d", file.INode),
					MatchBy: "id",
				},
			})
		}

		if groupCanRead(file.Mode) {
			edges = append(edges, model.Edge{
				Kind: "CanRead",
				Start: model.Connection{
					Value:   fmt.Sprintf("gid-%d", file.GID),
					MatchBy: "id",
				},
				End: model.Connection{
					Value:   fmt.Sprintf("inode-%d", file.INode),
					MatchBy: "id",
				},
			})
		}

		if othersCanExecute(file.Mode) {
			edges = append(edges, model.Edge{
				Kind: "CanExecute",
				Start: model.Connection{
					Value:   "uid-other",
					MatchBy: "id",
				},
				End: model.Connection{
					Value:   fmt.Sprintf("inode-%d", file.INode),
					MatchBy: "id",
				},
			})
		}

		if othersCanWrite(file.Mode) {
			edges = append(edges, model.Edge{
				Kind: "CanWrite",
				Start: model.Connection{
					Value:   "uid-other",
					MatchBy: "id",
				},
				End: model.Connection{
					Value:   fmt.Sprintf("inode-%d", file.INode),
					MatchBy: "id",
				},
			})
		}

		if othersCanRead(file.Mode) {
			edges = append(edges, model.Edge{
				Kind: "CanRead",
				Start: model.Connection{
					Value:   "uid-other",
					MatchBy: "id",
				},
				End: model.Connection{
					Value:   fmt.Sprintf("inode-%d", file.INode),
					MatchBy: "id",
				},
			})
		}

	}
	return model.GraphEnvelope{
		Graph: model.Graph{
			Nodes: nodes,
			Edges: edges,
		},
	}
}

func ownerCanExecute(mode os.FileMode) bool {
	return mode&0o100 != 0
}

func ownerCanWrite(mode os.FileMode) bool {
	return mode&0o200 != 0
}

func ownerCanRead(mode os.FileMode) bool {
	return mode&0o400 != 0
}

func groupCanExecute(mode os.FileMode) bool {
	return mode&0o010 != 0
}

func groupCanWrite(mode os.FileMode) bool {
	return mode&0o020 != 0
}

func groupCanRead(mode os.FileMode) bool {
	return mode&0o040 != 0
}

func othersCanExecute(mode os.FileMode) bool {
	return mode&0o001 != 0
}

func othersCanWrite(mode os.FileMode) bool {
	return mode&0o002 != 0
}

func othersCanRead(mode os.FileMode) bool {
	return mode&0o004 != 0
}
