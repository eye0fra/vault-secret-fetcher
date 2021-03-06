package fetcher

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/fsnotify/fsnotify"
	"github.com/robfig/cron"
	"path/filepath"
)

var log = logrus.New()

func initLog() {
	level, err := logrus.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.Fatalln(err)
	} else {
		log.Level = level
	}

}

func validateConfig() {

	env := []string{"vault-capath", "vault-addr", "vault-token", "vault-secret", "properties-file"}

	for _, value := range env {
		if viper.GetString(value) == "" {
			log.WithFields(logrus.Fields{
				"environment": value,
			}).Fatalln("Must be set and non-empty")
		}
	}
	log.Debugln("Configuration is valid")
}

func Start() {
	initLog()
	log.Infoln("Starting token fetcher...")

	validateConfig()

	propertiesFile := viper.GetString("properties-file")
	tokenHandler := TokenHandler{
		VaultAddr: viper.GetString("vault-addr"),
		Token: Token{
			Path: viper.GetString("vault-token"),
		},
		Properties: Properties{
			Path:   propertiesFile,
			Format: filepath.Ext(propertiesFile),
		}}

	newCron(tokenHandler)
	newWatcher(tokenHandler)
}

func newCron(tokenHandler TokenHandler) {
	c := cron.New()
	c.AddFunc(viper.GetString("vault-token-fetcher-cron"), func() { tokenHandler.readToken() })
	c.Start()
	tokenHandler.readToken()
}

func newWatcher(tokenHandler TokenHandler) {
	// creates a new file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorln(err)
	}
	defer watcher.Close()

	//
	done := make(chan bool)

	//
	go func() {
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				log.Infoln(event)
				tokenHandler.readToken()
				// watch for errors
			case err := <-watcher.Errors:
				log.Errorln(err)
			}
		}
	}()

	// out of the box fsnotify can watch a single file, or a single directory
	if err := watcher.Add(viper.GetString("vault-token")); err != nil {
		log.Fatal(err)
	}

	<-done
}
