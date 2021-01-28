package main

import (
	"bytes"
	"image/color"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 320
	screenHeight = 180
	bpm          = 120
)

type gameEngine struct {
	frame        int
	audio        map[string][]*audio.Player
	sequences    map[string][][]int
	groups       []string
	startTime    time.Time
	beatCounter  int
	playing      bool
	currentGroup string
}

func (g *gameEngine) playSequence() {
	g.startTime = time.Now()
	g.beatCounter = -1
	g.playing = true
}

func (g *gameEngine) stopSequence() {
	g.playing = false
}

func (g *gameEngine) playAudio(group string, index int) {
	g.audio[group][index].Rewind()
	g.audio[group][index].Play()
}

func (g *gameEngine) Draw(screen *ebiten.Image) {
	offsetX := 35.0
	incrementX := (screenWidth - offsetX) / 16
	incrementY := 50.0 / 5
	offsetY := incrementY
	for i := 0.0; i < 16; i++ {
		ebitenutil.DrawLine(screen, offsetX+(i*incrementX), 0, offsetX+(i*incrementX), 50, color.White)
	}
	for i := 0.0; i < 5; i++ {
		ebitenutil.DrawLine(screen, offsetX, offsetY+(incrementY*i), screenWidth-incrementX, offsetY+(incrementY*i), color.White)
	}
	if g.beatCounter != -1 {
		sequenceIndex := g.beatCounter % 16
		for _, audioIndex := range g.sequences[g.currentGroup][sequenceIndex] {
			if audioIndex != -1 {
				ebitenutil.DrawRect(screen, offsetX+(float64(sequenceIndex)*incrementX)-5, offsetY+(float64(audioIndex)*incrementY)-5, 10, 10, color.RGBA{255, 0, 255, 255})
			}
		}
	}
}

func (g *gameEngine) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.playing {
			g.stopSequence()
		} else {
			g.playSequence()
		}
	}
	g.frame++
	return nil
}

func (g *gameEngine) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *gameEngine) sequencer() {
	for {
		if g.playing {
			elapsed := time.Now().Sub(g.startTime)
			bps := 60 / float64(bpm)
			beatCounter := int(elapsed.Seconds() / bps)
			if beatCounter != g.beatCounter {
				sequenceIndex := beatCounter % 16
				for _, group := range g.groups {
					for _, audioIndex := range g.sequences[group][sequenceIndex] {
						if audioIndex != -1 {
							g.playAudio(group, audioIndex)
						}
					}
				}
			}
			g.beatCounter = beatCounter
		}
		time.Sleep(time.Millisecond * 50)
	}
}

func main() {
	g := newGameEngine()
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowSize(screenWidth*3, screenHeight*3)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Studio Stupor")
	go g.sequencer()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func newGameEngine() *gameEngine {
	g := &gameEngine{
		audio:        map[string][]*audio.Player{},
		groups:       []string{"drum"},
		sequences:    map[string][][]int{},
		currentGroup: "drum",
	}

	for _, group := range g.groups {
		g.sequences[group] = make([][]int, 16)
		for i := 0; i < 16; i++ {
			g.sequences[group][i] = make([]int, 5)
			for j := 0; j < 5; j++ {
				g.sequences[group][i][j] = -1
			}
		}
	}

	g.sequences["drum"] = [][]int{
		{0}, {2}, {1, 3}, {3}, {0}, {3, 0}, {1, 3}, {4}, {0}, {2, 0}, {3, 1}, {3}, {0}, {3}, {1, 3}, {4},
	}
	// g.sequences["drum"] = [][]int{
	// 	{0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0},
	// }

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
			g.audio[group] = append(g.audio[group], player)
		}
	}
	return g
}
