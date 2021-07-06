#version 400 core
/*
This code comes from the shaders developed by: 
Tomas Öhberg: https://gitlab.com/tomasoh/100_procent_more_volume/-/blob/master/shaders/volumetric.frag
Alexandre Pestana: https://www.alexandre-pestana.com/volumetric-lights/
Jake Ryan: https://github.com/JuanDiegoMontoya/GLest-Rendererer/blob/main/glRenderer/Resources/Shaders/volumetric.fs

I have chosen the second option as it has less parameters to tune and is as fast as raw visibility accumulation
by Jake Ryan which you can also try by setting a uniform "volumetricAlgo" to 0 in rendering.go.

The one by Tomas Öhberg is for point and spot lights only, but implements a full physics based approach of which the other two
are rough approximations. This physics approach is described in

BibTeX
@inproceedings {egs.20091048,
booktitle = {Eurographics 2009 - Short Papers},
editor = {P. Alliez and M. Magnor},
title = {{Real-time Volumetric Lighting in Participating Media}},
author = {Toth, Balazs and Umenhoffer, Tamas},
year = {2009},
publisher = {The Eurographics Association},
DOI = {10.2312/egs.20091048}
}

Better see the actual code.

The problem is that it is too slow: on my GTX760 even the modest sample number 24 takes 6ms. of volumetric stage alone, not to mention
the shadow map part for point lights which may take 3ms., all this only for a single light! 
Realistic rays are also not very intense despite that the model has so many parameters to adjust, but Tomas Öhberg gets a nice 
absorption of point/spot lights which does not saturate/over expose lit colors as the light fades away. God rays however are not strong,
though it is possible to clamp them somewhat, but it all depends on the goal really. There is no magic in the sense that beefier god rays 
oversaturate colors in any of these methods.

Apparently, full ray marching based volumetrics was already present in some early commercial closed source engines such as 
F.E.A.R., 2005, Perfect Dark Zero, 2005, Condemned: Criminal Origins, 2006, Crysis, 2007... See

https://www.youtube.com/watch?v=G0sYTrX3VHI&t=1085s

There is also some work on pushing scattering samples to very low numbers like 3 in INSIDE 2016 or 8 in Killzone Shadow Fall 2013 (GDC 2014) with 
lower than it should be shadow map resolutions, and then fixing the damage and artifacts with bilateral upscaling and all sorts of temporal AA. 
I have experimented with the bilateral filtering, but did not want to go this way, yet more parameters, and really no miracles with very low numbers of 
samples and filters. Filtering is also expensive. However, see Anki3D engine which works this way, it is a solid  BSD-licensed Vulkan engine, 
a complex code base, but compiles well on Ubuntu:

https://community.arm.com/developer/tools-software/graphics/b/blog/posts/clustered-volumetric-fog

*/


in vec2 fragmentTexCoord;
out vec4 out_Color;

const float M_PI = 3.141592653589793238;

const int MAX_LIGHTS = 8; //must match its value in rendering.go

uniform sampler2D shadowMap;

uniform mat4 invProjView;
uniform vec3 camPos;
uniform float screenWidth;
uniform float screenHeight;

struct DirLight {
    vec3 dir;
    vec3 color;
    float intensity;
    mat4 dirWorldToProj;
    sampler2D txrUnit; //shadowmap
};

uniform int numDirLights;
uniform DirLight dirlights[MAX_LIGHTS];

uniform int volumetricAlgo; //0 - rawVisibility, else - Pestana.
uniform float scatteringZFar; //set this to some large value to ignore it.
uniform int scatteringSamples;


/* The following functions are taken directly from the great work by Tomas Öhberg 
whose further bits of code also spread in my implementation of the iterations used 
by Andre Pestana and Jake Ryan.
----------------------------------------------------------------------------------------------
*/
/**
 * 2D Pseudo random number generator from:
 * The Book of Shaders by Patricio Gonzalez Vivo and Jen Lowe
 * https://thebookofshaders.com/10/
 * Returns a pseudo random value between 0.0 and 1.0
 */
float random(vec2 co) {
    return fract(sin(dot(co.xy, vec2(12.9898, 78.233))) * 43758.5453123);
}

