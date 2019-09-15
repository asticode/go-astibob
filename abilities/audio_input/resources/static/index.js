let index = {
    init: function() {
        base.init({
            onLoad: index.onLoad,
        })
    },
    onLoad: function() {
        // Handle calibrate
        document.getElementById("btn-calibrate").addEventListener("click", index.handleCalibrate)

        // Finish
        base.finish()
    },
    handleCalibrate: function() {
        // Create text
        let c = document.createElement("div")
        c.style.textAlign = "center"
        c.innerText = "Say something..."

        // Show modal
        asticode.modaler.setWidth("300px")
        asticode.modaler.setContent(c)
        asticode.modaler.show()

        // Send calibrate request
        asticode.tools.sendHttp({
            method: "GET",
            url: "../routes/calibrate",
            error: function(data) {
                // Hide modal
                asticode.modaler.hide()

                // Update error
                if (typeof data.responseJSON !== "undefined" && typeof data.responseJSON.message !== "undefined") {
                    asticode.notifier.error(data.responseJSON.message)
                } else {
                    asticode.notifier.error("unknown error")
                }
            },
            success: function(data) {
                // Hide modal
                asticode.modaler.hide()

                // Add results
                index.addCalibrationResults(data.responseJSON)
            },
        })
    },
    addCalibrationResults: function(data) {
        // Set html
        document.getElementById("calibration-results").innerHTML = `<table>
    <tbody>
        <tr>
            <td>Max audio level</td>
            <td>` + Math.round(data.max_audio_level) + `</td>
        </tr>
        <tr>
            <td>Current max silence audio level</td>
            <td>` + Math.round(data.current_max_silence_audio_level) + `</td>
        </tr>
        <tr>
            <td>Suggested max silence audio level</td>
            <td>` + Math.round(data.suggested_max_silence_audio_level) + `</td>
        </tr>
    </tbody>
</table>
<canvas id='chart'></canvas>`

        // Add chart
        new Chart(document.getElementById("chart"), data.chart);
    }
}