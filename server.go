package main

import(
	"io"
	"os"
	"log"
	"fmt"
	"path"
	"bytes"
	"regexp"
	"net/http"
	"io/ioutil"
	"archive/zip"
	"html/template"
	"encoding/json"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/fsouza/go-dockerclient"
)

type Server struct {}

type Container struct {
	Name string  `json:"name"`
	State string `json:"state"`
	Version string `json:"version"`
	Ports []string `json:"ports"`
	ForgeVersion string `json:"forgeVersion"`
	HostConfig string `json:"hostConfig"`
}

type FormContainer struct {
	Name string  `json:"name"`
	Version string `json:"version"`
	Port string `json:"port"`
	Difficulty int `json:"difficulty"`
	Seed string `json:"seed"`
	ForgeVersion string `json:"forgeVersion"`
}

type WorkspaceFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	IsDir bool `json:"isDir"`
}

type FormMakeDir struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Config struct {
	Accounts map[string]string				`json:"accounts"`
	Containers map[string]ConfigContainer	`json:"containers"`
	Versions []string						`json:"versions"`
}

type ConfigContainer struct {
	Version string `json:"version"`
	Port string `json:"port"`
	ForgeVersion string `json:"forgeVersion"`
	HostConfig string `json:"host_config"`
}

const maxUploadSize = 1 << 25;

func loadConfig() Config {
	b, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Printf(err.Error())
	}

	var config Config

	if b != nil {
		if err = json.Unmarshal(b, &config); err != nil {
			panic(err)
		}
	} else {
		// TODO: copy config.json.example
	}

	return config
}

func (cfg *Config) flush() error {
	cfgByt, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile("./config.json", cfgByt, 0655); err != nil {
		return err
	}
	return nil
}

func compressDir(writer *zip.Writer, zipPath, dirPath string) error {
	// open a directory
	directory, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer directory.Close()

	// get filenames
	files, err := directory.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, fname := range files {
		// ignore minecraft_xxx.jar
		if regexp.MustCompile(`^minecraft_.*\.jar$`).Match([]byte(fname)) {
			log.Printf("ignore file: ", fname)
			continue
		}

		filePath := path.Join(dirPath, fname)
		hostFile, err := os.Open(filePath)
		if err != nil {
			log.Printf("Can't open file: %s", filePath)
			continue
		}
		defer hostFile.Close()
		info, err := hostFile.Stat()
		if err != nil {
			log.Printf("Can't stat file: %s", filePath)
			continue
		}

		// create zip header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			log.Printf("Can't create header: %s", err.Error())
		}
		header.Name = path.Join(zipPath, fname)
		if info.IsDir() {
			header.Name += string(os.PathSeparator)
		}
		w, err := writer.CreateHeader(header)
		if err != nil {
			log.Printf(err.Error())
			continue
		}

		if info.IsDir() {
			// compress a child directory
			compressDir(writer, path.Join(zipPath, fname), path.Join(dirPath, fname))
		} else {
			// header.Method = zip.Deflate

			// archive a file
			if _, err := io.Copy(w, hostFile); err != nil {
				log.Printf("Can't copy file: %s", filePath)
				continue
			}
		}
	}
	return nil
}

func (server *Server) Help() string {
	return "crafty server"
}

func (server *Server) Synopsis() string {
	return "Start crafty"
}

