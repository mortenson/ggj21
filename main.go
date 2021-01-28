package main

import (
	"bytes"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const (
	screenWidth  = 320
	screenHeight = 180
)

type gameEngine struct {
	Frame int
	Audio map[string][]*audio.Player
}

func (g *gameEngine) Draw(screen *ebiten.Image) {
}

func (g *gameEngine) Update() error {
	return nil
}

func (g *gameEngine) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	g := newGameEngine()
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowSize(screenWidth*3, screenHeight*3)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Studio Stupor")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func newGameEngine() *gameEngine {
	g := &gameEngine{
		Audio: map[string][]*audio.Player{},
	}

	audioContext := audio.NewContext(44100)

	audioFiles := map[string][]string{
		"drum": []string{
			"audio/drum/kick.wav",
			"audio/drum/snare.wav",
			"audio/drum/stick.wav",
			"audio/drum/hihat.wav",
			"audio/drum/hihat_open.wav",
		},
	}
	for group, fileNames := range audioFiles {
		for _, fileName := range fileNames {
			file, err := os.Open(fileName)
			defer file.Close()
			if err != nil {
				panic(err)
			}
			byteValue, err := ioutil.ReadAll(file)
			if err != nil {
				panic(err)
			}
			stream, err := wav.Decode(audioContext, bytes.NewReader(byteValue))
			if err != nil {
				panic(err)
			}
			player, err := audio.NewPlayer(audioContext, stream)
			if err != nil {
				panic(err)
			}
			g.Audio[group] = append(g.Audio[group], player)
		}
	}
	return g
}
