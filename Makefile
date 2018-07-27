all: deps

.PHONY: deps
deps:
	@rm -rf ./vendor && \
	echo "running dep ensure..." && \
	dep ensure -v && \
	$(MAKE) gxundo

.PHONY: gxundo
gxundo:
	@bash scripts/gxundo.sh vendor/

.PHONY: install/gxundo
install/gxundo:
	@wget https://raw.githubusercontent.com/c3systems/gxundo/master/gxundo.sh \
	-O scripts/gxundo.sh && \
	chmod +x scripts/gxundo.sh
