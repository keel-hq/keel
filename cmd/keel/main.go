package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"context"

	netContext "golang.org/x/net/context"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/bot"
	"github.com/keel-hq/keel/cache/kubekv"

	"github.com/keel-hq/keel/constants"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/provider/helm"
	"github.com/keel-hq/keel/provider/kubernetes"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/secrets"
	"github.com/keel-hq/keel/trigger/http"
	"github.com/keel-hq/keel/trigger/poll"
	"github.com/keel-hq/keel/trigger/pubsub"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/codecs"
	"github.com/keel-hq/keel/version"

	// notification extensions
	_ "github.com/keel-hq/keel/extension/notification/hipchat"
	_ "github.com/keel-hq/keel/extension/notification/mattermost"
	_ "github.com/keel-hq/keel/extension/notification/slack"
	_ "github.com/keel-hq/keel/extension/notification/webhook"

	// bots
	_ "github.com/keel-hq/keel/bot/hipchat"
	_ "github.com/keel-hq/keel/bot/slack"

	log "github.com/sirupsen/logrus"
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

	inCluster := kingpin.Flag("incluster", "use in cluster configuration (defaults to 'true'), use '--no-incluster' if running outside of the cluster").Default("true").Bool()
	kubeconfig := kingpin.Flag("kubeconfig", "path to kubeconfig (if not in running inside a cluster)").Default(filepath.Join(os.Getenv("HOME"), ".kube", "config")).String()

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version(ver.Version)
	kingpin.CommandLine.Help = "Automated Kubernetes deployment updates. Learn more on https://keel.sh."
	kingpin.Parse()

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
	k8sCfg := &kubernetes.Opts{
		ConfigPath: *kubeconfig,
	}

	if os.Getenv(EnvKubernetesConfig) != "" {
		k8sCfg.ConfigPath = os.Getenv(EnvKubernetesConfig)
	}

	k8sCfg.InCluster = *inCluster

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

	bot.Run(implementer, approvalsManager)

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
			bot.Stop()

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
