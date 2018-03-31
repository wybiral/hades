package api

import (
	"github.com/google/shlex"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type activeDaemon struct {
	api       *Api
	key       string
	pidMutex  *sync.Mutex
	pid       int
	exitMutex *sync.Mutex
	exit      bool
}

func newActiveDaemon(api *Api, key string) *activeDaemon {
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

func (ad *activeDaemon) setStatus(status string) {
	ad.api.db.Exec(`
		update Daemon
		set status = ?
		where key = ?
	`, status, ad.key)
}

func (ad *activeDaemon) cleanup() {
	api := ad.api
	api.activeMutex.Lock()
	defer api.activeMutex.Unlock()
	delete(api.active, ad.key)
	api.db.Exec(`
		update Daemon
		set status = 'stopped', disabled = 1
		where key = ?
	`, ad.key)
}

func (ad *activeDaemon) start() {
	defer ad.cleanup()
	api := ad.api
	key := ad.key
	var cmd string
	var dir string
	row := api.db.QueryRow(`
		select cmd, dir
		from Daemon
		where key = ?
	`, key)
	err := row.Scan(&cmd, &dir)
	if err != nil {
		return
	}
	parts, err := shlex.Split(cmd)
	if err != nil {
		return
	}
	for {
		c := exec.Command(parts[0], parts[1:]...)
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		c.Dir = dir
		c.Env = os.Environ()
		err := c.Start()
		if err != nil {
			continue
		}
		ad.pidMutex.Lock()
		ad.pid = c.Process.Pid
		ad.pidMutex.Unlock()
		c.Wait()
		ad.exitMutex.Lock()
		exit := ad.exit
		ad.exitMutex.Unlock()
		if exit {
			return
		}
	}
}

func (ad *activeDaemon) sigkill() error {
	err := syscall.Kill(-ad.pid, syscall.SIGKILL)
	if err != nil {
		return err
	}
	ad.exitMutex.Lock()
	ad.setStatus("stopping")
	defer ad.exitMutex.Unlock()
	ad.exit = true
	return nil
}

func (ad *activeDaemon) sigstop() error {
	err := syscall.Kill(-ad.pid, syscall.SIGSTOP)
	if err != nil {
		return err
	}
	ad.setStatus("paused")
	return nil
}

func (ad *activeDaemon) sigcont() error {
	err := syscall.Kill(-ad.pid, syscall.SIGCONT)
	if err != nil {
		return err
	}
	ad.setStatus("running")
	return nil
}
