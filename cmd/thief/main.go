package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"thief/internal/app"
	"thief/internal/library"
	"thief/internal/native"
)

func main() {
	runtime.LockOSThread()
	root := os.Getenv("THIEF_DATA_DIR")
	if root == "" {
		home, e := os.UserHomeDir()
		if e != nil {
			log.Fatal(e)
		}
		root = filepath.Join(home, "Library", "Application Support", "thief")
	}
	store, e := library.Open(root)
	if e != nil {
		log.Fatal(e)
	}
	controller, e := app.NewController(store, native.Measurer{})
	if e != nil {
		log.Fatal(e)
	}
	publish := func() {
		data, e := json.Marshal(controller.Snapshot())
		if e == nil {
			native.Apply(string(data))
		}
	}
	controller.SetNotify(publish)
	events := make(chan native.Event, 256)
	go func() {
		for event := range events {
			var cmd app.Command
			if e := json.Unmarshal([]byte(event.Payload), &cmd); e != nil {
				log.Print(e)
				continue
			}
			cmd.Kind = event.Kind
			switch event.Kind {
			case "started":
				publish()
			case "quit":
				if e := controller.Close(); e != nil {
					log.Printf("保存阅读位置失败: %v", e)
				}
				native.Quit()
				return
			case "chapters":
				rows, e := controller.Chapters(cmd.BookID, cmd.Query)
				payload := map[string]any{"BookID": cmd.BookID, "Chapters": rows}
				if e != nil {
					payload["Error"] = e.Error()
				} else {
					books, _ := store.List()
					for _, b := range books {
						if b.ID == cmd.BookID {
							payload["Title"] = b.Title
						}
					}
				}
				raw, _ := json.Marshal(payload)
				native.Chapters(string(raw))
			default:
				controller.Dispatch(cmd)
			}
		}
	}()
	native.Run(func(e native.Event) { events <- e })
}
