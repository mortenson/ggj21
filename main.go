package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/bitmapfont"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

const (
	screenWidth  = 320
	screenHeight = 180
	bpm          = 120
)

type sprite struct {
	image     *ebiten.Image
	frame     int
	numFrames int
	playing   bool
	x         float64
	y         float64
	width     int
	height    int
}

func (s *sprite) draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x, s.y)
	screen.DrawImage(s.image, op)
}

func (s *sprite) getRect() rect {
	return rect{
		x:      s.x,
		y:      s.y,
		width:  float64(s.width),
		height: float64(s.height),
	}
}

type gameEngine struct {
	frame         int
	audio         map[string][]*audio.Player
	audioLabels   map[string][]string
	sequenceRules map[string]map[int]int
	sequences     map[string][][]int
	groups        []string
	startTime     time.Time
	beatCounter   int
	playing       bool
	currentGroup  string
	cursor        [2]int
	images        map[string]*ebiten.Image
	sprites       map[string]*sprite
	dialogueOpen  bool
	dialogue      []string
	dialogueIndex int
}

type rect struct {
	x      float64
	y      float64
	width  float64
	height float64
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
	screen.DrawImage(g.images["bandroom"], &ebiten.DrawImageOptions{})
	g.sprites["pianoman"].draw(screen)
	g.sprites["drummer"].draw(screen)
	if g.dialogueOpen {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(5, screenHeight-50)
		screen.DrawImage(g.images["menubox"], op)
		text.Draw(screen, g.dialogue[g.dialogueIndex], bitmapfont.Gothic12r, 35, screenHeight-25, color.Black)
		textBounds := text.BoundString(bitmapfont.Gothic10r, "ENTER")
		text.Draw(screen, "ENTER", bitmapfont.Gothic10r, screenWidth-8-textBounds.Max.X, screenHeight-8, color.Black)
	}
	if g.currentGroup == "" {
		// if !g.dialogueOpen {
		// 	text.Draw(screen, "Click a band member", bitmapfont.Gothic10r, 5, screenHeight-5, color.Black)
		// }
		return
	}
	// text.Draw(screen, "Play/pause (ENTER)", bitmapfont.Gothic10r, 5, screenHeight-5, color.Black)
	offsetX := 55.0
	incrementX := (screenWidth - offsetX) / 16
	incrementY := 50.0 / 5
	offsetY := incrementY
	for i := 0; i < 16; i++ {
		clr := color.RGBA{255, 255, 255, 255}
		if int(i)%4 == 0 {
			clr = color.RGBA{100, 255, 0, 150}
		}
		ebitenutil.DrawLine(screen, offsetX+(float64(i)*incrementX), 0, offsetX+(float64(i)*incrementX), 50, clr)
	}
	for i := 0; i < 5; i++ {
		y := offsetY + (incrementY * float64(i))
		ebitenutil.DrawLine(screen, offsetX, y, screenWidth-incrementX, y, color.White)
		textBounds := text.BoundString(bitmapfont.Gothic10r, g.audioLabels[g.currentGroup][i])
		text.Draw(screen, g.audioLabels[g.currentGroup][i], bitmapfont.Gothic10r, int(offsetX)-textBounds.Max.X-8, int(y)+3, color.White)
	}
	if g.beatCounter != -1 {
		rects := g.getSequenceRects()
		sequenceIndex := g.beatCounter % 16
		for i, sequence := range g.sequences[g.currentGroup] {
			for _, audioIndex := range sequence {
				if audioIndex != -1 {
					cursor := [2]int{i, audioIndex}
					clr := color.RGBA{255, 0, 255, 150}
					if sequenceIndex == i {
						clr = color.RGBA{255, 0, 255, 255}
					}
					ebitenutil.DrawRect(screen, rects[cursor].x, rects[cursor].y, rects[cursor].width, rects[cursor].height, clr)
				}
			}
		}
		cursorRect, ok := rects[g.cursor]
		if ok {
			ebitenutil.DrawRect(screen, cursorRect.x, cursorRect.y, cursorRect.width, cursorRect.height, color.RGBA{255, 255, 0, 150})
		}
		ebitenutil.DrawRect(screen, offsetX+incrementX*float64(sequenceIndex)-2.5, 60, 5, 5, color.RGBA{255, 255, 0, 150})
	}
}

func (g *gameEngine) getSequenceRects() map[[2]int]rect {
	offsetX := 55.0
	incrementX := (screenWidth - offsetX) / 16
	incrementY := 50.0 / 5
	offsetY := incrementY
	rects := map[[2]int]rect{}
	for i := 0; i < 16; i++ {
		for j := 0; j < 5; j++ {
			cursor := [2]int{i, j}
			rects[cursor] = rect{
				offsetX + (float64(i) * incrementX) - 5,
				offsetY + (float64(j) * incrementY) - 5,
				10,
				10,
			}
		}
	}
	return rects
}

func isColliding(r1, r2 rect) bool {
	if r1.x < r2.x+r2.width &&
		r1.x+r1.width > r2.x &&
		r1.y < r2.y+r2.height &&
		r1.y+r1.height > r2.y {
		return true
	}
	return false
}

