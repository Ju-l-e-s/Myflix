package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type FFProbeOutput struct {
	Streams []Stream `json:"streams"`
}

type Stream struct {
	Index     int    `json:"index"`
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Tags      struct {
		Language string `json:"language"`
		Title    string `json:"title"`
	} `json:"tags"`
}

var (
	dryRun  = flag.Bool("dry-run", false, "Do not perform actions, just print")
	verbose = flag.Bool("v", false, "Verbose output")
	jobs    = flag.Int("j", 2, "Number of concurrent jobs")
)

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Println("Usage: mkvcleaner [options] <path>")
		os.Exit(1)
	}

	path := flag.Arg(0)
	
	// Use a worker pool
	filesChan := make(chan string, 100)
	var wg sync.WaitGroup

	// Start workers
	if *jobs <= 0 {
		*jobs = runtime.NumCPU()
	}
	for i := 0; i < *jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range filesChan {
				processMKV(file)
			}
		}()
	}

	err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.ToLower(filepath.Ext(p)) == ".mkv" {
			filesChan <- p
		}
		return nil
	})

	close(filesChan)
	wg.Wait()

	if err != nil {
		log.Fatalf("Error walking path: %v", err)
	}
}

func processMKV(file string) {
	fmt.Printf("Analyzing %s...\n", filepath.Base(file))

	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "stream=index,codec_type,codec_name:stream_tags=language,title", "-of", "json", file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  Error analyzing file %s: %v\n", file, err)
		return
	}

	var ff FFProbeOutput
	if err := json.Unmarshal(output, &ff); err != nil {
		fmt.Printf("  Error parsing JSON for %s: %v\n", file, err)
		return
	}

	var videoTracks []string
	var audioTracksToKeep []string
	var audioTracksToConvert []string
	var subtitleTracksToExtract []Stream
	var pgsVobsubFound bool
	var hdAudioFound bool

	for _, s := range ff.Streams {
		switch s.CodecType {
		case "video":
			videoTracks = append(videoTracks, fmt.Sprintf("%d", s.Index))
		case "audio":
			isHD := s.CodecName == "truehd" || s.CodecName == "dts-hd ma" || strings.Contains(strings.ToLower(s.CodecName), "hd")
			if isHD {
				hdAudioFound = true
				fmt.Printf("  Found HD audio: %s (track %d) in %s\n", s.CodecName, s.Index, filepath.Base(file))
				audioTracksToConvert = append(audioTracksToConvert, fmt.Sprintf("%d", s.Index))
			} else {
				audioTracksToKeep = append(audioTracksToKeep, fmt.Sprintf("%d", s.Index))
			}
		case "subtitle":
			if s.CodecName == "subrip" || s.CodecName == "srt" {
				subtitleTracksToExtract = append(subtitleTracksToExtract, s)
			} else if s.CodecName == "hdmv_pgs_subtitle" || s.CodecName == "dvd_subtitle" || s.CodecName == "vobsub" || s.CodecName == "mov_text" {
				pgsVobsubFound = true
				fmt.Printf("  Found incompatible subtitle: %s (track %d) in %s\n", s.CodecName, s.Index, filepath.Base(file))
			}
		}
	}

	// 1. Extract SRT subtitles
	for _, s := range subtitleTracksToExtract {
		lang := s.Tags.Language
		if lang == "" {
			lang = "und"
		}
		srtFile := fmt.Sprintf("%s.%s.srt", strings.TrimSuffix(file, filepath.Ext(file)), lang)
		if _, err := os.Stat(srtFile); os.IsNotExist(err) {
			fmt.Printf("  Extracting SRT subtitle track %d (%s) from %s...\n", s.Index, lang, filepath.Base(file))
			if !*dryRun {
				extractCmd := exec.Command("ffmpeg", "-y", "-i", file, "-map", fmt.Sprintf("0:%d", s.Index), srtFile)
				if out, err := extractCmd.CombinedOutput(); err != nil {
					fmt.Printf("    Error extracting subtitle from %s: %v\n%s\n", file, err, string(out))
				}
			}
		}
	}

	// 2. Decide what to do with Audio and Video
	needsAction := hdAudioFound || pgsVobsubFound

	if needsAction {
		fmt.Printf("  Cleaning %s (optimizing for Direct Play)...\n", filepath.Base(file))
		tempFile := file + ".clean.mkv"
		
		var args []string
		args = append(args, "-y", "-i", file)
		
		// Map Video
		for _, vID := range videoTracks {
			args = append(args, "-map", fmt.Sprintf("0:%s", vID))
		}

		// Map Audio
		if len(audioTracksToKeep) > 0 {
			for _, aID := range audioTracksToKeep {
				args = append(args, "-map", fmt.Sprintf("0:%s", aID))
			}
		} else if len(audioTracksToConvert) > 0 {
			// Convert the first HD track to AC3 5.1
			args = append(args, "-map", fmt.Sprintf("0:%s", audioTracksToConvert[0]))
		}

		// Set Codecs
		args = append(args, "-c:v", "copy")
		if len(audioTracksToKeep) > 0 {
			args = append(args, "-c:a", "copy")
		} else if len(audioTracksToConvert) > 0 {
			args = append(args, "-c:a:0", "ac3", "-b:a:0", "640k", "-ac:a:0", "6")
		}

		args = append(args, "-sn") // No internal subtitles
		args = append(args, tempFile)

		if !*dryRun {
			cleanCmd := exec.Command("ffmpeg", args...)
			if out, err := cleanCmd.CombinedOutput(); err != nil {
				fmt.Printf("    Error cleaning %s: %v\n%s\n", file, err, string(out))
			} else {
				if err := os.Rename(tempFile, file); err != nil {
					fmt.Printf("    Error replacing %s: %v\n", file, err)
				} else {
					fmt.Printf("    SUCCESS: %s optimized.\n", filepath.Base(file))
				}
			}
		} else {
			fmt.Printf("    Dry-run: would run ffmpeg for %s\n", filepath.Base(file))
		}
	} else {
		if *verbose {
			fmt.Printf("  %s is already optimized.\n", filepath.Base(file))
		}
	}
}
