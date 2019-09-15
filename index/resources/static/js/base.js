let base = {
    from: {
        type: consts.identifierTypes.ui,
    },

    init: function(options) {
        // Init astitools
        asticode.loader.init();
        asticode.notifier.init();
        asticode.modaler.init();

        // Get references
        asticode.loader.show();
        asticode.tools.sendHttp({
            url: "/api/references",
            method: "GET",
            error: base.httpError,
            success: function(data) {
                // Create websocket
                asticode.ws.init({
                    okRequest: {
                        url: "/api/ok",
                        method: "GET",
                    },
                    url: data.responseJSON.websocket.addr,
                    pingPeriod: data.responseJSON.websocket.ping_period,
                    offline: function() { asticode.notifier.error("Server is offline") },
                    open: function() { asticode.loader.hide() },
                    messageRaw: function(data) {
                        // Log
                        console.debug("received msg", data)

                        // Switch on name
                        switch (data.name) {
                            case consts.messageNames.uiWelcome:
                                // Update from
                                base.from.name = data.payload.name

                                // Get message names
                                let ms = [
                                    consts.messageNames.runnableCrashed,
                                    consts.messageNames.runnableStarted,
                                    consts.messageNames.runnableStopped,
                                    consts.messageNames.workerDisconnected,
                                    consts.messageNames.workerRegistered,
                                ]

                                // Add custom message names
                                if (typeof options.messageNames !== "undefined" ) {
                                    options.messageNames.forEach(function(m) { ms.push(m) })
                                }

                                // Send register message
                                base.sendWebsocketMessage({
                                    name: consts.messageNames.uiRegister,
                                    payload: {
                                        message_names: ms,
                                        name: base.from.name,
                                    },
                                    to: {type: consts.identifierTypes.worker},
                                })

                                // Init menu
                                menu.init(data.payload)

                                // Custom callback
                                if (typeof options.onLoad !== "undefined") {
                                    options.onLoad(data.payload)
                                } else {
                                    base.finish()
                                }
                                break
                        }

                        // Menu
                        menu.onMessage(data)

                        // Custom callback
                        if (typeof options.onMessage !== "undefined") options.onMessage(data)
                    },
                    pingFunc: function() {
                        base.sendWebsocketMessage({
                            name: consts.messageNames.uiPing,
                            to: {type: consts.identifierTypes.index},
                        })
                    },
                })
            }
        })
    },
    finish: function() {
        asticode.loader.hide()
    },
    httpError: function(data) {
        if (typeof data.responseJSON !== "undefined") asticode.notifier.error(data.responseJSON.message)
        asticode.loader.hide();
    },
    sendWebsocketMessage: function(m) {
        m.from = base.from
        asticode.ws.sendJSON(m)
    }
}