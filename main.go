package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/devfacet/gocmd/v3"
	"github.com/sirupsen/logrus"
)

const (
	ErrCodeSuccess = 0
	ErrCodeFailure = 1
)

type MKVTrackProperties struct {
	CodecId              string  `json:"codec_id"`
	TrackName            string  `json:"track_name"`
	Encoding             string  `json:"encoding"`
	Language             string  `json:"language"`
	Number               int     `json:"number"`
	Forced               bool    `json:"forced_track"`
	Default              bool    `json:"default_track"`
	Enabled              bool    `json:"enabled_track"`
	TextSubtitles        bool    `json:"text_subtitles"`
	NumberOfIndexEntries int     `json:"num_index_entries"`
	Duration             string  `json:"tag_duration"`
	UId                  big.Int `json:"uid"`
}

type MKVTrack struct {
	Codec      string             `json:"codec"`
	Id         int                `json:"id"`
	Type       string             `json:"type"`
	Properties MKVTrackProperties `json:"properties"`
}

type MKVContainer struct {
	Type string `json:"type"`
}

type MKVInfo struct {
	Tracks    []MKVTrack   `json:"tracks"`
	Container MKVContainer `json:"container"`
}

var subtitleExtensionByCodec = map[string]string{
	"S_TEXT/UTF8": "srt",
	"S_TEXT/ASS":  "ass",
	"S_HDMV/PGS":  "sup",
}

func isMKVFile(inputFileName string) bool {
	return strings.HasSuffix(strings.ToLower(inputFileName), ".mkv")
}

func buildSubtitlesFileName(inputFileName string, track MKVTrack) string {
	baseDir := path.Dir(inputFileName)
	fileName := path.Base(inputFileName)
	extension := path.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, extension)
	trackNo := fmt.Sprintf("%03s", strconv.Itoa(track.Properties.Number))
	outFileName := fmt.Sprintf("%s.%s.%s", baseName, track.Properties.Language, trackNo)
	if track.Properties.TrackName != "" {
		outFileName = fmt.Sprintf("%s.%s", outFileName, track.Properties.TrackName)
	}
	if track.Properties.Forced {
		outFileName = fmt.Sprintf("%s.%s", outFileName, ".forced")
	}
	outFileName = fmt.Sprintf("%s.%s", outFileName, subtitleExtensionByCodec[track.Properties.CodecId])
	outFileName = path.Join(baseDir, outFileName)
	return outFileName
}

func extractSubtitles(inputFileName string, track MKVTrack, outFileName string) error {
	cmd := exec.Command(
		"mkvextract",
		fmt.Sprintf("%v", inputFileName),
		"tracks",
		fmt.Sprintf("%d:%v", track.Id, outFileName),
	)
	output, cmdErr := cmd.Output()
	if cmdErr != nil {
		logrus.
			WithField("cmd", cmd).
			WithField("inputFileName", inputFileName).
			WithField("track", track).
			WithField("outFileName", outFileName).
			WithError(cmdErr).
			Error("Error executing extract command")
		fmt.Println(string(output))
		return cmdErr
	}
	logrus.
		WithField("outFileName", outFileName).
		Info("Subtitles extracted")
	return nil
}

func main() {
	logrus.Println("gmmmkvsubsextract - GMM MKV Subtitles Extract")
	flags := struct {
		Extract string `short:"x" long:"extract" description:"Extract subtitles from MKV file" required:"true"`
	}{}
	_, extractHandleFlagErr := gocmd.HandleFlag("Extract", func(cmd *gocmd.Cmd, args []string) error {
		var inputFileName = flags.Extract
		if ifs, statErr := os.Stat(inputFileName); os.IsNotExist(statErr) || ifs.IsDir() {
			logrus.
				WithError(statErr).
				WithField("inputFileName", inputFileName).
				Errorf("File does not exist or is a directory: %s", inputFileName)
			return statErr
		}
		if !isMKVFile(inputFileName) {
			logrus.
				WithField("inputFileName", inputFileName).
				Error("File is not an MKV file")
			return errors.New("file is not an MKV file")
		}
		out, cmdErr := exec.Command("mkvmerge", "-J", inputFileName).Output()
		if cmdErr != nil {
			logrus.
				WithError(cmdErr).
				Error("Error executing command")
			return cmdErr
		}
		var mkvInfo MKVInfo
		jsonErr := json.Unmarshal(out, &mkvInfo)
		if jsonErr != nil {
			logrus.
				WithError(jsonErr).
				Error("Error parsing JSON")
			return jsonErr
		}
		if !(strings.ToLower(strings.TrimSpace(mkvInfo.Container.Type)) == "matroska") {
			logrus.
				WithField("containerType", mkvInfo.Container.Type).
				Error("File is not a Matroska container")
			return errors.New("file is not a Matroska container")
		}
		for _, track := range mkvInfo.Tracks {
			if track.Type == "subtitles" {
				logrus.
					WithField("trackId", track.Id).
					WithField("trackNumber", track.Properties.Number).
					WithField("trackLanguage", track.Properties.Language).
					WithField("trackCodec", track.Codec).
					Infof("Extracting subtitles from track %d", track.Id)
				outFileName := buildSubtitlesFileName(inputFileName, track)
				extractSubsErr := extractSubtitles(inputFileName, track, outFileName)
				if extractSubsErr != nil {
					logrus.WithError(extractSubsErr).Error("Error extracting subtitles")
					return extractSubsErr
				}
			}
		}
		return nil
	})
	if extractHandleFlagErr != nil {
		logrus.
			WithError(extractHandleFlagErr).
			Errorf("Error handling flag")
		os.Exit(ErrCodeFailure)
	}
	_, cmdErr := gocmd.New(gocmd.Options{
		Name:        "gmmmkvsubsextract",
		Description: "GMM MKV Subtitles Extract",
		Version:     "1.0.0",
		Flags:       &flags,
		ConfigType:  gocmd.ConfigTypeAuto,
	})
	if cmdErr != nil {
		logrus.
			WithError(cmdErr).
			Error("Error creating command")
		return
	}
	os.Exit(ErrCodeSuccess)
}
