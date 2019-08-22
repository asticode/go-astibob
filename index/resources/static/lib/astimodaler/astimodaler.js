if (typeof asticode === "undefined") {
    var asticode = {};
}
asticode.modaler = {
    scriptDir: document.currentScript.src.match(/.*\//),
    close: function() {
        if (typeof asticode.modaler.onclose !== "undefined" && asticode.modaler.onclose !== null) {
            asticode.modaler.onclose();
        }
        asticode.modaler.hide();
    },
    hide: function() {
        document.getElementById("astimodaler").style.display = "none";
    },
    init: function() {
        document.body.innerHTML = `<div class="astimodaler" id="astimodaler">
            <div class="astimodaler-background"></div>
            <div class="astimodaler-table">
                <div class="astimodaler-wrapper">
                    <div id="astimodaler-body">
                        <img class="astimodaler-close" src="` + asticode.modaler.scriptDir + `/cross.png" onclick="asticode.modaler.close()"/>
                        <div id="astimodaler-content"></div>
                    </div>
                </div>
            </div>
        </div>` + document.body.innerHTML;
    },
    setContent: function(content) {
        document.getElementById("astimodaler-content").innerHTML = '';
        if (typeof content.node !== "undefined") content = content.node
        document.getElementById("astimodaler-content").appendChild(content);
    },
    setWidth: function(width) {
        document.getElementById("astimodaler-body").style.width = width;
    },
    show: function() {
        document.getElementById("astimodaler").style.display = "block";
    },
    newForm: function() {
        return {
            fields: [],
            node: document.createElement("div"),
            addTitle: function(text) {
                let t = document.createElement("div")
                t.className = "astimodaler-title"
                t.innerText = text
                this.node.appendChild(t)
            },
            addError: function() {
                let e = document.createElement("div")
                e.className = "astimodaler-error"
                this.node.appendChild(e)
                this.error = e
            },
            showError: function(text) {
                this.error.innerText = text
                this.error.style.display = "block"
            },
            hideError: function() {
                this.error.style.display = "none"
            },
            addField: function(options) {
                let that = this
                switch (options.type) {
                    case "submit":
                        // Create button
                        let b = document.createElement("div")
                        b.className = "astimodaler-field-submit" + (typeof options.className !== "undefined" ? " " + options.className : "")
                        b.innerText = options.label
                        this.node.appendChild(b)

                        // Handle click
                        b.addEventListener("click", function() {
                            // Hide error
                            that.hideError()

                            // Loop through fields
                            let fs = []
                            for (let i = 0; i < that.fields.length; i++) {
                                // Get field
                                const f = that.fields[i]
                                
                                // Get value
                                let v
                                switch (f.options.type) {
                                    case "email":
                                    case "text":
                                    case "textarea":
                                        v = f.node.value
                                        break
                                }

                                // Check required
                                if (typeof f.options.required !== "undefined" && f.options.required && v === "") {
                                    that.showError('Field "' + f.options.label + '" is required')
                                    return
                                }
                                
                                // Check email
                                if (f.options.type === "email" && !asticode.tools.isEmail(v)) {
                                    that.showError(v + " is not a valid email")
                                    return
                                }

                                // Append field
                                fs.push({
                                    name: f.options.name,
                                    value: v,
                                })
                            }

                            // Success callback
                            options.success(fs)

                        })
                        break
                    case "email":
                    case "text":
                    case "textarea":
                        // Create label
                        let l = document.createElement("label")
                        l.className = "astimodaler-label"
                        l.innerHTML = options.label + (typeof options.required !== "undefined" && options.required ? "<span class='astimodaler-required'>*</span>" : "")
                        this.node.appendChild(l)

                        // Create element
                        let i
                        switch (options.type) {
                            case "email":
                            case "text":
                                i = document.createElement("input")
                                i.className = "astimodaler-field-text"
                                i.type = "text"
                                break
                            case "textarea":
                                i = document.createElement("textarea")
                                i.className = "astimodaler-field-textarea"
                                break
                        }

                        // Append field
                        this.node.appendChild(i)
                        this.fields.push({
                            node: i,
                            options: options
                        })
                        break
                }
            },
        }
    }
};