let base = {
    finish: function() {
        asticode.loader.hide();
    },
    init: function(webSocketFunc, pageFunc) {
        // Init astitools
        asticode.loader.init();
        asticode.notifier.init();
        asticode.modaler.init();

        // Init buttons
        base.initButtons();

        // Get references
        asticode.loader.show();
        base.sendHttp("/api/references", "GET", function(data) {
            // Init the web socket
            base.initWebSocket(webSocketFunc, data.ws_ping_period, function() {
                // Get bob's information
                base.sendHttp("/api/bob", "GET", function(data) {
                    // Init menu
                    base.initMenu(data);

                    // Custom function
                    pageFunc();
                }, function() {
                    asticode.loader.hide();
                });
            });
        }, function() {
            asticode.loader.hide();
        });
    },
    initButtons: function() {
        // Stop Bob
        $("#btn-bob-stop").click(function() {
            base.sendHttp("/api/bob/stop", "GET");
        });
    },
    initMenu: function(data) {
        // Init html
        let html = `<div class="table">`;

        // Loop through abilities
        if (typeof data.abilities !== "undefined") {
            for (let k in data.abilities) {
                if (data.abilities.hasOwnProperty(k)) {
                    html += base.initToggle(k, data.abilities[k])
                }
            }
        }

        // Write html
        html += "</div>";
        $("#menu").html(html);
    },
    initToggle: function(key, data) {
        if (typeof data !== "undefined") {
            let state = (data.is_on ? "on" : "off");
            return `<div class="row">
                <div class="cell" style="padding-right: 10px">` + data.name + `</div>
                <div class="cell">
                    <label class="toggle ` + state + `" id="` + key + `" onclick="base.handleToggle('` + key + `')" data-state="` + state + `">
                        <span class="slider"></span>
                    </label>
                </div>
            </div>`;
        }
    },
    initWebSocket: function(webSocketFunc, pingPeriod, pageFunc) {
        // Init websocket
        base.ws = new WebSocket("ws://" + window.location.hostname + ":" + window.location.port +  "/websocket");

        // Declare functions
        let intervalPing;
        base.ws.onclose = function() {
            if (base.isOnline) {
                asticode.notifier.error("Bob is offline");
                base.isOnline = false;
            }
            clearInterval(intervalPing);
            setTimeout(function() { base.initWebSocket(webSocketFunc, pingPeriod, pageFunc) }, 1000);
        };
        base.ws.onopen = function() {
            base.isOnline = true;
            intervalPing = setInterval(function() { base.sendWs("ping"); }, pingPeriod * 1000);
            pageFunc();
        };
        base.ws.onmessage = function(event) {
            let data = JSON.parse(event.data);
            if (!base.webSocketFunc(data.event_name, data.payload)) {
                webSocketFunc(data.event_name, data.payload);
            }
        };
    },
    handleToggle: function(key) {
        base.sendHttp("/api/abilities/" + key + "/" + ($("#" + key).data("state") === "on" ? "off" : "on"), "GET");
    },
    sendHttp: function(url, method, successFunc, errorFunc) {
        $.ajax({
            url: url,
            type: method,
            dataType: "json",
            error: function(jqXHR) {
                // Get message
                let message = jqXHR.responseText;
                if (jqXHR.status === 0) {
                    message = "Bob is offline";
                } else if (jqXHR.status === 504) {
                    message = method + " request to " + url + " has timed out"
                } else if (typeof jqXHR.responseJSON !== "undefined") {
                    message = jqXHR.responseJSON.message
                }
                if (message !== "") {
                    asticode.notifier.error(message);
                }

                // Custom error handling
                if (typeof errorFunc !== "undefined") {
                    errorFunc();
                }
            },
            success: function(data) {
                if (typeof successFunc !== "undefined") {
                    successFunc(data);
                }
            }
        });
    },
    sendWs: function(event_name, payload) {
        base.ws.send(JSON.stringify({event_name: event_name, payload: payload}));
    },
    updateToggle: function(key, is_on) {
        let sw = $("#" + key);
        sw.removeClass(is_on ? "off" : "on");
        sw.addClass(is_on ? "on" : "off");
        sw.data("state", is_on ? "on" : "off")
    },
    webSocketFunc: function(event_name, payload) {
        switch (event_name) {
            case consts.webSocket.eventNames.abilityCrashed:
            case consts.webSocket.eventNames.abilityOff:
                base.updateToggle(payload, false);
                break;
            case consts.webSocket.eventNames.abilityOn:
                base.updateToggle(payload, true);
                break;
            default:
                return false;
        }
        return true;
    }
};