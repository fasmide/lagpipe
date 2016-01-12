VERSION = 0.0-$(shell date +"%Y%m%d-%H%M")-$(shell git log -n 1 --pretty="format:%h")
GOSRC = $(wildcard *.go)

all: lagpipe

clean:
	rm -f lagpipe
	rm -f *.deb

run: lagpipe
	./lagpipe

lagpipe: ${GOSRC}
	go build .

deb: clean lagpipe
	mkdir -p lagpipe_$(VERSION)/usr/sbin/
	cp -a DEBIAN lagpipe_$(VERSION)/DEBIAN
	sed -i 's/PACKAGEVERSION/$(VERSION)/g' lagpipe_$(VERSION)/DEBIAN/control
	cp -a lagpipe lagpipe_$(VERSION)/usr/sbin
	dpkg-deb --build lagpipe_$(VERSION) .
	rm -rf lagpipe_$(VERSION)