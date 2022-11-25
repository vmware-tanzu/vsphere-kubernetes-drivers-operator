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
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25/types"
)

var _ = Describe("vc session functions", func() {
	var (
		ctx    context.Context
		s      *simulator.Server
		finder *find.Finder
		pass   string
	)

	BeforeEach(func() {
		RegisterFailHandler(Fail)
		model := simulator.VPX()
		model.Host = 1 // ClusterHost only
		model.Pool = 1
		model.Datacenter = 2
		model.Datastore = 2
		model.Cluster = 1
		model.ClusterHost = 1

		err := model.Create()
		Expect(err).NotTo(HaveOccurred())
		model.Service.TLS = new(tls.Config)

		s = model.Service.NewServer()
		pass, _ = s.URL.User.Password()

		ctx = context.Background()

		var client, _ = newClient(ctx, s.URL, "")

		finder = find.NewFinder(client.Client)
	})

	AfterEach(func() {
		s.Close()
	})

	Context("when we fetch session", func() {

		It("should successfully retrieve the session", func() {
			authSession, err := GetOrCreate(
				ctx,
				s.Server.URL, []string{"/DC0"},
				s.URL.User.Username(), pass, "")
			Expect(err).To(BeNil())
			Expect(authSession).NotTo(BeNil())
			isActive, err := authSession.SessionManager.SessionIsActive(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(isActive).To(BeTrue())
		})

		It("should successfully retrieve session from cache", func() {
			_, err := GetOrCreate(
				ctx,
				s.Server.URL, []string{"/DC0"},
				s.URL.User.Username(), pass, "")
			Expect(err).To(BeNil())
			authSession, err := GetOrCreate(
				ctx,
				s.Server.URL, []string{"/DC0"},
				s.URL.User.Username(), "", "")
			Expect(err).To(BeNil())
			Expect(authSession).NotTo(BeNil())
			isActive, err := authSession.SessionManager.SessionIsActive(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(isActive).To(BeTrue())
		})
	})

	Context("when we fetch vm by IP", func() {
		It("should not return a vm for an existing IP", func() {

			vm0, _ := finder.VirtualMachine(ctx, "/DC0/vm/DC0_C0_RP0_VM0")

			_, err := vm0.PowerOff(ctx)
			Expect(err).NotTo(HaveOccurred())

			networkSpec := types.CustomizationSpec{
				NicSettingMap: []types.CustomizationAdapterMapping{
					{
						Adapter: types.CustomizationIPSettings{
							Ip: &types.CustomizationFixedIp{
								IpAddress: "192.168.1.100",
							},
							SubnetMask:    "255.255.255.0",
							Gateway:       []string{"192.168.1.1"},
							DnsServerList: []string{"192.168.1.1"},
							DnsDomain:     "ad.domain",
						},
					},
				},
				Identity: &types.CustomizationLinuxPrep{
					HostName: &types.CustomizationFixedName{
						Name: vm0.Name(),
					},
					Domain:     "ad.domain",
					TimeZone:   "Etc/UTC",
					HwClockUTC: types.NewBool(true),
				},
				GlobalIPSettings: types.CustomizationGlobalIPSettings{
					DnsSuffixList: []string{"ad.domain"},
					DnsServerList: []string{"192.168.1.1"},
				},
			}

			var task, _ = vm0.Customize(ctx, networkSpec)
			Expect(task.Wait(ctx)).NotTo(HaveOccurred())

			var powerOnTask, _ = vm0.PowerOn(ctx)
			Expect(powerOnTask.Wait(ctx)).NotTo(HaveOccurred())

			var ip, _ = vm0.WaitForIP(ctx, true)
			Expect(ip).NotTo(BeEmpty())

			dcs, _ := finder.DatacenterList(ctx, "/DC0")
			vm3, err := GetVMByIP(ctx, ip, dcs)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm3).NotTo(BeNil())
		})

		It("should not return a vm for non existent IP", func() {

			dcs, _ := finder.DatacenterList(ctx, "/DC0")
			vm3, err := GetVMByIP(ctx, "1.1.1.1", dcs)
			Expect(err).NotTo(HaveOccurred())
			Expect(vm3).To(BeNil())
		})
	})
})
