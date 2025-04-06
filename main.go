package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 810
	screenHeight = 456
	sampleRate   = 44100
	loopStart    = 6 * 60  // 6 seconds at 60fps
	loopEnd      = 21 * 60 // 22 seconds at 60fps
)

//go:embed assets/img/*.png
var pngAssets embed.FS

//go:embed assets/audio/*.wav
var wavAssets embed.FS

type Game struct {
	count        int
	textures     map[string]*ebiten.Image
	bubbleTypes  []BubbleType
	bubbles      []Bubble
	audioContext *audio.Context
	introPlayer  *audio.Player
	loopPlayer   *audio.Player
	introPlayed  bool
	debugMode    bool
}

type BubbleType struct {
	name   string
	width  float64
	height float64
	chance float64
}

type Bubble struct {
	typeID   int
	x        float64
	y        float64
	startX   float64
	startY   float64
	endY     float64
	alpha    float64
	scale    float64
	rotation float64
	start    int
	end      int
	length   int
}

type Element struct {
	name       string
	width      float64
	height     float64
	rotation   float64
	loop       bool
	scale      float64
	animateX   bool
	animateY   bool
	animSpeedX float64
	animSpeedY float64
	animRangeX float64
	animRangeY float64
}

func loadImage(path string) (io.Reader, error) {
	data, err := pngAssets.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

func loadWav(path string) (io.Reader, error) {
	data, err := wavAssets.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func NewGame() *Game {
	g := &Game{
		textures:    make(map[string]*ebiten.Image),
		debugMode:   true,
		introPlayed: false,
	}
	g.initAudio()
	g.loadTextures()
	g.setupBubbleTypes()
	g.generateBubbles()

	return g
}

func (g *Game) loadTextures() {
	texturePaths := []string{
		"banner_title.png", "white.png", "banner_wavea.png", "banner_waveb.png",
		"banner_wave1a.png", "banner_wave1b.png", "banner_shape2.png", "banner_fade.png",
		"abubble1.png", "abubble2.png", "abubble3.png", "abubble4.png", "abubble5.png",
		"abubble6.png", "bbubble1.png", "cbubble1.png", "cbubble2.png",
	}

	for _, path := range texturePaths {
		imgFile, err := loadImage("assets/img/" + path)
		if err != nil {
			log.Printf("Warning: Could not load texture %s: %v\n", path, err)
			img := ebiten.NewImage(64, 64)
			img.Fill(color.RGBA{255, 0, 255, 255})
			g.textures[path] = img
			continue
		}

		img, _, err := image.Decode(imgFile)
		if err != nil {
			log.Printf("Warning: Could not decode texture %s: %v\n", path, err)
			img := ebiten.NewImage(64, 64)
			img.Fill(color.RGBA{255, 0, 255, 255})
			g.textures[path] = img
			continue
		}
		log.Printf("Loaded texture: %s\n", path)

		g.textures[path] = ebiten.NewImageFromImage(img)
	}

	if _, ok := g.textures["white.png"]; !ok {
		whiteImg := ebiten.NewImage(1, 1)
		whiteImg.Fill(color.White)
		g.textures["white.png"] = whiteImg
	}
}

func (g *Game) initAudio() {
	g.audioContext = audio.NewContext(sampleRate)

	introData, err := loadWav("assets/audio/intro.wav")
	if err != nil {
		log.Printf("Warning: Could not load intro audio: %v\n", err)
	} else {
		introDec, err := wav.DecodeWithSampleRate(sampleRate, introData)
		if err != nil {
			log.Printf("Warning: Could not decode intro audio: %v\n", err)
		} else {
			g.introPlayer, err = g.audioContext.NewPlayer(introDec)
			if err != nil {
				log.Printf("Warning: Could not create intro player: %v\n", err)
			}
		}
	}

	loopData, err := loadWav("assets/audio/loop.wav")
	if err != nil {
		log.Printf("Warning: Could not load loop audio: %v\n", err)
	} else {
		loopDec, err := wav.DecodeWithSampleRate(sampleRate, loopData)
		if err != nil {
			log.Printf("Warning: Could not decode loop audio: %v\n", err)
		} else {
			loopLoop := audio.NewInfiniteLoop(loopDec, loopDec.Length())
			g.loopPlayer, err = g.audioContext.NewPlayer(loopLoop)
			if err != nil {
				log.Printf("Warning: Could not create loop player: %v\n", err)
			}
		}
	}

}

func (g *Game) setupBubbleTypes() {
	g.bubbleTypes = []BubbleType{
		{name: "abubble1.png", width: 48, height: 48, chance: 1},
		{name: "abubble2.png", width: 32, height: 32, chance: 1},
		{name: "abubble3.png", width: 16, height: 16, chance: 1},
		{name: "abubble4.png", width: 24, height: 24, chance: 1},
		{name: "abubble5.png", width: 32, height: 32, chance: 1},
		{name: "abubble6.png", width: 16, height: 16, chance: 1},
		{name: "bbubble1.png", width: 48, height: 48, chance: 1},
		{name: "cbubble1.png", width: 64, height: 64, chance: 1},
		{name: "cbubble2.png", width: 16, height: 16, chance: 1},
	}
}

func (g *Game) chooseBubbleType() int {
	var sumChances float64
	for _, bt := range g.bubbleTypes {
		sumChances += bt.chance
	}

	opt := rand.Float64() * sumChances
	for i, bt := range g.bubbleTypes {
		if bt.chance > opt {
			return i
		}
		opt -= bt.chance
	}

	return len(g.bubbleTypes) - 1
}

func (g *Game) generateBubbles() {
	g.bubbles = []Bubble{}

	bubbleBoom := 250

	for i := 0; i < 100; i++ {
		g.addBubble(bubbleBoom)
	}

	for i := 0; i < 280; i++ {
		start := int(rand.Float64()*float64(loopEnd-bubbleBoom)) + bubbleBoom
		g.addBubble(start)
	}

	filteredBubbles := []Bubble{}
	for _, b := range g.bubbles {
		if b.end <= loopEnd {
			filteredBubbles = append(filteredBubbles, b)
		}
	}
	g.bubbles = filteredBubbles

	extraBubbles := []Bubble{}
	for _, b := range g.bubbles {
		if b.start < loopStart && b.end > loopStart {
			new := b
			new.start = new.start - loopStart + loopEnd
			new.end = new.end - loopStart + loopEnd
			extraBubbles = append(extraBubbles, new)
		}
	}
	g.bubbles = append(g.bubbles, extraBubbles...)
}

func (g *Game) addBubble(start int) {
	x := rand.Float64()*(screenWidth+128) - 64 - screenWidth/2
	length := rand.Float64()*180 + 50

	yStart := float64(screenWidth)
	yEnd := 170.0

	bubble := Bubble{
		typeID:   g.chooseBubbleType(),
		x:        x,
		y:        yStart,
		startX:   x,
		startY:   yStart,
		endY:     yEnd,
		alpha:    0,
		scale:    1.0,
		rotation: rand.Float64() * math.Pi * 2,
		start:    start,
		end:      start + int(length),
		length:   int(length),
	}

	g.bubbles = append(g.bubbles, bubble)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)

	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Translate(0, 0)
	screen.DrawImage(g.textures["white.png"], bgOp)
	g.drawFade(screen)
	g.drawWaves(screen)
	g.drawBubbles(screen)
	g.drawTitle(screen)
	g.drawBoom(screen)

	if g.debugMode {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f, Frame: %d/%d", ebiten.ActualFPS(), g.count, loopEnd))
	}
}

func (g *Game) drawWaves(screen *ebiten.Image) {
	frame := g.count
	aniSpeedX := 1.0
	waveElements := []Element{
		{
			name:       "banner_wavea.png",
			width:      1024,
			height:     32,
			animateX:   true,
			animateY:   true,
			animRangeX: 200,
			animRangeY: 5,
			animSpeedX: aniSpeedX,
			animSpeedY: 6,
			loop:       true,
		},
		{
			name:       "banner_waveb.png",
			width:      1024,
			height:     32,
			animateX:   true,
			animateY:   true,
			animRangeX: 200,
			animRangeY: 5,
			animSpeedX: aniSpeedX * 2.0,
			animSpeedY: 8,
			loop:       true,
		},
		{
			name:       "banner_wave1a.png",
			width:      382,
			height:     32,
			animateX:   true,
			animateY:   true,
			animRangeX: 400,
			animRangeY: 20,
			animSpeedX: aniSpeedX * 2.0,
			animSpeedY: 6 * 0.2,
		},
		{
			name:       "banner_wave1b.png",
			width:      527,
			height:     37,
			animateX:   true,
			animateY:   true,
			animRangeX: 200,
			animRangeY: 13,
			animSpeedX: aniSpeedX * 2.2,
			animSpeedY: 6 * 0.2,
		},
		{
			name:       "banner_wave1b.png",
			width:      527,
			height:     37,
			animateX:   true,
			animateY:   true,
			animRangeX: 200,
			animRangeY: 20,
			animSpeedX: aniSpeedX * 2.7,
			animSpeedY: 6 * 0.2,
		},
		{
			name:       "banner_shape2.png",
			width:      644,
			height:     28,
			animateX:   true,
			animateY:   true,
			animRangeX: 280,
			animRangeY: 5,
			animSpeedX: aniSpeedX * 1.4,
			animSpeedY: 6 * 0.2,
		},
	}

	aniProgress := min(float64(g.count)/244.0, 1.0)
	aniProgress = math.Sin(aniProgress * math.Pi / 2)
	initialY := 140
	targetSize := float64((float64(initialY)-float64(screenHeight))*aniProgress + float64(screenHeight))
	startPositions := []struct{ x, y float64 }{
		{-100, targetSize + 10},
		{-100, targetSize + 15},
		{-200, targetSize + 40},
		{200, targetSize + 50},
		{-400, targetSize + 45},
		{-180, targetSize + 50},
	}

	for i, elem := range waveElements {
		if i >= len(startPositions) {
			continue
		}
		startX, startY := startPositions[i].x, startPositions[i].y
		x, y := startX, startY

		if elem.animateX {
			progress := math.Sin(float64(frame)/60.0*elem.animSpeedX)*0.5 + 0.5
			x = startX + progress*elem.animRangeX
		}

		if elem.animateY {
			progress := math.Sin(float64(frame)/60.0*elem.animSpeedY)*0.5 + 0.5
			y = startY + progress*elem.animRangeY
		}

		op := &ebiten.DrawImageOptions{}
		w, h := elem.width, elem.height
		op.GeoM.Translate(-w/2, -h/2)

		if elem.rotation != 0 {
			op.GeoM.Rotate(elem.rotation)
		}
		if elem.scale != 0 && elem.scale != 1 {
			op.GeoM.Scale(elem.scale, elem.scale)
		}
		op.GeoM.Translate(screenWidth/2+x, y)

		screen.DrawImage(g.textures[elem.name], op)
	}
}

func (g *Game) drawBubbles(screen *ebiten.Image) {
	frame := g.count

	for _, bubble := range g.bubbles {
		if frame < bubble.start || frame >= bubble.end {
			continue
		}

		progress := float64(frame-bubble.start) / float64(bubble.length)

		x := bubble.startX
		y := bubble.startY + (bubble.endY-bubble.startY)*progress

		var alpha float64
		fadePoint := 0.7

		if progress < 0.1 {
			alpha = progress * 10.0
		} else if progress > fadePoint {
			alpha = 1.0 - (progress-fadePoint)/(1.0-fadePoint)
		} else {
			alpha = 1.0
		}

		if alpha < 0 {
			alpha = 0
		} else if alpha > 1 {
			alpha = 1
		}

		bubbleType := g.bubbleTypes[bubble.typeID]
		texture := g.textures[bubbleType.name]

		op := &ebiten.DrawImageOptions{}
		w, h := bubbleType.width, bubbleType.height
		op.GeoM.Translate(-w/2, -h/2)
		rotation := bubble.rotation + progress*math.Pi*2*0.5

		op.GeoM.Rotate(rotation)
		op.GeoM.Translate(screenWidth/2+x, y)
		op.ColorScale.ScaleAlpha(float32(alpha))

		screen.DrawImage(texture, op)
	}
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	frame := g.count
	titleImg := g.textures["banner_title.png"]
	width := 400.0
	height := 180.0

	y := 32.0
	if frame >= 244 {
		oscY := math.Sin(float64(frame)/50*2) * 10.0
		y = 22.0 + oscY
	}

	alpha := 0.0
	if frame >= 244 {
		alpha = 1.0
	} else if frame >= 243 {
		alpha = float64(frame - 243)
	}

	op := &ebiten.DrawImageOptions{}

	op.GeoM.Translate(-width/2, height/2)
	op.GeoM.Translate(screenWidth/2, screenHeight/4+y)
	op.ColorScale.ScaleAlpha(float32(alpha))

	screen.DrawImage(titleImg, op)
}

func (g *Game) drawFade(screen *ebiten.Image) {
	fadeImg := g.textures["banner_fade.png"]
	width := float64(screenWidth)
	height := 256.0

	op1 := &ebiten.DrawImageOptions{}
	op1.GeoM.Scale(width/float64(fadeImg.Bounds().Dx()), height/float64(fadeImg.Bounds().Dy()))

	aniProgress := min(float64(g.count)/244.0, 1.0)
	aniProgress = math.Sin(aniProgress * math.Pi / 2)
	initialY := 200
	targetSize := float64((float64(initialY)-float64(screenHeight))*aniProgress + float64(screenHeight))
	op1.GeoM.Translate(0, targetSize)

	screen.DrawImage(fadeImg, op1)
}

func (g *Game) drawBoom(screen *ebiten.Image) {
	frame := g.count

	if !g.introPlayer.IsPlaying() && frame <= 256 {
		alpha := 0.0

		if frame <= 246 {
			alpha = 1.0
		} else {
			alpha = 1.0 - float64(frame-246)/10.0
		}

		op := &ebiten.DrawImageOptions{}
		op.ColorScale.ScaleAlpha(float32(alpha))

		whiteImg := g.textures["white.png"]

		op.GeoM.Scale(screenWidth, screenHeight)
		screen.DrawImage(whiteImg, op)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) Update() error {
	g.count++

	if g.count >= 1 && g.introPlayer != nil && !g.introPlayer.IsPlaying() {
		g.introPlayer.Play()
	}

	if g.count >= 237 && g.loopPlayer != nil && !g.loopPlayer.IsPlaying() {
		g.loopPlayer.Play()
	}

	if g.count >= loopEnd {
		g.count = loopStart
	}

	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.debugMode = !g.debugMode
	}

	return nil
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("lmao")
	ebiten.SetTPS(60)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
