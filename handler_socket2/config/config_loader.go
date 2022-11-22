package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var cfg_debug = false
var cfg_verbose = false

func CfgIsDebug() bool {
	return cfg_debug
}

func CfgIsVerbose() bool {
	return cfg_verbose
}

const DEFAULT_COMPRESSION_THRESHOLD = 4096

type cfg struct {
	config           map[string]string
	local_interfaces []string

	compression_threshold int

	cfg_file_path     string
	cfg_file_size     int64
	cfg_file_modified int64

	raw_data map[string]interface{}
}

// Load config from json file
func _cfg_load_config() (*cfg, error) {
	ret := cfg{}
	data := []byte(nil)

	// Load config
	{
		conf_path := "conf.json"
		if len(os.Args) >= 2 {
			conf_path = os.Args[1]
		}
		if strings.Index(conf_path, "/") == -1 {
			if path, err := os.Readlink("/proc/self/exe"); err == nil {
				path = filepath.Dir(path)
				conf_path = path + "/" + conf_path
			} else {
				fmt.Println("Can't find executable directory, using current dir for config!")
			}
		}

		fmt.Println("Reading configuration: " + conf_path)
		data_tmp, err := os.ReadFile(conf_path)
		if err != nil {
			return nil, err
		}
		data = data_tmp

		st, _ := os.Stat(conf_path)
		ret.cfg_file_path = conf_path
		ret.cfg_file_size = st.Size()
		ret.cfg_file_modified = st.ModTime().Unix()
	}

	var cfg_tmp map[string]interface{}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	d.Decode(&cfg_tmp)

	ret.config = make(map[string]string)
	for k, v := range cfg_tmp {

		switch v.(type) {
		case string:
			ret.config[k] = v.(string)
		case int:
			ret.config[k] = strconv.Itoa(v.(int))
		case float64:
			ret.config[k] = strconv.FormatFloat(v.(float64), 'f', 3, 64)
		case json.Number:
			ret.config[k] = v.(json.Number).String()
		case bool:
			if v.(bool) {
				ret.config[k] = "1"
			} else {
				ret.config[k] = "0"
			}
		}
	}

	ret.compression_threshold = DEFAULT_COMPRESSION_THRESHOLD
	if _ct, exists := ret.config["COMPRESSION_THRESHOLD"]; exists {
		ret.compression_threshold, _ = strconv.Atoi(_ct)
	}

	cfg_debug = ret.config["DEBUG"] == "1"
	cfg_verbose = ret.config["VERBOSE"] == "1"

	fmt.Println("Config: ", ret.config)
	ret.raw_data = cfg_tmp
	return &ret, nil
}

// Add local interfaces to config
func (this *cfg) _cfg_local_add(interfaces string) {

	if len(interfaces) == 0 {
		return
	}
	add := func(iface string) {
		if len(iface) == 0 {
			return
		}
		should_add := true
		for _, v := range this.local_interfaces {
			if strings.Compare(v, iface) != 0 {
				continue
			}
			should_add = false
			break
		}
		if should_add {
			this.local_interfaces = append(this.local_interfaces, iface)
		}
	}

	for _, v := range strings.Split(interfaces, ",") {
		v = strings.Trim(v, "\r\n\t uh")
		if len(v) == 0 {
			continue
		}
		v = strings.Split(v, ":")[0]
		if len(v) == 0 {
			continue
		}
		add(v)
	}
}

// Get all local interfaces
func (this *cfg) _cfg_local_interfaces() {

	getMatchInterfaces := func() []string {
		match_ifaces := make([]string, 0)
		ifaces, err := net.Interfaces()
		if err != nil {
			fmt.Println("Cannot read interfaces (0x2) ", err.Error())
			os.Exit(2)
		}

		for _, iface := range ifaces {

			addrs, err := iface.Addrs()
			if err != nil {
				fmt.Println("Cannot read interfaces (0x3) ", err.Error())
				os.Exit(3)
			}

			for _, addr := range addrs {

				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				default:
					continue
				}

				if ip.To4() != nil {
					// ipv4 processing
					pieces := strings.Split(ip.String(), ".")
					if cfg_verbose {
						fmt.Print("Interface V4 ", iface.Name, " ... ")
					}

					for i := 0; i < len(pieces); i++ {
						_m := strings.Join(pieces[i:], ".")
						match_ifaces = append(match_ifaces, _m)
						if cfg_verbose {
							fmt.Print(_m, " ")
						}
					}
				} else {
					if cfg_verbose {
						fmt.Print("Interface V6 ", iface.Name, " ... ", ip.String())
					}
					match_ifaces = append(match_ifaces, ip.String())
				}

				if cfg_verbose {
					fmt.Println()
				}
			}
		}

		return match_ifaces
	}

	all_interfaces := getMatchInterfaces()
	for _, v := range all_interfaces {
		this._cfg_local_add(v)
	}
	this._cfg_local_add(this.config["LOCAL_IP"])
}

// Run conditional config, PARAM_[LOCAL_IP], if local ip is matching add the param to
// the original parameter
func (this *cfg) _cfg_conditional_config() {

	fmt.Println(" --- Conditional config:")
	_do_conditional := func(param string) string {
		ret := make([]string, 0)
		ret_uniq := make(map[string]bool)
		_add := func(v string) {
			v = strings.Trim(v, "\r\n\t ")
			if ret_uniq[v] {
				return
			}
			ret_uniq[v] = true
			ret = append(ret, v)
		}

		for _, v := range strings.Split(this.config[param], ",") {
			_add(v)
		}

		for _, iface_match := range this.local_interfaces {

			_key := param + "_" + iface_match
			if v, exists := this.config[_key]; exists {
				vv := strings.Split(v, ",")
				fmt.Println(" ▶ Conditional config", _key, "adding -", vv)
				for _, vvv := range vv {
					_add(vvv)
				}

				if strings.Compare("LOCAL_IP", param) == 0 {
					this._cfg_local_add(this.config[_key])
				}
			}
		}

		return strings.Join(ret, ",")
	}

	for k := range this.config {
		this.config[k] = _do_conditional(k)
	}
	for _, force_key := range []string{"LOCAL_IP", "BIND_TO", "RUN_SERVICES", "SLAVE", "REPLICATION_MODE"} {
		if _, exists := this.config[force_key]; !exists {
			_do_conditional(force_key)
		}
	}
}
