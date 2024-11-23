package main

import (
	"errors"
	"flag"
	"reflect"
)

type cfg struct {
	region   string
	account  string
	tagKey   string
	tagValue string
	dryRun   bool
}

func getFlags() (*cfg, error) {
	var c cfg
	flag.StringVar(&c.region, "region", "", "AWS region")
	flag.StringVar(&c.account, "account", "", "AWS account-id")
	flag.StringVar(&c.tagKey, "tagKey", "", "resource-tag key to search for")
	flag.StringVar(&c.tagValue, "tagValue", "", "resource-tag value to search for")
	flag.BoolVar(&c.dryRun, "dryRun", true, "dry-run (do not delete resources)")
	flag.Parse()

	var el []error

	// lol
	v := reflect.ValueOf(c)
	for i := range v.NumField() {
		f := v.Field(i)
		sf := v.Type().Field(i)

		if sf.Type.Kind() == reflect.String && f.IsZero() {
			el = append(el, errors.New("flag -"+sf.Name+" is required"))
		}
	}

	if len(el) != 0 {
		return nil, errors.Join(el...)
	}

	return &c, nil
}
