package specutil

import (
	"strconv"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/configlayout"
)

const (
	ParameterActionSectionTitle     = "Parameter Action"
	MappingHeaderSectionTitle       = "Mapping Header"
	ParameterDefaultActionCardTitle = "Parameter Default Action"
)

func Section(
	key string,
	title string,
	description string,
	columns int,
	fieldKeys ...string,
) configlayout.SectionSpec {
	return configlayout.SectionSpec{
		Key:         strings.TrimSpace(key),
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		Columns:     columns,
		FieldKeys:   append([]string(nil), fieldKeys...),
	}
}

func IntOptions[T any](max int, build func(label string, value string) T) []T {
	items := make([]T, 0, max)
	for _, number := range HeaderRowNumbers(max) {
		label := strconv.Itoa(number)
		items = append(items, build(label, label))
	}
	return items
}

func ParameterActionSection(
	description string,
	columns int,
	fieldKeys ...string,
) configlayout.SectionSpec {
	return Section("parameter", ParameterActionSectionTitle, description, columns, fieldKeys...)
}

func MappingHeaderSection(
	description string,
	columns int,
	fieldKeys ...string,
) configlayout.SectionSpec {
	return Section("mapping", MappingHeaderSectionTitle, description, columns, fieldKeys...)
}
