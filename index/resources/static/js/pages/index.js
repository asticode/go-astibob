let index = {
    // Attributes
    workers: {},
    workersCount: 0,

    // Init

    init: function () {
        base.init(index.onMessage, function(data) {
            // Loop through workers
            if (typeof data.workers !== "undefined") {
                for (let k = 0; k < data.workers.length; k++) {
                    index.addWorker(data.workers[k])
                }
            }

            // Finish
            base.finish()
        })
    },

    // Websocket

    onMessage: function(data) {
        switch (data.name) {
            case consts.messageNames.eventWorkerDisconnected:
                index.removeWorker(data.payload)
                break
            case consts.messageNames.eventWorkerRegistered:
                index.addWorker(data.payload)
                break
        }
    },

    // Worker

    addWorker: function(data) {
        // Worker already exists
        if (typeof index.workers[data.name] !== "undefined") {
            return
        }

        // Create worker
        let worker = index.newWorker(data)

        // Add in alphabetical order
        asticode.tools.appendSorted(document.querySelector("#content"), worker, index.workers)

        // Append to pool
        index.workers[worker.name] = worker

        // Update workers count
        index.updateWorkersCount(1)
    },
    newWorker: function(data) {
        // Init
        let r = {
            abilities: {},
            html: {},
            name: data.name,
        }

        // Create wrapper
        r.html.wrapper = document.createElement("div")
        r.html.wrapper.className = "index-worker"

        // Create name
        let name = document.createElement("div")
        name.className = "index-worker-name header"
        name.innerText = data.name
        r.html.wrapper.appendChild(name)

        // Create flex
        r.html.flex = document.createElement("div")
        r.html.flex.className = "flex"
        r.html.wrapper.appendChild(r.html.flex)

        // Loop through abilities
        if (typeof data.abilities !== "undefined") {
            for (let k = 0; k < data.abilities.length; k++) {
                index.addAbility(r, data.abilities[k])
            }
        }
        return r
    },
    removeWorker: function(name) {
        // Fetch worker
        let worker = index.workers[name]

        // Worker exists
        if (typeof worker !== "undefined") {
            // Remove HTML
            worker.html.wrapper.remove()

            // Remove from pool
            delete(index.workers[name])

            // Update workers count
            index.updateWorkersCount(-1)
        }
    },
    updateWorkersCount: function(delta) {
        // Update workers count
        index.workersCount += delta

        // Hide worker name
        let items = document.querySelectorAll(".index-worker-name")
        if (index.workersCount > 1) {
            items.forEach(function(item) { item.style.display = "block" })
        } else {
            items.forEach(function(item) { item.style.display = "none" })
        }
    },

    // Ability

    addAbility: function(worker, data) {
        // Ability already exists
        if (typeof worker.abilities[data.name] !== "undefined") {
            return
        }

        // Create ability
        let ability = index.newAbility(worker, data)

        // Add in alphabetical order
        asticode.tools.appendSorted(worker.html.flex, ability, worker.abilities)

        // Append to pool
        worker.abilities[ability.name] = ability
    },
    newAbility: function(worker, data) {
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
        r.html.wrapper.className = "panel-wrapper"

        // Create link
        let wrapper = r.html.wrapper
        if (typeof r.web_homepage !== "undefined") {
            wrapper = document.createElement("a")
            wrapper.href = r.web_homepage
            r.html.wrapper.appendChild(wrapper)
        }

        // Create panel
        let panel = document.createElement("div")
        panel.className = "panel"
        wrapper.appendChild(panel)

        // Create name
        let name = document.createElement("div")
        name.className = "title"
        name.innerText = r.name
        panel.appendChild(name)

        // Create description
        let cell = document.createElement("div")
        cell.className = "description"
        cell.innerText = r.description
        panel.appendChild(cell)
        return r
    },
};