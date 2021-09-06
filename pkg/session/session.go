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
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"net/url"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

var sessionCache = map[string]Session{}
var sessionMU sync.Mutex

// Session is a vSphere session with a configured Finder.
type Session struct {
	*govmomi.Client
	datacenters    []*object.Datacenter
	VsphereVersion string
}

type VirtualMachine struct {
	*object.VirtualMachine
	Datacenter *object.Datacenter
}

// GetOrCreate gets a cached session or creates a new one if one does not
// already exist.
func GetOrCreate(
	ctx context.Context,
	server string, datacenters []string, username, password, thumbprint string) (*Session, error) {
	logger := log.FromContext(ctx).WithValues("session", "vcsession")
	sessionMU.Lock()
	defer sessionMU.Unlock()

	sessionKey := server + username
	if cachedSession, ok := sessionCache[sessionKey]; ok {
		if ok, _ := cachedSession.SessionManager.SessionIsActive(ctx); ok {
			logger.V(2).Info("found active cached vSphere client session", "server", server)

			var DCListCachedSession []string
			for _, dc := range cachedSession.datacenters {
				DCListCachedSession = append(DCListCachedSession, dc.Name())
			}
			if reflect.DeepEqual(datacenters, DCListCachedSession) {
				return &cachedSession, nil
			}
		}
	}

	soapURL, err := soap.ParseURL(server)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing vSphere URL %q", server)
	}
	if soapURL == nil {
		return nil, errors.Errorf("error parsing vSphere URL %q", server)
	}

	soapURL.User = url.UserPassword(username, password)
	client, err := newClient(ctx, soapURL, thumbprint)
	if err != nil {
		return nil, err
	}

	session := Session{Client: client}
	session.UserAgent = v1alpha1.GroupVersion.String()
	// Assign the finder to the session.
	finder := find.NewFinder(session.Client.Client, false)

	session.VsphereVersion = session.Client.ServiceContent.About.Version

	if len(datacenters) > 0 {
		for _, datacenter := range datacenters {

			// Assign the datacenter if one was specified.
			dc, err := finder.Datacenter(ctx, datacenter)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to find datacenter %q", datacenter)
			}
			if dc != nil {
				session.datacenters = append(session.datacenters, dc)
			}
			logger.V(2).Info("cached vSphere client session", "server", server, "datacenter", datacenter)
		}

	}
	sessionCache[sessionKey] = session

	return &session, nil
}

func newClient(ctx context.Context, url *url.URL, thumbprint string) (*govmomi.Client, error) {
	insecure := thumbprint == ""

	soapClient := soap.NewClient(url, insecure)

	if !insecure {
		soapClient.SetThumbprint(url.Host, thumbprint)
	}

	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return nil, err
	}

	c := &govmomi.Client{
		Client:         vimClient,
		SessionManager: session.NewManager(vimClient),
	}

	if err := c.Login(ctx, url.User); err != nil {
		return nil, err
	}

	return c, nil
}

func (s *Session) GetVMByIP(ctx context.Context, ipAddy string) (*VirtualMachine, error) {
	if len(s.datacenters) > 0 {
	dcloop:
		for _, datacenter := range s.datacenters {
			i := object.NewSearchIndex(datacenter.Client())
			ipAddy = strings.ToLower(strings.TrimSpace(ipAddy))
			svm, err := i.FindByIp(ctx, datacenter, ipAddy, true)
			if err != nil {
				return nil, err
			}
			if svm == nil {
				continue dcloop
			}
			virtualMachine := VirtualMachine{svm.(*object.VirtualMachine), datacenter}
			return &virtualMachine, nil
		}
	}
	return nil, nil
}
