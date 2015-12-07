"use strict";

(function() {
    // Model
    var docker = {};
    docker.Container = function(data) {
      this.name = m.prop(data.name || null);
      this.state = m.prop(data.state);
      this.ports = m.prop(data.ports || []);
      this.version = m.prop(data.version);
      this.forgeVersion= m.prop(data.forgeVersion || "");
    };
    docker.FormContainer = function(data) {
      this.id = m.prop(data.id || null); // 多分これいらない
      this.name = m.prop(data.name || null);
      this.port = m.prop(data.port || null);
      this.version = m.prop(data.version || "1.9");
      this.forgeVersion= m.prop(data.forgeVersion || "");
      this.seed = m.prop(data.seed || "");
      this.difficulty = m.prop(data.difficulty || 1);
      this.hostConfig = m.prop(data.hostConfig || "");
    };
    // Component
    docker.ModalComponent = function() {
      return {
        controller: function(args) {
          return args;
        },
        view: function(ctrl) {
          return m("div", {class: "modal-container pure-form-aligned"}, [
            m("div", {class: "pure-g"}, [
              m("div", {class: "pure-u-1-3"}),
              m("div", {class: "pure-u-1-3"}, [
                m("div", {class: "modal"}, [
                  m("form", {class: "pure-form"}, [
                    m("fieldset", ctrl.fields),
                    m("fieldset", [
                      m('button', {class: "pure-button cancel", onclick: ctrl.cancel}, "Cancel"),
                      m('button', {class: "pure-button submit", onclick: ctrl.submit}, ctrl.message)
                    ])
                  ])
                ])
              ])
            ])
          ]);
        }
      };
    };
    // View-Model
    docker.index = (function() {
      var index = {form: {}};
      index.init = function() {
        index.initForm();
        index.initContainers();
      };
      index.initContainers = function() {
        index.list = m.prop([]);
        m.request({
          method: "GET",
          url: "/api/containers/",
          type: docker.Container,
          unwrapSuccess: function(response) { return response.values; }
        }).then(index.list);
      };
      index.initForm = function() {
        index.form.container = null;
        index.form.versions = m.prop([]);
        index.form.errors = [];
        index.form.difficulties = m.prop([
          [0, "Peaceful"],
          [1, "Easy"],
          [2, "Normal"],
          [3, "Hard"]
        ]);
        m.request({
          method: "GET",
          url: "/api/versions",
          unwrapSuccess: function(response) { return response.values; }
        }).then(index.form.versions);
      };
      index.controller = function() {
        if (m.route() === "/") {
          index.init();
        }
        var redraw = function() {
          if (m.route() == "/") {
            index.initContainers();
            setTimeout(redraw, 5000);
          }
        };
        setTimeout(redraw, 5000);
      };
      index.start = function(container, evt) {
        evt.preventDefault();
        evt.stopPropagation();

        m.request({
          method: "PUT",
          url: "/api/containers/" + container.name() + "/start",
          background: true
        });
      };
      index.stop = function(container, evt) {
        evt.preventDefault();
        evt.stopPropagation();

        m.request({
          method: "PUT",
          url: "/api/containers/" + container.name() + "/stop",
          background: true
        });
      };
      index.restart = function(container, evt) {
        evt.preventDefault();
        evt.stopPropagation();

        m.request({
          method: "PUT",
          url: "/api/containers/" + container.name() + "/restart",
          background: true
        });
      };
      index.remove = function(container, evt) {
        evt.preventDefault();
        evt.stopPropagation();

        m.request({
          method: "DELETE",
          url: "/api/containers/" + container.name(),
          background: true
        });
      };
      index.download = function(container, evt) {
        evt.preventDefault();
        evt.stopPropagation();

        window.open("/workspace/" + container.name() + "/file");
      };
      index.workspace = function(container, evt) {
        evt.preventDefault();
        evt.stopPropagation();

        m.route("/workspace/" + container.name());
      }
      index.openContainerModal = function(container, evt) {
        evt.preventDefault();
        evt.stopPropagation();

        index.form.container = container;
      };
      index.closeContainerModal = function(evt) {
        evt.preventDefault();
        evt.stopPropagation();

        index.form.container = null;
      };

      // form
      index.form.create = function(evt) {
        evt.preventDefault();
        evt.stopPropagation();

        var container = index.form.container;
        index.form.errors = [];
        if (!container.name() || container.name().length < 6) {
          index.form.errors.push("Invalid server name");
        }
        if (!container.version()) {
          index.form.errors.push("Invalid version");
        }
        if (container.port() && (isNaN(container.port()) || Number(container.port()) < 10000 || 40000 < Number(container.port()))) {
          index.form.errors.push("Invalid port");
        }
        if (container.forgeVersion()) {
          // TODO: If forgeVersion is not empty, check exists on forge server.
        }
        if (isNaN(container.difficulty()) || Number(container.difficulty()) < 0 || 3 < Number(container.difficulty())) {
          index.form.errors.push("Invalid difficulty");
        }
        if (!index.form.errors.length) {
          m.request({
            method: "POST",
            url: "/api/containers/",
            data: {
              name: container.name(),
              version: container.version(),
              port: container.port(),
              forgeVersion: container.forgeVersion(),
              seed: container.seed(),
              difficulty: Number(container.difficulty())
            },
            unwrapSuccess: function(response) { return response.value; }
          }).then(function(container) {
            if (container != null) {
              m.route("/");
            }
          }, function(response) {
            index.form.errors = [response.error];
          });
        }
      };
      index.view = function() {
        return m("div", {class: "pure-g"}, [
          (function() {
            var container = index.form.container;
            if (container) {
              if (container.id()) {
                // TODO: edit
              } else {
                return m.component(new docker.ModalComponent, {
                  fields: [
                    (function() {
                      if (index.form.errors.length) {
                        return m('div', {class: "pure-control-group errors"}, [
                          index.form.errors.map(function(msg, _) {
                            return m("li", {innerHTML: msg});
                          })
                        ]);
                      }
                    }()),
                    m('div', {class: "pure-control-group"}, [
                      m('label[for="name"]', {}, "Server Name"),
                      m('input[placeholder="Server Name"]', {
                        name: "name",
                        onchange: m.withAttr("value", index.form.container.name),
                        onkeyup: m.withAttr("value", index.form.container.name),
                        value: index.form.container.name()
                      }),
                    ]),
                    m('div', {class: "pure-control-group"}, [
                      m('label[for="version"]', {}, "Version"),
                      m("select", {
                        name: "version",
                        onchange: m.withAttr("value", index.form.container.version),
                        value: index.form.container.version()
                      }, [
                        index.form.versions().map(function(version, _) {
                          return m("option", {value: version, innerHTML: version});
                        })
                      ]),
                    ]),
                    m('div', {class: "pure-control-group"}, [
                      m('label[for="port"]', {}, "Port"),
                      m('input[placeholder="ex) 25565"][type="number"]', {
                        name: "port",
                        onchange: m.withAttr("value", index.form.container.port),
                        onkeyup: m.withAttr("value", index.form.container.port),
                        value: index.form.container.port()
                      }),
                    ]),
                    m('div', {class: "pure-control-group"}, [
                      m('label[for="seed"]', {}, "Seed"),
                      m('input[placeholder="ex) 12345678seed"]', {
                        name: "seed",
                        onchange: m.withAttr("value", index.form.container.seed),
                        onkeyup: m.withAttr("value", index.form.container.seed),
                        value: index.form.container.seed()
                      })
                    ]),
                    m('div', {class: "pure-control-group"}, [
                      m('label[for="difficulty"]', {}, "Difficulty"),
                      m("select", {
                        name: "difficulty",
                        onchange: m.withAttr("value", index.form.container.difficulty),
                        value: index.form.container.difficulty()
                      }, [
                        index.form.difficulties().map(function(difficulty, _) {
                          return m("option", {value: difficulty[0], innerHTML: difficulty[1]});
                        })
                      ]),
                    ]),
                    m('div', {class: "pure-control-group"}, [
                      m('label[for="forgeVersion"]', {}, "Forge Version"),
                      m('input[placeholder="ex) 11.15.1.1764"]', {
                        name: "forgeVersion",
                        onchange: m.withAttr("value", index.form.container.forgeVersion),
                        onkeyup: m.withAttr("value", index.form.container.forgeVersion),
                        value: index.form.container.forgeVersion()
                      })
                    ]),
                  ],
                  message: "Create",
                  cancel: index.closeContainerModal,
                  submit: index.form.create
                });
              }
            } else {
              return [];
            }
          }()),
          m("div", {class: "pure-u-1"}, [
            m("button", {class: "pure-button", onclick: index.openContainerModal.bind(index, new docker.FormContainer({}))}, "New"),
          ]),
          m("div", {class: "pure-u-1"}, [
            m("table", {class: "pure-table pure-table-horizontal container-table"}, [
              m("thead", [
                m("tr", [
                  m("th", "Name"),
                  m("th", "State"),
                  m("th", "Version"),
                  m("th", "Forge Version"),
                  m("th", "Port"),
                  m("th", "")
                ])
              ]),
              m("tbody", [
                docker.index.list().map(function(container, i) {
                  return m("tr", [
                    m("td", [
                      m("a", {onclick: index.openContainerModal.bind(index, container)}, container.name())
                    ]),
                    m("td", container.state()),
                    m("td", container.version()),
                    m("td", container.forgeVersion()),
                    m("td", container.ports()),
                    m("td", {class: "actions"}, [
                      m("div", {class: "pure-menu pure-menu-horizontal"}, [
                        m("ul", {class: "pure-menu-list"}, [
                          m("li", {class: "pure-menu-item pure-menu-has-children pure-menu-allow-hover"}, [
                            m("a", {class: "pure-menu-link dropdown", href: "#"}, "Actions"),
                            m("ul", {class: "pure-menu-children"}, [
                              m('li', {class: "pure-menu-item"},[m("a", {class: "pure-menu-link", href: "#", onclick: index.start.bind(index, container)}, "Start")]),
                              m('li', {class: "pure-menu-item"},[m("a", {class: "pure-menu-link", href: "#", onclick: index.stop.bind(index, container)}, "Stop")]),
                              m('li', {class: "pure-menu-item"},[m("a", {class: "pure-menu-link", href: "#", onclick: index.restart.bind(index, container)}, "Restart")]),
                              m('li', {class: "pure-menu-item"},[m("a", {class: "pure-menu-link", href: "#", onclick: index.remove.bind(index, container)}, "Remove")]),
                              m('li', {class: "pure-menu-item"},[m("a", {class: "pure-menu-link", href: "#", onclick: index.workspace.bind(index, container)}, "Open")]),
                              m('li', {class: "pure-menu-item"},[m("a", {class: "pure-menu-link", href: "#", onclick: index.download.bind(index, container)}, "Download")])
                            ])
                          ])
                        ])
                      ])
                    ])
                  ]);
                })
              ])
            ])
          ])
        ]);
      };
      return index;
    }());
    docker.workspace = (function() {
      var workspace = {};
      workspace.init = function() {
        workspace.name = m.prop(m.route.param("name"));
        workspace.path = m.prop(m.route.param("path") || "");
        workspace.files = m.prop([]);
        workspace.makeDirModal = m.prop(false);
        workspace.dirName = m.prop("");
        workspace.uploadModal = m.prop(false);
        workspace.uploadFiles = m.prop([]);
        workspace.contextMenu = m.prop(null);
        m.request({
          method: "GET",
          url: "/api/containers/" + workspace.name() + "/workspace?path=" + workspace.path(),
          unwrapSuccess: function(response) { return response.values; }
        }).then(workspace.files);
      };
      workspace.controller = function() {
        if (/^\/workspace/.exec(m.route())) {
          workspace.init();
        }
      };
      workspace.dropFiles = function(evt) {
        evt.preventDefault();
        workspace.selectFiles(evt);
        workspace.upload();
      };
      workspace.click = function(file) {
        if (file.isDir) {
          m.route("/workspace/" + m.route.param("name"), { path: file.path });
        } else {
          window.open("/workspace/" + workspace.name() + "/file?path=" + file.path);
        }
      };
      workspace.upOneLevel = function() {
        var ary = workspace.path().split("/");
        ary.pop();
        m.route("/workspace/" + m.route.param("name"), { path: ary.join("/") });
      };
      workspace.home = function() {
        m.route("/");
      };
      workspace.openUploadModal = function() {
        workspace.uploadModal(true);
      };
      workspace.closeUploadModal = function() {
        workspace.uploadModal(false);
      };
      workspace.selectFiles = function(evt) {
        workspace.uploadFiles((evt.dataTransfer || evt.target).files);
      };
      workspace.upload = function() {
        workspace.errors = [];
        var files = workspace.uploadFiles();
        if (!files && files.length < 1) {
          workspace.errors.push("Invalid file");
        }
        if (!workspace.errors.length) {
          var formData = new FormData;
          for (var i = 0; i < files.length; i++) {
            formData.append("file" + i, files[i]);
          }
          formData.append("path", workspace.path());
          m.request({
            method: "POST",
            url: "/api/containers/" + workspace.name() + "/workspace/upload",
            data: formData,
            serialize: function(data) {return data;},
            unwrapSuccess: function(response) { return response.value; }
          }).then(function() {
            m.route("/workspace/" + m.route.param("name"), { path: workspace.path() });
          });
        }
      };
      workspace.openMakeDirModal = function() {
        workspace.makeDirModal(true);
      };
      workspace.closeMakeDirModal = function() {
        workspace.makeDirModal(false);
      };
      workspace.makeDir = function() {
        workspace.errors = [];
        if (!workspace.dirName()) {
          workspace.errors.push("Invalid directory name");
        }
        if (!workspace.errors.length) {
          m.request({
            method: "POST",
            url: "/api/containers/" + workspace.name() + "/workspace/mkdir",
            data: {name: workspace.dirName(), path: workspace.path()},
            unwrapSuccess: function(response) { return response.value; }
          }).then(function() {
            m.route("/workspace/" + m.route.param("name"), { path: workspace.path() });
          });
        }
      };
      workspace.openContextMenu = function(file, evt) {
        evt.preventDefault();
        evt.stopPropagation();
        workspace.contextMenu({file: file, x: evt.clientX, y: evt.clientY});
      };
      workspace.closeContextMenu = function(evt) {
        workspace.contextMenu(null);
      };
      workspace.ContextMenuComponent = function() {
        var component = {};
        component.delete = function(path, evt) {
          evt.preventDefault();
          evt.stopPropagation();
          workspace.closeContextMenu();
          m.request({
            method: "DELETE",
            url: "/api/containers/" + workspace.name() + "/workspace/file?path=" + encodeURIComponent(path),
            unwrapSuccess: function(response) { return response.value; }
          }).then(function() {
            m.route("/workspace/" + m.route.param("name"), { path: workspace.path() });
          });
          // handle result
        };
        component.controller = function(args) {
          return args;
        };
        component.view = function(ctrl) {
          return m("div", {class: "pure-menu workspace-context-menu", style: "top:" + ctrl.y + "px; left:" + ctrl.x + "px;"}, [
            m("ul", {class: "pure-menu-list"}, [
              m("li", {class: "pure-menu-item"}, [
                m("a", {href: "/workspace/" + workspace.name() + "/file?path=" + ctrl.file.path, class: "pure-menu-link"}, "Download")
              ]),
              m("li", {class: "pure-menu-item"}, [
                m("a", {href: "#", class: "pure-menu-link", onclick: component.delete.bind(component, ctrl.file.path)}, "Delete")
              ])
            ])
          ]);
        };
        return component;
      };
      workspace.FileComponent = function() {
        return {
          controller: function(args) {
            return args;
          },
          view: function(ctrl) {
            return m("div", {class: "pure-u-1-4 pure-u-md-1-6 pure-u-lg-1-12"}, [
              m("div", {class: "file" + (ctrl.disabled ? " disabled" : ""), onclick: ctrl.click, oncontextmenu: ctrl.contextMenu}, [
                (function() {
                  if (ctrl.disabled) {
                    return m("div", {class: "overlay"})
                  } else {
                    return [];
                  }
                }()),
                m("img", {class: "ico", src: ctrl.icon}),
                m("span", ctrl.message),
              ])
            ]);
          }
        };
      };
      workspace.view = function() {
        return m("div", {
          class: "workspace",
          onclick: workspace.closeContextMenu, oncontextmenu: workspace.closeContextMenu,
          ondrop: workspace.dropFiles, ondragover: function() { return false; }, ondragenter: function() { return false; }
        }, [
          (function() {
            if (workspace.makeDirModal()) {
              return m.component(new docker.ModalComponent, {
                fields: [
                  m('input[placeholder="Directory Name"]',  {
                    onchange: m.withAttr("value", workspace.dirName), value: workspace.dirName()
                  }),
                ],
                message: "Create",
                cancel: workspace.closeMakeDirModal,
                submit: workspace.makeDir
              });
            } else {
              return [];
            }
          }()),
          (function() {
            if (workspace.uploadModal()) {
              return m.component(new docker.ModalComponent, {
                fields: [
                  m('input[type="file"][multiple]', {onchange: workspace.selectFiles}),
                ],
                message: "Upload",
                cancel: workspace.closeUploadModal,
                submit: workspace.upload
              });
            } else {
              return [];
            }
          }()),
          m("div", {class: "pure-g menu"}, [
            m.component(new workspace.FileComponent, {icon: "/assets/img/home.png", click: workspace.home, message: "Home"}),
            (function() {
              if (workspace.path() == "") {
                return m.component(new workspace.FileComponent, {
                  icon: "/assets/img/up-one-level.png", click: workspace.upOneLevel, message: "..",
                  disabled: true
                });
              } else {
                return m.component(new workspace.FileComponent, {
                  icon: "/assets/img/up-one-level.png", click: workspace.upOneLevel, message: ".."
                });
              }
            }()),
            m.component(new workspace.FileComponent, {icon: "/assets/img/make_dir.png", click: workspace.openMakeDirModal, message: "Create Folder"}),
            m.component(new workspace.FileComponent, {icon: "/assets/img/upload.png", click: workspace.openUploadModal, message: "Upload"})
          ]),
          m("div", {class: "pure-g"}, [
            workspace.files().map(function(file, i) {
              return m.component(new workspace.FileComponent, {
                icon: "/assets/img/" + (file.isDir ? "folder.png" : "file.png"),
                click: workspace.click.bind(workspace, file), message: file.name,
                contextMenu: workspace.openContextMenu.bind(workspace, file)
              });
            })
          ]),
          (function() {
            if (workspace.contextMenu()) {
              return m.component(new workspace.ContextMenuComponent, workspace.contextMenu());
            } else {
              return [];
            }
          }())
        ]);
      };
      return workspace;
    }());

    m.route.mode = "hash";
    m.route(document.body, "/", {
      "/workspace/:name": docker.workspace,
      "/new": docker.form,
      "/": docker.index,
    });
}());
