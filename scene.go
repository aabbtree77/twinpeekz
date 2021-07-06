package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/nicholasblaskey/stbi"

	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/modeler"
)

type Scene struct {
	drawables []DrawableInstance
	lights    []Light
}

//These will be on the GPU, but first preload to CPU
type MeshGeometry struct {
	vertArray [][3]float32
	uvArray   [][2]float32
	normArray [][3]float32
	indArray  []uint32
}

type MeshGPUBufferIDs struct {
	vaoID          uint32
	vertexBufferID uint32
	normalBufferID uint32
	uvBufferID     uint32
	indexBufferID  uint32
	lenIndices     int32
}

type DrawableInstance struct {
	mesh                       *MeshGPUBufferIDs
	model                      *mgl32.Mat4
	baseColorTextureID         uint32
	metallicRoughnessTextureID uint32
}

type Light struct {
	dir         mgl32.Vec3
	color       mgl32.Vec3
	intensity   float32
	projView    mgl32.Mat4
	shMapWidth  int32
	shMapHeight int32
	txrID       uint32
	txrUnit     uint32
	FBOID       uint32
	rerender    bool
}

func initScene() Scene {

	//folderPath := "/home/tokyo/blenderProj/cube77_03/"
	//gltfFileToLoad := folderPath + "mycube077_embedded.gltf"

	//folderPath := "/home/tokyo/sponzaGLTF/Sponza/glTF/"
	//gltfFileToLoad := folderPath + "Sponza.gltf"

	//folderPath := "/home/tokyo/boomboxGLTF/BoomBox/glTF/"
	//gltfFileToLoad := folderPath + "BoomBox.gltf"

	//folderPath := "/home/tokyo/sponzaGLTFResc/"
	folderPath := "/home/tokyo/sponzaGLTFRescNoLights/"
	gltfFileToLoad := folderPath + "sponza.gltf"

	doc, err := gltf.Open(gltfFileToLoad)
	if err != nil {
		log.Fatal(err)
	}
	/*
		fmt.Println("Asset info:")
		fmt.Print(doc.Asset)
		fmt.Println("\nDone.")
	*/

	//Exercise: Get the bbox over all the assets to know the global scale

	var loadedMeshPtrs []*MeshGPUBufferIDs
	var loadedTextureBaseColorIDs []uint32
	var loadedTextureMetallicRoughnessIDs []uint32

	for it, primitive := range doc.Meshes[0].Primitives {
		fmt.Printf("Primitive No.%d: ", it)

		//One needs to pass three gates to enter Valhalla:

		meshGeom, err := loadMeshFromGLTF(doc, primitive)
		if err != nil {
			continue
		}
		meshGPU := uploadMeshToGPU(meshGeom)
		//fmt.Printf("meshGPU.lenIndices[%d] = %d.\n", it, meshGPU.lenIndices)

		textureBaseColorID, err := loadTextureGLTFBaseColor(doc, primitive, folderPath)
		if err != nil {
			fmt.Println("Texture file not loaded, err =", err)
			continue
		}
		//fmt.Println("Still OK here?")
		textureMetallicRoughnessID, err := loadTextureGLTFMetallicRoughness(doc, primitive, folderPath)

		if err != nil {
			fmt.Println("Not loaded, err =", err)
			continue
		}

		//We have all the loading data, now store it:
		loadedMeshPtrs = append(loadedMeshPtrs, &meshGPU)
		loadedTextureBaseColorIDs = append(loadedTextureBaseColorIDs, textureBaseColorID)
		loadedTextureMetallicRoughnessIDs = append(loadedTextureMetallicRoughnessIDs, textureMetallicRoughnessID)
		fmt.Printf("Loaded.\n")
	}

	//fmt.Printf("Loaded %d instances out of %d Sponza assets.\n", len(loadedMeshPtrs), len(fnames))

	mWorldCommon := mgl32.Ident4()

	var instances []DrawableInstance
	for i := 0; i < len(loadedMeshPtrs); i++ {
		inst := DrawableInstance{
			mesh:                       loadedMeshPtrs[i],
			model:                      &mWorldCommon,
			baseColorTextureID:         loadedTextureBaseColorIDs[i],
			metallicRoughnessTextureID: loadedTextureMetallicRoughnessIDs[i]}
		instances = append(instances, inst)
	}
	var lights []Light

	light0 := Light{
		dir:         mgl32.Vec3{1, -0.5, -0.5},
		color:       mgl32.Vec3{1.0 * 2.0, 0.8 * 2.0, 0.6 * 2.0},
		intensity:   900.0,
		shMapWidth:  4096,
		shMapHeight: 4096,
		rerender:    true}

	light1 := Light{
		dir:         mgl32.Vec3{1, -0.25, -0.25},
		color:       mgl32.Vec3{1.0 * 2.0, 0.8 * 2.0, 0.6 * 2.0},
		intensity:   900.0,
		shMapWidth:  4096,
		shMapHeight: 4096,
		rerender:    true}

	light0.initShadowMap()
	light1.initShadowMap()

	lights = append(lights, light0, light1)
	//lights = append(lights, light0)

	return Scene{drawables: instances, lights: lights}
}

