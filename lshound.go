package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
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

var (
	startPath     string
	doJSON        bool
	checkACL      bool
	followSymlink bool
	maxDepth      int
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
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err == nil {
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
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[2] == uidStr {
			return parts[0]
		}
	}
	return ""
}

func gidToGroup(gid uint32) string {
	g, err := user.LookupGroupId(strconv.FormatUint(uint64(gid), 10))
	if err == nil {
		return g.Name
	}
	f, ferr := os.Open("/etc/group")
	if ferr != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	gidStr := strconv.FormatUint(uint64(gid), 10)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[2] == gidStr {
			return parts[0]
		}
	}
	return ""
}

func detectACL(path string) (bool, error) {
	_, err := exec.LookPath("getfacl")
	if err != nil {
		return false, nil
	}

	cmd := exec.Command("getfacl", "p", path)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			out = append(out, ee.Stderr...)
		} else {
			return false, err
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasPrefix(line, "user:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 3 && parts[1] != "" {
				if parts[1] != "" && parts[1] != "0" {
					if parts[1] != "" {
						return true, nil
					}
				}
			}
		}
		if strings.HasPrefix(line, "group:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 3 && parts[1] != "" {
				return true, nil
			}
		}
		if strings.HasPrefix(line, "mask:") || strings.HasPrefix(line, "default:") {
			return true, nil
		}
	}
	return false, nil
}

func processPath(path string, info os.FileInfo) FileInfoRecord {
	rec := FileInfoRecord{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
	mode := info.Mode()
	rec.Mode = modeToStirng(mode)
	rec.ModeOctal = fmt.Sprintf("%#o", uint32(mode.Perm()))
	rec.IsSymlink = (mode & os.ModeSymlink) != 0
	if rec.IsSymlink {
		rec.Type = "syslink"
		if tgt, err := os.Readlink(path); err == nil {
			rec.LinkTarget = tgt
		}
	} else if mode.IsDir() {
		rec.Type = "dir"
	} else if mode.IsRegular() {
		rec.Type = "file"
	} else {
		rec.Type = "other"
	}

	var stat syscall.Stat_t
	if err := syscall.Lstat(path, &stat); err == nil {
		rec.UID = uint32(stat.Uid)
		rec.GID = uint32(stat.Gid)
		rec.User = uidToUser(rec.UID)
		rec.Group = gidToGroup(rec.GID)
		rec.INode = uint64(stat.Ino)
	} else {
		rec.Err = err.Error()
	}

	if checkACL {
		acl, err := detectACL(path)
		if err != nil {
			if rec.Err == "" {
				rec.Err = err.Error()
			} else {
				rec.Err = rec.Err + "; " + err.Error()
			}
		}
		rec.ACL = acl
	}

	return rec
}

func walk(root string) error {
	rootAbs, err := filepath.Abs(root)
	if err == nil {
		root = rootAbs
	}
	rootDepth := len(strings.Split(filepath.Clean(root), string(os.PathSeparator)))

	return filepath.WalkDir(root, func(path string, dEntry os.DirEntry, err error) error {
		if err != nil {
			rec := FileInfoRecord{Path: path, Err: err.Error()}
			emit(rec)
			return nil
		}

		if maxDepth >= 0 {
			curDepth := len(strings.Split(filepath.Clean(path), string(os.PathSeparator)))
			if curDepth-rootDepth > maxDepth {
				if dEntry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		var info os.FileInfo
		if followSymlink {
			info, err = os.Stat(path)
		} else {
			info, err = os.Lstat(path)
		}
		if err != nil {
			rec := FileInfoRecord{Path: path, Err: err.Error()}
			emit(rec)
			return nil
		}
		rec := processPath(path, info)
		emit(rec)
		return nil
	})
}

func getAllUsers() ([]User, error) {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var users []User
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

			users = append(users, User{
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

func getAllGroups() ([]Group, error) {
	file, err := os.Open("/etc//group")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var groups []Group
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

			groups = append(groups, Group{
				Name:    fields[0],
				GID:     uint32(gid),
				Members: members,
			})
		}
	}

	return groups, scanner.Err()
}

func emit(rec FileInfoRecord) {
	if doJSON {
		js, _ := json.Marshal(rec)
		fmt.Println(string(js))
	} else {
		fmt.Printf("%s\t%s\t%s\tuid:%d\tgid:%d\tuser:%s\tgroup:%s\tsize:%d\tacl:%t\n",
			rec.Path, rec.Type, rec.Mode, rec.UID, rec.GID, rec.User, rec.Group, rec.Size, rec.ACL)
	}
}

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
				emit(FileInfoRecord{Path: path, Err: statErr.Error()})
			} else {
				rec := processPath(path, info)
				emit(rec)
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
		if err := walk(startPath); err != nil {
			fmt.Fprintln(os.Stderr, "walk error: ", err)
			os.Exit(1)
		}
	}

	_, userErr := getAllUsers()
	if userErr != nil {
		log.Fatal(userErr)
	}
	_, groupErr := getAllGroups()
	if groupErr != nil {
		log.Fatal(groupErr)
	}
}
