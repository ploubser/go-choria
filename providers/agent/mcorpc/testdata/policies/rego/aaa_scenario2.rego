package io.choria.aaasvc

default allow = false

allow {
    input.agent == "myco"
    input.action == "deploy"
}
