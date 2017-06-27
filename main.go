package main

import (
	"os"
	"os/signal"
	"time"

	"context"

	netContext "golang.org/x/net/context"

	"github.com/rusenask/keel/bot"
	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/provider/kubernetes"
	"github.com/rusenask/keel/trigger/http"
	"github.com/rusenask/keel/trigger/pubsub"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/version"

	log "github.com/Sirupsen/logrus"
)

// gcloud pubsub related config
const (
	EnvTriggerPubSub = "PUBSUB" // set to 1 or something to enable pub/sub trigger
	EnvProjectID     = "PROJECT_ID"
	EnvSlackToken    = "SLACK_TOKEN"
	EnvSlackBotName  = "SLACK_BOT_NAME"
)

// kubernetes config, if empty - will default to InCluster
const (
	EnvKubernetesConfig = "KUBERNETES_CONFIG"
)

// EnvDebug - set to 1 or anything else to enable debug logging
const EnvDebug = "DEBUG"

func main() {

	ver := version.GetKeelVersion()
	log.WithFields(log.Fields{
		"os":         ver.OS,
		"build_date": ver.BuildDate,
		"revision":   ver.Revision,
		"version":    ver.Version,
		"go_version": ver.GoVersion,
		"arch":       ver.Arch,
	}).Info("Keel starting..")

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

	// setting up providers
	providers, teardownProviders := setupProviders(implementer)

	// setting up triggers
	ctx, cancel := netContext.WithCancel(context.Background())
	defer cancel()
	teardownTriggers := setupTriggers(ctx, implementer, providers)

	teardownBot, err := setupBot(implementer)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("main: failed to setup slack bot")
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

			teardownProviders()
			teardownTriggers()
			teardownBot()

			cleanupDone <- true
		}
	}()

	<-cleanupDone

}

// setupProviders - setting up available providers. New providers should be initialised here and added to
// provider map
func setupProviders(k8sImplementer kubernetes.Implementer) (providers map[string]provider.Provider, teardown func()) {
	k8sProvider, err := kubernetes.NewProvider(k8sImplementer)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("main.setupProviders: failed to create kubernetes provider")
	}
	go k8sProvider.Start()

	providers = make(map[string]provider.Provider)
	providers[k8sProvider.GetName()] = k8sProvider

	teardown = func() {
		k8sProvider.Stop()
	}

	return providers, teardown
}

func setupBot(k8sImplementer kubernetes.Implementer) (teardown func(), err error) {

	if os.Getenv(EnvSlackToken) != "" {
		botName := "keel"

		if os.Getenv(EnvSlackBotName) != "" {
			botName = os.Getenv(EnvSlackBotName)
		}

		token := os.Getenv(EnvSlackToken)
		slackBot := bot.New(botName, token, k8sImplementer)

		ctx, cancel := context.WithCancel(context.Background())

		err := slackBot.Start(ctx)
		if err != nil {
			cancel()
			return nil, err
		}

		teardown := func() {
			// cancelling context
			cancel()
		}

		return teardown, nil
	}

	return func() {}, nil
}

// setupTriggers - setting up triggers. New triggers should be added to this function. Each trigger
// should go through all providers (or not if there is a reason) and submit events)
func setupTriggers(ctx context.Context, k8sImplementer kubernetes.Implementer, providers map[string]provider.Provider) (teardown func()) {

	// setting up generic http webhook server
	whs := http.NewTriggerServer(&http.Opts{
		Port:      types.KeelDefaultPort,
		Providers: providers,
	})

	go whs.Start()

	// checking whether pubsub (GCR) trigger is enabled
	if os.Getenv(EnvTriggerPubSub) != "" {
		projectID := os.Getenv(EnvProjectID)
		if projectID == "" {
			log.Fatalf("main.setupTriggers: project ID env variable not set")
			return
		}

		ps, err := pubsub.NewPubsubSubscriber(&pubsub.Opts{
			ProjectID: projectID,
			Providers: providers,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("main.setupTriggers: failed to create gcloud pubsub subscriber")
			return
		}

		subManager := pubsub.NewDefaultManager(projectID, k8sImplementer, ps)
		go subManager.Start(ctx)
	}

	teardown = func() {
		whs.Stop()
	}

	return teardown
}
