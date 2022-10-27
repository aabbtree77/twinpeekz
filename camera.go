package main

import (
	"github.com/g3n/engine/math32"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

var Z_AXIS = mgl32.Vec3{0.0, 0.0, 1.0}

type Camera struct {
	view        mgl32.Mat4
	proj        mgl32.Mat4
	projView    mgl32.Mat4
	invProjView mgl32.Mat4

	pos mgl32.Vec3
	dir mgl32.Vec3
	up  mgl32.Vec3

	pitch float32
	yaw   float32

	fovDeg      float32
	aspectRatio float32
	zNear       float32
	zFar        float32

	shiftSpeed float32
	rotSpeed   float32
	mouseSpeed float32
	firstMouse bool
	lastX      float32
	lastY      float32
}

func makeCam() Camera {

	var cam Camera
	//Sponza takes approx 30x17x12 meters centered roughly at the origin
	//cam.UpdateOrientation(mgl32.Vec3{10.0, -3.0, 4}, mgl32.Vec3{-1.0, 0.0, 0.0}, Z_AXIS)
	//cam.UpdateOrientation(mgl32.Vec3{4.0, -3.0, 4}, mgl32.Vec3{-1.0, 0.0, 0.0}, Z_AXIS)
	cam.UpdateOrientation(mgl32.Vec3{10.0, -4.5, 4.0}, mgl32.Vec3{-1.0, 0.8, 0.0}, Z_AXIS)
	//cam.UpdateOrientation(mgl32.Vec3{9.0, -5.2, 4}, mgl32.Vec3{0.0, 1.0, 0.2}, Z_AXIS)
	//cam.UpdateOrientation(mgl32.Vec3{9.0, -2.2, 1}, mgl32.Vec3{0.0, 1.0, -0.2}, Z_AXIS)
	cam.aspectRatio = 1.0
	cam.UpdateProjection(45.0, 0.1, 100.0)
	cam.shiftSpeed = float32(1.5)
	cam.rotSpeed = float32(40.0)
	cam.mouseSpeed = float32(0.1)
	cam.firstMouse = true
	return cam
}

func (cam *Camera) UpdateOrientation(pos, dir, up mgl32.Vec3) {

	cam.pos = pos
	cam.dir = dir.Normalize()
	cam.up = up

	radius := math32.Sqrt(cam.dir[0]*cam.dir[0] + cam.dir[1]*cam.dir[1])
	cam.pitch = math32.RadToDeg(math32.Atan2(cam.dir[2], radius))
	cam.yaw = math32.RadToDeg(math32.Atan2(cam.dir[1], cam.dir[0]))

	lookAt := pos.Add(cam.dir)
	cam.view = mgl32.LookAtV(pos, lookAt, up)
	cam.projView = cam.proj.Mul4(cam.view)
	cam.invProjView = cam.projView.Inv()
}

func (cam *Camera) UpdateProjection(fov, zNear, zFar float32) {
	cam.fovDeg = fov
	cam.zNear = zNear
	cam.zFar = zFar
	cam.proj = mgl32.Perspective(mgl32.DegToRad(cam.fovDeg), cam.aspectRatio, zNear, zFar)
	cam.projView = cam.proj.Mul4(cam.view)
	cam.invProjView = cam.projView.Inv()
}

func (cam *Camera) UpdateAspectRatio(aspectRatio float32) {
	cam.aspectRatio = aspectRatio
	cam.proj = mgl32.Perspective(mgl32.DegToRad(cam.fovDeg), cam.aspectRatio, cam.zNear, cam.zFar)
	cam.projView = cam.proj.Mul4(cam.view)
	cam.invProjView = cam.projView.Inv()
}

func (cam *Camera) Translate(trans mgl32.Vec3) {
	cam.pos = cam.pos.Add(trans)
	lookAt := cam.pos.Add(cam.dir)
	cam.view = mgl32.LookAtV(cam.pos, lookAt, cam.up)
	cam.projView = cam.proj.Mul4(cam.view)
	cam.invProjView = cam.projView.Inv()
}

func (cam *Camera) Rotate(dpitch float32, dyaw float32) {
	cam.pitch = math32.Max(-89.5, math32.Min(cam.pitch+dpitch, 89.5))
	cam.yaw += dyaw

	cam.dir[0] = math32.Cos(math32.DegToRad(cam.pitch)) * math32.Cos(math32.DegToRad(cam.yaw))
	cam.dir[1] = math32.Cos(math32.DegToRad(cam.pitch)) * math32.Sin(math32.DegToRad(cam.yaw))
	cam.dir[2] = math32.Sin(math32.DegToRad(cam.pitch))
	cam.dir = cam.dir.Normalize()

	lookAt := cam.pos.Add(cam.dir)
	cam.view = mgl32.LookAtV(cam.pos, lookAt, cam.up)
	cam.projView = cam.proj.Mul4(cam.view)
	cam.invProjView = cam.projView.Inv()
}

func (cam *Camera) updateViaKeyboard(wn *glfw.Window, deltaT float64) {

	shiftIncr := float32(cam.shiftSpeed * float32(deltaT))
	rotIncr := float32(cam.rotSpeed * float32(deltaT))

	if wn.GetKey(glfw.KeyW) == glfw.Press {
		dir := cam.dir
		dir[2] = 0.0
		dir = dir.Normalize()
		cam.Translate(dir.Mul(shiftIncr))
	}
	if wn.GetKey(glfw.KeyS) == glfw.Press {
		dir := cam.dir.Mul(-1.0)
		dir[2] = 0.0
		dir = dir.Normalize()
		cam.Translate(dir.Mul(shiftIncr))
	}
	if wn.GetKey(glfw.KeyA) == glfw.Press {
		right := cam.dir.Cross(Z_AXIS)
		cam.Translate(right.Normalize().Mul(-shiftIncr))
	}
	if wn.GetKey(glfw.KeyD) == glfw.Press {
		right := cam.dir.Cross(Z_AXIS)
		cam.Translate(right.Normalize().Mul(shiftIncr))
	}
	if wn.GetKey(glfw.KeyE) == glfw.Press {
		cam.Translate(Z_AXIS.Mul(shiftIncr))
	}
	if wn.GetKey(glfw.KeyC) == glfw.Press {
		cam.Translate(Z_AXIS.Mul(-shiftIncr))
	}
	if wn.GetKey(glfw.KeyUp) == glfw.Press {
		cam.Rotate(rotIncr, 0.0)
	}
	if wn.GetKey(glfw.KeyLeft) == glfw.Press {
		cam.Rotate(0.0, rotIncr)
	}
	if wn.GetKey(glfw.KeyDown) == glfw.Press {
		cam.Rotate(-rotIncr, 0.0)
	}
	if wn.GetKey(glfw.KeyRight) == glfw.Press {
		cam.Rotate(0.0, -rotIncr)
	}
}

func (cam *Camera) mouseCamRotate(w *glfw.Window, xPos float64, yPos float64) {
	if cam.firstMouse {
		cam.lastX = float32(xPos)
		cam.lastY = float32(yPos)
		cam.firstMouse = false
	}

	xOffset := float32(xPos) - cam.lastX
	yOffset := cam.lastY - float32(yPos)

	cam.lastX = float32(xPos)
	cam.lastY = float32(yPos)

	xOffset *= cam.mouseSpeed
	yOffset *= cam.mouseSpeed
	cam.Rotate(yOffset, xOffset)
}

func (cam *Camera) mouseZoom(w *glfw.Window, xOffset float64, yOffset float64) {
	fovDeg := cam.fovDeg
	fovDeg += float32(yOffset)
	if fovDeg < 1.0 {
		cam.fovDeg = 1.0
	}
	if fovDeg > 45.0 {
		cam.fovDeg = 45.0
	}
	if fovDeg >= 1.0 && fovDeg <= 45.0 {
		cam.fovDeg = fovDeg
	}
	//fmt.Println(yOffset, cam.fovDeg)
	cam.UpdateProjection(cam.fovDeg, cam.zNear, cam.zFar)
}
