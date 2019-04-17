package sentry

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

const unknown string = "unknown"
const contextLines int = 5

var sourceReader = NewSourceReader()

// // The module download is split into two parts: downloading the go.mod and downloading the actual code.
// // If you have dependencies only needed for tests, then they will show up in your go.mod,
// // and go get will download their go.mods, but it will not download their code.
// // The test-only dependencies get downloaded only when you need it, such as the first time you run go test.
// //
// // https://github.com/golang/go/issues/26913#issuecomment-411976222

type Stacktrace struct {
	Frames        []Frame `json:"frames"`
	FramesOmitted [2]uint `json:"frames_omitted"`
}

func NewStacktrace() *Stacktrace {
	pcs := make([]uintptr, 100)
	n := runtime.Callers(1, pcs)

	if n == 0 {
		return nil
	}

	frames := extractFrames(pcs[:n])
	frames = filterFrames(frames)
	frames = contextifyFrames(frames)

	stacktrace := Stacktrace{
		Frames: frames,
	}

	return &stacktrace
}

func ExtractStacktrace(err error) *Stacktrace {
	// https://github.com/pkg/errors
	methodStackTrace := reflect.ValueOf(err).MethodByName("StackTrace")

	if methodStackTrace.IsValid() {
		errStacktrace := methodStackTrace.Call(make([]reflect.Value, 0))[0]

		if errStacktrace.Kind() != reflect.Slice {
			return nil
		}

		var pcs []uintptr
		for i := 0; i < errStacktrace.Len(); i++ {
			pcs = append(pcs, uintptr(errStacktrace.Index(i).Uint()))
		}

		frames := extractFrames(pcs)
		frames = filterFrames(frames)
		frames = contextifyFrames(frames)

		stacktrace := Stacktrace{
			Frames: frames,
		}

		return &stacktrace
	}

	return nil
}

// https://docs.sentry.io/development/sdk-dev/interfaces/stacktrace/
type Frame struct {
	Function    string                 `json:"function"`
	Symbol      string                 `json:"symbol"`
	Module      string                 `json:"module"`
	Package     string                 `json:"package"`
	Filename    string                 `json:"filename"`
	AbsPath     string                 `json:"abs_path"`
	Lineno      int                    `json:"lineno"`
	Colno       int                    `json:"colno"`
	PreContext  []string               `json:"pre_context"`
	ContextLine string                 `json:"context_line"`
	PostContext []string               `json:"post_context"`
	InApp       bool                   `json:"in_app"`
	Vars        map[string]interface{} `json:"vars"`
}

func NewFrame(f runtime.Frame) Frame {
	abspath := f.File
	filename := f.File
	function := f.Function
	var module string

	if filename != "" {
		filename = extractFilenameFromPath(filename)
	} else {
		filename = unknown
	}

	if abspath == "" {
		abspath = unknown
	}

	if function != "" {
		module, function = deconstructFunctionName(function)
	}

	frame := Frame{
		AbsPath:  abspath,
		Filename: filename,
		Lineno:   f.Line,
		Module:   module,
		Function: function,
	}

	frame.InApp = isInAppFrame(frame)

	return frame
}

func extractFrames(pcs []uintptr) []Frame {
	var frames []Frame
	callersFrames := runtime.CallersFrames(pcs)

	for {
		callerFrame, more := callersFrames.Next()

		frames = append([]Frame{
			NewFrame(callerFrame),
		}, frames...)

		if !more {
			break
		}
	}

	return frames
}

func filterFrames(frames []Frame) []Frame {
	filteredFrames := make([]Frame, 0, len(frames))

	for _, frame := range frames {
		if frame.Module == "runtime" || frame.Module == "testing" {
			continue
		}
		filteredFrames = append(filteredFrames, frame)
	}

	return filteredFrames
}

func contextifyFrames(frames []Frame) []Frame {
	contextifiedFrames := make([]Frame, 0, len(frames))

	for _, frame := range frames {
		lines, initial := sourceReader.ReadContextLines(frame.AbsPath, frame.Lineno, contextLines)

		if lines == nil {
			continue
		}

		for i, line := range lines {
			switch {
			case i < initial:
				frame.PreContext = append(frame.PreContext, string(line))
			case i == initial:
				frame.ContextLine = string(line)
			default:
				frame.PostContext = append(frame.PostContext, string(line))
			}
		}

		contextifiedFrames = append(contextifiedFrames, frame)
	}

	return contextifiedFrames
}

func extractFilenameFromPath(path string) string {
	_, file := filepath.Split(path)
	return file
}

func isInAppFrame(frame Frame) bool {
	if frame.Module == "main" {
		return true
	}

	if !strings.Contains(frame.Module, "vendor") && !strings.Contains(frame.Module, "third_party") {
		return true
	}

	return false
}

// Transform `runtime/debug.*T·ptrmethod` into `{ module: runtime/debug, function: *T.ptrmethod }`
func deconstructFunctionName(name string) (module string, function string) {
	if idx := strings.LastIndex(name, "."); idx != -1 {
		module = name[:idx]
		function = name[idx+1:]
	}
	function = strings.Replace(function, "·", ".", -1)
	return module, function
}
