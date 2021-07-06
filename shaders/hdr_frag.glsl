#version 400 core

//This code is adapted from the following MIT-licensed 3D shader code by Angel Ortiz:

//https://github.com/Angelo1211/HybridRenderingEngine/blob/master/assets/shaders/PBRClusteredShader.frag

//Here point and spot lights are discarded, and so is ambient IBL, emissive etc.
//No need for that clutter, at least for now. Multiple directional light support is added here.

//Why the PBR code from HybridRenderingEngine? I liked two things about it: (i) naming conventions for 
//shader variables, though I do not use them as there are only few variables here, at least for now, but those 
//_wS for "world space" surely make life easier, (ii) the only code I saw that had the option not to
//calculate tangent spaces, which reduces complexity.

//What I don't like: the light color instead of radiance, unclear if this handles intensity well, but similarly
//in some other PBR codes. Unclear how everything adds to the final color.

//Other things to pay attention to in PBR codes: Tangent spaces often get done with C++ Assimp lib which takes 
//a lot of time to compile, warnings about something missing... 

//See Arcane (MIT licensed engine by Brady Jessup) that has PBR for all three types of lights, uses Assimp though.

#define M_PI 3.1415926535897932384626433832795

const int MAX_DIRLIGHTS = 8; //must match its value in rendering.go

struct DirLight {
    vec3 dir;
    vec3 color;
    float intensity;
    mat4 dirWorldToProj;
    sampler2D txrUnit; //shadowmap
};

uniform int numDirLights;
uniform DirLight dirlights[MAX_DIRLIGHTS];

in vec3 fragPos;
in vec3 fragNormal;
in vec2 fragUV;

uniform vec3 camPos;

uniform sampler2D albedoMap;
uniform sampler2D metalRoughMap;

out vec4 fragColor;


float calcDirShadow(DirLight light, vec3 point) {
    //point is fragment position in world space 
    vec4 pointLightSpace = light.dirWorldToProj * vec4(point, 1.0);
    //vec3 lightProjCoords = pointLightSpace.xyz;
    vec3 lightProjCoords = pointLightSpace.xyz / pointLightSpace.w;
    lightProjCoords = lightProjCoords * 0.5 + 0.5;
    float shadowDist = texture(light.txrUnit, lightProjCoords.xy).r;
    float lightDist = lightProjCoords.z;
        
    //No PCF:    
    //return lightDist > shadowDist + 0.01 ? 1.0 : 0.0;
        
    
    //3x3 PCF: 
    vec2 texelSize = 1.0 / textureSize(light.txrUnit, 0);
    float shadow = 0.0;
    for (int y = -1; y <= 1; ++y) {
        for (int x = -1; x <= 1; ++x) {
            float sampledDepthPCF = texture(light.txrUnit, lightProjCoords.xy + (texelSize * vec2(x, y))).r;
            shadow += lightDist > sampledDepthPCF + 0.001 ? 1.0 : 0.0;
        }
    }
    shadow /= 9.0;
    return shadow;
    
}


// PBR functions from HybridRenderingEngine:
//------------------------------------------------------------------------------------------------------------------------
vec3 fresnelSchlick(float cosTheta, vec3 F0){
    float val = 1.0 - cosTheta;
    return F0 + (1.0 - F0) * (val*val*val*val*val); //Faster than pow
}

//This func is for ambient IBL to add later
vec3 fresnelSchlickRoughness(float cosTheta, vec3 F0, float roughness){
    float val = 1.0 - cosTheta;
    return F0 + (max(vec3(1.0 - roughness), F0) - F0) * (val*val*val*val*val); //Faster than pow
}

float distributionGGX(vec3 N, vec3 H, float rough){
    float a  = rough * rough;
    float a2 = a * a;

    float nDotH  = max(dot(N, H), 0.0);
    float nDotH2 = nDotH * nDotH;

    float num = a2; 
    float denom = (nDotH2 * (a2 - 1.0) + 1.0);
    denom = 1 / (M_PI * denom * denom);

    return num * denom;
}

float geometrySchlickGGX(float nDotV, float rough){
    float r = (rough + 1.0);
    float k = r*r / 8.0;

    float num = nDotV;
    float denom = 1 / (nDotV * (1.0 - k) + k);

    return num * denom;
}

float geometrySmith(float nDotV, float nDotL, float rough){
    float ggx2  = geometrySchlickGGX(nDotV, rough);
    float ggx1  = geometrySchlickGGX(nDotL, rough);

    return ggx1 * ggx2;
}


vec3 calcDirLight(DirLight light, vec3 normal, vec3 viewDir, vec3 albedo, float rough, float metal, float shadow, vec3 F0){
    //Variables common to BRDFs
    vec3 lightDir = normalize(-light.dir);
    vec3 halfway  = normalize(lightDir + viewDir);
    float nDotV = max(dot(normal, viewDir), 0.0);
    float nDotL = max(dot(normal, lightDir), 0.0);
    vec3 radianceIn = light.color;

    //Cook-Torrance BRDF
    float NDF = distributionGGX(normal, halfway, rough);
    float G   = geometrySmith(nDotV, nDotL, rough);
    vec3  F   = fresnelSchlick(max(dot(halfway,viewDir), 0.0), F0);

    //Finding specular and diffuse component
    vec3 kS = F;
    vec3 kD = vec3(1.0) - kS;
    kD *= 1.0 - metal;

    vec3 numerator = NDF * G * F;
    float denominator = 4.0 * nDotV * nDotL;
    vec3 specular = numerator / max (denominator, 0.0001);

    vec3 radiance = (kD * (albedo / M_PI) + specular ) * radianceIn * nDotL;
    radiance *= (1.0 - shadow);

    return radiance;
}
//---------------------------------------------------------------------------------------------------------------------------

void main(){
    
    vec4 color      =  texture(albedoMap, fragUV).rgba;  
    vec2 metalRough =  texture(metalRoughMap, fragUV).bg;
    float metallic  =  metalRough.x;
    float roughness =  metalRough.y;

    vec3 albedo = color.rgb;
    float alpha = color .a;
    /*
    if(alpha < 0.5){
        discard;
    }
    */
    //TD: Extract normal from normal map and go through tangent space and all that, but
    //this will suffice for now:
    vec3 norm = normalize(fragNormal);
    
    vec3 viewDir = normalize(camPos - fragPos);

    vec3 F0   = vec3(0.04);
    F0 = mix(F0, albedo, metallic);

    vec3 radianceOut = vec3(0.0);
    for(int i = 0; i < numDirLights; ++i) {
        float shadow = calcDirShadow(dirlights[i], fragPos);
        radianceOut += calcDirLight(dirlights[i], norm, viewDir, albedo, roughness, metallic, shadow, F0);
    }
    //Add a small part of baseColor as ambient light, for now
    //fragColor = vec4(radianceOut, 1.0);
    //Increase 0.01 to say 0.05 if you like brighter clearly visible scenes with vivid colors
    fragColor = vec4(radianceOut+0.01*color.rgb, 1.0);
    //fragColor = color;
}
