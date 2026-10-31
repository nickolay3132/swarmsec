package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

type HandlerFunc func(ctx context.Context, args []string) error

type Loop struct {
	handlers map[string]HandlerFunc
	in       io.Reader
	out      io.Writer
}

func NewLoop(in io.Reader, out io.Writer) *Loop {
	return &Loop{
		handlers: make(map[string]HandlerFunc),
		in:       in,
		out:      out,
	}
}

func (l *Loop) Register(name string, h HandlerFunc) error {
	if name == "" {
		return errors.New("command name must not be empty")
	}
	if strings.ContainsAny(name, " \t\n") {
		return fmt.Errorf("command name %q must not contain whitespace", name)
	}
	if h == nil {
		return fmt.Errorf("handler for %q is nil", name)
	}
	l.handlers[name] = h
	return nil
}

func (l *Loop) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(l.in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		fmt.Fprint(l.out, "> ")

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read input: %w", err)
			}
			fmt.Fprintln(l.out)
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "exit" || line == "quit" {
			return nil
		}

		l.dispatch(ctx, line)
	}
}

func (l *Loop) dispatch(ctx context.Context, line string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}

	cmd := fields[0]
	args := fields[1:]

	h, ok := l.handlers[cmd]
	if !ok {
		fmt.Fprintf(l.out, "unknown command: %q\n", cmd)
		return
	}

	if err := h(ctx, args); err != nil {
		fmt.Fprintf(l.out, "error: %v\n", err)
	}
}
