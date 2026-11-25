//go:build linux

// Package securefile owns the Linux credential filesystem boundary.
package securefile

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fosrl/cli/internal/redact"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const MaxSize = 1 << 20

// Parent traversal uses directory descriptors and never follows a symlink.
func parent(path string, create bool, rootOnly ...bool) (int, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return -1, "", err
	}
	parts := strings.Split(strings.TrimPrefix(filepath.Clean(abs), "/"), "/")
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, "", err
	}
	for _, part := range parts[:len(parts)-1] {
		next, e := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if errors.Is(e, unix.ENOENT) && create {
			e = unix.Mkdirat(fd, part, 0700)
			if e == nil || errors.Is(e, unix.EEXIST) {
				next, e = unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
			}
		}
		unix.Close(fd)
		if e != nil {
			return -1, "", fmt.Errorf("unsafe or inaccessible credential directory: %w", e)
		}
		fd = next
		var st unix.Stat_t
		if unix.Fstat(fd, &st) != nil {
			unix.Close(fd)
			return -1, "", errors.New("cannot inspect credential directory")
		}
		// Root-owned sticky directories (e.g. /tmp) are safe for a private child.
		if st.Uid != 0 && (len(rootOnly) > 0 && rootOnly[0] || !allowedOwner(st.Uid, false)) || st.Mode&0022 != 0 && (st.Mode&unix.S_ISVTX == 0 || st.Uid != 0) {
			unix.Close(fd)
			return -1, "", errors.New("credential directory has unsafe ownership or write permissions")
		}
	}
	return fd, parts[len(parts)-1], nil
}
func allowedOwner(uid uint32, rootOnly bool) bool {
	if rootOnly {
		return uid == 0
	}
	if int(uid) == os.Geteuid() {
		return true
	}
	if os.Geteuid() == 0 {
		n, e := strconv.ParseUint(os.Getenv("SUDO_UID"), 10, 32)
		return e == nil && uid == uint32(n)
	}
	return false
}
func check(fd int, rootOnly bool) error {
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Mode&0077 != 0 || !allowedOwner(st.Uid, rootOnly) || st.Nlink != 1 {
		return errors.New("credential file must be regular, owner-only, singly linked, and owned by the expected user")
	}
	if rootOnly && st.Mode&0777 != 0600 {
		return errors.New("service credential file must have mode 0600")
	}
	return nil
}
func Read(path string, rootOnly bool) ([]byte, error) {
	fd, name, err := parent(path, false, rootOnly)
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)
	leaf, err := unix.Openat(fd, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(leaf), name)
	defer f.Close()
	if err = check(leaf, rootOnly); err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxSize+1))
	if len(b) > MaxSize {
		return nil, errors.New("credential file exceeds 1 MiB")
	}
	return b, err
}
func EnsureDir(path string) error {
	fd, _, err := parent(filepath.Join(path, ".check"), true)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil {
		return err
	}
	if !allowedOwner(st.Uid, false) || st.Mode&0077 != 0 {
		return errors.New("private directory must be owner-only")
	}
	return nil
}
func Write(path string, data []byte) error {
	if len(data) > MaxSize {
		return errors.New("credential payload exceeds 1 MiB")
	}
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	fd, name, err := parent(path, true)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	existing, err := unix.Openat(fd, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err == nil {
		err = check(existing, false)
		unix.Close(existing)
		if err != nil {
			return err
		}
	} else if !errors.Is(err, unix.ENOENT) {
		return err
	}
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		return err
	}
	tmp := fmt.Sprintf(".ztarc-%x", nonce)
	leaf, err := unix.Openat(fd, tmp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	defer unix.Unlinkat(fd, tmp, 0)
	f := os.NewFile(uintptr(leaf), tmp)
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = unix.Renameat(fd, tmp, fd, name); err != nil {
		return err
	}
	return unix.Fsync(fd)
}
func Append(path string) (*os.File, error) { return appendFile(path, nil) }

// AppendForUser keeps elevated runtime logs readable by their initiating user.
// Ownership changes use the checked open descriptor, never a path-based chown.
func AppendForUser(path string, uid int) (*os.File, error) {
	if uid < 0 || !allowedOwner(uint32(uid), false) {
		return nil, errors.New("unexpected log owner")
	}
	return appendFile(path, &uid)
}
func appendFile(path string, owner *int) (*os.File, error) {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	fd, name, err := parent(path, true)
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)
	if owner != nil {
		var st unix.Stat_t
		if err = unix.Fstat(fd, &st); err != nil {
			return nil, err
		}
		if int(st.Uid) != *owner {
			return nil, errors.New("log directory must belong to the initiating user")
		}
	}
	leaf, err := unix.Openat(fd, name, unix.O_WRONLY|unix.O_APPEND|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	if err = check(leaf, false); err != nil {
		unix.Close(leaf)
		return nil, err
	}
	if owner != nil && os.Geteuid() == 0 {
		if err = unix.Fchown(leaf, *owner, -1); err != nil {
			unix.Close(leaf)
			return nil, err
		}
	}
	return os.NewFile(uintptr(leaf), path), nil
}

type Machine struct {
	ID       string `json:"id"`
	Secret   string `json:"secret"`
	Endpoint string `json:"endpoint"`
	Org      string `json:"org"`
}

func ReadMachine(path string, rootOnly bool) (Machine, error) {
	var c Machine
	b, err := Read(path, rootOnly)
	if err != nil {
		return c, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&c); err != nil {
		return Machine{}, errors.New("invalid machine credential JSON")
	}
	var trailing any
	if dec.Decode(&trailing) != io.EOF {
		return Machine{}, errors.New("trailing machine credential data")
	}
	if c.ID == "" || c.Secret == "" || c.Endpoint == "" || c.Org == "" {
		return Machine{}, errors.New("machine credential file requires id, secret, endpoint, org")
	}
	redact.Register(c.Secret)
	return c, nil
}
