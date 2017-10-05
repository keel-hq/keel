package main

import (
	"os"
	"os/signal"
	"time"

	"context"

	netContext "golang.org/x/net/context"

	"github.com/rusenask/keel/approvals"
	"github.com/rusenask/keel/bot"
	"github.com/rusenask/keel/cache/kubekv"

	"github.com/rusenask/keel/constants"
	"github.com/rusenask/keel/extension/notification"
	"github.com/rusenask/keel/provider"
	"github.com/rusenask/keel/provider/helm"
	"github.com/rusenask/keel/provider/kubernetes"
	"github.com/rusenask/keel/registry"
	"github.com/rusenask/keel/secrets"
	"github.com/rusenask/keel/trigger/http"
	"github.com/rusenask/keel/trigger/poll"
	"github.com/rusenask/keel/trigger/pubsub"
	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/codecs"
	"github.com/rusenask/keel/version"

	// extensions
	_ "github.com/rusenask/keel/extension/notification/slack"
	_ "github.com/rusenask/keel/extension/notification/webhook"

	log "github.com/Sirupsen/logrus"
)

// gcloud pubsub related config
const (
	EnvTriggerPubSub = "PUBSUB" // set to 1 or something to enable pub/sub trigger
	EnvTriggerPoll   = "POLL"   // set to 1 or something to enable poll trigger
	EnvProjectID     = "PROJECT_ID"

	EnvNamespace = "NAMESPACE" // Keel's namespace

	EnvHelmProvider      = "HELM_PROVIDER"  // helm provider
	EnvHelmTillerAddress = "TILLER_ADDRESS" // helm provider
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
	}).Info("keel starting...")

	if os.Getenv(EnvDebug) != "" {
		log.SetLevel(log.DebugLevel)
	}

	// setting up triggers
	ctx, cancel := netContext.WithCancel(context.Background())
	defer cancel()

	notificationLevel := types.LevelInfo
	if os.Getenv(constants.EnvNotificationLevel) != "" {
		parsedLevel, err := types.ParseLevel(os.Getenv(constants.EnvNotificationLevel))
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Errorf("main: got error while parsing notification level, defaulting to: %s", notificationLevel)
		} else {
			notificationLevel = parsedLevel
		}
	}

	notifCfg := &notification.Config{
		Attempts: 10,
		Level:    notificationLevel,
	}
	sender := notification.New(ctx)

	_, err := sender.Configure(notifCfg)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("main: failed to configure notification sender manager")
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

	keelsNamespace := constants.DefaultNamespace
	if os.Getenv(EnvNamespace) != "" {
		keelsNamespace = os.Getenv(EnvNamespace)
	}

	kkv, err := kubekv.New(implementer.ConfigMaps(keelsNamespace), "approvals")
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"namespace": keelsNamespace,
		}).Fatal("main: failed to initialise kube-kv")
	}

	serializer := codecs.DefaultSerializer()
	// mem := memory.NewMemoryCache(24*time.Hour, 24*time.Hour, 1*time.Minute)
	approvalsManager := approvals.New(kkv, serializer)

	go approvalsManager.StartExpiryService(ctx)

	// setting up providers
	providers := setupProviders(implementer, sender, approvalsManager)

	secretsGetter := secrets.NewGetter(implementer)

	teardownTriggers := setupTriggers(ctx, providers, secretsGetter, approvalsManager)

	teardownBot, err := setupBot(implementer, approvalsManager)
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

			// teardownProviders()
			providers.Stop()
			teardownTriggers()
			teardownBot()

			cleanupDone <- true
		}
	}()

	<-cleanupDone
}

// setupProviders - setting up available providers. New providers should be initialised here and added to
// provider map
func setupProviders(k8sImplementer kubernetes.Implementer, sender notification.Sender, approvalsManager approvals.Manager) (providers provider.Providers) {
	var enabledProviders []provider.Provider

	k8sProvider, err := kubernetes.NewProvider(k8sImplementer, sender, approvalsManager)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("main.setupProviders: failed to create kubernetes provider")
	}
	go k8sProvider.Start()
	enabledProviders = append(enabledProviders, k8sProvider)

	if os.Getenv(EnvHelmProvider) == "1" {
		tillerAddr := os.Getenv(EnvHelmTillerAddress)
		helmImplementer := helm.NewHelmImplementer(tillerAddr)
		helmProvider := helm.NewProvider(helmImplementer, sender, approvalsManager)

		go helmProvider.Start()
		enabledProviders = append(enabledProviders, helmProvider)
	}

	providers = provider.New(enabledProviders, approvalsManager)

	return providers
}

func setupBot(k8sImplementer kubernetes.Implementer, approvalsManager approvals.Manager) (teardown func(), err error) {

	if os.Getenv(constants.EnvSlackToken) != "" {
		botName := "keel"

		if os.Getenv(constants.EnvSlackBotName) != "" {
			botName = os.Getenv(constants.EnvSlackBotName)
		}

		token := os.Getenv(constants.EnvSlackToken)
		slackBot := bot.New(botName, token, k8sImplementer, approvalsManager)

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
func setupTriggers(ctx context.Context, providers provider.Providers, secretsGetter secrets.Getter, approvalsManager approvals.Manager) (teardown func()) {

	// setting up generic http webhook server
	whs := http.NewTriggerServer(&http.Opts{
		Port:            types.KeelDefaultPort,
		Providers:       providers,
		ApprovalManager: approvalsManager,
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

		subManager := pubsub.NewDefaultManager(projectID, providers, ps)
		go subManager.Start(ctx)
	}

	if os.Getenv(EnvTriggerPoll) != "0" {

		registryClient := registry.New()
		watcher := poll.NewRepositoryWatcher(providers, registryClient)
		pollManager := poll.NewPollManager(providers, watcher, secretsGetter)

		// start poll manager, will finish with ctx
		go watcher.Start(ctx)
		go pollManager.Start(ctx)
	}

	teardown = func() {
		whs.Stop()
	}

	return teardown
}
