package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	TEX_UNIT_BASECOLOR         = 0
	TEX_UNIT_METALLICROUGHNESS = 1
	TEX_UNIT_DEPTH             = 2
	TEX_UNIT_VOLUMETRIC        = 3
	TEX_UNIT_SHADOW_MAP_BASE   = 4
	MAX_LIGHTS                 = 8 //sync this value in shaders
)

type RenderEngine struct {
	width, height int32
	hdrProg       Shader
	volProg       Shader
	postprProg    Shader
	dirLightProg  Shader
	screenQuad    MeshGPUBufferIDs
	query         [4]uint32
	volFB         FrameBuffer
	hdrFB         FrameBuffer
}

type FrameBuffer struct {
	fboID         uint32
	colorBufferID uint32
	zBufferID     uint32
}

func (scn *Scene) initRendering(width int32, height int32) RenderEngine {

	hdrProg := MakeShaders("./shaders/hdr_vert.glsl", "./shaders/hdr_frag.glsl")
	volProg := MakeShaders("./shaders/vol_vert.glsl", "./shaders/vol_frag.glsl")
	postprProg := MakeShaders("./shaders/postpr_vert.glsl", "./shaders/postpr_frag.glsl")
	dirLightProg := MakeShaders("./shaders/lightDir_vert.glsl", "./shaders/lightDir_frag.glsl")

	//BTW, arrays on GPU turn out to be a nasty area. Simply declaring arrays of MAX_LIGHTS
	//and then sending only the needed type of light num < MAX_LIGHTS breaks things as the
	//unused arrays or array elements interfere in some hard to predict/debug ways even
	//when the shader needs for instance just the first 0-th element of a single cube texture
	//array.
	//The solution is to fill all the arrays completely with pregenerated texture units and
	//then save them into the light structs. The lights can then later be also removed during
	//the rendering in a safer way.

	gl.UseProgram(hdrProg.ID)
	hdrProg.SetInt("albedoMap", TEX_UNIT_BASECOLOR)
	hdrProg.SetInt("metalRoughMap", TEX_UNIT_METALLICROUGHNESS)
	//for it, lht := range scn.lights {
	//It's not completely clear if sending the same texture unit number into cube and 2d
	//texture samples is a good idea, but it works.
	for it := 0; it < MAX_LIGHTS; it++ {
		textureUnit := it + TEX_UNIT_SHADOW_MAP_BASE
		hdrProg.SetInt("dirlights["+strconv.Itoa(it)+"].txrUnit", int32(textureUnit))
		if it < len(scn.lights) {
			scn.lights[it].txrUnit = uint32(textureUnit) //save for the use in the render loop
		}
	}

	gl.UseProgram(volProg.ID)
	volProg.SetInt("shadowMap", TEX_UNIT_DEPTH)

	for it := 0; it < MAX_LIGHTS; it++ {
		textureUnit := it + TEX_UNIT_SHADOW_MAP_BASE
		volProg.SetInt("dirlights["+strconv.Itoa(it)+"].txrUnit", int32(textureUnit))
	}

	gl.UseProgram(postprProg.ID)
	postprProg.SetInt("hdrTexture", TEX_UNIT_BASECOLOR)
	postprProg.SetInt("volTexture", TEX_UNIT_VOLUMETRIC)

	quadVertices := [][3]float32{{-1.0, 1.0, 0.0}, {-1.0, -1.0, 0.0}, {1.0, 1.0, 0.0}, {1.0, -1.0, 0.0}}
	quadUvs := [][2]float32{{0.0, 1.0}, {0.0, 0.0}, {1.0, 1.0}, {1.0, 0.0}}
	quadIndices := []uint32{0, 1, 2, 1, 3, 2}
	screenQuad := uploadMeshToGPU(MeshGeometry{
		vertArray: quadVertices,
		uvArray:   quadUvs,
		normArray: [][3]float32{},
		indArray:  quadIndices})

	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
	//gl.DepthFunc(gl.LESS)

	var query [4]uint32
	gl.GenQueries(4, &query[0])

	volFB, err := makeFrameBuffer(width, height)
	check(err)
	hdrFB, err := makeFrameBuffer(width, height)
	check(err)

	return RenderEngine{
		width:        width,
		height:       height,
		hdrProg:      hdrProg,
		volProg:      volProg,
		postprProg:   postprProg,
		dirLightProg: dirLightProg,
		screenQuad:   screenQuad,
		query:        query,
		volFB:        volFB,
		hdrFB:        hdrFB}
}

