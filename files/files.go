// Package files contains the structures and methods used to deal with files
package files

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	model "github.com/mykeelium/lshound/model"
)

func modeToStirng(m os.FileMode) string {
	var b [9]byte
	perms := []os.FileMode{0o400, 0o200, 0o100, 0o040, 0o020, 0o010, 0o004, 0o002, 0o001}
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

func ProcessPath(path string, info os.FileInfo, checkACL bool) model.FileInfoRecord {
	rec := model.FileInfoRecord{
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

func Walk(root string, maxDepth int, followSymlink bool, checkACL bool, humanReadbable bool, out chan<- model.FileInfoRecord, emit model.Emit) error {
	rootAbs, err := filepath.Abs(root)
	if err == nil {
		root = rootAbs
	}
	rootDepth := len(strings.Split(filepath.Clean(root), string(os.PathSeparator)))

	return filepath.WalkDir(root, func(path string, dEntry os.DirEntry, err error) error {
		if err != nil {
			rec := model.FileInfoRecord{Path: path, Err: err.Error()}
			emit(out, humanReadbable, rec)
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
			rec := model.FileInfoRecord{Path: path, Err: err.Error()}
			emit(out, humanReadbable, rec)
			return nil
		}
		rec := ProcessPath(path, info, checkACL)
		emit(out, humanReadbable, rec)
		return nil
	})
}
