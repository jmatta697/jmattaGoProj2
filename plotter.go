/* See also examples on https://github.com/golang-ui/nuklear */
package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"runtime"
	"strconv"
	"time"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/sindbach/nuklear/nk"
	"github.com/xlab/closer"
)

const (
	winWidth         = 515
	winHeight        = 544
	maxVertexBuffer  = 100 * 1024 * 1024
	maxElementBuffer = 100 * 1024 * 1024
)

func init() {
	runtime.LockOSThread()
}

var incomingDataChannel = make(chan string)

func main() {

	go plotterClient()
	// fmt.Println(PlotData)
	////////////////////////////////////////////
	if err := glfw.Init(); err != nil {
		closer.Fatalln(err)
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	win, err := glfw.CreateWindow(winWidth, winHeight, "Active Clients", nil, nil)
	if err != nil {
		closer.Fatalln(err)
	}
	win.MakeContextCurrent()
	win.SetSizeLimits(winWidth, winHeight, winWidth, winHeight)
	width, height := win.GetSize()

	if err := gl.Init(); err != nil {
		closer.Fatalln("opengl: initialisation failed:", err)
	}
	gl.Viewport(0, 0, int32(width), int32(height))
	ctx := nk.NkPlatformInit(win, nk.PlatformInstallCallbacks)

	// Fonts
	atlas := nk.NewFontAtlas()
	nk.NkFontStashBegin(&atlas)
	sansFont := nk.NkFontAtlasAddDefault(atlas, 16, nil)
	nk.NkFontStashEnd()
	if sansFont != nil {
		nk.NkStyleSetFont(ctx, sansFont.Handle())
	}
	exitC := make(chan struct{}, 1)
	doneC := make(chan struct{}, 1)
	closer.Bind(func() {
		close(exitC)
		<-doneC
	})
	fpsTicker := time.NewTicker(time.Second)
	for {
		select {
		case <-exitC:
			nk.NkPlatformShutdown()
			glfw.Terminate()
			fpsTicker.Stop()
			close(doneC)
			return
		case <-fpsTicker.C:
			if win.ShouldClose() {
				close(exitC)
				continue
			}
			glfw.PollEvents()
			gfxMain(win, ctx)

		}
	}
}

func plotWidget(ctx *nk.Context, canvas *nk.CommandBuffer) {
	state := nk.NkWidget(nk.NewRect(), ctx)
	switch state {
	case nk.WidgetInvalid:
		return
	case nk.WidgetRom:
		// temporary state
	case nk.WidgetValid:
		// update by user input
	}
	// test plotting with mouse click feature
	// draw stuff....
	yModifier := <-incomingDataChannel
	// turn string plot data into f32
	yModifierf32, _ := strconv.ParseFloat(yModifier, 32)
	yModifierScaled := yModifierf32 * 15
	c1 := nk.NkRect(float32(100), float32(yModifierScaled), 10.0, 10.0)
	nk.NkFillCircle(canvas, c1, nk.NkRgb(171, 239, 29))

	// fmt.Println(PlotData)

	/*for fy := 50; fy <= 400; fy += 10 {
		for fx := 50; fx <= 400; fx += 10 {
			c1 := nk.NkRect(float32(fx), float32(fy), 5.0, 5.0)
			nk.NkFillCircle(canvas, c1, nk.NkRgb(171, 239, 29))
		}
	}*/
}

func gfxMain(win *glfw.Window, ctx *nk.Context) {

	nk.NkPlatformNewFrame()
	plotAreaWidth := float32(winWidth)
	plotAreaHeight := float32(winHeight)

	// Layout
	bounds := nk.NkRect(0, 0, plotAreaWidth, plotAreaHeight)
	if nk.NkBegin(ctx, " ", bounds, nk.WindowTitle|nk.WindowNoScrollbar) > 0 {
		canvas := nk.NkWindowGetCanvas(ctx)
		totalSpace := nk.NkWindowGetContentRegion(ctx)
		nk.NkLayoutSpaceBegin(ctx, nk.LayoutStatic, totalSpace.H(), 10000)
		//Grid
		gridColor := nk.NkRgb(80, 80, 80)
		gridSize := float32(128.0)
		winSize := nk.NkLayoutSpaceBounds(ctx)
		for x := float32(0.0); x < winSize.W(); x += gridSize {
			nk.NkStrokeLine(canvas, x+winSize.X(), winSize.Y(), x+winSize.X(), winSize.Y()+winSize.H(), 1.0, gridColor)
		}
		for y := float32(0.0); y < winSize.H(); y += gridSize {
			nk.NkStrokeLine(canvas, winSize.X(), y+winSize.Y(), winSize.X()+winSize.W(), y+winSize.Y(), 1.0, gridColor)
		}

		nk.NkLayoutSpaceEnd(ctx)
		plotWidget(ctx, canvas)
		//Axis

	}
	nk.NkEnd(ctx)

	// Render
	bg := make([]float32, 4)
	nk.NkColorFv(bg, nk.NkRgba(50, 50, 50, 255))
	width, height := win.GetSize()
	gl.Viewport(0, 0, int32(width), int32(height))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.ClearColor(bg[0], bg[1], bg[2], bg[3])
	nk.NkPlatformRender(nk.AntiAliasingOn, maxVertexBuffer, maxElementBuffer)
	win.SwapBuffers()
}

// plotter's own client
func plotterClient() {
	conn, err := net.Dial("tcp", "localhost:8001")
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})
	buf := new(bytes.Buffer)

	go func() {
		io.Copy(buf, conn) // NOTE: ignoring errors
		incomingDataChannel <- buf.String()
		log.Println("done with complicated stuff")
		done <- struct{}{} // signal the main goroutine
	}()
	conn.Close()
	<-done // wait for background goroutine to finish
}
