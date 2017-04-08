OUT  :=  punch

all: clean $(OUT)

$(OUT): $(OUT).go
	go build -o $@

clean:
	$(RM) $(OUT)

.PHONY: clean all
