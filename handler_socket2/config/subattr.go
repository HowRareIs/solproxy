package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (this *cfg) ValidateAttribs(attr string, subattrs []string) (bool, error) {
	if len(subattrs) == 0 {
		if _, exists := this.config[attr]; exists {
			return true, nil
		}
		return false, nil
	}

	has_any := false
	has_all := true
	missing := make([]string, 0, len(subattrs))
	for _, subattr := range subattrs {
		_, err := this._get_subattr_interface(attr, subattr)
		has_any = has_any || err == nil
		if err != nil {
			has_all = false
			missing = append(missing, subattr)
		}
	}

	if !has_any {
		return false, nil
	}
	if has_all {
		return true, nil
	}

	return false, errors.New("Attributes missing: " + strings.Join(missing, ", "))
}

func (this *cfg) _get_subattr_interface(attr string, subattr string) (interface{}, error) {
	i := (interface{})(nil)
	if val, exists := this.raw_data[attr]; exists {
		tmp := val.(map[string]interface{})
		if val, exists := tmp[subattr]; exists {
			i = val
		}
	}
	if i == nil {
		return 0, errors.New(fmt.Sprintf("Attribute %s/%s not found", attr, subattr))
	}
	return i, nil
}

func (this *cfg) GetSubattrInt(attr string, subattr string) (int, error) {
	i, err := this._get_subattr_interface(attr, subattr)
	if err != nil {
		return 0, err
	}

	switch i.(type) {
	case string:
		out, err := strconv.Atoi(i.(string))
		if err != nil {
			return 0, errors.New(fmt.Sprintf("Attribute %s/%s error:\n"+err.Error(), attr, subattr))
		}
		return out, nil
	case int:
		return i.(int), nil
	case float64:
		return int(i.(float64)), nil
	case json.Number:
		out, err := i.(json.Number).Int64()
		if err != nil {
			return 0, errors.New(fmt.Sprintf("Attribute %s/%s error:\n"+err.Error(), attr, subattr))
		}
		return int(out), nil
	case bool:
		if i.(bool) {
			return 1, nil
		} else {
			return 0, nil
		}
	default:
		return 0, errors.New(fmt.Sprintf("Attribute %s/%s is of wrong type (int required)", attr, subattr))
	}
}

func (this *cfg) GetSubattrString(attr string, subattr string) (string, error) {
	i, err := this._get_subattr_interface(attr, subattr)
	if err != nil {
		return "", err
	}

	switch i.(type) {
	case string:
		return i.(string), nil
	case int:
		return strconv.Itoa(i.(int)), nil
	case float64:
		return strconv.FormatFloat(i.(float64), 'f', 3, 64), nil
	case json.Number:
		return i.(json.Number).String(), nil
	case bool:
		if i.(bool) {
			return "1", nil
		} else {
			return "0", nil
		}
	default:
		return "", errors.New(fmt.Sprintf("Attribute %s/%s is of wrong type (int required)", attr, subattr))
	}
}
