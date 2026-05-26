package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func findSdlAudioSinkInputID() string {
	cmd := exec.Command("pactl", "list", "sink-inputs")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to list sink inputs: %v", err)
		return ""
	}

	parts := strings.Split(string(output), "Sink Input #")
	for _, part := range parts {
		if strings.Contains(part, `application.name = "SDL Application"`) {
			for _, line := range strings.Split(part, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "object.id = ") {
					idVal := strings.TrimPrefix(trimmed, "object.id = ")
					idVal = strings.Trim(idVal, `"`)
					return idVal
				}
			}
		}
	}
	return ""
}

func setPipeWireVolume(volume int64) {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}

	nodeID := findSdlAudioSinkInputID()
	if nodeID == "" {
		log.Println("PipeWire: no SDL Application sink input found, skipping volume change")
		return
	}

	// Unmute if volume is greater than 0
	if volume > 0 {
		if err := exec.Command("wpctl", "set-mute", nodeID, "0").Run(); err != nil {
			log.Printf("PipeWire: failed to unmute on volume change: %v", err)
		}
	}

	vol := fmt.Sprintf("%.2f", float64(volume)/100.0)
	if err := exec.Command("wpctl", "set-volume", nodeID, vol).Run(); err != nil {
		log.Printf("PipeWire: failed to set volume: %v", err)
		return
	}
	log.Printf("PipeWire: set sink input %s volume to %d%%", nodeID, volume)
}

func setPipeWireMute(mute bool) {
	nodeID := findSdlAudioSinkInputID()
	if nodeID == "" {
		log.Println("PipeWire: no SDL Application sink input found, skipping mute change")
		return
	}

	val := "0"
	if mute {
		val = "1"
	}
	if err := exec.Command("wpctl", "set-mute", nodeID, val).Run(); err != nil {
		log.Printf("PipeWire: failed to set mute: %v", err)
		return
	}
	log.Printf("PipeWire: set sink input %s mute to %v", nodeID, mute)

	if mute {
		Config.SavedUIState.Volume = 0
	} else {
		Config.SavedUIState.Volume = 100
	}
}

func getPipeWireVolume() int64 {
	nodeID := findSdlAudioSinkInputID()
	if nodeID == "" {
		log.Println("PipeWire: no SDL Application sink input found")
		return -1
	}

	cmd := exec.Command("wpctl", "get-volume", nodeID)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("PipeWire: failed to get volume: %v", err)
		return -1
	}

	parts := strings.Fields(string(output))
	if len(parts) < 2 {
		return -1
	}

	vol, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return -1
	}

	return int64(vol * 100)
}
