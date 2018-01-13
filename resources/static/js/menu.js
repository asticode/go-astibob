let menu = {
    // Attributes
    brains: {},
    brainsCount: 0,

    // Init

    init: function(data) {
        // Loop through brains
        if (typeof data.brains !== "undefined") {
            for (let k = 0; k < data.brains.length; k++) {
                menu.addBrain(data.brains[k]);
            }
        }
    },

    // Brain

    addBrain: function(data) {
        // Brain already exists
        if (typeof menu.brains[data.name] !== "undefined") {
            return
        }

        // Create brain
        let brain = menu.newBrain(data);

        // Add in alphabetical order
        base.addInAlphabeticalOrder($("#menu"), brain, menu.brains);

        // Append to pool
        menu.brains[brain.name] = brain;

        // Update brains count
        menu.updateBrainsCount(1);
    },
    newBrain: function(data) {
        // Init
        let r = {
            abilities: {},
            html: {},
            name: data.name,
        };

        // Create wrapper
        r.html.wrapper = $(`<div class="menu-brain"></div>`);

        // Create name
        let name = $(`<div class="menu-brain-name">` + data.name + `</div>`);
        name.appendTo(r.html.wrapper);

        // Create table
        r.html.table = $(`<div class="table"></div>`);
        r.html.table.appendTo(r.html.wrapper);

        // Loop through abilities
        if (typeof data.abilities !== "undefined") {
            for (let k = 0; k < data.abilities.length; k++) {
                menu.addAbility(r, data.abilities[k]);
            }
        }
        return r
    },
    removeBrain: function(data) {
        // Fetch brain
        let brain = menu.brains[data.name];

        // Brain exists
        if (typeof brain !== "undefined") {
            // Remove HTML
            brain.html.wrapper.remove();

            // Delete from pool
            delete(menu.brains[data.name]);

            // Update brains count
            menu.updateBrainsCount(-1);
        }
    },
    updateBrainsCount: function(delta) {
        // Update brains count
        menu.brainsCount += delta;

        // Hide brain name
        let sel = $(".menu-brain-name");
        if (menu.brainsCount > 1) {
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
        let ability = menu.newAbility(brain, data);

        // Add in alphabetical order
        base.addInAlphabeticalOrder(brain.html.table, ability, brain.abilities);

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
        let title = data.name;
        if (typeof r.ui !== "undefined") {
            if (r.ui.title !== "") title = r.ui.title;
            if (r.ui.description !== "") description = r.ui.description;
        }

        // Create wrapper
        r.html.wrapper = $(`<div class="row" title="` + description + `"></div>`);

        // Create name
        let name = $(`<div class="cell" style="padding-right: 10px">` + title + `</div>`);
        name.appendTo(r.html.wrapper);

        // Create toggle cell
        let cell = $(`<div class="cell"></div>`);
        cell.appendTo(r.html.wrapper);

        // Create toggle
        let state = (data.is_on ? "on" : "off");
        r.html.toggle = $(`<label class="toggle ` + state + `">
            <span class="slider"></span>
        </label>`);
        r.html.toggle.click(function() {
            base.sendWs(r.is_on ? consts.websocket.eventNames.abilityStop : consts.websocket.eventNames.abilityStart, {
                brain_name: r.brain_name,
                name: r.name,
            });
        });
        r.html.toggle.appendTo(cell);
        return r;
    },
    updateToggle: function(data) {
        // Fetch brain
        let brain = menu.brains[data.brain_name];

        // Brain exists
        if (typeof brain !== "undefined") {
            // Fetch ability
            let ability = brain.abilities[data.name];

            // Ability exists
            if (typeof ability !== "undefined") {
                // Update class
                ability.html.toggle.removeClass(data.is_on ? "off" : "on");
                ability.html.toggle.addClass(data.is_on ? "on" : "off");

                // Update attribute
                ability.is_on = data.is_on;
            }
        }
    },
};