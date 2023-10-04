package sentry

import (
	pkgErrors "github.com/pkg/errors"
)

// NOTE: if you modify this file, you are also responsible for updating LoC position in Stacktrace tests

func Trace() *Stacktrace {
	return NewStacktrace()
}

func RedPkgErrorsRanger() error {
	return BluePkgErrorsRanger()
}

func BluePkgErrorsRanger() error {
	return pkgErrors.New("this is bad from pkgErrors")
}
