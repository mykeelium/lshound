// Package groups is used to contain the data and methods that pertain to groups
package groups

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	model "github.com/mykeelium/lshound/model"
)

func GetAllGroups() ([]model.Group, error) {
	file, err := os.Open("/etc//group")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var groups []model.Group
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) >= 3 {
			gid, err := strconv.ParseUint(fields[2], 10, 32)
			if err != nil {
				log.Printf("Warning: invalid GID for group %s: %v", fields[0], err)
				continue
			}

			var members []string
			if len(fields) >= 4 && fields[3] != "" {
				members = strings.Split(fields[3], ",")
			}

			groups = append(groups, model.Group{
				Name:    fields[0],
				GID:     uint32(gid),
				Members: members,
			})
		}
	}

	return groups, scanner.Err()
}
