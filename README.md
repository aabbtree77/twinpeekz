> “Extinguished by light, I turn on the night <br>
Wear its darkness with an empty smile”<br>
&ndash; Pink Floyd "Wearing The Inside Out", 1994

> “There are things you can't get anywhere... but we dream they can be found in other people.”<br>
&ndash; Twin Peaks, 1991

<table align="center">
    <tr>
    <th align="center">Sponza under Two Suns</th>
    </tr>
    <tr>
    <td>
    <img src="./screenshots_volumetrics/screentwodirlightsSept2022.png"  alt="Sponza with two dir lights" width="100%" >
    </td>
    </tr>
</table>

## Introduction

This is a real time rendering of the Sponza demo scene, written in Go (Golang) and GLSL. Details:

1. i7, 16GB of RAM, GTX 760. Ubuntu, GLFW3, OpenGL, GLTF 2.0, MIT license.

2. Forward directional light(s), shadow mapping (PCF 3x3), basic PBR (no baking, no ambient term), 3D ray marched volumetric lighting.

Why yet another rendering code? In a narrow sense, this is a study of volumetric lighting and Go/GC in 3D. More broadly, Unreal/Unity/Godot types are hardly a joy to work with. Consider this attempt as a starting point towards a lightweight graphics engine whose rendering pipeline one could actually control.

## Setup

