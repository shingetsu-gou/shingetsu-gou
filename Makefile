
update-bindata:
	go-bindata -pkg util -o util/bindata.go $(shell find www/ file/ gou_template/ -type d)

