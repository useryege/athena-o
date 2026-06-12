//go:build windows

package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	hotkeyStartID = 1
	hotkeyStopID  = 2

	modNoRepeat = 0x4000

	vkF1 = 0x70
	vkF2 = 0x71

	wmHotkey = 0x0312
	wmQuit   = 0x0012

	inputMouse     = 0
	mouseEventLeft = 0x0002
	mouseEventUp   = 0x0004
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterHotKey    = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey  = user32.NewProc("UnregisterHotKey")
	procGetMessage        = user32.NewProc("GetMessageW")
	procPostThreadMessage = user32.NewProc("PostThreadMessageW")
	procSendInput         = user32.NewProc("SendInput")
	procGetCurrentThread  = kernel32.NewProc("GetCurrentThreadId")
)

type point struct {
	x int32
	y int32
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type mouseInput struct {
	dx          int32
	dy          int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type input struct {
	inputType uint32
	mi        mouseInput
}

type clicker struct {
	interval time.Duration

	mu      sync.Mutex
	running bool
	stop    chan struct{}
}

func run(interval time.Duration) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := procGetCurrentThread.Call()
	if err := registerHotkey(hotkeyStartID, vkF1); err != nil {
		return fmt.Errorf("register F1 hotkey: %w", err)
	}
	defer unregisterHotkey(hotkeyStartID)

	if err := registerHotkey(hotkeyStopID, vkF2); err != nil {
		return fmt.Errorf("register F2 hotkey: %w", err)
	}
	defer unregisterHotkey(hotkeyStopID)

	c := &clicker{interval: interval}
	defer c.stopClicking()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	go func() {
		<-signals
		postThreadQuit(uint32(threadID))
	}()

	fmt.Println("auto-clicker is running")
	fmt.Printf("F1: start clicking every %s\n", interval)
	fmt.Println("F2: stop clicking")
	fmt.Println("Ctrl+C: exit")

	for {
		var m msg
		ret, _, err := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		switch int32(ret) {
		case -1:
			return fmt.Errorf("read Windows message: %w", err)
		case 0:
			return nil
		}

		if m.message != wmHotkey {
			continue
		}

		switch m.wParam {
		case hotkeyStartID:
			c.startClicking()
		case hotkeyStopID:
			c.stopClicking()
		}
	}
}

func registerHotkey(id int, vk uint32) error {
	ret, _, err := procRegisterHotKey.Call(0, uintptr(id), modNoRepeat, uintptr(vk))
	if ret == 0 {
		return err
	}
	return nil
}

func unregisterHotkey(id int) {
	procUnregisterHotKey.Call(0, uintptr(id))
}

func postThreadQuit(threadID uint32) {
	procPostThreadMessage.Call(uintptr(threadID), wmQuit, 0, 0)
}

func (c *clicker) startClicking() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return
	}

	stop := make(chan struct{})
	c.stop = stop
	c.running = true
	fmt.Println("clicking started")

	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			sendLeftClick()
			select {
			case <-ticker.C:
			case <-stop:
				return
			}
		}
	}()
}

func (c *clicker) stopClicking() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	close(c.stop)
	c.stop = nil
	c.running = false
	fmt.Println("clicking stopped")
}

func sendLeftClick() {
	inputs := []input{
		{inputType: inputMouse, mi: mouseInput{dwFlags: mouseEventLeft}},
		{inputType: inputMouse, mi: mouseInput{dwFlags: mouseEventUp}},
	}
	procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
}
