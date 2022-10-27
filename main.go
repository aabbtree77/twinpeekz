package main

/*
To test Golang's GC, comment out fmt.Println("timeOpenGLms=", timeOpenGLms) some place below,
and then run this in a terminal:
go build
GODEBUG=gctrace=1 ./twinpeekz
*/

import (
	"fmt"
	_ "image/png"
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type windowState struct {
	pos           [2]int
	size          [2]int
	fullScrState  bool
	fullScrChange bool
	cam           Camera
	rengine       RenderEngine
}

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func (winState *windowState) KeyHandler(ww *glfw.Window, key glfw.Key, scan int, action glfw.Action, mods glfw.ModifierKey) {

	if key == glfw.KeyEscape && action == glfw.Press {
		if false {
			fmt.Println("Escape pressed")
			fmt.Printf("proj: \n")
			fmt.Printf("%v\n", winState.cam.proj)
			fmt.Printf("view: \n")
			fmt.Printf("%v\n", winState.cam.view)
			fmt.Printf("projView: \n")
			fmt.Printf("%v\n", winState.cam.projView)
			fmt.Printf("invProjView: \n")
			fmt.Printf("%v\n", winState.cam.invProjView)
		}
		ww.SetShouldClose(true)
	}

	if key == glfw.KeyF11 && action == glfw.Press {
		fmt.Println("F11 pressed")
		// If this callback was not defined as windowState method, but just a function KeyHandler, then this could be the way 
		// to pass data into this callback:
		//var ptr unsafe.Pointer
		//ptr := w.GetUserPointer()
		//winState := (*(*windowState)(ptr))
		// Or just use a static/global variable, anything above this function should be visible anyway.
		winState.fullScrChange = true
	}
	//winState.cam = updateCameraFromKeyboard(winState.cam, winState.deltaT, key, action)
}

func (winState *windowState) FbResize(w *glfw.Window, width int, height int) {
	winState.cam.UpdateAspectRatio(float32(width) / float32(height))
	winState.rengine.width = int32(width)
	winState.rengine.height = int32(height)
	resizeRendering(winState.rengine)
	//gl.Viewport(0, 0, int32(width), int32(height))
}

/*
func WinResize(w *glfw.Window, width int, height int) {
	//winStateLocal := w.GetUserPointer()
	gl.Viewport(0, 0, int32(width), int32(height))
}
func WinRepos(w *glfw.Window, width int, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
}
*/

func main() {

	// Set up GLFW and OpenGL
	//--------------------------------------------------------------------------------------------------------------------------
	if err := glfw.Init(); err != nil {
		log.Fatalln("Failed to initialize glfw:", err)
	}
	defer glfw.Terminate()
	defer runtime.UnlockOSThread() // From g3n glfw.go: Important when using the execution tracer

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	//glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	var mon *glfw.Monitor
	mon = glfw.GetPrimaryMonitor()
	var vmode *glfw.VidMode
	vmode = mon.GetVideoMode()
	windowWidth, windowHeight := vmode.Width, vmode.Height
	//window, err := glfw.CreateWindow(windowWidth, windowHeight, "Sponza", glfw.GetPrimaryMonitor(), nil)
	window, err := glfw.CreateWindow(windowWidth, windowHeight, "Sponza", nil, nil)
	check(err)
	window.MakeContextCurrent()

	// Init Glow
	err = gl.Init()
	check(err)

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL versionZ", version)

	var flags int32
	gl.GetIntegerv(gl.CONTEXT_FLAGS, &flags)

	var winState windowState
	winState.size[0], winState.size[1] = window.GetSize()
	winState.pos[0], winState.pos[1] = window.GetPos()
	winState.fullScrState = false
	//window.SetUserPointer(unsafe.Pointer(&winState))

	//window.SetsizeCallback(WinResize)
	//window.SetposCallback(WinRepos)

	//Init scene, rendering
	//-------------------------------------------------------------------------------------------------------------------------
	scene := initScene()
	fmt.Println("Drawables =", len(scene.drawables))
	fmt.Println("Lights =", len(scene.lights))

	fbWidth, fbHeight := window.GetFramebufferSize()

	winState.cam = makeCam()
	winState.cam.UpdateAspectRatio(float32(fbWidth) / float32(fbHeight))

	winState.rengine = scene.initRendering(int32(fbWidth), int32(fbHeight))
	resizeRendering(winState.rengine)

	//Lock and hide mouse cursor
	//window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	window.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
	//window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)

	//Set up callbacks
	window.SetKeyCallback(winState.KeyHandler)
	window.SetFramebufferSizeCallback(winState.FbResize)
	window.SetCursorPosCallback(glfw.CursorPosCallback(winState.cam.mouseCamRotate))
	window.SetScrollCallback(glfw.ScrollCallback(winState.cam.mouseZoom))
	if false {
		fmt.Println("Cam matrices before Rendering Loop:")
		fmt.Printf("proj: \n")
		fmt.Printf("%v\n", winState.cam.proj)
		fmt.Printf("view: \n")
		fmt.Printf("%v\n", winState.cam.view)
		fmt.Printf("projView: \n")
		fmt.Printf("%v\n", winState.cam.projView)
		fmt.Printf("invProjView: \n")
		fmt.Printf("%v\n", winState.cam.invProjView)
	}
	//Rendering loop
	//-------------------------------------------------------------------------------------------------------------------------
	/*Enable VSync (0-off, 1-on in glfw.SwapInterval)
	You might also need adjust your card settings, i.e. executing this helped for my GTX 760 setup:
	nvidia-settings --assign CurrentMetaMode="nvidia-auto-select +0+0 { ForceFullCompositionPipeline = On }"
	See https://github.com/godlikepanos/anki-3d-engine/issues/59
	*/
	glfw.SwapInterval(0)

	var timeOpenGLms [5]float64

	timePassedSec := glfw.GetTime()
	timeStartSec := timePassedSec
	frames := int(0)
	fps := float64(0.0)

	framePrevSec := glfw.GetTime()
	deltaT := float64(0.0)

	for !window.ShouldClose() {

		//Measure FPS, average time per frame
		timePassedSec = glfw.GetTime()
		if (timePassedSec-timeStartSec > 1.0) && (frames > 10) {
			fps = float64(frames) / (timePassedSec - timeStartSec)
			timeStartSec = timePassedSec
			frames = 0
			//fmt.Printf("deltaT = %.2fms, FPS = %.0f.\n", deltaT*1000.0, fps)
			//fmt.Printf("timeOpenGLms=%.2f\n", timeOpenGLms)
			_ = fps
			_ = timeOpenGLms[0]
		}
		frames++

		//Measure  deltaT
		frameCurrSec := glfw.GetTime()
		deltaT = frameCurrSec - framePrevSec
		framePrevSec = frameCurrSec
		winState.cam.updateViaKeyboard(window, deltaT)
		//fmt.Printf("deltaT = %.2fms\n", deltaT*1000.0)

		//gl.ClearColor(0.2, 0.3, 0.3, 1.0)
		//gl.Clear(gl.COLOR_BUFFER_BIT)

		if winState.fullScrChange {
			if !winState.fullScrState {
				//Save current window state
				winState.size[0], winState.size[1] = window.GetSize()
				winState.pos[0], winState.pos[1] = window.GetPos()
				mon = glfw.GetPrimaryMonitor()
				vmode = mon.GetVideoMode()
				window.SetMonitor(mon, 0, 0, vmode.Width, vmode.Height, vmode.RefreshRate)
				//TD: update camera, viewport gets updated automatically via resize callback?!
			} else {
				mon = glfw.GetPrimaryMonitor()
				vmode = mon.GetVideoMode()
				window.SetMonitor(nil, winState.pos[0], winState.pos[1], winState.size[0], winState.size[1], vmode.RefreshRate)
				//TD: update camera, viewport gets updated automatically via resize callback?!
			}
			winState.fullScrState = !winState.fullScrState
			winState.fullScrChange = false
		}

		timeOpenGLms = scene.mainRendering(winState.rengine, winState.cam)
		//fmt.Println("timeOpenGLms=", timeOpenGLms)

		window.SwapBuffers()
		glfw.PollEvents()
	}
}
