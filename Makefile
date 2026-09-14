SUBDIRS := stmtdate stmtpng2tsv goquery docker-override

.PHONY: build test tidy

build test tidy:
	@for dir in $(SUBDIRS); do \
		$(MAKE) -C $$dir $@ || exit 1; \
	done