func makeFrameBuffer(width int32, height int32) (FrameBuffer, error) {

	var fboID uint32
	gl.GenFramebuffers(1, &fboID)

	var colorBufferID uint32
	gl.GenTextures(1, &colorBufferID)
	gl.BindTexture(gl.TEXTURE_2D, colorBufferID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, width, height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	var zBufferID uint32
	gl.GenTextures(1, &zBufferID)
	gl.BindTexture(gl.TEXTURE_2D, zBufferID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT, width, height, 0, gl.DEPTH_COMPONENT, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	// Attach buffers to FBO
	gl.BindFramebuffer(gl.FRAMEBUFFER, fboID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, colorBufferID, 0)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, zBufferID, 0)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		return FrameBuffer{}, fmt.Errorf("incomplete frame buffer: %v", gl.CheckFramebufferStatus(gl.FRAMEBUFFER))
	}

	// Bind hdr framebuffer
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	return FrameBuffer{fboID: fboID, colorBufferID: colorBufferID, zBufferID: zBufferID}, nil
}

func resizeFrameBuffer(fb FrameBuffer, width int32, height int32) {
	gl.BindTexture(gl.TEXTURE_2D, fb.colorBufferID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F, width, height, 0, gl.RGBA, gl.FLOAT, nil)
	gl.BindTexture(gl.TEXTURE_2D, fb.zBufferID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT, width, height, 0, gl.DEPTH_COMPONENT, gl.UNSIGNED_BYTE, nil)
}

func resizeRendering(reng RenderEngine) {
	resizeFrameBuffer(reng.volFB, reng.width, reng.height)
	resizeFrameBuffer(reng.hdrFB, reng.width, reng.height)
	gl.Viewport(0, 0, reng.width, reng.height)
}

