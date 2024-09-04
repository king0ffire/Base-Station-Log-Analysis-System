package util

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

var ConfigMap map[string]map[string]string

func init() {
	var err error
	ConfigMap, err = IniToMap("config.ini")
	if err != nil {
		logrus.Panic(fmt.Sprintf("启动ini失败:%s", err))
	}
}

func IniToMap(filename string) (map[string]map[string]string, error) {
	cfg, err := ini.Load(filename)
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]string)
	for _, section := range cfg.Sections() {
		result[section.Name()] = make(map[string]string)
		for _, key := range section.Keys() {
			result[section.Name()][key.Name()] = key.Value()
		}
	}
	return result, nil
}
