package main

import (
	"os"
	"os/signal"
	"time"

	"golang.org/x/net/context"

	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/provider/kubernetes"
	"github.com/rusenask/keel/trigger/pubsub"
	"github.com/rusenask/keel/trigger/webhook"
	"github.com/rusenask/keel/types"

	log "github.com/Sirupsen/logrus"
)

// gcloud pubsub related config
const (
	EnvTriggerPubSub = "PUBSUB" // set to 1 or something to enable pub/sub trigger
	EnvProjectID     = "PROJECT_ID"
)

// kubernetes config, if empty - will default to InCluster
const (
	EnvKubernetesConfig = "KUBERNETES_CONFIG"
)

// EnvDebug - set to 1 or anything else to enable debug logging
const EnvDebug = "DEBUG"

func main() {

	if os.Getenv(EnvDebug) != "" {
		log.SetLevel(log.DebugLevel)
	}

	// getting k8s provider
	k8sCfg := &kubernetes.Opts{}
	if os.Getenv(EnvKubernetesConfig) != "" {
		k8sCfg.ConfigPath = os.Getenv(EnvKubernetesConfig)
	} else {
		k8sCfg.InCluster = true
	}
	implementer, err := kubernetes.NewKubernetesImplementer(k8sCfg)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"config": k8sCfg,
		}).Fatal("main: failed to create kubernetes implementer")
	}

	k8sProvider, err := kubernetes.NewProvider(implementer)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("main: failed to create kubernetes provider")
	}
	go k8sProvider.Start()

	providers := make(map[string]provider.Provider)
	providers[k8sProvider.GetName()] = k8sProvider

	whs := webhook.NewTriggerServer(&webhook.Opts{
		Port:      types.KeelDefaultPort,
		Providers: providers,
	})

	go whs.Start()

	if os.Getenv(EnvTriggerPubSub) != "" {
		projectID := os.Getenv(EnvProjectID)
		if projectID == "" {
			log.Fatalf("main: project ID env variable not set")
			return
		}

		ps, err := pubsub.NewSubscriber(&pubsub.Opts{
			ProjectID: projectID,
			Providers: providers,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("main: failed to create gcloud pubsub subscriber")
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		subManager := pubsub.NewDefaultManager(projectID, implementer, ps)
		go subManager.Start(ctx)
	}

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
			// whs.Stop()
			cleanupDone <- true
		}
	}()

	<-cleanupDone

}
