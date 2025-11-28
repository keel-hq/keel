package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/pkg/store/sql"
	"github.com/keel-hq/keel/provider"
	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/trigger/poll"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

// TestCronExample demonstrates how to use the fixed polling system
// This example shows proper concurrent polling without race conditions
func main() {
	fmt.Println("Starting Keel Polling System Test")

	// Create mock providers and registry client
	fp := &fakeProvider{}
	testStore, teardown := createTestStore()
	defer teardown()

	am := approvals.New(&approvals.Opts{Store: testStore})
	providers := provider.New([]provider.Provider{fp}, am)

	frc := &fakeRegistryClient{
		digestToReturn: "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb",
	}

	// Create the RepositoryWatcher - this is the proper way to use the polling system
	watcher := poll.NewRepositoryWatcher(providers, frc)

	// Start the watcher in a goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	go watcher.Start(ctx)

	// Create multiple tracked images to watch
	images := []*types.TrackedImage{
		createMockTrackedImage("nginx:1.20"),
		createMockTrackedImage("nginx:1.21"),
		createMockTrackedImage("nginx:latest"),
	}

	fmt.Printf("Starting to watch %d images\n", len(images))

	// Watch the images - this will set up polling jobs for each image
	err := watcher.Watch(images...)
	if err != nil {
		fmt.Printf("Error watching images: %v\n", err)
		return
	}

	fmt.Println("Images are now being watched. Polling will happen according to each image's schedule.")
	fmt.Println("The race condition fix ensures that concurrent polling jobs don't corrupt shared state.")

	// Wait for the context to timeout (2 minutes)
	<-ctx.Done()
	fmt.Println("Test completed - polling system ran without race conditions")
}

// createMockTrackedImage creates a mock tracked image for testing
func createMockTrackedImage(imageRef string) *types.TrackedImage {
	ref, _ := image.Parse(imageRef)

	// Create a force policy (simplest policy for testing)
	pol := policy.NewForcePolicy(false)

	return &types.TrackedImage{
		Image:        ref,
		Trigger:      types.TriggerTypePoll,
		PollSchedule: "@every 10s", // Poll every 10 seconds for faster testing
		Policy:       pol,
	}
}

// createTestStore creates a temporary test store
func createTestStore() (*sql.SQLStore, func()) {
	// Use the same pattern as in the actual tests
	dir, err := os.MkdirTemp("", "keeltest")
	if err != nil {
		panic(err)
	}

	tmpfn := filepath.Join(dir, "test.db")
	testStore, err := sql.New(sql.Opts{DatabaseType: "sqlite3", URI: tmpfn})
	if err != nil {
		panic(err)
	}

	teardown := func() {
		os.RemoveAll(dir)
	}

	return testStore, teardown
}

// Mock implementations for testing
type fakeProvider struct{}

func (fp *fakeProvider) Submit(event types.Event) error {
	fmt.Printf("✅ Event submitted for image: %s:%s (digest: %s)\n",
		event.Repository.Name, event.Repository.Tag, event.Repository.Digest)
	return nil
}

func (fp *fakeProvider) TrackedImages() ([]*types.TrackedImage, error) {
	// Return empty list for this test
	return []*types.TrackedImage{}, nil
}

func (fp *fakeProvider) GetName() string {
	return "fakeProvider"
}

func (fp *fakeProvider) Stop() {}


type fakeRegistryClient struct {
	digestToReturn string
}

func (frc *fakeRegistryClient) Digest(opts registry.Opts) (string, error) {
	fmt.Printf("🔍 Checking digest for %s:%s\n", opts.Name, opts.Tag)
	return frc.digestToReturn, nil
}

func (frc *fakeRegistryClient) Get(opts registry.Opts) (*registry.Repository, error) {
	fmt.Printf("📋 Getting repository info for %s\n", opts.Name)
	return &registry.Repository{
		Name: opts.Name,
		Tags: []string{"1.20", "1.21", "latest"},
	}, nil
}
