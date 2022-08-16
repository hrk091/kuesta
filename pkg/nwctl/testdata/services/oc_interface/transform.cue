package oc_interface

import (
	ocdemo "github.com/hrk091/nwctl/testdata/pkg/ocdemo"
)

#Input: {
	device: string
	port:   uint16
	noShut: bool
	desc:   string | *""
	mtu:    uint16 | *9000
}

#Template: {
	input: #Input

	let _portName = "Ethernet\(input.port)"

	output: devices: "\(input.device)": config: {
		ocdemo.#Device
		Interface: "\(_portName)": {
			Name:        _portName
			Description: input.desc
			Enabled:     input.noShut
			AdminStatus: 1
			OperStatus:  1
			Type:        ocdemo.#IETFInterfaces_InterfaceType_ethernetCsmacd
			Mtu:         input.mtu
			...
		}
		...
	}
}