func (scn *Scene) mainRendering(rengine RenderEngine, cam Camera) [5]float64 {

	var timeOpenGLms [5]float64

	// Update shadow maps, they will be drawn for each light into its corresponding lht.FBOID
	//framebuffer with depth texture attachment created with initShadowMap() in scene.go.
	//TD: Think about framebuffer objects spread in: (i) scn.lights Light struct and (ii) rengine RenderEngine struct.
	//This is alright, but shows how low level graphics API concepts spread into different things.
	//Scene and RenderEngine has no clear borders as to what belongs where, e.g. this is not a rigorous math question:
	//Do shadow map framebuffer ids belong to the light struct in the scene, or RenderEngine's shadow map stage, which
	//is not clearly delineated in RenderEngine struct BTW.
	//----------------------------------------------------------------------------------------------------------------
	gl.BeginQuery(gl.TIME_ELAPSED, rengine.query[0])
	gl.Enable(gl.DEPTH_TEST)

	for it, lht := range scn.lights {

		gl.BindFramebuffer(gl.FRAMEBUFFER, lht.FBOID)
		gl.Viewport(0, 0, lht.shMapWidth, lht.shMapHeight)
		gl.Clear(gl.DEPTH_BUFFER_BIT)

		var proj mgl32.Mat4
		zNear := float32(0.0) //should these be different per light, adaptive?
		zFar := float32(100.0)
		proj = mgl32.Ortho(-20.0, 20.0, -15.0, 15.0, zNear, zFar)
		//If dir and up are aligned LookAtV will output NaN matrices!!!
		//Here pos dpes not matter, test that
		pos := mgl32.Vec3{-4.0, 2.0, 5.0}
		up := mgl32.Vec3{1.0, 0.0, 1.0}
		projView := proj.Mul4(mgl32.LookAtV(pos, pos.Add(lht.dir), up))

		gl.UseProgram(rengine.dirLightProg.ID)
		rengine.dirLightProg.SetMat4("projView", projView)

		for _, instance := range scn.drawables {
			drawMeshVertOnly(instance, rengine.dirLightProg)
		}
		scn.lights[it].projView = projView
	}

	//(0, 0) arguments would change for "multi-viewport" settings, "the lower left corner of the viewport rectangle, in pixels".
	//Framebuffer texture resolution is different for shadow map stages and hdr/vol passes.
	//hdr/vol pass resolution reacts to glfw window resizes, but shadowmap resolutions are fixed when setting up lights in scene.go.
	gl.Viewport(0, 0, rengine.width, rengine.height)
	gl.EndQuery(gl.TIME_ELAPSED)

	// Draw scene to the HDR frame buffer
	//---------------------------------------------------------------------------------------------
	gl.BeginQuery(gl.TIME_ELAPSED, rengine.query[1])
	gl.BindFramebuffer(gl.FRAMEBUFFER, rengine.hdrFB.fboID)
	//gl.BindFramebuffer(gl.FRAMEBUFFER, 0) //Screen

	bgColor := mgl32.Vec3{0.0, 0.0, 1.0}
	gl.ClearColor(bgColor[0], bgColor[1], bgColor[2], 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	gl.UseProgram(rengine.hdrProg.ID)
	rengine.hdrProg.SetVec3("camPos", cam.pos)
	rengine.hdrProg.SetInt("numDirLights", int32(len(scn.lights)))

	//Activate and bind depth textures, quite a ceremony, but they get into a .glsl file via texture ints.
	//These ints are set in initRendering and are esentially global texture ids/variables.
	//In a GLSL program they become sampler2D/samplerCube, find a concrete variable in .glsl first, and
	//then search for its texture associations in higher level programs.
	//TD: need to better manage those ints to remove max lights bounds and such, but this exists due to
	//a static nature of GPU shaders, probably a hassle to use an array with a variable number of structs.

	for it, lht := range scn.lights {
		gl.ActiveTexture(gl.TEXTURE0 + lht.txrUnit)
		gl.BindTexture(gl.TEXTURE_2D, lht.txrID)

		str := "dirlights[" + strconv.Itoa(it) + "]"
		lht.setLightParams(rengine.hdrProg, str)
	}
	//glCheckError()
	//return
	for _, instance := range scn.drawables {
		rengine.hdrProg.SetMat4("projViewModel", cam.projView.Mul4(*(instance.model)))
		rengine.hdrProg.SetMat4("model", *(instance.model))
		//TD if there is no texture file upload material.diffuseColor to the shaders instead
		//the field and value is set there in the scene just in case, but not used for now.
		//Meshes without all the proper textures are simply not loaded at the moment.

		gl.ActiveTexture(gl.TEXTURE0 + TEX_UNIT_BASECOLOR)
		gl.BindTexture(gl.TEXTURE_2D, instance.baseColorTextureID)

		gl.ActiveTexture(gl.TEXTURE0 + TEX_UNIT_METALLICROUGHNESS)
		gl.BindTexture(gl.TEXTURE_2D, instance.metallicRoughnessTextureID)

		drawMesh(*(instance.mesh), rengine.hdrProg.ID)
	}

	gl.EndQuery(gl.TIME_ELAPSED)

	// Draw vol pass
	//---------------------------------------------------------------------------------------------
	gl.BeginQuery(gl.TIME_ELAPSED, rengine.query[2])
	gl.Disable(gl.DEPTH_TEST)
	gl.BindFramebuffer(gl.FRAMEBUFFER, rengine.volFB.fboID)

	gl.UseProgram(rengine.volProg.ID)
	//gl.ActiveTexture(gl.TEXTURE0 + TEX_UNIT_BASECOLOR)
	//gl.BindTexture(gl.TEXTURE_2D, rengine.hdrFB.colorBufferID)

	gl.ActiveTexture(gl.TEXTURE0 + TEX_UNIT_DEPTH)
	gl.BindTexture(gl.TEXTURE_2D, rengine.hdrFB.zBufferID)

	rengine.volProg.SetInt("numDirLights", int32(len(scn.lights)))

	//Activate and bind depth textures
	for it, lht := range scn.lights {
		gl.ActiveTexture(gl.TEXTURE0 + lht.txrUnit)
		gl.BindTexture(gl.TEXTURE_2D, lht.txrID)

		str := "dirlights[" + strconv.Itoa(it) + "]"
		lht.setLightParams(rengine.volProg, str)
	}

	rengine.volProg.SetMat4("invProjView", cam.invProjView)
	rengine.volProg.SetVec3("camPos", cam.pos)
	rengine.volProg.SetFloat("screenWidth", float32(rengine.width))
	rengine.volProg.SetFloat("screenHeight", float32(rengine.height))
	rengine.volProg.SetInt("volumetricAlgo", 1) //0 for visibility accumulation experiments
	rengine.volProg.SetFloat("scatteringZFar", float32(100.0))
	rengine.volProg.SetInt("scatteringSamples", 64)

	drawMesh(rengine.screenQuad, rengine.volProg.ID)
	gl.EndQuery(gl.TIME_ELAPSED)

	// Draw resulting frame buffer to screen with gamma correction and tone mapping
	//----------------------------------------------------------------------------------------------
	gl.BeginQuery(gl.TIME_ELAPSED, rengine.query[3])
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0) //Screen
	gl.UseProgram(rengine.postprProg.ID)

	//fmt.Printf("ns=%v, interleave=%v, shader=%v\n", 24, 3, "science")
	gl.ActiveTexture(gl.TEXTURE0 + TEX_UNIT_BASECOLOR)
	gl.BindTexture(gl.TEXTURE_2D, rengine.hdrFB.colorBufferID)

	gl.ActiveTexture(gl.TEXTURE0 + TEX_UNIT_VOLUMETRIC)
	gl.BindTexture(gl.TEXTURE_2D, rengine.volFB.colorBufferID)

	rengine.postprProg.SetInt("hdrVolMixType", 0)
	rengine.postprProg.SetFloat("clampPower", 0.8) //adjustable only when hdrVolMixType !=0
	rengine.postprProg.SetFloat("gamma", 2.2)
	rengine.postprProg.SetFloat("exposure", 5.0)

	drawMesh(rengine.screenQuad, rengine.postprProg.ID)
	gl.EndQuery(gl.TIME_ELAPSED)

	var elapsedTime uint64
	totalms := float64(0.0)
	for i := 0; i < 4; i++ {
		gl.GetQueryObjectui64v(rengine.query[i], gl.QUERY_RESULT, &elapsedTime)
		timeOpenGLms[i] = float64(elapsedTime) / 1000000.0
		totalms += timeOpenGLms[i]
	}
	timeOpenGLms[4] = totalms
	return timeOpenGLms
}

