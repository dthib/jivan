///////////////////////////////////////////////////////////////////////////////
//
// The MIT License (MIT)
// Copyright (c) 2018 Jivan Amara
// Copyright (c) 2018 Tom Kralidis
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
// OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE
// USE OR OTHER DEALINGS IN THE SOFTWARE.
//
///////////////////////////////////////////////////////////////////////////////

// jivan project main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/go-spatial/jivan/config"
	"github.com/go-spatial/jivan/data_provider"
	"github.com/go-spatial/jivan/server"
	"github.com/go-spatial/jivan/util"
	"github.com/go-spatial/jivan/wfs3"
	"github.com/go-spatial/tegola/dict"
	tegola_provider "github.com/go-spatial/tegola/provider"
	"github.com/go-spatial/tegola/provider/gpkg"
	"github.com/go-spatial/tegola/provider/postgis"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	var bindIp string
	var bindPort int
	var serveAddress string
	var dataSource string
	var configFile string
	var err error

	flag.StringVar(&bindIp, "b", "", "IP address for the server to listen on")
	flag.IntVar(&bindPort, "p", 8080, "port for the server to listen on")
	flag.StringVar(&serveAddress, "s", "", "IP:Port that result urls will be constructed with (defaults to the IP:Port used in request)")
	flag.StringVar(&dataSource, "d", "", "data source (path to .gpkg file or connection string to PostGIS database i.e 'user={user} password={password} dbname={dbname} host={host} port={port}')")
	flag.StringVar(&configFile, "c", "", "config (path to .toml file)")

	flag.Parse()

	// Configuration logic
	// 1. config.Configuration gets set at startup (via config.init())
	// 2. if -c is passed, config file overrides
	// 3. if other command line arguments are passed, they override previous settings
	// 4. If no data provider is supplied by any of these means, the working directory
	//    is scanned for .gpkg files, then the 'data/' and 'test_data/' directories.

	if configFile != "" { // load config from command line
		config.Configuration, err = config.LoadConfigFromFile(configFile)
		if err != nil {
			panic(err)
		}
	}

	err = envconfig.Process("JIVAN", &config.Configuration.Server)
	if err != nil {
		log.Printf("Error while reading environment configs: %s", err.Error())
	}

	config.Configuration.Server.BindHost = bindIp
	config.Configuration.Server.BindPort = bindPort

	if serveAddress != "" {
		config.Configuration.Server.URLHostPort = serveAddress
	}

	// If not set in command line, use configuration file
	if dataSource == "" {
		dataSource = config.Configuration.Providers.Data
	}

	if !path.IsAbs(dataSource) {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		dataSource = path.Join(wd, dataSource)
	}

	var autoconfig func(ds string) (map[string]interface{}, error)
	var ntp func(config dict.Dicter) (tegola_provider.Tiler, error)
	if dataSource != "" {
		// Is this a PostGIS conn string or GeoPackage path?
		if _, err := os.Stat(dataSource); os.IsNotExist(err) {
			// TODO: to be fixed when this pull request is merged https://github.com/go-spatial/tegola/pull/412
			//autoconfig = postgis.AutoConfig
			ntp = postgis.NewTileProvider
		} else {
			autoconfig = gpkg.AutoConfig
			ntp = gpkg.NewTileProvider
		}
	}
	if dataSource == "" {
		dataSource = util.DefaultGpkg()
		autoconfig = gpkg.AutoConfig
		ntp = gpkg.NewTileProvider
	}
	if dataSource == "" {
		panic("no datasource")
	}
	config.Configuration.Providers.Data = dataSource

	dataConfig, err := autoconfig(dataSource)
	if err != nil {
		panic(fmt.Sprintf("data provider auto-config failure for '%v': %v", dataSource, err))
	}

	dataProvider, err := ntp(dict.Dicter(dict.Dict(dataConfig)))
	if err != nil {
		panic(fmt.Sprintf("data provider creation error for '%v': %v", dataSource, err))
	}

	p := data_provider.Provider{Tiler: dataProvider}
	wfs3.GenerateOpenAPIDocument()

	server.StartServer(p)
}