vec3 fragmentWorldPos(float depthValue) {
    vec4 ndcCoords = vec4(
        2.0*(gl_FragCoord.x / screenWidth) - 1.0,
        2.0*(gl_FragCoord.y / screenHeight) - 1.0,
        2.0*depthValue - 1.0,
        1.0);

    vec4 worldCoords = invProjView * ndcCoords;
    return worldCoords.xyz / worldCoords.w;
}

float phaseFunction(vec3 inDir, vec3 outDir) {
    float anisotropy = 0.0;
    float cosAngle = dot(inDir, outDir) / (length(inDir)*length(outDir));
    float nom = 1 - anisotropy*anisotropy;
    float denom = 4 * M_PI * pow(1 + anisotropy*anisotropy - 2*anisotropy*cosAngle, 1.5);
    return nom/denom;
}
//---------------------------------------------------------------------------------------------

float shadowFactor(DirLight light, vec3 point) {
    vec4 pointLightSpace = light.dirWorldToProj * vec4(point, 1.0);
    vec3 lightProjCoords = pointLightSpace.xyz / pointLightSpace.w;
    lightProjCoords = lightProjCoords * 0.5 + 0.5;
    float shadowDist = texture(light.txrUnit, lightProjCoords.xy).r;
    float lightDist = lightProjCoords.z;
    return lightDist > shadowDist + 0.001 ? 1.0 : 0.0; //0.00004, 0.001
}


vec3 volScattering_rawVisibility(vec3 fragPosition, DirLight light) {
    
    vec3 camToFrag = fragPosition - camPos;
    if(length(camToFrag) > scatteringZFar) {
        camToFrag = normalize(camToFrag) * scatteringZFar;
    }
    vec3 deltaStep = camToFrag / (scatteringSamples+1);
    vec3 fragToCamNorm = normalize(camPos - fragPosition);
    vec3 x = camPos;
    
    //Why this randomization of an initial step improves things? See
    //Michal Valient, GDC 2014 which explains it in one picture.
    float rand = random(fragPosition.xy+fragPosition.z);
    x += (deltaStep*(1.0+rand));
    
    float result = 0.0;
    for(int i = 0; i < scatteringSamples; ++i) {
        float visibility = 1.0 - shadowFactor(light, x);
        result += visibility;
        x += deltaStep;
    }
    
    //This is from Jake Ryan's code:
    float d = result * length(camToFrag)/scatteringSamples;
    //float powder = 1.0 - exp(-d * 10.0);
    float powder = 1.0; //no need for that powder term really
    float beer = exp(-d * 0.01); //increasing exp const strengthens rays, but overexposes colors

    return (1.0 - beer) * powder * light.color;
    
}

vec3 volScattering_Pestana(vec3 fragPosition, DirLight light) {
    
    vec3 camToFrag = fragPosition - camPos;
    if(length(camToFrag) > scatteringZFar) {
        camToFrag = normalize(camToFrag) * scatteringZFar;
    }
    vec3 deltaStep = camToFrag / (scatteringSamples+1);
    vec3 fragToCamNorm = normalize(camPos - fragPosition);
    vec3 x = camPos;
    
    float rand = random(fragPosition.xy+fragPosition.z);
    x += (deltaStep*(1.0+rand));
    
    float result = 0.0;
    for(int i = 0; i < scatteringSamples; ++i) {
        float visibility = 1.0 - shadowFactor(light, x);
        result += visibility * phaseFunction(normalize(light.dir), fragToCamNorm);
        x += deltaStep;
    }
    
    return result/scatteringSamples * light.color;
}


void main(void) {
    vec4 volColor = vec4(0.0, 0.0, 0.0, 1.0);
    float depthValue = texture(shadowMap, fragmentTexCoord).r;
    vec3 fragPosition = fragmentWorldPos(depthValue);
    if (volumetricAlgo == 0){ 
        for(int i = 0; i < numDirLights; ++i) {
            volColor += vec4(volScattering_rawVisibility(fragPosition, dirlights[i]), 0.0);
        }
    } else {
        for(int i = 0; i < numDirLights; ++i) {
            volColor += vec4(volScattering_Pestana(fragPosition, dirlights[i]), 0.0);
        }
    }
    
    out_Color = volColor;

}
