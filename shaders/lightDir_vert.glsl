#version 400 core

layout(location = 0) in vec3 inPosition;

uniform mat4 model;
uniform mat4 projView;

void main() {
    gl_Position = projView * model * vec4(inPosition, 1.0);
}
