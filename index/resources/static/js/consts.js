let consts = {
    runnableStatuses: {
        running: "running",
        stopped: "stopped",
    },
    identifierTypes: {
        runnable: "runnable",
        index: "index",
        ui: "ui",
        worker: "worker",
    },
    messageNames: {
        cmdRunnableStart: "cmd.runnable.start",
        cmdRunnableStop: "cmd.runnable.stop",
        cmdUIPing: "cmd.ui.ping",
        cmdUIRegister: "cmd.ui.register",
        eventRunnableCrashed: "event.runnable.crashed",
        eventRunnableStarted: "event.runnable.started",
        eventRunnableStopped: "event.runnable.stopped",
        eventUIWelcome: "event.ui.welcome",
        eventWorkerDisconnected: "event.worker.disconnected",
        eventWorkerRegistered: "event.worker.registered",
    },
}