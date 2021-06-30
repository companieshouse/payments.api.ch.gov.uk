package config

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitGet(t *testing.T) {

	Convey("Config already defined", t, func() {
		cfg = DefaultConfig()
		config, err := Get()
		So(config, ShouldResemble, DefaultConfig())
		So(err, ShouldBeNil)
	})

	Convey("Successful get config", t, func() {
		cfg = nil // reset after previous tests
		config, err := Get()
		So(config, ShouldResemble, DefaultConfig())
		So(err, ShouldBeNil)
	})

}
