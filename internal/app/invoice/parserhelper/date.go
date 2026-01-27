package parserhelper

import (
	"time"

	"github.com/sieryo/invoice-extractor/pkg/helper"
)

func ParseDateValue(val string) *time.Time {
	if val == "" {
		return nil
	}

	t, err := helper.ParseDateValue(val)
	if err != nil {
		return nil
	}

	return t
}
