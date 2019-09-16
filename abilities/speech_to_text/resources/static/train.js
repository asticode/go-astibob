let train = {
    init: function() {
        base.init({
            messageNames: train.messageNames,
            onLoad: train.onLoad,
            onMessage: train.onMessage,
        })
    },
    onLoad: function() {
        // Get references
        asticode.tools.sendHttp({
            method: "GET",
            url: "../routes/references/train",
            error: base.httpError,
            success: function(data) {
                // Handle cancel
                document.getElementById("btn-cancel").addEventListener("click", train.cancel)

                // Handle train
                document.getElementById("btn-train").addEventListener("click", train.train)

                // Create progresser
                train.progresser = asticode.progresser.new({
                    error: function(error) {
                        let e = document.getElementById("error")
                        e.innerText = error
                        e.style.display = "block"
                    },
                    root: document.getElementById("progress"),
                })

                // Update progress
                train.updateProgress(data.responseJSON.progress)

                // Finish
                base.finish()
            }
        })
    },
    messageNames: [
        "speech_to_text.progress",
    ],
    onMessage: function(data) {
        switch (data.name) {
            case "speech_to_text.progress":
                // Refresh error
                document.getElementById("error").style.display = "none"

                // Update progress
                train.updateProgress(data.payload)
                break
        }
    },
    train: function() {
        asticode.tools.sendHttp({
            method: "GET",
            url: "../routes/train",
            error: base.httpError,
        })
    },
    cancel: function() {
        asticode.tools.sendHttp({
            method: "GET",
            url: "../routes/train/cancel",
            error: base.httpError,
        })
    },
    updateProgress: function(progress) {
        // Hide/Show buttons
        if (typeof progress === "undefined" || typeof progress.error !== "undefined" || (progress.progress === 100 && progress.current_step === progress.steps[progress.steps.length - 1])) {
            document.getElementById("btn-cancel").style.display = "none"
            document.getElementById("btn-train").style.display = "block"
        } else {
            document.getElementById("btn-cancel").style.display = "block"
            document.getElementById("btn-train").style.display = "none"
        }

        // Update progresser
        if (typeof progress !== "undefined") {
            train.progresser.update(progress)
        }
    }
}