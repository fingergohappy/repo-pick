// Package output 提供普通终端 CLI 使用的文本输出能力。
package output

import (
	"fmt"
	"io"
)

// ProgressPrinter 负责输出脚本友好的进度消息。
type ProgressPrinter struct {
	out io.Writer
}

// NewProgressPrinter 创建进度打印器。out 为空时打印操作会被忽略，便于测试或静默模式复用。
func NewProgressPrinter(out io.Writer) ProgressPrinter {
	return ProgressPrinter{out: out}
}

// Start 输出一个正在执行的步骤。
func (p ProgressPrinter) Start(message string) {
	p.printf("[..] %s\n", message)
}

// Success 输出一个已成功完成的步骤。
func (p ProgressPrinter) Success(message string) {
	p.printf("[ok] %s\n", message)
}

// Failure 输出一个失败步骤及失败原因。
func (p ProgressPrinter) Failure(message string, err error) {
	if err == nil {
		p.printf("[!!] %s\n", message)
		return
	}
	p.printf("[!!] %s: %v\n", message, err)
}

func (p ProgressPrinter) printf(format string, args ...any) {
	if p.out == nil {
		return
	}
	_, _ = fmt.Fprintf(p.out, format, args...)
}
