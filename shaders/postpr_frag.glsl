#version 400 core

in vec2 fragmentTexCoord;
out vec4 out_Color;

uniform sampler2D hdrTexture;
uniform sampler2D volTexture;

uniform int hdrVolMixType; //0 - linear, 1 - nonlinear
uniform float clampPower;
uniform float gamma;
uniform float exposure;

void main(void) {

    vec3 hdrColor = texture(hdrTexture, fragmentTexCoord).rgb;
    vec4 volColor = texture(volTexture, fragmentTexCoord);
   
    if(hdrVolMixType == 0) {
        hdrColor += volColor.rgb;
        //hdrColor += 0.0;
    } else {
        hdrColor = mix(hdrColor, vec3(1.0, 1.0, 1.0), clamp(pow(volColor.r,clampPower), 0.0, 1.0) );
    }

    hdrColor = vec3(1.0) - exp(-hdrColor*exposure);
    hdrColor = pow(hdrColor, vec3(1.0/gamma));
    out_Color = vec4(hdrColor, 1.0);
}
