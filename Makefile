VERSION=$(shell git describe --exact-match --tags 2>/dev/null | sed 's/^v//')
ifeq ($(VERSION),)
    VERSION=$(shell git rev-list -1 HEAD)
endif

export CGO_ENABLED=0
LDFLAGS=-X main.VERSION=$(VERSION) -s -w
GCFLAGS=

BINARIES=
define compile
dist/cidr-merger-$(1)-$(or $(3),$(2))$(if $(filter windows,$(1)),.exe):
	mkdir -p $$(@D)
	GOOS=$(1) GOARCH=$(2) $(4) go build -ldflags "$$(LDFLAGS)" -gcflags "$$(GCFLAGS)" -o $$@
BINARIES+=dist/cidr-merger-$(1)-$(or $(3),$(2))$(if $(filter windows,$(1)),.exe)
endef

dist/cidr-merger:
	mkdir -p $(@D)
	go build -ldflags "$(LDFLAGS)" -gcflags "$(GCFLAGS)" -o $@

$(eval $(call compile,darwin,amd64))
$(eval $(call compile,darwin,arm64))
$(eval $(call compile,dragonfly,amd64))
$(eval $(call compile,freebsd,386))
$(eval $(call compile,freebsd,amd64))
$(eval $(call compile,linux,386))
$(eval $(call compile,linux,amd64))
$(eval $(call compile,linux,arm,arm5,GOARM=5))
$(eval $(call compile,linux,arm,arm6,GOARM=6))
$(eval $(call compile,linux,arm,arm7,GOARM=7))
$(eval $(call compile,linux,arm64))
$(eval $(call compile,linux,mips,mips-hard,GOMIPS=hardfloat))
$(eval $(call compile,linux,mips,mips-soft,GOMIPS=softfloat))
$(eval $(call compile,linux,mipsle,mipsle-hard,GOMIPS=hardfloat))
$(eval $(call compile,linux,mipsle,mipsle-soft,GOMIPS=softfloat))
$(eval $(call compile,linux,mips64,mips64-hard,GOMIPS64=hardfloat))
$(eval $(call compile,linux,mips64,mips64-soft,GOMIPS64=softfloat))
$(eval $(call compile,linux,mips64le,mips64le-hard,GOMIPS64=hardfloat))
$(eval $(call compile,linux,mips64le,mips64le-soft,GOMIPS64=softfloat))
$(eval $(call compile,netbsd,386))
$(eval $(call compile,netbsd,amd64))
$(eval $(call compile,openbsd,386))
$(eval $(call compile,openbsd,amd64))
$(eval $(call compile,windows,386))
$(eval $(call compile,windows,amd64))

all: $(BINARIES)
.PHONY: all

clean:
	rm -rf dist
.PHONY: clean
