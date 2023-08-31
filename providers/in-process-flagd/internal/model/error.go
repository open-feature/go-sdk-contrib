package model

import (
	"fmt"
	flagdModels "github.com/open-feature/flagd/core/pkg/model"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
)

func FlagdErrorCodeToResolutionError(flagdCode string, msg string) of.ResolutionError {
	switch flagdCode {
	case flagdModels.FlagNotFoundErrorCode:
		return of.NewFlagNotFoundResolutionError(msg)
	case flagdModels.ParseErrorCode:
		return of.NewParseErrorResolutionError(msg)
	case flagdModels.TypeMismatchErrorCode:
		return of.NewTypeMismatchResolutionError(msg)
	}

	resErrMsg := flagdCode
	if msg != "" {
		resErrMsg = fmt.Sprintf("%s: %s", resErrMsg, msg)
	}
	return of.NewGeneralResolutionError(resErrMsg)
}
