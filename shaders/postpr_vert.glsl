#version 400 core

in vec3 inPosition;
in vec2 inTexCoord;

out vec2 fragmentTexCoord;
    
void main(void) {
    fragmentTexCoord = inTexCoord;
    gl_Position = vec4(inPosition, 1.0);
}