1. Install [Go](https://go.dev/doc/install).

2. Install Ubuntu dependencies:
  ```console
  sudo apt install xorg-dev libgl1-mesa-dev
  ```

3. Download Sponza from github to "Sponza_GLTF": 
  ```console
  sudo apt install subversion
  svn export https://github.com/KhronosGroup/glTF-Sample-Models.git/trunk/2.0/Sponza/glTF Sponza_GLTF
  ```

4. Install Blender:
  ```console
  sudo snap install blender --classic
  sudo snap refresh blender
  ``` 

5. Run Blender, delete the cube by pressing "x", import GLTF 2.0, select "Sponza_GLTF". If you press "n" or open the transform panel, "Dimensions" will show these values:
    ```console
    X = 29.8 m.
    Y = 18.3 m.
    Z = 12.4 m.
    ```
    
    Above "Dimensions" you will see "Scale" values, all of which will be 0.008, which is not right. Select the Sponza building in the Object mode, press ctrl+A and apply the scale. The transform panel's "Scale" values should now become 1.0. The transform panel is toggled by pressing "n".

    Export now the scene in GLTF 2.0 to Sponza_GLTF2, select "Format" as "GLTF Separate (.gltf + .bin + textures). Uncheck "+Y Up" in the "Transform" tab on the right side of the exporting prompt window, otherwise Y and Z axes will be switched and the camera will be messed up. Just in case, the initial camera position and orientation is set inside the function "makeCam()" in the file "camera.go": 
    ```go
    cam.UpdateOrientation(mgl32.Vec3{10.0, -4.5, 4.0}, mgl32.Vec3{-1.0, 0.9, 0.0}, Z_AXIS)
    ```
  
6. Clone this repo:
  ```console
  git clone https://github.com/aabbtree77/twinpeekz.git
  cd twinpeekz
  ```
  
7. Open "scene.go" and change the Sponza path to the one you reexported in Blender. In my case it is 
  ```console
  folderPath := "/home/tokyo/Sponza_GLTF2/"
  ```

8. Compile and run:
  ```console
  go build
  ./twinpeekz
  ```

## Code Organization and Render Pipeline

There are four layers of code:

1. **main.go**, **camera.go** - mostly GLFW stuff, window, mouse, keyboard management. Initial camera position in makeCam() (camera.go). AWSDEC  and arrow keys to move in 3D, holding+LMB changes camera view direction. Scrolling zooms in and out, F11 toggles full screen. Window resizes change certain frame buffers, but the shadow map resolution remains the same.

    Elementary camera-related math functions are relegated to the external dependency, i.e. the g3n engine (BSD-2), which one could "vendor" directly. The pain point will not be a Go package though, but the system libs like "xorg-dev" or "libgl1-mesa-dev" on Ubuntu. See [Fyne](https://github.com/fyne-io/fyne/blob/master/.github/workflows/platform_tests.yml) and g3n repos for more of these precise OpenGL/AL Ubuntu layer lib names.

2. **scene.go** - GLTF loading and some GPU buffer preparation. Each model must have .gltf, .bin and its texture image files in the same folder, tested mostly on an enhanced Sponza scene. There is no missing data filling and excessive checking, if the model does not have all the data it needs to have then it is simply not loaded or some error might occur. You know what is in your GLTF file and adjust scene.go to use the data you need.

3. **rendering.go**, **shader.go** - the rendering pipeline follows the C++ code of Tomas Öhberg. The file shader.go is the work by Nicholas Blaskey (MIT license), I leave a geometry shader compilation there in case the point lights would need to be added later (they are very expensive!).

4. Shaders:

    **hdr_frag.glsl** - this is the PBR code which I took from Angel Ortiz (MIT licensed), I removed light baking and the abient light contribution term.

    **vol_frag.glsl** - implements two approaches, the one by Andre Pestana (ray marches over visibility * phase func) and by Jake Ryan (marches just over visibility and postprocesses it into a color), see the links in those files. 

    **postpr_frag.glsl** - simply adds the hdr and volumetric colors and postprocesses them in some standard ways. There is an option to clamp a bit the 
    volumetric part to emphasize "god rays", but this also saturates colors, and one can also strengthen rays by adjusting Beer-related parameters in vol_frag.glsl,
    or treating color as radiance in Pestana's method (simply multiplying light.color by some intensity value).

The pipeline splits into

shadow mapping -> hdr (PBR) -> volumetrics -> postprocessing.

The execution time will depend largely on the number of lights and the shadow map texture sizes. Given a single directional light source and 4096x4096 textures, pure OpenGL rendering stage (measurement taken randomly, the last value summarizes all the stages):

timeOpenGLms = [1.190624 3.639968 2.843136 0.210208 7.883936]

squeezes a volumetrically rendered Sponza frame under 8ms.

This is the raw OpenGL part. The overall frame rendering in Go takes deltaT = 16.69ms, which I measure in the keyboard update function
inside "main.go".

Timing with two directional lights:

timeOpenGLms= [1.918592 3.783936 4.978624 0.210656 10.891808],

with the overall deltaT = 17.15ms.

## Rendering Discussion I: Shadows

I have initially ported the PCSS code taken from the MIT-licensed C++ (GLSL) code by Dihara Wijetunga:

https://github.com/diharaw/PercentageCloserSoftShadows

However, I was severely disappointed by its characteristics and timing, see the images below.

<table>
<tr>
<th> PCF 3x3, time = 1.1+2.2 ms </th>
<th> PCSS(128, 128), time = 1.1+75 ms</th>
</tr>
<tr>
<td>

<img src="./screenshots_PCSS/pcf3x3.jpeg"  alt="PCF 3x3" width="100%" >

</td>
<td>

<img src="./screenshots_PCSS/pcss128128.jpeg"  alt="PCF(128, 128)" width="100%" >

</td>
</tr>
</table>

<table>
<tr>
<th> PCSS(16, 16), time = 1.1+6.5 ms</th>
<th> PCSS(4, 25), time = 1.1+5.8 ms</th>
</tr>
<tr>
<td>

<img src="./screenshots_PCSS/pcss1616.jpeg"  alt="PCSS(16, 16)" width="100%" >

</td>
<td>

<img src="./screenshots_PCSS/pcss425.jpeg"  alt="PCSS(4, 25)" width="100%" >

</td>
</tr>
</table>

Shadow map resolution is 4096 in all the cases. The bare PCF 3x3 can be even better as PCSS makes certain shadows vanish.
Look into the shadow stripe on the left side, on the wall. It comes from the real bar-like mesh attached to the wall in the central part of Sponza which is not visible in the image. Notice how even ideal PCSS, which takes ages to compute (whole 75ms.), totally blurs away that shadow stripe.

Smaller internal blocker search and PCSS PCF values (which already take 3-4 extra ms.) produce artifacts which barely improve a raw shadow mapping with PCF 3x3.
So PCSS is a no go for me. It is probably a tool for some baking/prerendering that might give a smooth visual feel to the scene, or rendering at smaller shadow map resolutions followed by some filters, but it is better to simply increase the shadow map resolution as this stage takes roughly only 1ms. for 4K on my GTX 760, albeit for a single light.

VSM, ESM as faster smooth shadow alternatives? I deeply suspect many already went that way and did not find peace. So instead of trying every soft shadow technique one should probably just bite the bullet and go with an industry standard CSM. However, CSM introduces a significant jump in complexity. The code leaks from shaders to higher levels with all sorts of frustum computations and cascade stabilization and AA to avoid various "frame shimmering" effects. Considering all this baggage, the basic shadow mapping with PCF 3x3 at higher resolution such as 4K seems to be OK, until better, ray tracing times...

## Rendering Discussion II: PBR

Let us see some little PBR in action. 

<table>
<tr>
<th style="text-align:center"> Pseudo-PBR Rendering by Tomas Öhberg with a Dim Point Light Setup </th>
<th style="text-align:center"> PBR Rendering with Full Volumetrics </th>
</tr>
<tr>
<td>

<img src="./screenshots_PBR/NoPBR_tinyVol.jpeg"  alt="Non-PBR rendering by Tomas Öhberg with a weak point light setup." width="100%" >

</td>
<td>

<img src="./screenshots_PBR/PBR_Vol.jpeg"  alt="PBR rendering with full volumetrics in this code." width="100%" >

</td>
</tr>
</table>

Despite an overexposure of colors due to the light rays, you can see the specular/metallic reflections from the vases and the curtains, while these are less visible in the pseudo-PBR rendering (Tomas Öhberg code) even in the low volumetric exposure. These highlights also move as the camera view changes, enhancing 
the 3D experience in the both cases. The pseudo-PBR also emphasizes the edges of the curtain better.

What is a pseudo PBR? It is a shortcut I use for Tomas Öhberg's code. It has its own simplistic physics that includes specular highlights, but it does not use
the metallic roughness textures, only the base color which is also called "diffuse color".

Let us evaluate some more images.

<table>
<tr>
<th style="text-align:center"> PBR, Darker Setup, Non-Volumetric</th>
</tr>
<tr>
<td>
<img src="./screenshots_PBR/PBR.jpeg"  alt="PBR, darker setup, non-volumetric" width="100%" >
</td>
</tr>
</table>

<table>
<tr>
<th style="text-align:center"> Pseudo-PBR, Darker Setup, Non-Volumetric</th>
</tr>
<tr>
<td>
<img src="./screenshots_PBR/pseudoPBRdarker.jpeg"  alt="PBR, darker setup, non-volumetric" width="100%" >
</td>
</tr>
</table>

<table>
<tr>
<th style="text-align:center"> Pseudo-PBR, Bright Setup, Non-Volumetric</th>
</tr>
<tr>
<td>
<img src="./screenshots_PBR/pseudoPBRbright.jpeg"  alt="PBR, bright setup, non-volumetric" width="100%" >
</td>
</tr>
</table>

With a similar light intensity, pseudo-PBR is less playful, and the material is more dry and digital, but with a lot more light it exhibits some highlight generation. That proper light intensity region however is very narrow as with more light the shadow-light diffuse effects begin to dominate specular/metallic reflections which become barely visible, while in low light intensity settings the material has no highlights. Pseudo-PBR highlights are also bigger and more blurry, the surfaces do not have that thin glass layer that plays nicely with light.

On the other hand, somewhat brute shadows and mesh resolutions destroy the realism and mood if you start zooming or getting close to the objects.
So it is a fine tool which must go hand in hand with some other improvements. At the moment, no IBL and no ambient terms, I consider this as an unnecessary code complexity. Ambience is achieved by simply adding the base color weighted with some 0.01 factor and such, this is too crude, but I like things look a bit darker and do not mind underexposure and lack of detail in dark areas.

A skybox is still the basic need to implement, but I would not like some 3ms. wasted on just a nicer background, anything "samplerCube" related is quite slow on GTX 760.

In addition to baking and ambience, a proper PBR would bring one more complication, which is some tangent-bitangent space usually relegated either to Assimp, or custom C++ implementations. So the mesh essentially gets one more GPU tangent buffer in addition to normals. However, it is still possible to compute tangents directly in the shader, but this might reduce the performance. 

For some future implementation, I will mention three references here:

1. C++ Assimp, e.g. used by Arcane (MIT licensed):

    https://github.com/Ershany/Arcane-Engine/blob/78c9e6931704a36875f325f0449ddb04c9335032/Arcane%20Engine%20Core/src/graphics/mesh/Model.cpp

    const aiScene *scene = import.ReadFile(path, aiProcess_Triangulate | aiProcess_FlipUVs | aiProcess_CalcTangentSpace);

    Interesingly, CSM implementation intent is on Brady Jessup's Trello list since 2018, which just shows there is no simple answer to the shadow problem. See perhaps the CSM implementation by Dihara Wijetunga which was used by the Skylicht engine.

2. This custom C++ implementation

    https://github.com/Snowapril/gl_shaded_gltfscene/blob/5c6c766b4bf6efe7096184339173495606fc4d7c/Sources/Core/GLTFScene.cpp

    references the actual algorithm:

    //! Implementation in "Foundations of Game Engine Development : Volume2 Rendering"

3. Shader-based, my favorite most likely, exemplified by Lugdunum (MIT-licensed):

    https://github.com/Lugdunum3D/Lugdunum/blob/b6d6907d034fdba1ffc278b96598eba1d860f0d4/resources/shaders/forward/shader.frag
    ```console
    # if IN_TANGENT
        const vec3 tangent = normalize(inTangent.xyz);
        const vec3 bitangent = normalize(cross(inNormalWorldSpace, tangent) * inTangent.w);
        # else
        const vec3 deltaPosX = dFdx(inPositionWorldSpace);
        const vec3 deltaPosY = dFdy(inPositionWorldSpace);
        const vec2 deltaUvX = dFdx(TEXTURE_NORMAL_UV);
        const vec2 deltaUvY = dFdy(TEXTURE_NORMAL_UV);

        const vec3 tangent = normalize(deltaPosX * deltaUvY.t - deltaPosY * deltaUvX.t);
        const vec3 bitangent = -normalize(cross(inNormalWorldSpace, tangent));
        # endif

        const mat3 tbn = mat3(tangent, bitangent, inNormalWorldSpace);
    ```
    Here dFdx, dFdy are standard OpenGL functions, "return the partial derivative of an argument with respect to x or y".

    Another potentially useful reference for this method is this MIT-licensed code:

    https://github.com/kosowski/SimplePBR/blob/master/data/shaders/pbr/simplepbr.frag

    It uses Java-based Processing framework where the uniforms are set up, the function is "computeTangentFrame".

## Rendering Discussion III: Volumetric Light

<table>
<tr>
<th style="text-align:center"> Volumetric Lighting: Andre Pestana, Time = [1.06 3.78 2.52 0.21 7.58] ms </th>
<th style="text-align:center"> Volumetric Lighting: Jake Ryan, Time = [1.06 4.07 2.54 0.35 8.03] ms </th>
</tr>
<tr>
<td>

<img src="./screenshots_volumetrics/vol_andrepestana.jpeg"  alt="Volumetric Lighting: Andre Pestana" width="100%" >

</td>
<td>

<img src="./screenshots_volumetrics/vol_jakeryan.jpeg"  alt="Volumetric Lighting: Andre Pestana" width="100%" >

</td>
</tr>
</table>

The timings are provided for each rendering stage separately, supplied with the final total value of the rendered frame duration.

## Golang Experience Report

Golang is great at the level of main.go, no need to use unsafe pointers to pass user 
data with GLFW callbacks, just pass a struct/object method using an object as an implicit first argument (as in UFCS).

Golang's GC is not such a big problem, at least not for this scene, but 2-3x universal slow-down on raw loops compared 
to C/C++/Nim is not attractive.

Here is the log of Golang's GC:
```console
gc 3 @7.479s 0%: 0.051+24+0.004 ms clock, 0.41+0.47/0.22/0.14+0.036 ms cpu, 17->17->0 MB, 18 MB goal, 8 P
gc 4 @15.446s 0%: 0.042+0.73+0.047 ms clock, 0.34+0.18/0.60/0+0.38 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 5 @23.529s 0%: 0.038+0.39+0.009 ms clock, 0.30+0.15/0.47/0.35+0.079 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 6 @31.596s 0%: 0.037+0.60+0.005 ms clock, 0.30+0.15/0.58/0.033+0.046 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 7 @39.612s 0%: 0.038+0.48+0.032 ms clock, 0.30+0.15/0.51/0.54+0.25 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 8 @47.679s 0%: 0.043+0.59+0.10 ms clock, 0.35+0.081/0.64/0.52+0.81 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 9 @55.745s 0%: 0.037+0.34+0.007 ms clock, 0.30+0.32/0.49/0.15+0.058 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 10 @63.729s 0%: 0.040+0.67+0.14 ms clock, 0.32+0.16/0.56/0.33+1.1 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 11 @71.812s 0%: 0.047+0.63+0.008 ms clock, 0.38+0.14/0.62/0.094+0.068 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 12 @79.878s 0%: 0.039+0.76+0.033 ms clock, 0.31+0.18/0.66/0.030+0.26 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 13 @87.962s 0%: 0.040+0.61+0.009 ms clock, 0.32+0.18/0.59/0.42+0.073 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 14 @96.045s 0%: 0.044+25+0.006 ms clock, 0.35+0.33/0.42/0.14+0.051 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 15 @104.045s 0%: 0.040+1.3+0.062 ms clock, 0.32+0.81/0.70/1.5+0.50 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 16 @112.129s 0%: 0.036+0.60+0.005 ms clock, 0.29+0.17/0.61/0.029+0.046 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 17 @120.128s 0%: 0.040+0.44+0.11 ms clock, 0.32+0.18/0.53/0.51+0.94 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 18 @128.178s 0%: 0.036+0.52+0.008 ms clock, 0.28+0.17/0.59/0.58+0.070 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 19 @136.195s 0%: 0.054+0.81+0.007 ms clock, 0.43+0.088/0.36/0.63+0.062 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 20 @144.261s 0%: 0.037+0.58+0.005 ms clock, 0.30+0.14/0.59/0.045+0.045 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 21 @152.345s 0%: 0.038+0.48+0.006 ms clock, 0.30+0.15/0.50/0.52+0.054 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 22 @160.361s 0%: 0.038+0.48+0.020 ms clock, 0.31+0.17/0.56/0.33+0.16 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 23 @168.411s 0%: 0.038+0.67+0.013 ms clock, 0.31+0.16/0.60/0.021+0.10 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 24 @176.511s 0%: 0.036+0.39+0.006 ms clock, 0.29+0.14/0.53/0.19+0.053 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 25 @184.594s 0%: 0.039+0.66+0.006 ms clock, 0.31+0.16/0.68/0.032+0.049 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 26 @192.711s 0%: 0.038+0.55+0.10 ms clock, 0.30+0.34/0.59/0.28+0.80 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 27 @200.728s 0%: 0.037+0.35+0.006 ms clock, 0.30+0.32/0.47/0.19+0.053 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 28 @208.761s 0%: 0.036+0.41+0.006 ms clock, 0.29+0.23/0.47/0.48+0.052 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 29 @216.793s 0%: 0.036+0.40+0.007 ms clock, 0.28+0.13/0.47/0.48+0.061 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 30 @224.843s 0%: 0.045+0.52+0.063 ms clock, 0.36+0.33/0.53/0.41+0.50 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 31 @232.844s 0%: 0.038+24+0.004 ms clock, 0.31+0.27/0.42/0.28+0.035 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 32 @240.827s 0%: 0.029+18+0.006 ms clock, 0.23+0.27/0.34/0.15+0.055 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 33 @248.944s 0%: 0.036+0.49+0.015 ms clock, 0.29+0.13/0.53/0.38+0.12 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 34 @256.977s 0%: 0.036+0.61+0.009 ms clock, 0.29+0.17/0.53/0.51+0.074 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 35 @265.027s 0%: 0.037+0.43+0.011 ms clock, 0.30+0.30/0.48/0.48+0.088 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 36 @273.077s 0%: 0.096+24+0.005 ms clock, 0.77+0.40/0.29/0.088+0.046 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 37 @281.144s 0%: 0.038+0.53+0.12 ms clock, 0.30+0.15/0.63/0+0.99 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 38 @289.227s 0%: 0.061+0.56+0.10 ms clock, 0.49+0.17/0.48/0.47+0.83 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 39 @297.243s 0%: 0.037+18+0.006 ms clock, 0.29+0.33/0.47/0.24+0.050 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 40 @305.260s 0%: 0.050+15+0.005 ms clock, 0.40+0.48/0.33/0.23+0.045 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 41 @313.327s 0%: 0.031+0.63+0.006 ms clock, 0.24+0.12/0.63/0.049+0.049 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 42 @321.326s 0%: 0.049+0.56+0.006 ms clock, 0.39+0.15/0.56/0.022+0.048 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 43 @329.409s 0%: 0.042+0.51+0.072 ms clock, 0.34+0.18/0.57/0.66+0.57 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 44 @337.425s 0%: 0.046+0.42+0.006 ms clock, 0.37+0.14/0.49/0.60+0.054 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 45 @345.492s 0%: 0.037+14+0.007 ms clock, 0.30+0.41/0.31/0.29+0.057 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
gc 46 @353.542s 0%: 0.045+15+0.006 ms clock, 0.36+0.25/0.58/0.27+0.051 ms cpu, 4->4->0 MB, 5 MB goal, 8 P
```

You can see that the default GC setup does not consume more than 1ms. and it gets invoked every 8s. or so. However, a spike wasting whole 24ms. may occur once a minute or so. Dropping a frame or two per minute does not break any smooth 3D experience, but with heavier codes 
taking place in Go its GC could become a problem or would need a special focus and experiments.

Pointers are troublesome, and we get them without the ability to control stack vs heap 
allocations. They also overlap with some other purposes: a mutable function argument qualifier, or a test if the structure field exists after loading 
a struct from *.json, by checking for the nil value if the structure element has been defined as a pointer.

Also $GOPATH in the past, variable capitalization to mark visibility, unused variable errors, these are warts.

I also use these math functions as the external dependency:

https://github.com/g3n/engine/blob/master/math32/math.go

If you look into the code such as
```console
func Atan(v float32) float32 {
	return float32(math.Atan(float64(v)))
}
```
you might think how terrible Go is as this calls for generic types. It is actually on the contrary! This is readable, does not slow down the compilation, does not spit out crazy C++ template errors. Edit 2022: Go now has generic types since version 1.18. However, this is totally not an issue. [Pointers](https://github.com/g3n/engine/issues/163) are troublesome though.

The tools are good. I could use go-vim, and mostly just :GoDef and ctrl+O to get back, sometimes :GoRename. My GLTF code was written just by looking at Quim Muntal's GLTF library with :GoDef.

Why is Go so little used in 3D, is it worth pushing Go there? The GC spikes will always be there, and so will [cgo](https://zchee.github.io/golang-wiki/cgo/) due to king of the hill. A modern non-GC C-frontend such as Nim/Zig seems to be more suitable, at first glance. However, Go is remarkable in that it is one of so very few languages with simple polymorphism (at least prior to Go 1.18), not to mention a good parallelism story. These do not help taming graphics APIs, but might become handy with asset loading and code "at scale". History shows that anything static and non-GC is always a mess.

## OpenGL Experience Report

The amazing part is that OpenGL works, given that it is an API implemented by a graphics card maker (NVIDIA in my case) for Linux (Ubuntu). Programming is very tiresome though, as there is not much help from the compiler.

I was hunting down one bug for days which was rendering a black screen without a crash. The problem turned out to be gl.BufferData function storing mesh index array on a GPU. I was using
type uint whose analogue in C++ takes 4 bytes of memory, but in Go it's 8! Switching to uint32 simply solved the problem, but to actually locate 
that I had to use RenderDoc. Luckily, I had a working C++ alternative code by Tomas Öhberg, so I could compare the raw data differences in the both cases by loading a simple 3D cube.

An unused variable warning is actually an error and looks like this:
```console
2021/07/02 11:59:36 Could not find uniform:  metalRoughMap
```
Instead of the variable location in code we get the day of the month.

Tricky: Reading fragment's world position from the depth buffer of the hdr stage in the volumetric ligthing shader.

## To Do:

Automatic mesh scale and more scenes, tighter frustum, fix the rusty chain bug which is likely Sponza primitive No. 12 which has no 
MetallicRoughnessTexture in Sponza.gltf. I simply do not load the whole primitive in this case as the code does not have a fall down 
option to use PBR without such textures, whereas it should just load the base color and use it.

## Credits, Rendering Frameworks I Have Tried, Many Thanks To:

1. **Tomas Öhberg**. Initially I was simply translating his marvelous C++ work written in the Autumn of 2017:

    https://gitlab.com/tomasoh/100_procent_more_volume

    [https://youtu.be/Bafoekti34Y](https://youtu.be/Bafoekti34Y)

    It is a reasonable POD-mostly clean C++ that completely implements the method of Balázs Tóth and Tamás Umenhoffer (EUROGRAPHICS 2009). The translation was not smooth due to the bug I mentioned above and some OpenGL issues with sending an array of light structs to the shaders.

    Eventually, a few problems became visible. The lack of license, 

    https://gitlab.com/tomasoh/100_procent_more_volume/-/issues/1

    Some 3x slower than it should be shadow map stage and 3x slower volumetrics, caused mostly by the use of point lights and numerous distance-based functions. The rays were somewhat bleak and numerous scattering parameters barely changed anything visually. I was clamping the ray intensity artificially initially, and the code for that is still included in the postprocessing shader, though with the other two volumetric methods this problem is gone now. Parsing .obj files instead of GLTF 2.0, suboptimal PBR... An amazing work nonetheless.

2. **Baldur Karlsson** (and Crytek?) for RenderDoc which helped me to locate wrong mesh buffer indexing in "scene.go" (uint vs uint32!).

3. **Andre Pestana** for volumetrics and metallicRoughness:

    https://www.alexandre-pestana.com/volumetric-lights/

    https://www.alexandre-pestana.com/pbr-textures-sponza/

4. **Jake Ryan** for volumetrics:

    https://github.com/JuanDiegoMontoya/GLest-Rendererer/blob/main/glRenderer/Resources/Shaders/volumetric.fs

    Both of these works do not have licenses, but this should not be a problem. Andre Pestana is also the one who added metallicRoughness textures to Sponza via Subtance Designer.

5. **Dihara Wijetunga** for PCSS which I initially incorporated in my Go-controlled shaders, but eventually dropped as it is a wrong tool for real time rendering. His dwSampleFramework (MIT-licensed) is also quite a candidate to dissect and rewrite in more productive languages. It has OpenGL and Vulkan codes going side by side, and some interesting extensions scattered in his github repos. It compiles on Ubuntu with an error which is not hard to fix by googling (it comes from VS2019 use), and his models are not included in some places and need Blender to say generate "plane.obj" or "rescale lucy.obj".

6. **Panagiotis Christopoulos Charitos**, who helped me to solve the screen tearing issue which must be pretty common and good to know:

    https://github.com/godlikepanos/anki-3d-engine/issues/59

    His BSD-licensed Anki3d engine is some 2M LOC of C++ with Vulkan API (around 100KLOC of original code and the rest is 3rd party), but it compiles cleanly on Ubuntu, believe it or not, as many folders get some neat localized minimal cmake files. Contrast this to Confetti/The-Forge, (Apache 2.0) which also tries to be Linux friendly (kudos for that!), but distributes only LiteCode project files which are a lot more problematic and Windows-idiomatic, I had some issues with some -m32 flag that I do not remember anymore and also see my comment somewhere below

    https://www.youtube.com/watch?v=uFDa4M4ZBPs 

    Initially I was hoping to learn from any of these two engines, as they were both modern Vulkan-based and had volumetric lights I wanted above anything else. However, my excitement quickly evaporated as I could not get through the jungle of C++ and Vulkan.

7. **Finn Bear** and **Quim Muntal** on this github issue

    https://github.com/qmuntal/gltf/issues/26

    helped me to load the basic GLTF. Without their open correspondence I would not even be able to understand how to use the library. I then guessed the indexing pattern in some other places and relied on vim-go's :GoDef which allows to trace the GLTF structures. The problem is still there, that higher level layer to load everything into some structure is missing, but I understand why it is not there. The GLTF as such is some vast container and there can be so many features added or missing that it is perhaps better to leave it for a user to extract anything he or she wants. You know your GLTF, you write the code. The problem, however, is that nobody knows this all in the beginning, and Blender for instance manages to load everything, so a beginner expects any library to work this way. 
    
    A good clean GLTF code to start porting/extending is [this github code by Ryan Oldenburg](https://github.com/guzba/gltfviewer/blob/master/src/gltfviewer/gltf.nim), written in Nim. 

8. **Maximilian Mader**, for posing some modern C++ Windows-to-Ubuntu portability challenges:

    https://github.com/Max1412/Graphics2/issues/24

    This repo has no license, its main demo window is non-resizable, but it implements a full Wronski's SIGGRAPH 2014 Ubisoft approach to volumetrics, which I was hoping to be a great learning resource, but it turned out to be too complex for me, though it might be worth revisiting some day. Some quite advanced stuff is there: Volumetric textures, compute shaders, C++17 with some meta programming, a simple max function is quite polymorphic there and was causing complex template compilation errors, which was fixed by Maximilian Mader. C++ wise, I am only in the read-only, compile-only mode. 

    Then I encountered an OpenGL error and had the usual long blind dark alley walk which ended in a success without RenderDoc, but I do not know if RenderDoc would have helped me in this case. Again, why this was the problem on Ubuntu, but worked on Windows is beyond me, see the issue.

    This approach is heavier than one might think considering clusters and tiles in use, read the issue for some details and my further link to a specific Unity-related volumetrics discussion below. 

    There is a similar WebGL 2.0 implementation of clustered volumetrics:

    https://github.com/rms13/WebGL2-Volumetric-Renderer

    The code has no license too. Clustered ray marched volumetric ligthing is also implemented in Armory3D (Zlib-licensed), Lumix (MIT-licensed), Flax (commercial OSS), and Anki3d (BSD), but these are big engines and extracting/isolating the necessary bits could be difficult.
    There was someone experimenting with 3D ray marched volumetrics also in the Urho3D community, but the code did not manage to get into the main repo, please correct me if I am wrong.

9. **Michal Skalsky** and his Unity forum discussion:

    https://github.com/SlightlyMad/VolumetricLights

    https://forum.unity.com/threads/true-volumetric-lights-now-open-source.390818/page-5

    An interesting reading with a lot of inspiring images and insights into the problems of the Unity community, despite everything being Windows-centered and non-OpenGL. In order not to get lost in a big discussion, I will cite directly the most important and insightful bit of information here:

    Jun 16, 2016

    *gian-reto alig*:

    Didn't had the chance yet to watch the Adam short... any indication if Unitys volumetric fog approach is any good?

    *Michal_*:

    It is supposedly the same technique as the one introduced in Assassin's Creed 4 (and used in many other games since then). The paper is available online or in GPU Pro 6 book. It is a more general approach but both techniques has their advantages and disadvantages.

    Classic" raymarching approach (this, Killzone etc.) - volumetric effect is computed and immediatelly stored in 2d texture that is later applied to the final image.

    - You can render at native resolution and get best possible quality with crisp edges.
    - Keeps quality even for distant lights
    - Works with pretty much any shadow technique
    - Doesn't support transparent objects out of the box
    - Trading quality for performance is problematic. You can render at lower resolutions but you have to deal with all the "mixed resolution" rendering problems (halos, bleeding, etc.)
    - Probably more expensive per light but can give you better quality.

    Voxel based approach (Unity?, Assassin's Creed, Frostbite etc.) - volumetric effect is computed for "every" point in front of the camera. It is stored in 3d texture for future use.

    - You must render at very low resolution to make it practical. No crisp edges or shadow details. For example, AC4 used 160x90x64 resolution. Roughly 1 000 000 voxels. It would require hundreds of millions of voxels to match the highest quality of the classic technique.
    - Probably not very practical for long drawing distances because of the limited number of depth slices. The range for AC4 was only 100 meters.
    - Shouldn't work very well with current Unity shadows. The shadows have high frequency information and the voxel resolution is very low. That should lead to under-sampling artifacts, aliasing, flickering etc. AC4 used blurred exponential shadows to remove high frequency signal. I'm curious what Unity will do here. Maybe just a temporal super-sampling?
    - Works with transparent object. 3d texture stores light information for every point so you can easily apply this to transparent objects.
    - It is very simple to trade quality for performance. You simply change number of voxels. 3d texture filtering will take care of the rest.
    Whatever they do, I doubt it will work for everybody. There is so many different projects out there. I'm curious if they make it integral part of Unity or keep it as a separate asset.

10. **Daniel Salvadori**, g3n and Gokoban projects. This shows how much one can do with a GC language in 3D, and even though I am not much of a gamer, Gokoban is an awesome little 3D puzzle game that so deserves to be better known. It is sort of a prototype or a vast simplification of Stephen's Sausage Roll so appreciated by **Jonathan Blow**. It is too bad that the game requires some knowledge of how to set up 
Go and $GOPATH and is thus not available for non-programmers. I helped solving this issue, which I keep as a memo whenever I need to install this game on Ubuntu:

    https://github.com/danaugrs/gokoban/issues/14

    Edit 2022: This is no longer an issue as the repository now uses Go modules and compiles without gymnastics.

    TBH, I largely ignored the g3n engine code as I found it quite complex, but I was perhaps too unprepared at the time.
