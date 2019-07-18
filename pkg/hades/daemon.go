package hades

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/shlex"
)

// time to wait between restarts of failed daemon.
const timeout = time.Second * 10

// Daemon represents a single daemon process.
type Daemon struct {
	ID       uint64 `json:"id"`
	Cmd      string `json:"cmd"`
	Dir      string `json:"dir,omitempty"`
	Status   string `json:"status"`
	Disabled bool   `json:"disabled"`
}

// activeDaemon represents a running daemon.
type activeDaemon struct {
	h         *Hades
	id        uint64
	pidMutex  *sync.Mutex
	pid       int
	exitMutex *sync.Mutex
	exit      bool
}

// newActiveDaemon returns new activeDaemon, starting the process.
func newActiveDaemon(h *Hades, id uint64) *activeDaemon {
	ad := &activeDaemon{
		h:         h,
		id:        id,
		pidMutex:  &sync.Mutex{},
		pid:       0,
		exitMutex: &sync.Mutex{},
		exit:      false,
	}
	ad.setStatus("running")
	go ad.start()
	return ad
}

// setStatus updates daemon status in DB.
func (ad *activeDaemon) setStatus(status string) {
	ad.h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		v := b.Get(itob(ad.id))
		d := &Daemon{}
		err := json.Unmarshal(v, d)
		if err != nil {
			return err
		}
		d.Status = status
		enc, err := json.Marshal(d)
		if err != nil {
			return err
		}
		return b.Put(itob(ad.id), enc)
	})
}

// cleanup called after daemon is stopped to update DB and remove from active.
func (ad *activeDaemon) cleanup() {
	h := ad.h
	h.activeMutex.Lock()
	defer h.activeMutex.Unlock()
	delete(h.active, ad.id)
	h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		v := b.Get(itob(ad.id))
		d := &Daemon{}
		err := json.Unmarshal(v, d)
		if err != nil {
			return err
		}
		d.Disabled = true
		d.Status = "stopped"
		enc, err := json.Marshal(d)
		if err != nil {
			return err
		}
		return b.Put(itob(ad.id), enc)
	})
}

// start starts a daemon process and schedules cleanup when it's stopped.
func (ad *activeDaemon) start() {
	defer ad.cleanup()
	h := ad.h
	id := ad.id
	d, err := h.Get(id)
	if err != nil {
		return
	}
	parts, err := shlex.Split(d.Cmd)
	if err != nil {
		return
	}
	if len(parts) == 0 {
		return
	}
	dir := d.Dir
	if strings.HasPrefix(dir, "~") {
		// expand relative home paths
		usr, err := user.Current()
		if err != nil {
			return
		}
		dir = filepath.Join(usr.HomeDir, dir[1:])
	}
	// make paths absolute
	dir, err = filepath.Abs(dir)
	if err != nil {
		return
	}
	for {
		ad.exitMutex.Lock()
		exit := ad.exit
		ad.exitMutex.Unlock()
		if exit {
			return
		}
		c := exec.Command(parts[0], parts[1:]...)
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		c.Dir = dir
		c.Env = os.Environ()
		err := c.Start()
		if err != nil {
			ad.setStatus("failed")
			log.Printf("%d: %s\n", ad.id, err)
			time.Sleep(timeout)
			continue
		}
		ad.pidMutex.Lock()
		ad.pid = c.Process.Pid
		ad.pidMutex.Unlock()
		err = c.Wait()
		if err != nil {
			continue
		}
		ad.setStatus("running")
	}
}

// sigkill sends KILL signal to activeDaemon and updates status.
func (ad *activeDaemon) sigkill() error {
	ad.exitMutex.Lock()
	defer ad.exitMutex.Unlock()
	ad.setStatus("stopping")
	ad.exit = true
	if ad.pid == 0 {
		// This happens when the process fails
		return errors.New("daemon: bad pid")
	}
	err := syscall.Kill(-ad.pid, syscall.SIGKILL)
	if err != nil {
		return err
	}
	return nil
}

// sigstop sends STOP signal to activeDaemon and updates status.
func (ad *activeDaemon) sigstop() error {
	err := syscall.Kill(-ad.pid, syscall.SIGSTOP)
	if err != nil {
		return err
	}
	ad.setStatus("paused")
	return nil
}

// sigcont sends CONT signal to activeDaemon and updates status.
func (ad *activeDaemon) sigcont() error {
	err := syscall.Kill(-ad.pid, syscall.SIGCONT)
	if err != nil {
		return err
	}
	ad.setStatus("running")
	return nil
}
