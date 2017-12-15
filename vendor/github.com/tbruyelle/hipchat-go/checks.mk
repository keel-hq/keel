checks:
	go test -race $(SRC_DIR)
	@$(call checkbin,go tool vet,golang.org/x/tools/cms/vet)
	go tool vet $(SRC_DIR)
	@$(call checkbin,golint,github.com/golang/lint/golint)
	golint -set_exit_status $(SRC_DIR)
	@$(call checkbin,errcheck,github.com/kisielk/errcheck)
	errcheck -ignore 'Close' -ignoretests $(SRC_DIR)
	@$(call checkbin,structcheck,github.com/opennota/check/cmd/structcheck)
	structcheck $(SRC_DIR)
	@$(call checkbin,varcheck,github.com/opennota/check/cmd/varcheck)
	varcheck $(SRC_DIR)

checkbin = $1 2> /dev/null; if [ $$? -eq 127 ]; then\
					 	echo "Retrieving missing tool $1...";\
				 		go get $2; \
					fi;

