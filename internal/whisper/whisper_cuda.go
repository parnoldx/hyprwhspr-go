//go:build cuda

package whisper

/*
#cgo CFLAGS: -I${SRCDIR}/../../whisper.cpp/include -I${SRCDIR}/../../whisper.cpp/ggml/include
#cgo LDFLAGS: -L${SRCDIR}/../../whisper.cpp/build/src -L${SRCDIR}/../../whisper.cpp/build/ggml/src -L${SRCDIR}/../../whisper.cpp/build/ggml/src/ggml-cuda -L/opt/cuda/lib64 -lwhisper -lggml -lggml-cuda -lggml-cpu -lggml-base -lm -lstdc++ -lpthread -lgomp -lcublas -lcublasLt -lcudart -lcuda
*/
import "C"

const cudaEnabled = true
