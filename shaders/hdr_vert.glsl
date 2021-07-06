#version 400 core

in vec3 inPosition;
in vec3 inNormal;
in vec2 inTexCoord;

uniform mat4 model;
uniform mat4 projViewModel;

out vec3 fragPos;
out vec3 fragNormal;
out vec2 fragUV;

void main() {
    fragPos = vec3(model * vec4(inPosition, 1.0));
    fragNormal = mat3(model)*inNormal;
    fragUV = inTexCoord;
    
    gl_Position = projViewModel*vec4(inPosition, 1.0);
}
