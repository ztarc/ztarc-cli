package securefile

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestAtomicCredentialBoundary(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	path := filepath.Join(dir, "client.json")
	payload := []byte(`{"id":"client","secret":"test-secret","endpoint":"https://test.invalid","org":"org"}`)
	if err := Write(path, payload); err != nil {
		t.Fatal(err)
	}
	for p, want := range map[string]os.FileMode{dir: 0700, path: 0600} {
		st, e := os.Stat(p)
		if e != nil || st.Mode().Perm() != want {
			t.Fatalf("unsafe mode %s: %v", p, e)
		}
	}
	if _, err := ReadMachine(path, false); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, payload); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadMachine(path, false); err == nil {
		t.Fatal("accepted world-readable secret")
	}
	if err := Write(path, payload); err == nil {
		t.Fatal("overwrote unsafe existing credential")
	}
}
func TestRejectSymlinksHardlinksAndLargeFiles(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	target := filepath.Join(dir, "target")
	if err := Write(target, []byte("keep")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, e := Read(link, false); e == nil {
		t.Fatal("read symlink")
	}
	if e := Write(link, []byte("bad")); e == nil {
		t.Fatal("wrote symlink")
	}
	via := filepath.Join(dir, "via")
	os.Symlink(dir, via)
	if _, e := Read(filepath.Join(via, "target"), false); e == nil {
		t.Fatal("followed directory symlink")
	}
	os.Link(target, filepath.Join(dir, "hard"))
	if _, e := Read(target, false); e == nil {
		t.Fatal("accepted hardlink")
	}
	large := filepath.Join(dir, "large")
	os.WriteFile(large, []byte(strings.Repeat("x", MaxSize+1)), 0600)
	if _, e := Read(large, false); e == nil {
		t.Fatal("accepted oversized file")
	}
	b, _ := os.ReadFile(target)
	if string(b) != "keep" {
		t.Fatal("modified target")
	}
}
func TestMachineJSONRejectsAmbiguity(t *testing.T) {
	for _, content := range []string{`{}`, `{"id":"i","secret":"s","endpoint":"e","org":"o","user-token":"bad"}`, `{"id":"i","secret":"s","endpoint":"e","org":"o"} {}`} {
		dir := t.TempDir()
		os.Chmod(dir, 0700)
		path := filepath.Join(dir, "credential")
		if err := Write(path, []byte(content)); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadMachine(path, false); err == nil {
			t.Fatal("accepted malformed credentials")
		}
	}
}

// Run as root as well to exercise the actual elevated descriptor ownership change.
func TestElevatedLogOwnershipAcrossRotation(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root; also run in isolated container")
	}
	const uid = 1000
	t.Setenv("SUDO_UID", "1000")
	dir, err := os.MkdirTemp("", "ztarc-log-owner-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	// The user prepared these directories before elevation.
	if err := os.Chmod(dir, 0711); err != nil {
		t.Fatal(err)
	}
	private := filepath.Join(dir, "logs")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(private, uid, -1); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(private, "client.log")
	for i := 0; i < 2; i++ {
		f, err := AppendForUser(path, uid)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.WriteString("log entry\n"); err != nil {
			t.Fatal(err)
		}
		f.Close()
		assertUserCanReadLog(t, path, uid)
		if i == 0 {
			rotated := filepath.Join(private, "client-yesterday.log")
			if err := os.Rename(path, rotated); err != nil {
				t.Fatal(err)
			}
			assertUserCanReadLog(t, rotated, uid)
		}
	}
	// Repair an existing root-owned 0600 log from an older runtime.
	if err := os.Chown(path, 0, -1); err != nil {
		t.Fatal(err)
	}
	f, err := AppendForUser(path, uid)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	assertUserCanReadLog(t, path, uid)
}
func assertUserCanReadLog(t *testing.T, path string, uid int) {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0600 || int(st.Sys().(*syscall.Stat_t).Uid) != uid {
		t.Fatalf("unexpected log ownership/mode: %v", st)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestLogReadHelper$")
	cmd.Env = append(os.Environ(), "ZTARC_TEST_LOG="+path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(uid)}}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("initiating user cannot read log: %v %s", err, out)
	}
}
func TestLogReadHelper(t *testing.T) {
	if path := os.Getenv("ZTARC_TEST_LOG"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil || string(b) != "log entry\n" {
			t.Fatalf("read log: %q %v", b, err)
		}
	}
}
func TestAppendForUserRejectsUnsafeTargets(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	target := filepath.Join(dir, "target")
	os.WriteFile(target, []byte("keep"), 0600)
	link := filepath.Join(dir, "link")
	os.Symlink(target, link)
	if f, err := AppendForUser(link, os.Geteuid()); err == nil {
		f.Close()
		t.Fatal("accepted symlink")
	}
	os.Link(target, filepath.Join(dir, "hard"))
	if f, err := AppendForUser(target, os.Geteuid()); err == nil {
		f.Close()
		t.Fatal("accepted hardlink")
	}
}