func loadMeshFromGLTF(doc *gltf.Document, primitive *gltf.Primitive) (MeshGeometry, error) {

	positionIndex := primitive.Attributes["POSITION"]
	positionAccessor := doc.Accessors[positionIndex]
	positions, err := modeler.ReadPosition(doc, positionAccessor, nil)
	if err != nil {
		log.Fatal(err)
	}
	/*
		fmt.Println("Positions:")
		fmt.Printf("%#v", positions)
		fmt.Println("\nDone.")
	*/
	normalIndex := primitive.Attributes["NORMAL"]
	normalAccessor := doc.Accessors[normalIndex]
	normals, err := modeler.ReadNormal(doc, normalAccessor, nil)
	if err != nil {
		log.Fatal(err)
	}
	/*
		fmt.Println("Normals:")
		fmt.Printf("%#v", normals)
		fmt.Println("\nDone.")
	*/
	uvIndex := primitive.Attributes["TEXCOORD_0"]
	uvAccessor := doc.Accessors[uvIndex]
	uvs, err := modeler.ReadTextureCoord(doc, uvAccessor, nil)
	if err != nil {
		log.Fatal(err)
	}
	/*
		fmt.Println("uvs:")
		fmt.Printf("%#v", uvs)
		fmt.Println("\nDone.")
	*/
	indexIndex := *(primitive.Indices)
	indexAccessor := doc.Accessors[indexIndex]
	indices, err := modeler.ReadIndices(doc, indexAccessor, nil)
	if err != nil {
		log.Fatal(err)
	}
	/*
		fmt.Println("Indices:")
		fmt.Printf("%#v", indices)
		fmt.Println("\nDone.")
	*/
	return MeshGeometry{positions, uvs, normals, indices}, nil

}

func uploadMeshToGPU(mshIN MeshGeometry) MeshGPUBufferIDs {

	var mshOUT MeshGPUBufferIDs
	mshOUT.lenIndices = int32(len(mshIN.indArray))

	gl.GenVertexArrays(1, &mshOUT.vaoID)
	gl.BindVertexArray(mshOUT.vaoID)

	// Buffer for vertices
	gl.GenBuffers(1, &mshOUT.vertexBufferID)
	gl.BindBuffer(gl.ARRAY_BUFFER, mshOUT.vertexBufferID)
	gl.BufferData(gl.ARRAY_BUFFER, len(mshIN.vertArray)*int(reflect.TypeOf(mshIN.vertArray).Elem().Size()),
		gl.Ptr(mshIN.vertArray), gl.STATIC_DRAW)

	// Buffer for uvs
	if len(mshIN.uvArray) > 0 {
		gl.GenBuffers(1, &mshOUT.uvBufferID)
		gl.BindBuffer(gl.ARRAY_BUFFER, mshOUT.uvBufferID)
		gl.BufferData(gl.ARRAY_BUFFER, len(mshIN.uvArray)*int(reflect.TypeOf(mshIN.uvArray).Elem().Size()),
			gl.Ptr(mshIN.uvArray), gl.STATIC_DRAW)
	}

	// Buffer for normals
	if len(mshIN.normArray) > 0 {
		gl.GenBuffers(1, &mshOUT.normalBufferID)
		gl.BindBuffer(gl.ARRAY_BUFFER, mshOUT.normalBufferID)
		gl.BufferData(gl.ARRAY_BUFFER, len(mshIN.normArray)*int(reflect.TypeOf(mshIN.normArray).Elem().Size()),
			gl.Ptr(mshIN.normArray), gl.STATIC_DRAW)
	}

	// Buffer for vertex indecies
	gl.GenBuffers(1, &mshOUT.indexBufferID)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, mshOUT.indexBufferID)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(mshIN.indArray)*int(reflect.TypeOf(mshIN.indArray).Elem().Size()),
		gl.Ptr(mshIN.indArray), gl.STATIC_DRAW)

	/*
		Two things to note here:
		Danger-1
		There was a bug here which I located only with the renderdoc by inspecting a cube.
		indArray initially was of type uint which takes 8 bytes, not 4.
		However, the function gl.BufferData works correctly only with 4-byte sized ints (uint32).
		It's very hard to see this as underneath it's just void* and bytenumber so you can pass any types.
		Set indArray as uint and all you would get is a a black screen without any compiler or glcheck errors.
		//indArray := []uint{0 1 2 1 3 2} would get uploaded as {0 0 1 0 2 0} wrongly bypassing any checks.
		Danger-2
		len and Elem is needed here as the direct .Size() on a slice does not produce the correct size in bytes.
	*/

	return mshOUT
}

