a:
	go run cmd/operatable/main.go -v -as abilities

b:
	go run cmd/index/main.go -v

i:
	go run demo/index/main.go

w1:
	go run demo/workers/1/main.go

w2:
	go run demo/workers/2/main.go

w3:
	CGO_CXXFLAGS="-I${CURDIR}/demo/tmp/deepspeech/include" \
	LIBRARY_PATH=${CURDIR}/demo/tmp/deepspeech/lib:${LIBRARY_PATH} \
	LD_LIBRARY_PATH=${CURDIR}/demo/tmp/deepspeech/lib:${LD_LIBRARY_PATH} \
	go run demo/workers/3/main.go