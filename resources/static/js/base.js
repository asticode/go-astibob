let base = {
    init: function(webSocketFunc, pageFunc) {
        // Get references
        $.ajax({
            url: "/api/references",
            type: "GET",
            dataType: 'json',
            success: function(data) {
                // Init the web socket
                base.initWebSocket(webSocketFunc, data.ws_ping_period, pageFunc);
            }
        });
    },
    initWebSocket: function(webSocketFunc, pingPeriod, pageFunc) {
        // Try to init the websocket until it doesn't fail
        let stop = false;
        while(!stop) {
            try {
                base.ws = new WebSocket("ws://" + window.location.hostname + ":" + window.location.port +  "/websocket");
                stop = true
            } catch(err) {
                console.log(err);
                base.sleep(1000);
            }
        }

        // Declare functions
        let intervalPing;
        base.ws.onclose = function() {
            clearInterval(intervalPing);
            base.initWebSocket(webSocketFunc, pingPeriod, pageFunc);
        };
        base.ws.onopen = function() {
            intervalPing = setInterval(function() { base.send("ping"); }, pingPeriod * 1000)
            pageFunc();
        };
        base.ws.onmessage = function(event) {
            let data = JSON.parse(event.data);
            webSocketFunc(data.event_name, data.payload);
        };
    },
    send: function(event_name, payload) {
        base.ws.send(JSON.stringify({event_name: event_name, payload: payload}));
    },
    sleep: function(milliseconds){
        let waitUntil = new Date().getTime() + milliseconds;
        while(new Date().getTime() < waitUntil) true;
    },
};