func (g *gameEngine) Update() error {
	if g.dialogueOpen {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.dialogueIndex++
			if g.dialogueIndex >= len(g.dialogue) {
				g.dialogueOpen = false
			}
		}
	} else {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && g.currentGroup != "" {
			if g.playing {
				g.stopSequence()
			} else {
				g.playSequence()
			}
		}
		mouseX, mouseY := ebiten.CursorPosition()
		mouseRect := rect{
			x:      float64(mouseX),
			y:      float64(mouseY),
			width:  1,
			height: 1,
		}
		hovering := false
		for cursor, sequenceRect := range g.getSequenceRects() {
			if isColliding(sequenceRect, mouseRect) {
				g.cursor = cursor
				hovering = true
			}
		}
		if !hovering {
			g.cursor = [2]int{-1, -1}
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.cursor != [2]int{-1, -1} {
			sliceIndex := -1
			for i, audioIndex := range g.sequences[g.currentGroup][g.cursor[0]] {
				if audioIndex == g.cursor[1] {
					sliceIndex = i
					break
				}
			}
			if sliceIndex != -1 {
				g.sequences[g.currentGroup][g.cursor[0]] = append(g.sequences[g.currentGroup][g.cursor[0]][:sliceIndex], g.sequences[g.currentGroup][g.cursor[0]][sliceIndex+1:]...)
			} else {
				g.sequences[g.currentGroup][g.cursor[0]] = append(g.sequences[g.currentGroup][g.cursor[0]], g.cursor[1])
			}
		}
		showDialogue := false
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && isColliding(mouseRect, g.sprites["drummer"].getRect()) {
			g.currentGroup = "drum"
			showDialogue = true
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && isColliding(mouseRect, g.sprites["pianoman"].getRect()) {
			g.currentGroup = "piano"
			showDialogue = true
		}
		if showDialogue {
			g.dialogueIndex = 0
			g.dialogueOpen = true
			g.dialogue = []string{}
			for audioIndex, count := range g.sequenceRules[g.currentGroup] {
				g.dialogue = append(g.dialogue, fmt.Sprintf("I remember %s being played %d times", g.audioLabels[g.currentGroup][audioIndex], count))
			}
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
			bps := 60 / float64(bpm) / 4
			beatCounter := int(elapsed.Seconds() / bps)
			if beatCounter != g.beatCounter {
				sequenceIndex := beatCounter % 16
				for _, group := range g.groups {
					for _, audioIndex := range g.sequences[group][sequenceIndex] {
						if audioIndex != -1 {
							go g.playAudio(group, audioIndex)
						}
					}
				}
			}
			g.beatCounter = beatCounter
		}
		time.Sleep(time.Millisecond)
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

func loadImage(filename string) *ebiten.Image {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(bytes.NewReader(byteValue))
	if err != nil {
		panic(err)
	}
	return ebiten.NewImageFromImage(img)
}

func newGameEngine() *gameEngine {
	rand.Seed(time.Now().Unix())

	g := &gameEngine{
		audio:         map[string][]*audio.Player{},
		audioLabels:   map[string][]string{},
		groups:        []string{"drum", "piano"},
		sequences:     map[string][][]int{},
		sequenceRules: map[string]map[int]int{},
		currentGroup:  "",
		beatCounter:   0,
		cursor:        [2]int{-1, -1},
		images:        map[string]*ebiten.Image{},
		sprites:       map[string]*sprite{},
		dialogueOpen:  true,
		dialogue: []string{
			"Can't believe we have to record today...",
			"We drank so much last night.",
			"I remember writing a song.",
			"I can't remember a thing...",
			"Well we gotta use the studio time!",
			"Let's wing it and remember as we go.",
			"[Click on a band member to write music]",
			"[Sequence notes with your mouse]",
			"[Play/restart sequence with the ENTER key]",
			"[Make sure your tune matches their memory]",
			"[Set your own challenges and get creative]",
		},
		dialogueIndex: 0,
	}

	for _, group := range g.groups {
		g.sequences[group] = make([][]int, 16)
		for i := 0; i < 16; i++ {
			g.sequences[group][i] = make([]int, 5)
			for j := 0; j < 5; j++ {
				g.sequences[group][i][j] = -1
			}
		}
		g.sequenceRules[group] = map[int]int{}
		audioIndex := rand.Intn(5)
		g.sequenceRules[group][audioIndex] = rand.Intn(9) + 1
		for {
			newIndex := rand.Intn(5)
			if newIndex != audioIndex {
				audioIndex = newIndex
				break
			}
		}
		g.sequenceRules[group][audioIndex] = rand.Intn(9) + 1
	}

	// g.sequences["drum"] = [][]int{
	// 	{0}, {2}, {1, 3}, {3}, {0}, {3, 0}, {1, 3}, {4}, {0}, {2, 0}, {3, 1}, {3}, {0}, {3}, {1, 3}, {4},
	// }
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
		"piano": []string{
			"audio/piano/f.wav",
			"audio/piano/d.wav",
			"audio/piano/b.wav",
			"audio/piano/g.wav",
			"audio/piano/e.wav",
		},
	}
	g.audioLabels = map[string][]string{
		"drum": {
			"Kick",
			"Snare",
			"Stick",
			"HiHat",
			"OpenHat",
		},
		"piano": {
			"f",
			"d",
			"b",
			"g",
			"e",
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

	g.images["bandroom"] = loadImage("images/bandroom.png")
	g.images["menubox"] = loadImage("images/menubox.png")

	img := loadImage("images/drummer.png")
	width, height := img.Size()
	g.sprites["drummer"] = &sprite{
		image:     img,
		frame:     0,
		numFrames: 1,
		x:         218,
		y:         100,
		width:     width,
		height:    height,
	}
	img = loadImage("images/pianoman.png")
	width, height = img.Size()
	g.sprites["pianoman"] = &sprite{
		image:     img,
		frame:     0,
		numFrames: 1,
		x:         148,
		y:         97,
		width:     width,
		height:    height,
	}

	return g
}
