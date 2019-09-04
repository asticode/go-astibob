let build = {
    audio: new Audio(),
    newCount: 0,
    speeches: {},
    validatedCount: 0,

    init: function() {
        base.init(build.onMessage, function() {
            // Get references
            asticode.tools.sendHttp({
                method: "GET",
                url: "../routes/references/build",
                error: base.httpError,
                success: function(data) {
                    // Update options
                    build.options = data.responseJSON.options

                    // Update store new speeches
                    let e = document.querySelector("#store-new-speeches")
                    e.className = "toggle " + (build.options.store_new_speeches ? "on": "off")
                    e.addEventListener("click", build.updateOptions)

                    // Loop through speeches
                    data.responseJSON.speeches.forEach(function(s) {
                        // Add speech
                        build.addSpeech(s)
                    })

                    // Finish
                    base.finish()
                }
            })
        })
    },
    onMessage: function(data) {
        switch(data.name) {
            case "speech_to_text.options.build.updated":
                build.options = data.payload
                document.querySelector("#store-new-speeches").className = "toggle " + (build.options.store_new_speeches ? "on": "off")
                break
            case "speech_to_text.speech.created":
                build.addSpeech(data.payload)
                break
            case "speech_to_text.speech.deleted":
                build.delSpeech(data.payload)
                break
            case "speech_to_text.speech.updated":
                build.updateSpeech(data.payload)
                break
        }
    },
    updateOptions: function() {
        asticode.tools.sendHttp({
            method: "PATCH",
            url: "../routes/options/build",
            payload: JSON.stringify({
                store_new_speeches: !build.options.store_new_speeches,
            }),
            error: base.httpError,
        })
    },
    playAudio: function(s) {
        build.audio.pause()
        build.audio.currentTime = 0
        build.audio.src = "../routes/speeches/" + s.name + ".wav"
        build.audio.play()
    },
    addSpeech: function(i) {
        // Speech already exists
        if (typeof build.speeches[i.name] !== "undefined") return

        // Create speech
        let s = {
            created_at: i.created_at,
            html: {},
            is_validated: i.is_validated,
            name: i.name,
            text: i.text,
        }

        // Get container
        let c = document.querySelector("#new-speeches .speech-grid")
        if (s.is_validated) c = document.querySelector("#validated-speeches .speech-grid")

        // Create wrapper
        s.html.wrapper = document.createElement("div")
        s.html.wrapper.className = "speech-wrapper table"
        c.appendChild(s.html.wrapper)

        // Create cell
        c = document.createElement("div")
        c.className = "cell"
        s.html.wrapper.appendChild(c)

        // Create input
        s.html.input = document.createElement("input")
        s.html.input.value = s.text
        c.appendChild(s.html.input)

        // Handle focus
        s.html.input.addEventListener("focus", function() { build.playAudio(s) })

        // Handle key up
        s.html.input.addEventListener("keyup", function(e) {
            if (e.key === "Enter") {
                if (e.ctrlKey) {
                    asticode.tools.sendHttp({
                        method: "DELETE",
                        url: "../routes/speeches/" + s.name,
                        error: base.httpError,
                    })
                } else {
                    asticode.tools.sendHttp({
                        method: "PATCH",
                        url: "../routes/speeches/" + s.name,
                        payload: JSON.stringify({
                            is_validated: true,
                            text: s.html.input.value,
                        }),
                        error: base.httpError,
                    })
                }
            }
        })

        // Create cell
        c = document.createElement("div")
        c.className = "cell"
        s.html.wrapper.appendChild(c)

        // Create play icon
        let e = document.createElement("img")
        e.src = "../routes/static/play.png"
        c.appendChild(e)

        // Handle click
        e.addEventListener("click", function() { build.playAudio(s) })

        // Append speech
        build.speeches[s.name] = s

        // Update count
        build.updateCount(s, 1)
    },
    delSpeech: function(i) {
        // Get speech
        let s = build.speeches[i.name]

        // Speech doesn't exist
        if (typeof s === "undefined") return

        // Get next wrapper
        let w = s.html.wrapper.nextSibling

        // Remove html
        s.html.wrapper.remove()

        // Remove from pool
        delete(build.speeches[i.name])

        // Update count
        build.updateCount(s, -1)

        // Focus next input
        if (w !== null) w.querySelector("input").focus()
    },
    updateSpeech: function(i) {
        // Get speech
        let s = build.speeches[i.name]

        // Speech doesn't exist
        if (typeof s === "undefined") return

        // Update speech
        if (!s.is_validated && i.is_validated) {
            // Delete new speech
            build.delSpeech(s)

            // Add validated speech
            build.addSpeech(i)
        } else {
            // Update input
            if (s.text !== i.text) s.html.input.value = i.text

            // Update speech
            build.speeches[i.name].text = i.text
        }
    },
    updateCount: function(s, delta) {
        // Update count
        s.is_validated ? build.validatedCount -= delta : build.newCount -= delta

        // Update new html
        if (build.newCount === 0) {
            document.getElementById("new-speeches").style.display = "none"
        } else {
            document.getElementById("new-speeches").style.display = "block"
        }

        // Update validated html
        if (build.validatedCount === 0) {
            document.getElementById("validated-speeches").style.display = "none"
        } else {
            document.getElementById("validated-speeches").style.display = "block"
        }
    },
}