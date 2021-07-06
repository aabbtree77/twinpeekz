#version 400 core

void main() {
    gl_FragDepth = gl_FragCoord.z;
}
