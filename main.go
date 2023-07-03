package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"os"
	"runtime"
	"sync"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	ebiten "github.com/hajimehoshi/ebiten/v2"
	im "gopkg.in/gographics/imagick.v3/imagick"
)

var (
	choices = [][]int{}
)

type Game struct {
	jiff     []*ebiten.Image
	duration []int
	tick     int
	curr     int
	lastUp   time.Time
	pos      []image.Rectangle
}

func (g *Game) Update() error {
	if 0 == len(gm.jiff) {
		return nil
	}
	if gm.tick == 0 {
		gm.lastUp = time.Now()
	}
	cx, cy := ebiten.CursorPosition()
	w, h := ebiten.WindowSize()
	if cx > 0 && cx < w && cy > 0 && cy < h {
		die := rand.Int31n(4)
		ebiten.SetWindowPosition(choices[die][0], choices[die][1])
	}
	if gm.lastUp.Before(time.Now()) {
		gm.curr++
		gm.curr %= len(gm.jiff)
		gm.lastUp = time.Now().Add(time.Duration(gm.duration[gm.curr]) * time.Second / 100)
	}
	gm.tick++
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if len(gm.jiff) == 0 {
		g = zero
		return
	}
	opt := ebiten.DrawImageOptions{}
	opt.GeoM.Translate(float64(gm.pos[gm.curr].Min.X), float64(gm.pos[gm.curr].Min.Y))
	screen.DrawImage(gm.jiff[gm.curr], &opt)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func gameFromGif(path string) (*Game, int, int, error) {
	var (
		data []byte
		err  error
	)
	data, err = ioutil.ReadFile(path)
	if err != nil {
		resp, err := http.Get(path)
		if err != nil {
			return nil, 0, 0, err
		}
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, 0, 0, err
		}
	}
	mw := im.NewMagickWand()
	defer mw.Destroy()
	err = mw.ReadImageBlob(data)
	if err != nil {
		return nil, 0, 0, err
	}
	mw = mw.CoalesceImages()
	defer mw.Destroy()
	data = mw.GetImagesBlob()
	jiff, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		panic(err)
		return nil, 0, 0, err
	}
	game := &Game{jiff: []*ebiten.Image{}}
	game.duration = jiff.Delay
	game.pos = []image.Rectangle{}
	for _, img := range jiff.Image {
		i := ebiten.NewImageFromImage(img)
		game.jiff = append(game.jiff, i)
		game.pos = append(game.pos, img.Bounds())
	}

	w := jiff.Config.Width
	h := jiff.Config.Height
	return game, w, h, nil
}

var doneChan = make(chan bool, 1)
var (
	gm *Game = &Game{
		jiff:     []*ebiten.Image{},
		duration: []int{},
		pos:      []image.Rectangle{},
	}
	zero   = gm
	vw, vh int
)

func RunGame(gifChan chan string, wg *sync.WaitGroup) {
	wg.Add(1)
	ebiten.SetScreenClearedEveryFrame(true)
	vw, vh = ebiten.ScreenSizeInFullscreen()
	go func() {
		doneChan <- true
		for range doneChan {
			file, ok := <-gifChan
			if !ok {
				doneChan <- true
				continue
			}

			game, w, h, err := gameFromGif(file)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if ebiten.IsWindowMinimized() {
				ebiten.RestoreWindow()
			}
			gm = game
			choices = [][]int{
				{0, 0},
				{vw - w, 0},
				{0, vh - h},
				{vw - w, vh - h},
			}
			ebiten.SetWindowSize(w, h)
			ebiten.SetWindowTitle(file)
			ebiten.SetWindowPosition(choices[0][0], choices[0][1])
		}
		wg.Done()
	}()
	ebiten.SetScreenTransparent(true)
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	if err := ebiten.RunGame(gm); err != nil {
		fmt.Println(err)
	}
	wg.Wait()
}

func main() {
	runtime.LockOSThread()
	gifChan := make(chan string, 100)
	wg := &sync.WaitGroup{}
	gifChan <- os.Args[1]
	im.Initialize()
	defer im.Terminate()
	go func() {
		rdr := bufio.NewReader(os.Stdin)
		for {
			line, err := rdr.ReadString('\n')
			if err != nil {
				close(gifChan)
				os.Exit(0)
			}
			gifChan <- line[:len(line)-1]
			doneChan <- true
		}
	}()
	RunGame(gifChan, wg)
}
