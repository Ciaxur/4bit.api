package filewatcher

import (
	"log"
	"os"
	"time"
)

type FileWatcher struct {
	Filepath          string      // File to watch for changes.
	ChangeTriggerChan chan uint64 // Channel which holds an 8-bit counter for file changes.
	IsRunning         bool        // State of watcher running.
	successfulClose   chan bool   // Channel that gets triggered on a successful go routine close.
}

func watchFile(fw *FileWatcher) {
	initialStat, err := os.Stat(fw.Filepath)
	changeCounter := uint64(0)
	if err != nil {
		log.Printf("failed to watch file '%s': %v", fw.Filepath, err)
		return
	}

	for fw.IsRunning {
		stat, err := os.Stat(fw.Filepath)
		if err != nil {
			log.Printf("failed to get file '%s''s stat: %v", fw.Filepath, err)
			return
		}

		if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
			changeCounter++
			fw.ChangeTriggerChan <- changeCounter
			initialStat = stat
		}

		time.Sleep(1 * time.Second)
	}

	fw.successfulClose <- true
}

func NewFileWatcher(filepath string) *FileWatcher {
	fw := FileWatcher{
		Filepath:          filepath,
		IsRunning:         true,
		ChangeTriggerChan: make(chan uint64),
		successfulClose:   make(chan bool),
	}

	go watchFile(&fw)
	return &fw
}

func (fw *FileWatcher) Close() {
	if !fw.IsRunning {
		return
	}

	// Break the watcher's loop and wait for the go routine to exit safely.
	fw.IsRunning = false
	<-fw.successfulClose
}