func loadTextureGLTFBaseColor(doc *gltf.Document, primitive *gltf.Primitive, folderPath string) (uint32, error) {

	materialIndex := *(primitive.Material)
	material := doc.Materials[materialIndex]
	intermediateStruct := *(material.PBRMetallicRoughness)
	textureInfo := *(intermediateStruct.BaseColorTexture)
	textureIndex := textureInfo.Index
	texture := doc.Textures[textureIndex]
	imageIndex := *(texture.Source)
	image := doc.Images[imageIndex]
	textureFileToLoad := folderPath + image.URI
	/*
		fmt.Println("Texture file:")
		fmt.Printf("%#v", textureFileToLoad)
		fmt.Println("\nDone.")
	*/
	flipVertical := false
	desiredChannels := 0 // leave 0 for it to decide

	data, width, height, nChannels, cleanup, err := stbi.Load(textureFileToLoad, flipVertical, desiredChannels)
	//fmt.Println("width=", width, "height=", height, "nchannels=", nChannels)
	check(err)
	defer cleanup()

	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_2D, textureID)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	if nChannels == 3 {
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.SRGB, int32(width), int32(height), 0,
			gl.RGB, gl.UNSIGNED_BYTE, data)
	}
	if nChannels == 4 {
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.SRGB_ALPHA, int32(width), int32(height), 0,
			gl.RGBA, gl.UNSIGNED_BYTE, data)
	}
	if nChannels != 3 && nChannels != 4 {
		return 0, fmt.Errorf("3 or 4 nChannels required, received %v", nChannels)
	}

	gl.BindTexture(gl.TEXTURE_2D, 0)
	return textureID, nil
}

func loadTextureGLTFMetallicRoughness(doc *gltf.Document, primitive *gltf.Primitive, folderPath string) (uint32, error) {

	materialIndex := *(primitive.Material)
	material := doc.Materials[materialIndex]
	intermediateStruct := *(material.PBRMetallicRoughness)

	//This works as the struct field under the check is a pointer, otherwise one would need some reflection?!
	if intermediateStruct.MetallicRoughnessTexture == nil {
		return 0, fmt.Errorf("Check %s, its MetallicRoughnessTexture field does not exist in .gltf.", material.Name)
	}

	textureInfo := *(intermediateStruct.MetallicRoughnessTexture)

	textureIndex := textureInfo.Index
	texture := doc.Textures[textureIndex]
	imageIndex := *(texture.Source)
	image := doc.Images[imageIndex]
	textureFileToLoad := folderPath + image.URI

	/*
		fmt.Println("Texture file:")
		fmt.Printf("%#v", textureFileToLoad)
		fmt.Println("\nDone.")
	*/
	flipVertical := false
	desiredChannels := 0 // leave 0 for it to decide

	data, width, height, nChannels, cleanup, err := stbi.Load(textureFileToLoad, flipVertical, desiredChannels)
	//fmt.Println("width=", width, "height=", height, "nchannels=", nChannels)
	check(err)
	defer cleanup()

	var textureID uint32
	gl.GenTextures(1, &textureID)
	gl.BindTexture(gl.TEXTURE_2D, textureID)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	if nChannels == 3 {
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.SRGB, int32(width), int32(height), 0,
			gl.RGB, gl.UNSIGNED_BYTE, data)
	}
	if nChannels == 4 {
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.SRGB_ALPHA, int32(width), int32(height), 0,
			gl.RGBA, gl.UNSIGNED_BYTE, data)
	}
	if nChannels != 3 && nChannels != 4 {
		return 0, fmt.Errorf("3 or 4 nChannels required, received %v", nChannels)
	}

	gl.BindTexture(gl.TEXTURE_2D, 0)
	return textureID, nil
}

//Fill in texture and framebuffer IDs
func (lht *Light) initShadowMap() {

	gl.GenTextures(1, &(lht.txrID))

	// Init shadow map textures
	gl.BindTexture(gl.TEXTURE_2D, lht.txrID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT, lht.shMapWidth, lht.shMapHeight,
		0, gl.DEPTH_COMPONENT, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

	// Init FBO
	gl.GenFramebuffers(1, &(lht.FBOID))
	gl.BindFramebuffer(gl.FRAMEBUFFER, lht.FBOID)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, lht.txrID, 0)

	gl.ReadBuffer(gl.NONE) //No color texture will be used
	gl.DrawBuffer(gl.NONE) //No color texture will be used
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

}
