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
	EnvTriggerPubSub  = "PUBSUB" // set to 1 or something to enable pub/sub trigger
	EnvProjectID      = "PROJECT_ID"
	EnvSubscriptionID = "SUBSCRIPTION_ID"
	EnvTopic          = "TOPIC"
)

// kubernetes config, if empty - will default to InCluster
const (
	EnvKubernetesConfig = "KUBERNETES_CONFIG"
)

func main() {

	// getting k8s provider
	k8sCfg := &kubernetes.Opts{}
	if os.Getenv(EnvKubernetesConfig) != "" {
		k8sCfg.ConfigPath = os.Getenv(EnvKubernetesConfig)
	} else {
		k8sCfg.InCluster = true
	}
	k8sProvider, err := kubernetes.NewProvider(k8sCfg)
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
		subscriptionID := os.Getenv(EnvSubscriptionID)
		if subscriptionID == "" {
			log.Fatalf("main: subscription ID env variable not set")
			return
		}
		topic := os.Getenv(EnvTopic)
		if topic == "" {
			log.Fatalf("main: top env variable not set")
			return
		}

		ps, err := pubsub.NewSubscriber(&pubsub.Opts{
			Project:      projectID,
			Subscription: subscriptionID,
			Topic:        topic,
			Providers:    providers,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("main: failed to create gcloud pubsub subscriber")
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go ps.Subscribe(ctx)
		log.WithFields(log.Fields{
			"project":      projectID,
			"subscription": subscriptionID,
			"topic":        topic,
		}).Info("main: gcloud pubsub trigger for gcr enabled")
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
