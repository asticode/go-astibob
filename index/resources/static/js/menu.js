let menu = {
    // Attributes
    workers: {},
    workersCount: 0,

    // Init

    init: function(data) {
        // Loop through workers
        if (typeof data.workers !== "undefined") {
            for (let k = 0; k < data.workers.length; k++) {
                menu.addWorker(data.workers[k])
            }
        }
    },

    // Websocket

    onMessage: function(data) {
        switch (data.name) {
            case consts.messageNames.runnableCrashed:
            case consts.messageNames.runnableStarted:
            case consts.messageNames.runnableStopped:
                // Update toggle
                menu.updateToggle(data)
                break
            case consts.messageNames.workerDisconnected:
                // Remove worker from menu
                menu.removeWorker(data.payload)
                break
            case consts.messageNames.workerRegistered:
                // Add worker to menu
                menu.addWorker(data.payload)
                break
        }
    },

    // Worker

    addWorker: function(data) {
        // Worker already exists
        if (typeof menu.workers[data.name] !== "undefined") {
            return
        }

        // Create worker
        let worker = menu.newWorker(data)

        // Add in alphabetical order
        asticode.tools.appendSorted(document.querySelector("#menu"), worker, menu.workers)

        // Append to pool
        menu.workers[worker.name] = worker

        // Update workers count
        menu.updateWorkersCount(1)
    },
    newWorker: function(data) {
        // Init
        let r = {
            runnables: {},
            html: {},
            name: data.name,
        }

        // Create wrapper
        r.html.wrapper = document.createElement("div")
        r.html.wrapper.className = "menu-worker"

        // Create name
        let name = document.createElement("div")
        name.className = "menu-worker-name"
        name.innerText = data.name
        r.html.wrapper.appendChild(name)

        // Create table
        r.html.table = document.createElement("div")
        r.html.table.className = "table"
        r.html.wrapper.appendChild(r.html.table)

        // Loop through runnables
        if (typeof data.runnables !== "undefined") {
            for (let k = 0; k < data.runnables.length; k++) {
                menu.addRunnable(r, data.runnables[k])
            }
        }
        return r
    },
    removeWorker: function(name) {
        // Fetch worker
        let worker = menu.workers[name]

        // Worker exists
        if (typeof worker !== "undefined") {
            // Remove HTML
            worker.html.wrapper.remove()

            // Delete from pool
            delete(menu.workers[name])

            // Update workers count
            menu.updateWorkersCount(-1)
        }
    },
    updateWorkersCount: function(delta) {
        // Update workers count
        menu.workersCount += delta

        // Hide worker name
        let items = document.querySelectorAll(".menu-worker-name")
        if (menu.workersCount > 1) {
            items.forEach(function(item) { item.style.display = "block" })
        } else {
            items.forEach(function(item) { item.style.display = "none" })
        }
    },

    // Runnable

    addRunnable: function(worker, data) {
        // Runnable already exists
        if (typeof worker.runnables[data.name] !== "undefined") {
            return
        }

        // Create runnable
        let runnable = menu.newRunnable(worker, data)

        // Add in alphabetical order
        asticode.tools.appendSorted(worker.html.table, runnable, worker.runnables)

        // Append to pool
        worker.runnables[runnable.name] = runnable
    },
    newRunnable: function(worker, data) {
        // Create results
        let r = {
            description: data.description,
            html: {},
            name: data.name,
            status: data.status,
            web_homepage: data.web_homepage,
            worker_name: worker.name,
        }

        // Create wrapper
        r.html.wrapper = document.createElement("div")
        r.html.wrapper.className = "row"
        r.html.wrapper.title = r.description

        // Create name
        let name = document.createElement("div")
        name.className = "cell"
        name.style.paddingRight = "10px"
        r.html.wrapper.appendChild(name)

        // Create title
        let title
        if (typeof r.web_homepage !== "undefined") {
            title = document.createElement("a")
            title.href = r.web_homepage
            title.innerText = r.name
        } else {
            title = document.createElement("span")
            title.innerText = r.name
        }
        name.appendChild(title)

        // Create toggle cell
        let cell = document.createElement("div")
        cell.class = "cell"
        cell.style.fontSize = "11px"
        cell.style.textAlign = "right"
        r.html.wrapper.appendChild(cell)

        // Create toggle
        r.html.toggle = document.createElement("label")
        r.html.toggle.className = "toggle " + menu.toggleClass(data.status)
        r.html.toggle.innerHTML = '<span class="slider"></span>'
        r.html.toggle.addEventListener("click", function() {
            // Create message
            let m = {
                to: {
                    name: r.worker_name,
                    type: consts.identifierTypes.worker,
                },
                payload: r.name,
            }

            // Add name
            if (r.status === consts.runnableStatuses.stopped) {
                m.name = consts.messageNames.runnableStart
            } else {
                m.name = consts.messageNames.runnableStop
            }

            // Send message
            base.sendWebsocketMessage(m)
        })
        cell.appendChild(r.html.toggle)
        return r
    },
    updateToggle: function(data) {
        // Fetch worker
        let worker = menu.workers[data.from.worker]

        // Worker exists
        if (typeof worker !== "undefined") {
            // Fetch runnable
            let runnable = worker.runnables[data.from.name]

            // Runnable exists
            if (typeof runnable !== "undefined") {
                // Update status
                runnable.status = (data.name === consts.messageNames.runnableStarted ? consts.runnableStatuses.running : consts.runnableStatuses.stopped)

                // Update class
                runnable.html.toggle.className = "toggle " + menu.toggleClass(runnable.status)
            }
        }
    },
    toggleClass: function(status) {
        return status === "running" ? "on" : "off"
    }
}