func drawMesh(msh MeshGPUBufferIDs, shdrProgID uint32) {
	gl.BindVertexArray(msh.vaoID)

	loc := uint32(gl.GetAttribLocation(shdrProgID, gl.Str("inPosition\x00")))
	if loc >= 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, msh.vertexBufferID)
		gl.VertexAttribPointer(loc, 3, gl.FLOAT, false, 0, gl.PtrOffset(0))
		gl.EnableVertexAttribArray(loc)
	} else {
		log.Fatalln("drawInstance could not find uniform: inPosition")
	}

	if msh.normalBufferID > 0 && (msh.normalBufferID != gl.INVALID_VALUE) {
		loc = uint32(gl.GetAttribLocation(shdrProgID, gl.Str("inNormal\x00")))
		if loc >= 0 {
			gl.BindBuffer(gl.ARRAY_BUFFER, msh.normalBufferID)
			gl.VertexAttribPointer(loc, 3, gl.FLOAT, false, 0, gl.PtrOffset(0))
			gl.EnableVertexAttribArray(loc)
		} else {
			log.Fatalln("drawInstance could not find uniform: inNormal")
		}
	}
	if msh.uvBufferID > 0 && (msh.uvBufferID != gl.INVALID_VALUE) {
		loc = uint32(gl.GetAttribLocation(shdrProgID, gl.Str("inTexCoord\x00")))
		if loc >= 0 {
			gl.BindBuffer(gl.ARRAY_BUFFER, msh.uvBufferID)
			gl.VertexAttribPointer(loc, 2, gl.FLOAT, false, 0, gl.PtrOffset(0))
			gl.EnableVertexAttribArray(loc)
		} else {
			log.Fatalln("drawInstance could not find uniform: inTexCoord")
		}
	}
	gl.DrawElements(gl.TRIANGLES, msh.lenIndices, gl.UNSIGNED_INT, gl.PtrOffset(0))
}

func drawMeshVertOnly(instance DrawableInstance, shdrProg Shader) {

	shdrProg.SetMat4("model", *(instance.model))
	gl.BindVertexArray((*(instance.mesh)).vaoID)

	gl.BindBuffer(gl.ARRAY_BUFFER, (*(instance.mesh)).vertexBufferID)
	//Vert attr loc 0:
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	gl.DrawElements(gl.TRIANGLES, (*(instance.mesh)).lenIndices, gl.UNSIGNED_INT, gl.PtrOffset(0))
}

func (lht Light) setLightParams(shd Shader, str string) {

	shd.SetVec3(str+".dir", lht.dir)
	shd.SetVec3(str+".color", lht.color)
	shd.SetFloat(str+".intensity", lht.intensity)
	shd.SetMat4(str+".dirWorldToProj", lht.projView)
}
