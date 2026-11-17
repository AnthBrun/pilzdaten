// Package appbundle baut das Daten-Bundle der nativen App (Repo pilzkorb).
//
// Die App lädt beim ersten Öffnen manifest.json und die Dateien darin, prüft
// sha256 und schaltet erst danach um. Alle Dateien außer manifest.json tragen
// ihre Version im Namen und ändern sich nie:
//
//	manifest.json
//	pack-<version>.json.gz            Artdaten (web.AppPackage)
//	model-<version>/meta.json         Bildmodell (wie MODEL_DIR)
//	model-<version>/model.int8.onnx
//
// Das Format liest pilzkorb/lib/data/bundle_store.dart; beide gemeinsam ändern.
package appbundle

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Format zählt hoch, wenn sich der Aufbau inkompatibel ändert.
const Format = 1

type File struct {
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Size    int    `json:"size"`
	Version string `json:"version"`
	Gzip    bool   `json:"gzip,omitempty"`
	Species int    `json:"species,omitempty"`
}

type Manifest struct {
	Format  int             `json:"format"`
	Version string          `json:"version"`
	Created string          `json:"created"`
	Files   map[string]File `json:"files"`
}

// Write legt das Bundle in outDir an. pack ist die Ausgabe von
// web.AppPackage, modelDir enthält meta.json und model.int8.onnx.
func Write(outDir string, pack []byte, modelDir string) (Manifest, error) {
	var header struct {
		Version string            `json:"version"`
		Species []json.RawMessage `json:"species"`
	}
	if err := json.Unmarshal(pack, &header); err != nil || header.Version == "" {
		return Manifest{}, fmt.Errorf("App-Paket ohne Version: %v", err)
	}
	meta, err := os.ReadFile(filepath.Join(modelDir, "meta.json"))
	if err != nil {
		return Manifest{}, err
	}
	onnx, err := os.ReadFile(filepath.Join(modelDir, "model.int8.onnx"))
	if err != nil {
		return Manifest{}, err
	}
	var model struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(meta, &model); err != nil || model.Version == "" {
		return Manifest{}, fmt.Errorf("meta.json ohne Version: %v", err)
	}

	var packed bytes.Buffer
	writer, _ := gzip.NewWriterLevel(&packed, gzip.BestCompression)
	if _, err := writer.Write(pack); err != nil {
		return Manifest{}, err
	}
	if err := writer.Close(); err != nil {
		return Manifest{}, err
	}

	packFile := fmt.Sprintf("pack-%s.json.gz", header.Version)
	modelPrefix := fmt.Sprintf("model-%s/", model.Version)
	files := map[string][]byte{packFile: packed.Bytes(), modelPrefix + "meta.json": meta, modelPrefix + "model.int8.onnx": onnx}
	for name, data := range files {
		target := filepath.Join(outDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return Manifest{}, err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return Manifest{}, err
		}
	}
	manifest := Manifest{
		Format:  Format,
		Version: header.Version + "-" + model.Version,
		Created: time.Now().UTC().Format(time.RFC3339),
		Files: map[string]File{
			"pack":       describe(packFile, packed.Bytes(), header.Version, true, len(header.Species)),
			"model_meta": describe(modelPrefix+"meta.json", meta, model.Version, false, 0),
			"model":      describe(modelPrefix+"model.int8.onnx", onnx, model.Version, false, 0),
		},
	}
	raw, err := json.MarshalIndent(manifest, "", " ")
	if err != nil {
		return Manifest{}, err
	}
	// manifest.json zuletzt: Wer es sieht, findet alle Dateien vor.
	return manifest, os.WriteFile(filepath.Join(outDir, "manifest.json"), append(raw, '\n'), 0o644)
}

func describe(path string, data []byte, version string, packed bool, species int) File {
	sum := sha256.Sum256(data)
	return File{Path: path, SHA256: hex.EncodeToString(sum[:]), Size: len(data), Version: version, Gzip: packed, Species: species}
}
