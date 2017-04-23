OUT           :=  punch
GO_SRC         = $(wildcard *.go)
GDOC_WHERE    := -X main.VersionUrl=`git remote get-url origin`
GDOC_TAG      := -X main.VersionRef=`git rev-parse HEAD | cut -c 1-15`
GDOC_STAMP    := -X main.VersionDate=`date --iso-8601=minutes`
GDOC_NAME     := -X main.AppName=$(OUT)
GO_LINK_DOCS  := "$(GDOC_NAME) $(GDOC_TAG) $(GDOC_STAMP) $(GDOC_WHERE)"
XTRA_BLD_FLGS := -x

all: clean $(OUT)

$(OUT): $(GO_SRC)
	go build $(XTRA_BLD_FLGS) -ldflags $(GO_LINK_DOCS) -o $@

clean:
	$(RM) $(OUT)

.PHONY: clean all
