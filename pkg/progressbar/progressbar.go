package progressbar

import (
	"fmt"
	"strings"
)

type ProgressBar struct {
	total int
}

const barWidth = 80

func New(total int) *ProgressBar {
	return &ProgressBar{total: total}
}

func (pb *ProgressBar) Start() {
	bar := pb.getBarString(0)
	fmt.Printf("%s", bar)
}

func (pb *ProgressBar) Update(current int) {
	bar := pb.getBarString(current)
	fmt.Printf("\r%s", bar)
}

func (pb *ProgressBar) Finish() {
	fmt.Println()
}

func (pb *ProgressBar) getBarString(completed int) string {
	percent := float64(completed) / float64(pb.total) * 100
	completeWidth := int(percent) * barWidth / 100

	var sb strings.Builder

	sb.WriteString("Downloading  ")
	sb.WriteString("[")
	sb.WriteString(strings.Repeat("#", completeWidth))
	sb.WriteString(strings.Repeat(".", barWidth-completeWidth))
	sb.WriteString("]")
	sb.WriteString(fmt.Sprintf(" %0.2f%%", percent))
	sb.WriteString(fmt.Sprintf("   (%d/%d)", completed, pb.total))

	return sb.String()
}
