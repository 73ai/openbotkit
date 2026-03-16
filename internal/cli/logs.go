package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

func printTail(f *os.File, n int) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	if size == 0 {
		return nil
	}

	// Scan backward to find the last n newlines.
	buf := make([]byte, 1)
	found := 0
	pos := size
	for pos > 0 && found <= n {
		pos--
		if _, err := f.ReadAt(buf, pos); err != nil {
			return err
		}
		if buf[0] == '\n' {
			found++
		}
	}
	if pos > 0 || found > n {
		pos++ // skip past the newline we stopped on
	}

	if _, err := f.Seek(pos, io.SeekStart); err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, f)
	return err
}

func followFile(ctx context.Context, f *os.File) error {
	scanner := bufio.NewScanner(f)
	for {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			// Poll for new data — lightweight compared to fsnotify dep.
			time.Sleep(200 * time.Millisecond)
			scanner = bufio.NewScanner(f)
		}
	}
}
