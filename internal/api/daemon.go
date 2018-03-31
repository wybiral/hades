package api

import (
	"errors"
	"github.com/google/shlex"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const timeout = time.Second * 10

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

func (ad *activeDaemon) sigkill() error {
	ad.exitMutex.Lock()
	ad.setStatus("stopping")
	defer ad.exitMutex.Unlock()
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
