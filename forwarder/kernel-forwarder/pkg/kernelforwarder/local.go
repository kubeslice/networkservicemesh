// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kernelforwarder

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
        "github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/forwarder/kernel-forwarder/pkg/monitoring"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"
)

// handleLocalConnection either creates or deletes a local connection - same host
func (k *KernelForwarder) handleLocalConnection(crossConnect *crossconnect.CrossConnect, connect bool) (map[string]monitoring.Device, error) {
	logrus.Info("local: connection type - local source/local destination")
	var devices map[string]monitoring.Device
	var err error
	if connect {
		/* 2. Create a connection */
		devices, err = k.createLocalConnection(crossConnect)
		if err != nil {
			logrus.Errorf("local: failed to create connection - %v", err)
			devices = nil
		}
	} else {
		/* 3. Delete a connection */
		devices, err = k.deleteLocalConnection(crossConnect)
		if err != nil {
			logrus.Errorf("local: failed to delete connection - %v", err)
			devices = nil
		}
	}
	return devices, err
}

func (k *KernelForwarder) findLocalSrcConnection(crossConnect *crossconnect.CrossConnect) (bool, error) {
	hostNs, err := netns.Get()
	if err != nil {
		logrus.Errorf("find_local: failed getting host namespace: %v", err)
		return false, err
	}
	defer func() {
		if err = hostNs.Close(); err != nil {
			logrus.Error("common: failed closing host namespace handle: ", err)
		}
		logrus.Debug("common: closed host namespace handle: ", hostNs)
	}()
	/* Don't forget to switch back to the host namespace */
	defer func() {
		if err = netns.Set(hostNs); err != nil {
			logrus.Errorf("common: failed switching back to host namespace: %v", err)
		}
	}()

	srcNetNsInode := crossConnect.GetSource().GetMechanism().GetParameters()[common.NetNsInodeKey]
	srcName := crossConnect.GetSource().GetMechanism().GetParameters()[common.InterfaceNameKey]
	srcNetNsHandle, err := fs.GetNsHandleFromInode(srcNetNsInode)
	if err != nil {
		logrus.Errorf("find_local: failed to get source namespace handle - %v", err)
		return false, nil
	}
	defer func() {
		if err = srcNetNsHandle.Close(); err != nil {
			logrus.Errorf("local: error when closing source namespace handle: %v", err)
		}
	}()

	if err = netns.Set(srcNetNsHandle); err != nil {
		logrus.Errorf("find_local: failed switching to source namespace: %v", err)
		return false, err
	}

	links, err := netlink.LinkList()
	for _, link := range links {
		if link.Type() == "veth" && link.Attrs().Name == srcName {
			return true, nil
		}
	}

	return false, nil
}

// createLocalConnection handles creating a local connection
func (k *KernelForwarder) createLocalConnection(crossConnect *crossconnect.CrossConnect) (map[string]monitoring.Device, error) {
	logrus.Info("local: creating connection...")
	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	rand.Seed(time.Now().UnixNano())

	var err error
	var tempName string
	var srcNetNsInode string
	var dstNetNsInode string

        srcName := crossConnect.GetSource().GetMechanism().GetParameters()[common.InterfaceNameKey]
	dstName := crossConnect.GetDestination().GetMechanism().GetParameters()[common.InterfaceNameKey]

	/*
	 * Check if the interface by the same name exists in the source namespace.
	 * If it does, we need to clean up the interface so that the new interface creation and
	 * setup request will not fail due to its presence.
	 * The interface is one part of the veth pair that connects the NSC (source) and 
	 * NSE (destination). So if we get a new create request with a source interface that already
	 * exists, it would most likely have a different destination interface. Hence we are clearing
	 * up the existing source interface, veth pair as well as a result, to start the process of
	 * creating interfaces and cross connects on a clean slate.
	 */
	foundLocalSrcConn, err := k.findLocalSrcConnection(crossConnect)
	if err != nil {
		return nil, err
	}
	if foundLocalSrcConn {
		logrus.Infof("local: src connection already exists. deleting... %v", srcName)
                _, err := ClearInterfaceSetup(srcName, crossConnect.GetSource())
		if err != nil {
			logrus.Errorf("local: failed to clear source intf: %v", err)
			return nil, err
		}
	        err = k.localConnect.DeleteInterfaces(srcName)
                if err != nil {
			logrus.Errorf("local: failed to delete intf in host: %v", err)
			return nil, err
		}
	}

	/* In case source and destination names are the same, use temporary values during the setup of the VETH interfaces */
	if dstName == srcName {
		tempName = srcName
		srcName = fmt.Sprintf("nsm%d", rand.Uint32())
		dstName = fmt.Sprintf("nsm%d", rand.Uint32())
		logrus.Infof("local: source and destination use the same name - %s, the configuration will proceed with temporary names: %s, %s", tempName, srcName, dstName)
	}

	logrus.Infof("local: creating connection for: %s, %s", srcName, dstName)

	if err = k.localConnect.CreateInterfaces(srcName, dstName); err != nil {
		logrus.Errorf("local: %v", err)
		return nil, err
	}

	if srcNetNsInode, err = SetupInterface(srcName, tempName, crossConnect.GetSource(), false); err != nil {
		return nil, err
	}

	crossConnect.GetDestination().GetContext().IpContext = crossConnect.GetSource().GetContext().GetIpContext()
	if dstNetNsInode, err = SetupInterface(dstName, tempName, crossConnect.GetDestination(), true); err != nil {
		return nil, err
	}

	/* Return to desired names in case of name conflict, for ex. src and dst name are both eth10 */
	if tempName != "" {
		srcName, dstName = tempName, tempName
	}

	logrus.Infof("local: creation completed for devices - source: %s, destination: %s", srcName, dstName)

	srcDevice := monitoring.Device{Name: srcName, XconName: "SRC-" + crossConnect.GetId()}
	dstDevice := monitoring.Device{Name: dstName, XconName: "DST-" + crossConnect.GetId()}
	return map[string]monitoring.Device{srcNetNsInode: srcDevice, dstNetNsInode: dstDevice}, nil
}

// deleteLocalConnection handles deleting a local connection
func (k *KernelForwarder) deleteLocalConnection(crossConnect *crossconnect.CrossConnect) (map[string]monitoring.Device, error) {
	logrus.Info("local: deleting connection...")
	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	srcName := crossConnect.GetSource().GetMechanism().GetParameters()[common.InterfaceNameKey]
	dstName := crossConnect.GetDestination().GetMechanism().GetParameters()[common.InterfaceNameKey]

	srcNetNsInode, srcErr := ClearInterfaceSetup(srcName, crossConnect.GetSource())
	dstNetNsInode, dstErr := ClearInterfaceSetup(dstName, crossConnect.GetDestination())

	err := k.localConnect.DeleteInterfaces(srcName)

	if srcErr != nil || dstErr != nil || err != nil {
		return nil, errors.Errorf("local: %v - %v", srcErr, dstErr)
	}

	logrus.Infof("local: deletion completed for devices - source: %s, destination: %s", srcName, dstName)
	srcDevice := monitoring.Device{Name: srcName, XconName: "SRC-" + crossConnect.GetId()}
	dstDevice := monitoring.Device{Name: dstName, XconName: "DST-" + crossConnect.GetId()}
	return map[string]monitoring.Device{srcNetNsInode: srcDevice, dstNetNsInode: dstDevice}, nil
}
