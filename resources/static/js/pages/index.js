let index = {
    // Attributes
    brains: {},
    
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

        // Append to pool
        index.brains[brain.name] = brain;
    },
    newBrain: function(data) {
        // Init
        let r = {
            abilities: {},
            name: data.name,
        };

        // Loop through abilities
        if (typeof data.abilities !== "undefined") {
            for (let k = 0; k < data.abilities.length; k++) {
                index.addAbility(r, data.abilities[k]);
            }
        }
        return r
    },
    removeBrain: function(data) {
        let brain = index.brains[data.name];
        if (typeof brain !== "undefined") {
            for (let k = 0; k < data.abilities.length; k++) {
                index.removeAbility(brain, data.abilities[k]);
            }
            delete(index.brains[data.name])
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

        // TODO Add in alphabetical order
        base.addInAlphabeticalOrder($("#index"), ability, brain.abilities);

        // Append to pool
        brain.abilities[ability.name] = ability;
    },
    newAbility: function(brain, data) {
        // Create results
        let r = {
            brain_name: brain.name,
            html: {},
            is_on: data.is_on,
            name: data.name,
            ui: data.ui,
        };

        // Create ui items
        let description = data.name;
        let homepage = "";
        let title = data.name;
        if (typeof r.ui !== "undefined") {
            if (r.ui.description !== "") description = r.ui.description;
            if (r.ui.homepage !== "") homepage = "<a href='" + r.ui.homepage + "' style='position: absolute; right: 0;'><i class='fa fa-cog'></i></a>";
            if (r.ui.title !== "") title = r.ui.title;
        }
        title += " (" + brain.name + ")";

        // Create wrapper
        r.html.wrapper = $(`<div class="panel"></div>`);

        // Create name
        let name = $(`<div class="title">` + title + homepage + `</div>`);
        name.appendTo(r.html.wrapper);

        // Create description
        let cell = $(`<div class="description">` + description + `</div>`);
        cell.appendTo(r.html.wrapper);
        return r;
    },
    removeAbility: function(brain, data) {
        let ability = brain.abilities[data.name];
        if (typeof ability !== "undefined") {
            ability.html.wrapper.remove();
            delete(brain.abilities[data.name])
        }
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