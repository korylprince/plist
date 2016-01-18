package plist

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

var decodeTests = []struct {
	out interface{}
	in  string
}{
	{"foo", fooRef},
	{"UTF-8 ☼", utf8Ref},
	{uint64(0), zeroRef},
	{uint64(1), oneRef},
	{1.2, realRef},
	{false, falseRef},
	{true, trueRef},
	{[]interface{}{"a", "b", "c", uint64(4), true}, arrRef},
	{time.Date(1900, 01, 01, 12, 00, 00, 0, time.UTC), time1900Ref},
	{map[string]interface{}{
		"foo":  "bar",
		"bool": true},
		dictRef},
}

func TestDecodeEmptyInterface(t *testing.T) {
	for _, tt := range decodeTests {
		var out interface{}
		if err := Unmarshal([]byte(tt.in), &out); err != nil {
			t.Error(err)
			continue
		}
		eq := reflect.DeepEqual(out, tt.out)
		if !eq {
			t.Errorf("Unmarshal(%v) = \n%v, want %v", tt.in, out, tt.out)
		}
	}
}

func TestDecodeDict(t *testing.T) {
	// Test struct
	expected := struct {
		InfoDictionaryVersion string `plist:"CFBundleInfoDictionaryVersion"`
		BandSize              uint64 `plist:"band-size"`
		BackingStoreVersion   int    `plist:"bundle-backingstore-version"`
		DiskImageBundleType   string `plist:"diskimage-bundle-type"`
		Size                  uint64 `plist:"size"`
	}{
		InfoDictionaryVersion: "6.0",
		BandSize:              8388608,
		Size:                  4 * 1048576 * 1024 * 1024,
		DiskImageBundleType:   "com.apple.diskimage.sparsebundle",
		BackingStoreVersion:   1,
	}

	var sparseBundleHeader struct {
		InfoDictionaryVersion string `plist:"CFBundleInfoDictionaryVersion"`
		BandSize              uint64 `plist:"band-size"`
		BackingStoreVersion   int    `plist:"bundle-backingstore-version"`
		DiskImageBundleType   string `plist:"diskimage-bundle-type"`
		Size                  uint64 `plist:"size"`
	}

	if err := Unmarshal([]byte(indentRef), &sparseBundleHeader); err != nil {
		t.Fatal(err)
	}
	if sparseBundleHeader != expected {
		t.Error("Expected", expected, "got", sparseBundleHeader)
	}

	// Test Map
	var mapHeader = map[string]interface{}{}
	// Output map[CFBundleInfoDictionaryVersion:6.0 band-size:8388608 bundle-backingstore-version:1 diskimage-bundle-type:com.apple.diskimage.sparsebundle size:4398046511104]
	if err := Unmarshal([]byte(indentRef), &mapHeader); err != nil {
		t.Fatal(err)
	}
	if mapHeader["CFBundleInfoDictionaryVersion"] != "6.0" {
		t.Fatal("Expected", "6.0", "got", mapHeader["CFBundleInfoDictionaryVersion"])
	}
}
func TestDecodeArray(t *testing.T) {
	const input = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><array><string>foo</string><string>bar</string></array></plist>`
	var data []string
	expected := []string{"foo", "bar"}
	if err := Unmarshal([]byte(input), &data); err != nil {
		t.Fatal(err)
	}
	if eq := reflect.DeepEqual(data, expected); !eq {
		t.Error("Expected", expected, "got", data)
	}
}

func TestDecodeBoolean(t *testing.T) {
	const input = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><true/></plist>`
	var data bool
	expected := true
	if err := Unmarshal([]byte(input), &data); err != nil {
		t.Fatal(err)
	}
	if data != expected {
		t.Error("Expected", expected, "got", data)
	}
}

func TestDecodeReal(t *testing.T) {
	const input = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><real>1.2</real></plist>`
	var data float64
	expected := 1.2
	if err := Unmarshal([]byte(input), &data); err != nil {
		t.Fatal(err)
	}
	if data != expected {
		t.Error("Expected", expected, "got", data)
	}
}

func TestDecodeDate(t *testing.T) {
	const input = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <date>2011-05-12T01:00:00Z</date>
</plist>`
	var data time.Time
	expected, _ := time.Parse(time.RFC3339, "2011-05-12T01:00:00Z")
	if err := Unmarshal([]byte(input), &data); err != nil {
		t.Fatal(err)
	}
	if data != expected {
		t.Error("Expected", expected, "got", data)
	}
}

func TestDecodeData(t *testing.T) {
	expected := `PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4KPCFET0NUWVBFIHBsaXN0IFBVQkxJQyAiLS8vQXBwbGUvL0RURCBQTElTVCAxLjAvL0VOIiAiaHR0cDovL3d3dy5hcHBsZS5jb20vRFREcy9Qcm9wZXJ0eUxpc3QtMS4wLmR0ZCI+CjxwbGlzdCB2ZXJzaW9uPSIxLjAiPjxzdHJpbmc+Zm9vPC9zdHJpbmc+PC9wbGlzdD4=`
	type data []byte
	out := data{}
	if err := Unmarshal([]byte(dataRef), &out); err != nil {
		t.Fatal(err)
	}
	if string(out) != expected {
		t.Error("Expected", expected, "got", string(out))
	}
}

// Unknown struct fields should return an error
func TestDecodeUnknownStructField(t *testing.T) {
	var sparseBundleHeader struct {
		InfoDictionaryVersion string `plist:"CFBundleInfoDictionaryVersion"`
		BandSize              uint64 `plist:"band-size"`
		BackingStoreVersion   int    `plist:"bundle-backingstore-version"`
		DiskImageBundleType   string `plist:"diskimage-bundle-type"`
		Size                  uint64 `plist:"unknownKey"`
	}
	if err := Unmarshal([]byte(indentRef), &sparseBundleHeader); err == nil {
		t.Error("Expected error `plist: unknown struct field unknownKey`, got nil")
	}
}

func TestHTTPDecoding(t *testing.T) {
	const raw = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><string>bar</string></plist>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(raw))
	}))
	defer ts.Close()
	res, err := http.Get(ts.URL)
	if err != nil {
		log.Fatalf("GET failed: %v", err)
	}
	defer res.Body.Close()
	var foo string
	d := NewDecoder(res.Body)
	err = d.Decode(&foo)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if foo != "bar" {
		t.Errorf("decoded %q; want \"bar\"", foo)
	}
	err = d.Decode(&foo)
	if err != io.EOF {
		t.Errorf("err = %v; want io.EOF", err)
	}
}
