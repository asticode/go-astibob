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
                // Handle train
                document.getElementById("btn-train").addEventListener("click", train.train)

                // Create progresser
                train.progresser = asticode.progresser.new({
                    error: function(error) { asticode.notifier.error(error) },
                    root: document.getElementById("progress"),
                })

                // Check progress
                if (typeof data.responseJSON.progress !== "undefined") {
                    // Update progress
                    train.progresser.update(data.responseJSON.progress)
                }

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
                // Update progress
                train.progresser.update(data.payload)
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
}