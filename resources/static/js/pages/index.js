let index = {
    init: function () {
        base.init(index.webSocketCallback, function() {
            // Get bob's status
            $.ajax({
                url: "/api/bob",
                type: "GET",
                dataType: 'json',
                success: function(data) {
                    // Init html
                    let html = `<div class="table">`;

                    // Loop through abilities
                    if (typeof data.abilities !== "undefined") {
                        for (let k in data.abilities) {
                            if (data.abilities.hasOwnProperty(k)) {
                                html += index.initSwitch(k, data.abilities[k])
                            }
                        }
                    }

                    // Write html
                    html += "</div>";
                    document.getElementById("index").innerHTML = html;
                }
            });
        });
    },
    initSwitch: function(key, data) {
        if (typeof data !== "undefined") {
            let state = (data.is_on ? "on" : "off");
            return `<div class="row">
                <div class="cell">` + data.name + `</div>
                <div class="cell">
                    <label class="switch ` + state + `" id="` + key + `" style="margin-left: 10px" onclick="index.handleSwitch('` + key + `')" data-state="` + state + `">
                        <span class="slider round"></span>
                    </label>
                </div>
            </div>`;
        }
    },
    handleSwitch: function(key) {
        $.ajax({
            url: "/api/abilities/" + key + "/" + ($("#" + key).data("state") === "on" ? "off" : "on"),
            type: "GET",
            dataType: 'json'
        })
    },
    updateSwitch: function(key, is_on) {
        let sw = $("#" + key);
        sw.removeClass(is_on ? "off" : "on");
        sw.addClass(is_on ? "on" : "off");
        sw.data("state", is_on ? "on" : "off")
    },
    webSocketCallback: function(event_name, payload) {
        switch (event_name) {
            case consts.webSocket.eventNames.abilityCrashed:
            case consts.webSocket.eventNames.abilityOff:
                index.updateSwitch(payload, false);
                break;
            case consts.webSocket.eventNames.abilityOn:
                index.updateSwitch(payload, true);
                break;
        }
    }
};