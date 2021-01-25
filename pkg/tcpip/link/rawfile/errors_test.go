// Copyright 2020 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build linux

package rawfile

import (
	"syscall"
	"testing"

	"gvisor.dev/gvisor/pkg/tcpip"
)

func TestTranslateErrno(t *testing.T) {
	for _, test := range []struct {
		errno      syscall.Errno
		translated func(tcpip.Error) tcpip.Error
	}{
		{
			errno: syscall.Errno(0),
			translated: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrInvalidEndpointState); ok {
					return nil
				}
				return &tcpip.ErrInvalidEndpointState{}
			},
		},
		{
			errno: syscall.Errno(maxErrno),
			translated: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrInvalidEndpointState); ok {
					return nil
				}
				return &tcpip.ErrInvalidEndpointState{}
			},
		},
		{
			errno: syscall.Errno(514),
			translated: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrInvalidEndpointState); ok {
					return nil
				}
				return &tcpip.ErrInvalidEndpointState{}
			},
		},
		{
			errno: syscall.EEXIST,
			translated: func(err tcpip.Error) tcpip.Error {
				if _, ok := err.(*tcpip.ErrDuplicateAddress); ok {
					return nil
				}
				return &tcpip.ErrDuplicateAddress{}
			},
		},
	} {
		got := TranslateErrno(test.errno)
		if want := test.translated(got); want != nil {
			t.Errorf("TranslateErrno(%q) = %q, want %q", test.errno, got, want)
		}
	}
}
