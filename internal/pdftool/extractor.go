package pdftool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

func ExtractText(
	ctx context.Context,
	pdfPath string,
	opts ExtractOptions,
) (string, error) {

	bin, err := ResolvePDFToTextPath()
	if err != nil {
		return "", err
	}

	args := buildArgs(opts, pdfPath)

	cmd := exec.CommandContext(ctx, bin, args...)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf(
			"%w: %s",
			ErrExtractFailed,
			stderr.String(),
		)
	}

	return out.String(), nil
}

func buildArgs(opts ExtractOptions, pdfPath string) []string {
	args := []string{}

	if opts.Layout {
		args = append(args, "-layout")
	}
	if opts.NoPgBrk {
		args = append(args, "-nopgbrk")
	}
	if opts.Encoding != "" {
		args = append(args, "-enc", opts.Encoding)
	}

	args = append(args, pdfPath, "-")
	return args
}
