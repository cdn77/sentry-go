package sentry

import "io"

type writerAdapter struct {
	w   io.Writer
	hub *Hub
}

func GetExceptionWriter(w io.Writer, hub *Hub) io.Writer {
	return &writerAdapter{w, hub}
}

func (wa *writerAdapter) Write(p []byte) (int, error) {
	wa.w.Write(p)
	wa.hub.CaptureEvent(&Event{Level: LevelError, Message: string(p)})
	return len(p), nil
}
