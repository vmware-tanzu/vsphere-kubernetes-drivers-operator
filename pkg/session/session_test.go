/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package session

import (
	"context"
	"crypto/tls"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware/govmomi/simulator"
	"testing"
)

func TestSession(t *testing.T) {
	RegisterFailHandler(Fail)
	model := simulator.VPX()
	model.Host = 0 // ClusterHost only

	defer model.Remove()
	err := model.Create()
	if err != nil {
		t.Fatal(err)
	}
	model.Service.TLS = new(tls.Config)

	s := model.Service.NewServer()
	defer s.Close()
	pass, _ := s.URL.User.Password()

	authSession, err := GetOrCreate(
		context.Background(),
		s.Server.URL, []string{},
		s.URL.User.Username(), pass, "")
	Expect(err).To(BeNil())
	Expect(authSession).NotTo(BeNil())
	isActive, err := authSession.SessionManager.SessionIsActive(context.Background())
	Expect(err).NotTo(HaveOccurred())
	Expect(isActive).To(BeTrue())
}
