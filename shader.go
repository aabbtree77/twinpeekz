// Based on
// https://github.com/NicholasBlaskey/gophergl/blob/main/Open/gl/shader.go
// which is released under the MIT license.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Shader struct {
	ID uint32
}

func MakeShaders(vertexPath string, fragmentPath string) Shader {
	// Read the source code into strings
	vertexCodeBytes, err := ioutil.ReadFile(vertexPath)
	if err != nil {
		panic(err)
	}
	vertexCode := string(vertexCodeBytes)

	fragmentCodeBytes, err := ioutil.ReadFile(fragmentPath)
	if err != nil {
		panic(err)
	}
	fragmentCode := string(fragmentCodeBytes)

	// Compile the shaders
	vertexShader := gl.CreateShader(gl.VERTEX_SHADER)
	shaderSource, freeVertex := gl.Strs(vertexCode + "\x00")
	defer freeVertex()
	gl.ShaderSource(vertexShader, 1, shaderSource, nil)
	gl.CompileShader(vertexShader)
	checkCompileErrors(vertexShader, "VERTEX")

	fragmentShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	shaderSource, freeFragment := gl.Strs(fragmentCode + "\x00")
	defer freeFragment()
	gl.ShaderSource(fragmentShader, 1, shaderSource, nil)
	gl.CompileShader(fragmentShader)
	checkCompileErrors(fragmentShader, "FRAGMENT")

	// Create a shader program
	ID := gl.CreateProgram()
	gl.AttachShader(ID, vertexShader)
	gl.AttachShader(ID, fragmentShader)
	gl.LinkProgram(ID)

	checkCompileErrors(ID, "PROGRAM")

	// Delete shaders
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return Shader{ID: ID}
}

func MakeGeomShaders(vertexPath, fragmentPath, geoPath string) Shader {
	// Read the source code into strings
	vertexCodeBytes, err := ioutil.ReadFile(vertexPath)
	if err != nil {
		panic(err)
	}
	vertexCode := string(vertexCodeBytes)

	fragmentCodeBytes, err := ioutil.ReadFile(fragmentPath)
	if err != nil {
		panic(err)
	}
	fragmentCode := string(fragmentCodeBytes)

	geoCodeBytes, err := ioutil.ReadFile(geoPath)
	if err != nil {
		panic(err)
	}
	geoCode := string(geoCodeBytes)

	// Compile the shaders
	vertexShader := gl.CreateShader(gl.VERTEX_SHADER)
	shaderSource, freeVertex := gl.Strs(vertexCode + "\x00")
	defer freeVertex()
	gl.ShaderSource(vertexShader, 1, shaderSource, nil)
	gl.CompileShader(vertexShader)
	checkCompileErrors(vertexShader, "VERTEX")

	fragmentShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	shaderSource, freeFragment := gl.Strs(fragmentCode + "\x00")
	defer freeFragment()
	gl.ShaderSource(fragmentShader, 1, shaderSource, nil)
	gl.CompileShader(fragmentShader)
	checkCompileErrors(fragmentShader, "FRAGMENT")

	geoShader := gl.CreateShader(gl.GEOMETRY_SHADER)
	shaderSource, freeGeo := gl.Strs(geoCode + "\x00")
	defer freeGeo()
	gl.ShaderSource(geoShader, 1, shaderSource, nil)
	gl.CompileShader(geoShader)
	checkCompileErrors(geoShader, "GEOMETRY")

	// Create a shader program
	ID := gl.CreateProgram()
	gl.AttachShader(ID, vertexShader)
	gl.AttachShader(ID, fragmentShader)
	gl.AttachShader(ID, geoShader)
	gl.LinkProgram(ID)

	checkCompileErrors(ID, "PROGRAM")

	// Delete shaders
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	gl.DeleteShader(geoShader)

	return Shader{ID: ID}
}

func (s Shader) Use() {
	gl.UseProgram(s.ID)
}

func (s Shader) SetBool(name string, value bool) {
	var intValue int32 = 0
	if value {
		intValue = 1
	}
	gl.Uniform1i(gl.GetUniformLocation(s.ID, gl.Str(name+"\x00")),
		intValue)
}

func (s Shader) SetInt(name string, value int32) {

	loc := gl.GetUniformLocation(s.ID, gl.Str(fmt.Sprintf("%s\x00", name)))
	if loc == -1 {
		log.Fatalln("Could not find uniform: ", name)
	}
	gl.Uniform1i(loc, value)
}

func (s Shader) SetFloat(name string, value float32) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(fmt.Sprintf("%s\x00", name)))
	if loc == -1 {
		log.Fatalln("Could not find uniform: ", name)
	}
	gl.Uniform1f(loc, value)
}

func (s Shader) SetVec2(name string, value mgl32.Vec2) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(fmt.Sprintf("%s\x00", name)))
	if loc == -1 {
		log.Fatalln("Could not find uniform: ", name)
	}
	gl.Uniform2fv(loc, 1, &value[0])
}

func (s Shader) SetVec3(name string, value mgl32.Vec3) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(fmt.Sprintf("%s\x00", name)))
	if loc == -1 {
		log.Fatalln("Could not find uniform: ", name)
	}
	gl.Uniform3fv(loc, 1, &value[0])
}

func (s Shader) SetMat4(name string, value mgl32.Mat4) {
	loc := gl.GetUniformLocation(s.ID, gl.Str(fmt.Sprintf("%s\x00", name)))
	if loc == -1 {
		log.Fatalln("Could not find uniform: ", name)
	}
	gl.UniformMatrix4fv(loc, 1, false, &value[0])
}

func checkCompileErrors(shader uint32, shaderType string) {
	var success int32
	var infoLog [1024]byte

	var status uint32 = gl.COMPILE_STATUS
	stageMessage := "Shader_Compilation_error"
	errorFunc := gl.GetShaderInfoLog
	getIV := gl.GetShaderiv
	if shaderType == "PROGRAM" {
		status = gl.LINK_STATUS
		stageMessage = "Program_link_error"
		errorFunc = gl.GetProgramInfoLog
		getIV = gl.GetProgramiv
	}

	getIV(shader, status, &success)
	if success != 1 {
		test := &success
		errorFunc(shader, 1024, test, (*uint8)(unsafe.Pointer(&infoLog)))
		log.Fatalln(stageMessage + shaderType + "|" + string(infoLog[:1024]) + "|")
	}
}
