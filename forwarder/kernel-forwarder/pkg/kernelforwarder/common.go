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
	"net"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"github.com/mpvl/unique"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"
)

type linkRoutes struct {
	routes  []*connectioncontext.Route
	nextHop string
}

// LinkData instance
type LinkData struct {
	nsHandle  netns.NsHandle // Desired namespace handler
	name      string
	tempName  string // Used in case src and dst name are the same causing the VETH creation to fail
	alias     string
	ip        string
	routes    linkRoutes
	neighbors []*connectioncontext.IpNeighbor
}

// SetupInterface - setup interface to namespace
func SetupInterface(ifaceName, tempName string, conn *connection.Connection, isDst bool) (string, error) {
	var err error
	link := &LinkData{name: ifaceName, tempName: tempName}
	netNsInode := conn.GetMechanism().GetParameters()[common.NetNsInodeKey]
	link.neighbors = conn.GetContext().GetIpContext().GetIpNeighbors()
	installRoutes := linkRoutes{}
	if isDst {
		link.ip = conn.GetContext().GetIpContext().GetDstIpAddr()
		installRoutes.routes = conn.GetContext().GetIpContext().GetSrcRoutes()
		installRoutes.nextHop = conn.GetContext().GetIpContext().GetSrcIpAddr()
	} else {
		link.ip = conn.GetContext().GetIpContext().GetSrcIpAddr()
		installRoutes.routes = conn.GetContext().GetIpContext().GetDstRoutes()
		installRoutes.nextHop = conn.GetContext().GetIpContext().GetDstIpAddr()
	}

	link.routes = installRoutes
	link.alias = conn.GetLabels()["podName"]

	/* Get namespace handler - source */
	link.nsHandle, err = fs.GetNsHandleFromInode(netNsInode)
	if err != nil {
		logrus.Errorf("local: failed to get source namespace handle - %v", err)
		return netNsInode, err
	}
	/* If successful, don't forget to close the handler upon exit */
	defer func() {
		if err = link.nsHandle.Close(); err != nil {
			logrus.Error("local: error when closing source handle: ", err)
		}
		logrus.Debug("local: closed source handle: ", link.nsHandle, netNsInode)
	}()
	logrus.Debug("local: opened source handle: ", link.nsHandle, netNsInode)

	/* Setup interface - source namespace */
	if err = setupLinkInNs(link, true); err != nil {
		logrus.Errorf("local: failed to setup interface - source - %q: %v", link.name, err)
		return netNsInode, err
	}

	return netNsInode, nil
}

// ClearInterfaceSetup - deletes interface setup
func ClearInterfaceSetup(ifaceName string, conn *connection.Connection) (string, error) {
	var err error
	link := &LinkData{name: ifaceName}
	netNsInode := conn.GetMechanism().GetParameters()[common.NetNsInodeKey]
	link.ip = conn.GetContext().GetIpContext().GetSrcIpAddr()

	delRoutes := linkRoutes{}
	delRoutes.routes = conn.GetContext().GetIpContext().GetDstRoutes()
	link.routes = delRoutes

	/* Get namespace handler - source */
	link.nsHandle, err = fs.GetNsHandleFromInode(netNsInode)
	if err != nil {
		return "", errors.Errorf("failed to get source namespace handle - %v", err)
	}
	/* If successful, don't forget to close the handler upon exit */
	defer func() {
		if err = link.nsHandle.Close(); err != nil {
			logrus.Error("local: error when closing source handle: ", err)
		}
		logrus.Debug("local: closed source handle: ", link.nsHandle, netNsInode)
	}()
	logrus.Debug("local: opened source handle: ", link.nsHandle, netNsInode)

	/* Extract interface - source namespace */
	if err = setupLinkInNs(link, false); err != nil {
		return "", errors.Errorf("failed to extract interface - source - %q: %v", link.name, err)
	}

	return netNsInode, nil
}

// setupLinkInNs is responsible for configuring an interface inside a given namespace - assigns IP address, routes, etc.
func setupLinkInNs(link *LinkData, inject bool) error {
	if inject {
		/* Get a link object for the interface */
		ifaceLink, err := netlink.LinkByName(link.name)
		if err != nil {
			logrus.Errorf("common: failed to get link for %q - %v", link.name, err)
			return err
		}
		/* Set alias for the interface */
		if link.alias != "" {
			if err = netlink.LinkSetAlias(ifaceLink, link.alias); err != nil {
				logrus.Errorf("common: failed to set alias to the interface: %v, alias: %v - %v", link.name, link.alias, err)
				return err
			}
		}

		logrus.Debug("common: interface alias: ", link.alias)
		/* Inject the interface into the desired namespace */
		if err = netlink.LinkSetNsFd(ifaceLink, int(link.nsHandle)); err != nil {
			logrus.Errorf("common: failed to inject %q in namespace - %v", link.name, err)
			return err
		}
	}
	/* Save current network namespace */
	hostNs, err := netns.Get()
	if err != nil {
		logrus.Errorf("common: failed getting host namespace: %v", err)
		return err
	}
	logrus.Debug("common: host namespace: ", hostNs)
	defer func() {
		if err = hostNs.Close(); err != nil {
			logrus.Error("common: failed closing host namespace handle: ", err)
		}
		logrus.Debug("common: closed host namespace handle: ", hostNs)
	}()

	/* Switch to the desired namespace */
	if err = netns.Set(link.nsHandle); err != nil {
		logrus.Errorf("common: failed switching to desired namespace: %v", err)
		return err
	}
	logrus.Debug("common: switched to desired namespace: ", link.nsHandle)

	/* Don't forget to switch back to the host namespace */
	defer func() {
		if err = netns.Set(hostNs); err != nil {
			logrus.Errorf("common: failed switching back to host namespace: %v", err)
		}
		logrus.Debug("common: switched back to host namespace: ", hostNs)
	}()

	/* Get a link for the interface name */
	l, err := netlink.LinkByName(link.name)
	if err != nil {
		logrus.Errorf("common: failed to lookup %q, %v", link.name, err)
		return err
	}
	if inject {
		if err = setupLink(l, link); err != nil {
			logrus.Errorf("common: failed to setup link %s: %v", link.name, err)
			return err
		}
	} else {
		/* Delete routes */
		if err = deleteRoutes(l, link.routes); err != nil {
			logrus.Errorf("common: failed deleting routes:", err)
		}
		/* Bring the interface DOWN */
		if err = netlink.LinkSetDown(l); err != nil {
			logrus.Errorf("common: failed to bring %q down: %v", link.name, err)
			return err
		}
		/* Inject the interface back to current namespace */
		if err = netlink.LinkSetNsFd(l, int(hostNs)); err != nil {
			logrus.Errorf("common: failed to inject %q back to host namespace - %v", link.name, err)
			return err
		}
	}
	return nil
}

