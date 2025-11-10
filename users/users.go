// Package users contians functions and data pertaining to users of the system.
package users

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	model "github.com/mykeelium/lshound/model"
)

func GetAllUsers() ([]model.User, error) {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var users []model.User
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) >= 7 {
			uid, err := strconv.ParseUint(fields[2], 10, 32)
			if err != nil {
				log.Printf("Warning: invalid UID for user %s: %v", fields[0], err)
				continue
			}

			gid, err := strconv.ParseUint(fields[3], 10, 32)
			if err != nil {
				log.Printf("Warning: invalid GID for user %s: %v", fields[0], err)
				continue
			}

			users = append(users, model.User{
				Username: fields[0],
				UID:      uint32(uid),
				GID:      uint32(gid),
				Home:     fields[5],
				Shell:    fields[6],
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return users, nil
}
