GOBIN ?= $(shell go env GOPATH)/bin
BINARY = openbotkit
ALIAS = obk
SKILLS_DIR = $(HOME)/.obk/skills
ASSISTANT_SKILLS = assistant/.claude/skills

.PHONY: build build-obkmacos install uninstall update-local

build:
	go build -o $(BINARY) .

build-obkmacos:
ifeq ($(shell uname),Darwin)
	@mkdir -p $(HOME)/.obk/bin
	swiftc -O -o $(HOME)/.obk/bin/obkmacos swift/obkmacos.swift
	@echo "Built obkmacos -> $(HOME)/.obk/bin/obkmacos"
endif

install: build-obkmacos
	go install .
	ln -sf $(GOBIN)/$(BINARY) $(GOBIN)/$(ALIAS)
	mkdir -p $(SKILLS_DIR)
	$(GOBIN)/$(ALIAS) update --skills-only
	@if [ -d assistant ]; then \
		rm -f $(ASSISTANT_SKILLS); \
		ln -sf $(SKILLS_DIR) $(ASSISTANT_SKILLS); \
		echo "Linked $(ASSISTANT_SKILLS) -> $(SKILLS_DIR)"; \
	fi
	@if pgrep -f "$(BINARY)\|$(ALIAS)" > /dev/null 2>&1; then \
		echo "Restarting running services..."; \
		$(GOBIN)/$(ALIAS) service restart 2>/dev/null || true; \
	fi

update-local: install
	$(GOBIN)/$(ALIAS) service restart

uninstall:
	rm -f $(GOBIN)/$(BINARY) $(GOBIN)/$(ALIAS)
