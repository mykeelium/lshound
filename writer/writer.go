// Package writer handles the logic for writing data, whether it be open graph objects or intermediate objects
package writer

import (
	// "encoding/json"
	"fmt"

	model "github.com/mykeelium/lshound/model"
)

func EmitStdOut(_ chan<- model.FileInfoRecord, humanReadable bool, rec model.FileInfoRecord) {
	if !humanReadable {
		// js, _ := json.Marshal(rec)
		// fmt.Println(string(js))
	} else {
		// fmt.Printf("%s\t%s\t%s\tuid:%d\tgid:%d\tuser:%s\tgroup:%s\tsize:%d\tacl:%t\n",
		// 	rec.Path, rec.Type, rec.Mode, rec.UID, rec.GID, rec.User, rec.Group, rec.Size, rec.ACL)
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
			Kind: "MemberOf",
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

	// 	for file := range fileChannel {
	// 	}
	return model.GraphEnvelope{
		Graph: model.Graph{
			Nodes: nodes,
			Edges: edges,
		},
	}
}
