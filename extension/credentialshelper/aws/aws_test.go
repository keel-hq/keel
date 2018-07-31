package aws

import (
	"os"
	"testing"

	"github.com/keel-hq/keel/registry"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
)

func TestAWS(t *testing.T) {

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip()
	}

	ch := New()

	// image
	imgRef, _ := image.Parse("528670773427.dkr.ecr.us-east-2.amazonaws.com/webhook-demo:master")

	creds, err := ch.GetCredentials(&types.TrackedImage{
		Image: imgRef,
	})
	if err != nil {
		t.Fatalf("cred helper got error: %s", err)
	}

	rc := registry.New()

	currentDigest, err := rc.Digest(registry.Opts{
		Registry: imgRef.Scheme() + "://" + imgRef.Registry(),
		Name:     imgRef.ShortName(),
		Tag:      imgRef.Tag(),
		Username: creds.Username,
		Password: creds.Password,
	})

	if err != nil {
		t.Fatalf("failed to get digest: %s", err)
	}

	if currentDigest != "sha256:7712aa425c17c2e413e5f4d64e2761eda009509d05d0e45a26e389d715aebe23" {
		t.Errorf("unexpected digest: %s", currentDigest)
	}
}

func TestCredentialsCaching(t *testing.T) {

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip()
	}
	ch := New()
	imgRef, _ := image.Parse("528670773427.dkr.ecr.us-east-2.amazonaws.com/webhook-demo:master")
	for i := 0; i < 200; i++ {
		_, err := ch.GetCredentials(&types.TrackedImage{
			Image: imgRef,
		})
		if err != nil {
			t.Fatalf("cred helper got error: %s", err)
		}
	}
}

func TestAWSRegistryParse(t *testing.T) {
	registry := "528670773427.dkr.ecr.us-east-2.amazonaws.com"
	registryID, region, err := parseRegistry(registry)
	if err != nil {
		t.Fatalf("parseRegistry got error: %s", err)
	}
	if registryID != "528670773427" {
		t.Fatalf("parseRegistry parse registryID(528670773427) not as expected: %s", registryID)
	}
	if region != "us-east-2" {
		t.Fatalf("parseRegistry parse region(us-east-2) not as expected: %s", region)
	}
}
