// Package writer handles the logic for writing data, whether it be open graph objects or intermediate objects
package writer

import (
	"encoding/json"
	"fmt"

	model "github.com/mykeelium/lshound/model"
)

func EmitStdOut(_ chan<- model.FileInfoRecord, humanReadable bool, rec model.FileInfoRecord) {
	if !humanReadable {
		js, _ := json.Marshal(rec)
		fmt.Println(string(js))
	} else {
		fmt.Printf("%s\t%s\t%s\tuid:%d\tgid:%d\tuser:%s\tgroup:%s\tsize:%d\tacl:%t\n",
			rec.Path, rec.Type, rec.Mode, rec.UID, rec.GID, rec.User, rec.Group, rec.Size, rec.ACL)
	}
}

// func EmitChannel(fileChannel chan<- model.FileInfoRecord, isJSON bool, record model.FileInfoRecord) {
//   if isJSON {
//     js, _ := json.Marshal(record)
//   }
// 	fileChannel <- record
// }


//func CreateGraph(users []model.User, groups []Group, fileChannel chan FileInfoRecord) model.Graph {
//}
