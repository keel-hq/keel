SRC_DIR=./hipchat

include checks.mk

default: test checks

# test runs the unit tests and vets the code
test:
	go test -v $(SRC_DIR) $(TESTARGS) -timeout=30s -parallel=4
