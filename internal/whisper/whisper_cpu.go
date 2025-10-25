//go:build !cuda

package whisper

/*
#cgo CFLAGS: -I${SRCDIR}/../../whisper.cpp/include -I${SRCDIR}/../../whisper.cpp/ggml/include
#cgo LDFLAGS: -L${SRCDIR}/../../whisper.cpp/build/src -L${SRCDIR}/../../whisper.cpp/build/ggml/src -lwhisper -lggml -lggml-cpu -lggml-base -lm -lstdc++ -lpthread -lgomp
*/
import "C"

const cudaEnabled = false