// setupLink configures the link - name, IP, routes, etc.
func setupLink(l netlink.Link, link *LinkData) error {
	var err error
	var addr *netlink.Addr
	/* Rename back the interface in case there was a naming conflict */
	if link.tempName != "" {
		if err = netlink.LinkSetName(l, link.tempName); err != nil {
			logrus.Errorf("common: failed to rename link %s -> %s: %v",
				link.name, link.tempName, err)
			return err
		}
		link.name = link.tempName
	}
	/* Parse the IP address */
	addr, err = netlink.ParseAddr(link.ip)
	if err != nil {
		logrus.Errorf("common: failed to parse IP %q: %v", link.ip, err)
		return err
	}
	/* Set IP address */
	if err = netlink.AddrAdd(l, addr); err != nil {
		logrus.Errorf("common: failed to set IP %q: %v", link.ip, err)
		return err
	}
	/* Bring the interface UP */
	if err = netlink.LinkSetUp(l); err != nil {
		logrus.Errorf("common: failed to bring %q up: %v", link.name, err)
		return err
	}
	/* Add routes */
	if err = addRoutes(l, addr, link.routes); err != nil {
		logrus.Error("common: failed adding routes:", err)
		return err
	}
	/* Add neighbors - applicable only for source side */
	if err = addNeighbors(l, link.neighbors); err != nil {
		logrus.Error("common: failed adding neighbors:", err)
		return err
	}
	return err
}

// addRoutes adds routes
func addRoutes(link netlink.Link, addr *netlink.Addr, lroutes linkRoutes) error {
	prefixList := []string{}
	for _, installRoute := range lroutes.routes {
		prefixList = append(prefixList, installRoute.GetPrefix())
	}
	unique.Strings(&prefixList)
	nextHop := lroutes.nextHop

	for _, prefix := range prefixList {
		_, routeNet, err := net.ParseCIDR(prefix)
		nextHopIp, _ := netlink.ParseAddr(nextHop)
		if err != nil {
			logrus.Error("common: failed parsing route CIDR:", err)
			return err
		}
		route := netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst: &net.IPNet{
				IP:   routeNet.IP,
				Mask: routeNet.Mask,
			},
			Gw: nextHopIp.IP,
		}
		if err = netlink.RouteAdd(&route); err != nil {
			logrus.Error("common: failed adding routes:", err)
			return err
		}
	}

	return nil
}

// deleteRoutes deletes routes
func deleteRoutes(link netlink.Link, lroutes linkRoutes) error {
	installedRoutes, err := netlink.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		return err
	}

	routeMap := make(map[string][]netlink.Route)
	for _, installedRoute := range installedRoutes {
		routeMap[installedRoute.Dst.String()] = append(routeMap[installedRoute.Dst.String()], installedRoute)
	}

	prefixList := []string{}
	for _, delRoute := range lroutes.routes {
		prefixList = append(prefixList, delRoute.GetPrefix())
	}
	unique.Strings(&prefixList)

	for _, prefix := range prefixList {
		_, ok := routeMap[prefix]
		if !ok {
			logrus.Infof("common: prefix %v not present in routing table. Ignoring..", prefix)
			continue
		}
		for _, r := range routeMap[prefix] {
			err := netlink.RouteDel(&r)
			if err != nil {
				logrus.Errorf("common: failed to delete route: %v, err: %v", r, err)
				return err
			}
			logrus.Infof("Route deleted: %v", r)
		}
	}

	return nil
}

// addNeighbors adds neighbors
func addNeighbors(link netlink.Link, neighbors []*connectioncontext.IpNeighbor) error {
	for _, neighbor := range neighbors {
		mac, err := net.ParseMAC(neighbor.GetHardwareAddress())
		if err != nil {
			logrus.Error("common: failed parsing the MAC address for IP neighbors:", err)
			return err
		}
		neigh := netlink.Neigh{
			LinkIndex:    link.Attrs().Index,
			State:        0x02, // netlink.NUD_REACHABLE, // the constant is somehow not being found in the package in case of using a darwin based machine
			IP:           net.ParseIP(neighbor.GetIp()),
			HardwareAddr: mac,
		}
		if err = netlink.NeighAdd(&neigh); err != nil {
			logrus.Error("common: failed adding neighbor:", err)
			return err
		}
	}
	return nil
}
