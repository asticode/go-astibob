let base = {
    finish: function() {
        asticode.loader.hide();
    },
    init: function(webSocketFunc, pageFunc) {
        // Init astitools
        asticode.loader.init();
        asticode.notifier.init();
        asticode.modaler.init();

        // Before unload
        window.onbeforeunload = function() {
            base.showOfflineMessage = false;
            if (typeof base.ws !== "undefined") {
                base.ws.close();
            }
        };

        // Init buttons
        base.initButtons();

        // Get references
        asticode.loader.show();
        base.sendHttp("/api/references", "GET", function(data) {
            // Init the web socket
            base.initWebSocket(webSocketFunc, data.ws_url, data.ws_ping_period, function() {
                // Get bob's information
                base.sendHttp("/api/bob", "GET", function(data) {
                    // Init menu
                    menu.init(data);

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
    initWebSocket: function(webSocketFunc, url, pingPeriod, pageFunc) {
        // Try pinging the API
        $.ajax({
            url: "/api/ok",
            type: "GET",
            error: function() {
                setTimeout(function() {
                    base.initWebSocket(webSocketFunc, url, pingPeriod, pageFunc);
                }, 1000);
            },
            success: function() {
                // Init websocket
                base.ws = new WebSocket(url);

                // Declare functions
                let intervalPing;
                base.ws.onclose = function() {
                    if (base.showOfflineMessage) {
                        asticode.notifier.error("Bob is offline");
                        base.showOfflineMessage = false;
                    }
                    clearInterval(intervalPing);
                    setTimeout(function() {
                        base.initWebSocket(webSocketFunc, url, pingPeriod, pageFunc);
                    }, 1000);
                };
                base.ws.onopen = function() {
                    base.showOfflineMessage = true;
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
        });
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
    webSocketFunc: function(event_name, payload) {
        switch (event_name) {
            case consts.webSocket.eventNames.abilityCrashed:
            case consts.webSocket.eventNames.abilityStopped:
                menu.updateToggle(payload, false);
                break;
            case consts.webSocket.eventNames.abilityStarted:
                menu.updateToggle(payload, true);
                break;
            case consts.webSocket.eventNames.brainDisconnected:
                menu.removeBrain(payload);
                break;
            case consts.webSocket.eventNames.brainRegistered:
                menu.addBrain(payload);
                break;
            default:
                return false;
        }
        return true;
    },
    addInAlphabeticalOrder: function(rootSelector, data, map) {
        // Find proper key
        let key;
        for (let k in map) {
            if (map.hasOwnProperty(k)) {
                if (map[k].name > data.name && (typeof key === "undefined" || map[key].name > map[k].name)) {
                    key = k;
                    break;
                }
            }
        }

        // Update html
        if (typeof key !== "undefined") {
            map[key].html.wrapper.before(data.html.wrapper);
        } else {
            rootSelector.append(data.html.wrapper);
        }
    }
};