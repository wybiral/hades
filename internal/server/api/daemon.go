package api

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
	"github.com/google/shlex"
	"github.com/wybiral/hades/pkg/types"
)

// time to wait between restarts of failed daemon.
const timeout = time.Second * 10

// activeDaemon represents a running daemon.
type activeDaemon struct {
	api       *API
	key       string
	pidMutex  *sync.Mutex
	pid       int
	exitMutex *sync.Mutex
	exit      bool
}

// newActiveDaemon returns new activeDaemon, starting the process.
func newActiveDaemon(api *API, key string) *activeDaemon {
	ad := &activeDaemon{
		api:       api,
		key:       key,
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
	ad.api.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		v := b.Get([]byte(ad.key))
		d := &types.Daemon{}
		err := json.Unmarshal(v, d)
		if err != nil {
			return err
		}
		d.Status = status
		enc, err := json.Marshal(d)
		if err != nil {
			return err
		}
		return b.Put([]byte(ad.key), enc)
	})
}

// cleanup called after daemon is stopped to update DB and remove from active.
func (ad *activeDaemon) cleanup() {
	api := ad.api
	api.activeMutex.Lock()
	defer api.activeMutex.Unlock()
	delete(api.active, ad.key)
	ad.api.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(daemonBucket)
		v := b.Get([]byte(ad.key))
		d := &types.Daemon{}
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
		return b.Put([]byte(ad.key), enc)
	})
}

// start starts a daemon process and schedules cleanup when it's stopped.
func (ad *activeDaemon) start() {
	defer ad.cleanup()
	api := ad.api
	key := ad.key
	d, err := api.GetDaemon(key)
	if err != nil {
		return
	}
	parts, err := shlex.Split(d.Cmd)
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
		c.Dir = d.Dir
		c.Env = os.Environ()
		err := c.Start()
		if err != nil {
			log.Println(ad.key+":", err)
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