func (server *Server) Run(args []string) int {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		log.Fatal("Err: %v", err)
	}

	config := loadConfig()

	// Set a mode for gin
	ginMode := os.Getenv("GIN_MODE")
	if ginMode != "" {
		gin.SetMode(ginMode)
	}

	r := gin.Default()
	r.Static("/assets", "./assets")

	// use authentication middleware
	if len(config.Accounts) > 0 {
		r.Use(gin.BasicAuth(config.Accounts))
	}

	// load templates
	html := template.Must(template.ParseFiles("views/index.tpl"))
	r.SetHTMLTemplate(html)

	// TOP
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tpl", gin.H{})
	})
	r.GET("/workspace/:name/file", func(c *gin.Context) {
		workspacePath, err := filepath.Abs(path.Join("workspace", c.Param("name")))
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		filePath, err := filepath.Abs(path.Join(workspacePath, c.Query("path")))
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		// TODO: directory check

		file, err := os.Open(filePath)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		defer file.Close()

		// get file information
		fileInfo, err := file.Stat()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		if fileInfo.IsDir() {
			// archive directory
			buf := new(bytes.Buffer)
			writer := zip.NewWriter(buf)
			if err := compressDir(writer, "", filePath); err != nil {
				log.Printf("Can't compress data: %s", err.Error())
			}

			// flush writer
			if err := writer.Flush(); err != nil {
				log.Printf("Can't flush data: %s" + err.Error())
			}
			if err := writer.Close(); err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			// send zip
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename='%s.zip'", filepath.Base(filePath)))
			c.Data(http.StatusOK, "application/zip", buf.Bytes())
		} else {
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}

			// send file
			contentType := http.DetectContentType(data)
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename='%s'", filepath.Base(filePath)))
			c.Data(http.StatusOK, contentType, data)
		}
	})

	// API
	apiRoute := r.Group("/api")
	{
		apiRoute.GET("/versions", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"values": config.Versions,
				"error": nil,
			})
		})
		containerRoute := apiRoute.Group("/containers")
		{
			containerRoute.GET("/", func(c *gin.Context) {
				var containers []Container

				for name, c := range config.Containers {
					container := Container{
						Name: name,
						HostConfig: c.HostConfig,
						Version: c.Version,
						ForgeVersion: c.ForgeVersion,
					}
					inspect, err := client.InspectContainer(container.Name)
					if err != nil {
						log.Printf(err.Error())
						continue
					}

					portBindings := inspect.NetworkSettings.Ports["25565/tcp"]
					if portBindings != nil && len(portBindings) > 0 {
						container.Ports = make([]string, len(portBindings))
						for i, portBinding := range portBindings {
							container.Ports[i] = portBinding.HostPort
						}
					}
					container.State = inspect.State.String()
					switch {
					case inspect.State.Running:
						container.State = "Running"
					case inspect.State.Paused:
						container.State = "Paused"
					case inspect.State.Restarting:
						container.State = "Restring..."
					default:
						container.State = "Stopped"
					}
					containers = append(containers, container)
				}

				if containers != nil {
					c.JSON(http.StatusOK, gin.H{
						"values": containers,
						"error": nil,
					})
				} else {
					c.JSON(http.StatusOK, gin.H{
						"values": []Container{},
						"error": nil,
					})
				}
			})
			containerRoute.POST("/", func(c *gin.Context) {
				// bind json
				var form FormContainer
				if c.BindJSON(&form) != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": "Invalid json",
					})
					return
				}

				// create directories
				workspacePath, err := filepath.Abs("workspace/" + form.Name)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}
				os.MkdirAll(workspacePath, 0755)

				// make options
				labels := map[string]string {"crafty": form.Version,}
				var environments []string
				environments = append(environments, "DIFFICULTY=" + string(form.Difficulty))
				if form.Version != "" {
					environments = append(environments, "VERSION=" + form.Version)
				}
				if form.ForgeVersion != "" {
					environments = append(environments, "FORGE_VERSION=" + form.ForgeVersion)
				}
				if form.Seed != "" {
					environments = append(environments, "SEED=" + form.Seed)
				}
				// remove
				environments = append(environments, "EULA=yes")
				dockerConfig := docker.Config{
					Image: "zak1ck/minecraft:java8",
					Labels: labels,
					Env: environments,
				}
				portBindings := map[docker.Port][]docker.PortBinding{
					"25565/tcp": {{HostIP: "0.0.0.0", HostPort: form.Port}},
				}
				binds := []string{workspacePath + ":/minecraft/data",}
				hostConfig := docker.HostConfig{
					PortBindings: portBindings,
					Privileged: false,
					PublishAllPorts: false,
					Binds: binds,
				}

				// persist a record
				hostConfigJson, err := json.Marshal(hostConfig)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}
				config.Containers[form.Name] = ConfigContainer{
					Version: form.Version,
					ForgeVersion: form.ForgeVersion,
					Port: form.Port,
					HostConfig: string(hostConfigJson),
				}
				config.flush()

				// make a container option
				opts := docker.CreateContainerOptions{
					Name: form.Name,
					Config: &dockerConfig,
					HostConfig: &hostConfig,
				}

				// create a container
				if _, err := client.CreateContainer(opts); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"value": true,
					"error": nil,
				})
			})
			containerRoute.GET("/:name", func(c *gin.Context) {
				container, err := client.InspectContainer(c.Param("name"))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"value": container,
					"error": nil,
				})
			})
			containerRoute.PUT("/:name/start", func(c *gin.Context) {
				// get host_config
				var hostConfigJson string
				if c, exists := config.Containers[c.Param("name")]; exists {
					hostConfigJson = c.HostConfig
				}

				// bind json
				var hostConfig docker.HostConfig
				if err := json.Unmarshal([]byte(hostConfigJson), &hostConfig); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				// start container
				if err := client.StartContainer(c.Param("name"), nil); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{ "success": true,})
			})
			containerRoute.PUT("/:name/stop", func(c *gin.Context) {
				// stop container
				if err := client.StopContainer(c.Param("name"), 10); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"value": true,
					"error": nil,
				})
			})
			containerRoute.PUT("/:name/restart", func(c *gin.Context) {
				// restart container
				if err := client.RestartContainer(c.Param("name"), 5); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": nil,
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"value": true,
					"error": nil,
				})
			})
			containerRoute.DELETE("/:name", func(c *gin.Context) {
				// make options
				opts := docker.RemoveContainerOptions{
					ID: c.Param("name"),
				}

				// remove container
				err := client.RemoveContainer(opts)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
					return
				}

				// drop record
				if _, exists := config.Containers[c.Param("name")]; exists {
					delete(config.Containers, c.Param("name"))
					config.flush()
				}

				workspacePath, err := filepath.Abs(path.Join("workspace", c.Param("name")))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				// remove directory
				if _, err := os.Stat(workspacePath); err == nil {
					if err := os.RemoveAll(workspacePath); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{
							"value": nil,
							"error": err.Error(),
						})
						return
					}
				}

				c.JSON(http.StatusOK, gin.H{
					"values": true,
					"error": nil,
				})
			})
			containerRoute.GET("/:name/workspace", func(c *gin.Context) {
				workspacePath, err := filepath.Abs(path.Join("workspace", c.Param("name")))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				// TODO: directory check

				// get file informations
				fileInfos, err := ioutil.ReadDir(path.Join(workspacePath, c.Query("path")))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				var files []WorkspaceFile
				for _, info := range fileInfos {
					if regexp.MustCompile(`^minecraft_.*\.jar$`).Match([]byte(info.Name())) {
						log.Printf("ignore file: ", info.Name())
						continue
					}

					filePath, err := filepath.Abs(path.Join(workspacePath, c.Query("path"), info.Name()))
					if err != nil {
						// continue, if can't get absolute path
						continue
					}
					relPath, err := filepath.Rel(workspacePath, filePath)
					if err != nil {
						// continue, if can't get relative path
						continue
					}

					files = append(files, WorkspaceFile{
						Name: info.Name(),
						Path: relPath,
						IsDir: info.IsDir(),
					})
				}

				if files != nil && len(files) > 0 {
					c.JSON(http.StatusOK, gin.H{
						"values": files,
						"error": nil,
					})
				} else {
					c.JSON(http.StatusOK, gin.H{
						"values": []WorkspaceFile{},
						"error": nil,
					})
				}
			})
			containerRoute.POST("/:name/workspace/mkdir", func(c *gin.Context) {
				workspacePath, err := filepath.Abs(path.Join("workspace", c.Param("name")))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				// bind json
				var form FormMakeDir
				if c.BindJSON(&form) != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": "Invalid json",
					})
					return
				}

				directoryPath, err := filepath.Abs(path.Join(workspacePath, form.Path, form.Name))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				// TODO: check directory

				// make dir
				if err := os.MkdirAll(directoryPath, 0755); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"value": form,
					"error": nil,
				})
			})
			containerRoute.POST("/:name/workspace/upload", func(c *gin.Context) {
				workspacePath, err := filepath.Abs(path.Join("workspace", c.Param("name")))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				// open a file for form
				c.Request.ParseMultipartForm(maxUploadSize)

				for _, fheaders := range c.Request.MultipartForm.File {
					for _, handler := range fheaders {
						file, err := handler.Open()
						if err != nil {
							log.Printf(err.Error())
							continue
						}
						defer file.Close()

						// get a file path
						filePath, err := filepath.Abs(path.Join(workspacePath, c.Request.FormValue("path"), handler.Filename))
						if err != nil {
							log.Printf(err.Error())
							continue
						}

						// TODO: check directory

						// store a file
						f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0755)
						if err != nil {
							log.Printf(err.Error())
							continue
						}
						defer f.Close()
						io.Copy(f, file)
					}
				}

				c.JSON(http.StatusOK, gin.H{
					"value": true,
					"error": nil,
				})
			})
			containerRoute.DELETE("/:name/workspace/file", func(c *gin.Context) {
				workspacePath, err := filepath.Abs(path.Join("workspace", c.Param("name")))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				filePath, err := filepath.Abs(path.Join(workspacePath, c.Query("path")))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				// TODO: directory check

				log.Printf(filePath)
				if err := os.RemoveAll(filePath); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"value": nil,
						"error": err.Error(),
					})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"value": true,
					"error": nil,
				})
			})
		}
	}


	port := os.Getenv("CRAFTY_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listen %s", port)
	r.Run(":" + port)

	return 0
}
