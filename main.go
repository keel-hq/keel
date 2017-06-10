package main

import (
	"os"
	"os/signal"
	"time"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/provider/kubernetes"
	"github.com/rusenask/keel/trigger/webhook"
	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
)

func main() {

	// getting k8s provider
	k8sProvider, err := kubernetes.NewProvider(&kubernetes.Opts{InCluster: true})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("main: failed to create kubernetes provider")
		return
	}

	go k8sProvider.Start()

	providers := make(map[string]provider.Provider)
	providers[k8sProvider.GetName()] = k8sProvider

	whs := webhook.NewTriggerServer(&webhook.Opts{
		Port:      types.KeelDefaultPort,
		Providers: providers,
	})

	go whs.Start()

	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			log.Info("received an interrupt, closing connection...")

			go func() {
				select {
				case <-time.After(10 * time.Second):
					log.Info("connection shutdown took too long, exiting... ")
					close(cleanupDone)
					return
				case <-cleanupDone:
					return
				}
			}()

			k8sProvider.Stop()
			whs.Stop()
			cleanupDone <- true
		}
	}()

	<-cleanupDone

}
