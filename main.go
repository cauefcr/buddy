package main

import (
	"bytes"
	"image"
	"image/gif"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	ebiten "github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	jiff     []*ebiten.Image
	duration []int
	tick     int
	curr     int
	start    chan bool
	pos      []image.Rectangle
}

func (g *Game) Update() error {
	cx, cy := ebiten.CursorPosition()
	w, h := ebiten.WindowSize()
	if cx > 0 && cx < w && cy > 0 && cy < h {
		die := rand.Int31n(4)
		ebiten.SetWindowPosition(choices[die][0], choices[die][1])
	}
	g.tick++

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	opt := ebiten.DrawImageOptions{}
	opt.GeoM.Translate(float64(g.pos[g.curr].Min.X), float64(g.pos[g.curr].Min.Y))
	select {
	case g.start <- true:
	default:
	}
	screen.DrawImage(g.jiff[g.curr], &opt)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

var choices [][]int

func gameFromGif(path string) (*Game, int, int) {
	var (
		data []byte
		err  error
	)
	data, err = ioutil.ReadFile(os.Args[1])
	if err != nil {
		resp, err := http.Get(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)

		}
	}
	jiff, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	game := &Game{jiff: []*ebiten.Image{}}
	game.duration = jiff.Delay
	game.pos = []image.Rectangle{}
	game.start = make(chan bool, 1)
	for _, img := range jiff.Image {
		i := ebiten.NewImageFromImage(img)
		game.jiff = append(game.jiff, i)
		game.pos = append(game.pos, img.Bounds())
	}

	w := jiff.Config.Width
	h := jiff.Config.Height
	return game, w, h
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./jiffer <gif>")
	}
	game, w, h := gameFromGif(os.Args[1])
	vw, vh := ebiten.ScreenSizeInFullscreen()
	choices = [][]int{
		{0, 0},
		{vw - w, 0},
		{0, vh - h},
		{vw - w, vh - h},
	}
	ebiten.SetWindowPosition(choices[0][0], choices[0][1])
	ebiten.SetWindowSize(w, h)
	ebiten.SetScreenTransparent(true)
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	ebiten.SetWindowTitle(os.Args[1])
	go func() {
		for ready := range game.start {
			if ready {
				break
			}
		}
		for range time.Tick(time.Second / 100 * time.Duration(game.duration[game.curr])) {
			game.curr++
			game.curr %= len(game.jiff)
		}
	}()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
