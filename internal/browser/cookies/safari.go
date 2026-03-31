package cookies

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// macAbsoluteTimeEpoch is the Unix timestamp of the Mac absolute time epoch
// (2001-01-01 00:00:00 UTC).
const macAbsoluteTimeEpoch = 978307200

type Cookie struct {
	Host    string
	Name    string
	Path    string
	Value   string
	Expires time.Time
	Flags   uint32
}

func ParseBinaryCookies(r io.Reader) ([]Cookie, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	if len(data) < 8 {
		return nil, fmt.Errorf("file too short (%d bytes)", len(data))
	}

	magic := string(data[:4])
	if magic != "cook" {
		return nil, fmt.Errorf("bad magic %q, expected \"cook\"", magic)
	}

	numPages := int(binary.BigEndian.Uint32(data[4:8]))
	if numPages < 0 || numPages > 100000 {
		return nil, fmt.Errorf("unreasonable page count: %d", numPages)
	}

	offset := 8
	if len(data) < offset+numPages*4 {
		return nil, fmt.Errorf("file too short for page sizes")
	}

	pageSizes := make([]int, numPages)
	for i := range numPages {
		pageSizes[i] = int(binary.BigEndian.Uint32(data[offset:]))
		offset += 4
	}

	var cookies []Cookie
	for i := range numPages {
		pageEnd := offset + pageSizes[i]
		if pageEnd > len(data) {
			return nil, fmt.Errorf("page %d extends beyond file", i)
		}

		pageCookies, err := parsePage(data[offset:pageEnd])
		if err != nil {
			return nil, fmt.Errorf("parse page %d: %w", i, err)
		}
		cookies = append(cookies, pageCookies...)
		offset = pageEnd
	}

	return cookies, nil
}

func parsePage(page []byte) ([]Cookie, error) {
	if len(page) < 8 {
		return nil, fmt.Errorf("page too short")
	}

	// Page header: 4 bytes (0x00000100), then cookie count (LE).
	numCookies := int(binary.LittleEndian.Uint32(page[4:8]))
	if numCookies < 0 || numCookies > 100000 {
		return nil, fmt.Errorf("unreasonable cookie count: %d", numCookies)
	}

	if len(page) < 8+numCookies*4 {
		return nil, fmt.Errorf("page too short for cookie offsets")
	}

	cookieOffsets := make([]int, numCookies)
	for i := range numCookies {
		cookieOffsets[i] = int(binary.LittleEndian.Uint32(page[8+i*4:]))
	}

	var cookies []Cookie
	for _, off := range cookieOffsets {
		c, err := parseCookieRecord(page, off)
		if err != nil {
			continue
		}
		cookies = append(cookies, c)
	}

	return cookies, nil
}

func parseCookieRecord(page []byte, offset int) (Cookie, error) {
	if offset+44 > len(page) {
		return Cookie{}, fmt.Errorf("cookie record too short")
	}

	rec := page[offset:]

	flags := binary.LittleEndian.Uint32(rec[8:12])
	urlOffset := int(binary.LittleEndian.Uint32(rec[16:20]))
	nameOffset := int(binary.LittleEndian.Uint32(rec[20:24]))
	pathOffset := int(binary.LittleEndian.Uint32(rec[24:28]))
	valueOffset := int(binary.LittleEndian.Uint32(rec[28:32]))

	expiryRaw := binary.LittleEndian.Uint64(rec[40:48])
	expiryFloat := float64FromBits(expiryRaw)
	expiry := time.Unix(int64(expiryFloat)+macAbsoluteTimeEpoch, 0)

	host := readNullTerminated(rec, urlOffset)
	name := readNullTerminated(rec, nameOffset)
	path := readNullTerminated(rec, pathOffset)
	value := readNullTerminated(rec, valueOffset)

	return Cookie{
		Host:    host,
		Name:    name,
		Path:    path,
		Value:   value,
		Expires: expiry,
		Flags:   flags,
	}, nil
}

func readNullTerminated(data []byte, offset int) string {
	if offset < 0 || offset >= len(data) {
		return ""
	}
	end := offset
	for end < len(data) && data[end] != 0 {
		end++
	}
	return string(data[offset:end])
}

func float64FromBits(bits uint64) float64 {
	return math.Float64frombits(bits)
}

func ExtractSafariCookie(hosts []string, names []string) (map[string]string, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("safari cookie extraction only supported on macOS")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	paths := []string{
		filepath.Join(home, "Library", "Cookies", "Cookies.binarycookies"),
		filepath.Join(home, "Library", "Containers", "com.apple.Safari", "Data", "Library", "Cookies", "Cookies.binarycookies"),
	}

	hostSet := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		hostSet[h] = true
	}
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}

	var lastErr error
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			lastErr = err
			continue
		}

		cookies, err := ParseBinaryCookies(f)
		f.Close()
		if err != nil {
			lastErr = err
			continue
		}

		result := make(map[string]string)
		for _, c := range cookies {
			if hostSet[c.Host] && nameSet[c.Name] && c.Value != "" {
				if _, exists := result[c.Name]; !exists {
					result[c.Name] = c.Value
				}
			}
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("safari cookie extraction failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no matching cookies found in Safari")
}
