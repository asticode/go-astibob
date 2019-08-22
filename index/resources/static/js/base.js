let base = {
    from: {
        type: consts.identifierTypes.ui,
    },

    init: function(messageHandler, onLoad) {
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
                        switch (data.name) {
                            case consts.messageNames.eventUIWelcome:
                                // Update from
                                base.from.name = data.payload.name

                                // Init menu
                                menu.init(data.payload)

                                // Custom callback
                                if (typeof onLoad !== "undefined") onLoad(data.payload)
                                break
                        }
                    },
                    pingFunc: function(ws) {
                        ws.sendJSON({
                            from: base.from,
                            name: consts.messageNames.cmdUIPing,
                            to: {type: consts.identifierTypes.index},
                        })
                    },
                })
            }
        })
    },
    httpError: function(data) {
        asticode.notifier.error(data.responseJSON.message)
        asticode.loader.hide();
    }
}