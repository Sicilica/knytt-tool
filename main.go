package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

type World struct {
	Name  string
	Files map[string][]byte
}

func main() {
	outFile := flag.String("o", "", "Output file")
	flag.Parse()

	inFile := flag.Arg(0)
	if *outFile == "" || inFile == "" {
		// TODO this is just awful tbh
		flag.Usage()
	}

	if err := decompress(inFile, *outFile); err != nil {
		log.Fatal(err)
	}
}

func decompress(inFile, outFile string) error {
	w, err := loadKnyttBin(inFile)
	if err != nil {
		return err
	}
	return w.SaveFolder(outFile)
}

// Adapted in part from https://github.com/andrewmd5/KnyttSharp
func loadKnyttBin(file string) (*World, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	b := make([]byte, 4)

	// Header
	if _, err := io.ReadFull(r, b[:2]); err != nil {
		return nil, err
	}
	if string(b[:2]) != "NF" {
		return nil, errors.New("not a valid knytt world file (missing NF header)")
	}

	w := World{}

	// Level name
	w.Name, err = r.ReadString(0)
	if err != nil {
		return nil, err
	}
	w.Name = strings.ReplaceAll(w.Name, "\u0000", "")

	// File count
	if _, err := io.ReadFull(r, b[:4]); err != nil {
		return nil, err
	}
	numFiles := binary.LittleEndian.Uint32(b)

	// Parse files
	// (Note: the file count read from the file is 53 for my test, but with 47 actual files; no clue why)
	w.Files = make(map[string][]byte, numFiles)
	for {
		// Each entry has its own header
		if n, err := io.ReadFull(r, b[:2]); err != nil {
			if err == io.EOF && n == 0 {
				break
			}
			return nil, err
		}
		if string(b[:2]) != "NF" {
			return nil, errors.New("invalid entry header")
		}

		// Name
		name, err := r.ReadString(0)
		if err != nil {
			return nil, err
		}
		name = strings.ReplaceAll(name, "\u0000", "")

		// Size
		if _, err := io.ReadFull(r, b[:4]); err != nil {
			return nil, err
		}
		size := binary.LittleEndian.Uint32(b)

		// Data
		w.Files[name] = make([]byte, size)
		if _, err := io.ReadFull(r, w.Files[name]); err != nil {
			return nil, err
		}
	}

	return &w, nil
}

func (w *World) SaveFolder(base string) error {
	base = strings.TrimSpace(path.Join(base, w.Name))

	for n, d := range w.Files {
		file := path.Join(base, strings.ReplaceAll(n, "\\", "/"))

		if err := os.MkdirAll(path.Dir(file), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(file, d, 0644); err != nil {
			return err
		}
	}

	return nil
}
