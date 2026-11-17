package appbundle

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteBundle(t *testing.T) {
	modelDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(modelDir, "meta.json"), []byte(`{"version":"20261116-1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "model.int8.onnx"), []byte("onnx"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	pack := []byte(`{"version":"abc","lang":"de","species":[{"s":"A"},{"s":"B"}]}`)
	manifest, err := Write(out, pack, modelDir)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "abc-20261116-1" || manifest.Files["pack"].Species != 2 || !manifest.Files["pack"].Gzip {
		t.Fatalf("manifest = %+v", manifest)
	}
	var stored Manifest
	raw, _ := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err := json.Unmarshal(raw, &stored); err != nil || stored.Version != manifest.Version {
		t.Fatalf("stored manifest %s: %v", raw, err)
	}
	for key, file := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(out, file.Path))
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA256 || len(data) != file.Size {
			t.Errorf("%s: checksum or size mismatch", key)
		}
	}
	gz, _ := os.ReadFile(filepath.Join(out, "pack-abc.json.gz"))
	reader, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		t.Fatal(err)
	}
	if unpacked, _ := io.ReadAll(reader); !bytes.Equal(unpacked, pack) {
		t.Error("pack does not round-trip")
	}
}
