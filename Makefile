# Defaults that should probably be overriden in a Config.mk
RUMPCC ?= x86_64-rumprun-netbsd-gcc
export RUMPCC

DISTDIR ?= $(PWD)/dist
export DISTDIR

DIRS = worker controller scripts

BUILDDIRS = $(DIRS:%=build-%)
DISTDIRS = $(DIRS:%=dist-%)
CLEANDIRS = $(DIRS:%=clean-%)

all: dist

build: $(BUILDDIRS)
$(DIRS): $(BUILDDIRS)
$(BUILDDIRS):
	$(MAKE) -C $(@:build-%=%)

dist: build $(DISTDIR) $(DISTDIRS)

$(DISTDIR):
	mkdir -p $(DISTDIR)

$(DISTDIRS):
	$(MAKE) -C $(@:dist-%=%) dist

clean: $(CLEANDIRS) clean-dist
clean-dist:
	rm -rf $(DISTDIR)

$(CLEANDIRS): 
	$(MAKE) -C $(@:clean-%=%) clean


.PHONY: subdirs $(DIRS)
.PHONY: subdirs $(BUILDDIRS)
.PHONY: subdirs $(INSTALLDIRS)
.PHONY: subdirs $(CLEANDIRS)
.PHONY: all build dist clean clean-dist
