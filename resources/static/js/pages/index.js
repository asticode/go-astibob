let index = {
    // Attributes
    brains: {},
    brainsCount: 0,
    
    // Init
    
    init: function () {
        base.init(index.websocketFunc, function(data) {
            // Loop through brains
            if (typeof data.brains !== "undefined") {
                for (let k = 0; k < data.brains.length; k++) {
                    index.addBrain(data.brains[k]);
                }
            }

            // Finish
            base.finish();
        });
    },

    // Brain

    addBrain: function(data) {
        // Brain already exists
        if (typeof index.brains[data.name] !== "undefined") {
            return
        }

        // Create brain
        let brain = index.newBrain(data);

        // Add in alphabetical order
        base.addInAlphabeticalOrder($("#content"), brain, index.brains);

        // Append to pool
        index.brains[brain.name] = brain;

        // Update brains count
        index.updateBrainsCount(1);
    },
    newBrain: function(data) {
        // Init
        let r = {
            abilities: {},
            html: {},
            name: data.name,
        };

        // Create wrapper
        r.html.wrapper = $(`<div class="index-brain"></div>`);

        // Create name
        let name = $(`<div class="index-brain-name header">` + data.name + `</div>`);
        name.appendTo(r.html.wrapper);

        // Create flex
        r.html.flex = $(`<div class="flex"></div>`);
        r.html.flex.appendTo(r.html.wrapper);

        // Loop through abilities
        if (typeof data.abilities !== "undefined") {
            for (let k = 0; k < data.abilities.length; k++) {
                index.addAbility(r, data.abilities[k]);
            }
        }
        return r
    },
    removeBrain: function(data) {
        // Fetch brain
        let brain = index.brains[data.name];

        // Brain exists
        if (typeof brain !== "undefined") {
            // Remove HTML
            brain.html.wrapper.remove();

            // Remove from pool
            delete(index.brains[data.name]);

            // Update brains count
            index.updateBrainsCount(-1);
        }
    },
    updateBrainsCount: function(delta) {
        // Update brains count
        index.brainsCount += delta;

        // Hide brain name
        let sel = $(".index-brain-name");
        if (index.brainsCount > 1) {
            sel.show();
        } else {
            sel.hide();
        }
    },

    // Ability

    addAbility: function(brain, data) {
        // Ability already exists
        if (typeof brain.abilities[data.name] !== "undefined") {
            return
        }

        // Create ability
        let ability = index.newAbility(brain, data);

        // Add in alphabetical order
        base.addInAlphabeticalOrder(brain.html.flex, ability, brain.abilities);

        // Append to pool
        brain.abilities[ability.name] = ability;
    },
    newAbility: function(brain, data) {
        // Create results
        let r = {
            brain_name: brain.name,
            description: data.description,
            html: {},
            is_on: data.is_on,
            name: data.name,
            web_homepage: data.web_homepage,
        };

        // Create wrapper
        r.html.wrapper = $(`<div class="panel-wrapper"></div>`);

        // Create link
        let wrapper = r.html.wrapper;
        if (typeof r.web_homepage !== "undefined") {
            wrapper = $(`<a href="` + r.web_homepage + `"></a>`);
            wrapper.appendTo(r.html.wrapper);
        }

        // Create panel
        let panel = $(`<div class="panel"></div>`);
        panel.appendTo(wrapper);

        // Create name
        let name = $(`<div class="title">` + r.name + `</div>`);
        name.appendTo(panel);

        // Create description
        let cell = $(`<div class="description">` + r.description + `</div>`);
        cell.appendTo(panel);
        return r;
    },

    // Websocket

    websocketFunc: function(event_name, payload) {
        switch (event_name) {
            case consts.websocket.eventNames.brainDisconnected:
                index.removeBrain(payload);
                break;
            case consts.websocket.eventNames.brainRegistered:
                index.addBrain(payload);
                break;
        }
    }
};