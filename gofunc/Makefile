
# Copyright 2009 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

include $(GOROOT)/src/Make.inc

TARG=gofunc
GOFILES=\
	gofunc.go\

include $(GOROOT)/src/Make.cmd

test: $(TARG)
	./test.sh

smoketest: $(TARG)
	./test.sh $(GOROOT)/src/pkg/go/parser/parser.go

