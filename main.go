package main

import (
	"log"
	"os"
)

func main() {
	// set logfile
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	app := NewApp()
	app.gui.setGui(app.awsr)
	go func() {
		app.awsr.getLogGroups(app.gui.logGroup)
		app.gui.tvApp.QueueUpdateDraw(func() {
			app.gui.setLogGroupToGui(app.awsr)
		})
	}()
	app.Run()
}
