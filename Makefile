GOBIN ?= $(shell go env GOPATH)/bin
BINARY = openbotkit
ALIAS = obk
SKILLS_DIR = $(HOME)/.obk/skills
ASSISTANT_SKILLS = assistant/.claude/skills

.PHONY: FORCE build build-obkmacos install uninstall update-local

build:
	go build -o $(BINARY) .

OBKMACOS = $(HOME)/.obk/bin/obkmacos

ifeq ($(shell uname),Darwin)
SIGN_IDENTITY ?= $(shell security find-identity -v -p codesigning 2>/dev/null | sed -n 's/.*"\(Developer ID Application[^"]*\)".*/\1/p' | head -1)
OBKMACOS_RESIGN = $(if $(SIGN_IDENTITY),$(shell codesign -v -R='identifier "com.73ai.obkmacos" and anchor apple generic' $(OBKMACOS) 2>/dev/null || echo FORCE))
build-obkmacos: $(OBKMACOS)
else
build-obkmacos:
endif

define sign
	@if [ "$$(uname)" = "Darwin" ]; then \
		if [ -n "$(SIGN_IDENTITY)" ]; then \
			codesign -f -s "$(SIGN_IDENTITY)" -i $(2) $(1) && echo "Signed $(1) with $(SIGN_IDENTITY)"; \
		else \
			echo "warning: no Developer ID signing identity, macOS will ask for permissions again after every install"; \
		fi; \
	fi
endef

FORCE:

$(OBKMACOS): swift/obkmacos.swift $(OBKMACOS_RESIGN)
	@mkdir -p $(dir $@)
	swiftc -O -o $@ swift/obkmacos.swift
	$(call sign,$@,com.73ai.obkmacos)
	@echo "Built obkmacos -> $@"

install: build-obkmacos
	go install .
	$(call sign,$(GOBIN)/$(BINARY),com.73ai.openbotkit)
	ln -sf $(GOBIN)/$(BINARY) $(GOBIN)/$(ALIAS)
	mkdir -p $(SKILLS_DIR)
	$(GOBIN)/$(ALIAS) update --skills-only
	@if [ -d assistant ]; then \
		rm -f $(ASSISTANT_SKILLS); \
		ln -sf $(SKILLS_DIR) $(ASSISTANT_SKILLS); \
		echo "Linked $(ASSISTANT_SKILLS) -> $(SKILLS_DIR)"; \
	fi
	@if pgrep -f "$(GOBIN)/($(BINARY)|$(ALIAS)) service" > /dev/null 2>&1; then \
		echo "Restarting running services..."; \
		$(GOBIN)/$(ALIAS) service restart 2>/dev/null || true; \
	fi

update-local: install
	$(GOBIN)/$(ALIAS) service restart

uninstall:
	rm -f $(GOBIN)/$(BINARY) $(GOBIN)/$(ALIAS)
