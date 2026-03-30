package cookies

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

func TestParseBinaryCookies_Basic(t *testing.T) {
	data := buildBinaryCookies(t, []testCookie{
		{host: ".x.com", name: "auth_token", path: "/", value: "abc123", expiry: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
		{host: ".x.com", name: "ct0", path: "/", value: "csrf456", expiry: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
	})

	cookies, err := ParseBinaryCookies(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseBinaryCookies: %v", err)
	}

	if len(cookies) != 2 {
		t.Fatalf("got %d cookies, want 2", len(cookies))
	}

	if cookies[0].Host != ".x.com" || cookies[0].Name != "auth_token" || cookies[0].Value != "abc123" {
		t.Errorf("cookie 0: got host=%q name=%q value=%q", cookies[0].Host, cookies[0].Name, cookies[0].Value)
	}
	if cookies[1].Host != ".x.com" || cookies[1].Name != "ct0" || cookies[1].Value != "csrf456" {
		t.Errorf("cookie 1: got host=%q name=%q value=%q", cookies[1].Host, cookies[1].Name, cookies[1].Value)
	}
}

func TestParseBinaryCookies_BadMagic(t *testing.T) {
	data := []byte("badm\x00\x00\x00\x00")
	_, err := ParseBinaryCookies(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestParseBinaryCookies_EmptyFile(t *testing.T) {
	_, err := ParseBinaryCookies(bytes.NewReader(nil))
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestParseBinaryCookies_MultipleCookies_Filtering(t *testing.T) {
	data := buildBinaryCookies(t, []testCookie{
		{host: ".x.com", name: "auth_token", path: "/", value: "tok123", expiry: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
		{host: ".google.com", name: "NID", path: "/", value: "goog", expiry: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
		{host: ".twitter.com", name: "ct0", path: "/", value: "csrf789", expiry: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)},
	})

	cookies, err := ParseBinaryCookies(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseBinaryCookies: %v", err)
	}

	if len(cookies) != 3 {
		t.Fatalf("got %d cookies, want 3", len(cookies))
	}

	// Test filtering logic (simulating what ExtractSafariCookie does)
	hosts := map[string]bool{".x.com": true, ".twitter.com": true}
	names := map[string]bool{"auth_token": true, "ct0": true}

	result := make(map[string]string)
	for _, c := range cookies {
		if hosts[c.Host] && names[c.Name] && c.Value != "" {
			if _, exists := result[c.Name]; !exists {
				result[c.Name] = c.Value
			}
		}
	}

	if result["auth_token"] != "tok123" {
		t.Errorf("auth_token = %q, want %q", result["auth_token"], "tok123")
	}
	if result["ct0"] != "csrf789" {
		t.Errorf("ct0 = %q, want %q", result["ct0"], "csrf789")
	}
	if _, ok := result["NID"]; ok {
		t.Error("NID should have been filtered out")
	}
}

func TestParseBinaryCookies_TooShortForPageSizes(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("cook")
	binary.Write(&buf, binary.BigEndian, uint32(5)) // 5 pages
	// but no page sizes follow
	_, err := ParseBinaryCookies(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- helpers ---

type testCookie struct {
	host   string
	name   string
	path   string
	value  string
	expiry time.Time
}

func buildBinaryCookies(t *testing.T, cookies []testCookie) []byte {
	t.Helper()

	// Build all cookies into a single page.
	page := buildPage(t, cookies)

	var buf bytes.Buffer
	buf.WriteString("cook")
	binary.Write(&buf, binary.BigEndian, uint32(1)) // 1 page
	binary.Write(&buf, binary.BigEndian, uint32(len(page)))
	buf.Write(page)

	// Footer (checksum + trailer)
	binary.Write(&buf, binary.BigEndian, uint32(0)) // checksum placeholder
	binary.Write(&buf, binary.BigEndian, uint64(0x071720050000004F))

	return buf.Bytes()
}

func buildPage(t *testing.T, cookies []testCookie) []byte {
	t.Helper()

	var cookieRecords [][]byte
	for _, c := range cookies {
		cookieRecords = append(cookieRecords, buildCookieRecord(t, c))
	}

	numCookies := len(cookieRecords)

	// Page header: magic (4) + count (4) + offsets (numCookies * 4) + end marker (4)
	headerSize := 4 + 4 + numCookies*4 + 4
	offsets := make([]int, numCookies)
	off := headerSize
	for i, rec := range cookieRecords {
		offsets[i] = off
		off += len(rec)
	}

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(0x00000100)) // page magic
	binary.Write(&buf, binary.LittleEndian, uint32(numCookies))
	for _, o := range offsets {
		binary.Write(&buf, binary.LittleEndian, uint32(o))
	}
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // end marker

	for _, rec := range cookieRecords {
		buf.Write(rec)
	}

	return buf.Bytes()
}

func buildCookieRecord(t *testing.T, c testCookie) []byte {
	t.Helper()

	// Cookie record layout (all LE):
	// 0-3: size (will be filled at end)
	// 4-7: unknown (0)
	// 8-11: flags
	// 12-15: unknown (0)
	// 16-19: url offset (relative to record start)
	// 20-23: name offset
	// 24-27: path offset
	// 28-31: value offset
	// 32-39: comment (empty, 0)
	// 40-47: expiry (float64 LE, Mac absolute time)
	// 48-55: creation (float64 LE, Mac absolute time)
	// then: url\0 name\0 path\0 value\0

	headerSize := 56
	urlBytes := append([]byte(c.host), 0)
	nameBytes := append([]byte(c.name), 0)
	pathBytes := append([]byte(c.path), 0)
	valueBytes := append([]byte(c.value), 0)

	urlOff := headerSize
	nameOff := urlOff + len(urlBytes)
	pathOff := nameOff + len(nameBytes)
	valueOff := pathOff + len(pathBytes)
	totalSize := valueOff + len(valueBytes)

	rec := make([]byte, totalSize)

	binary.LittleEndian.PutUint32(rec[0:4], uint32(totalSize))
	binary.LittleEndian.PutUint32(rec[8:12], 0) // flags
	binary.LittleEndian.PutUint32(rec[16:20], uint32(urlOff))
	binary.LittleEndian.PutUint32(rec[20:24], uint32(nameOff))
	binary.LittleEndian.PutUint32(rec[24:28], uint32(pathOff))
	binary.LittleEndian.PutUint32(rec[28:32], uint32(valueOff))

	// Expiry as Mac absolute time (float64 LE)
	macExpiry := float64(c.expiry.Unix() - macAbsoluteTimeEpoch)
	binary.LittleEndian.PutUint64(rec[40:48], math.Float64bits(macExpiry))

	// Creation time
	binary.LittleEndian.PutUint64(rec[48:56], math.Float64bits(0))

	copy(rec[urlOff:], urlBytes)
	copy(rec[nameOff:], nameBytes)
	copy(rec[pathOff:], pathBytes)
	copy(rec[valueOff:], valueBytes)

	return rec
}
