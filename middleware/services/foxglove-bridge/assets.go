package main

import (
	_ "embed"
	"encoding/binary"
	"log"

	"github.com/gorilla/websocket"
)

//go:embed meshes/bunny.stl
var bunnySTL []byte

// assetMap maps package:// URIs to their embedded byte data and media type.
var assetMap = map[string]struct {
	data      []byte
	mediaType string
}{
	"package://robot_pkg/meshes/bunny.stl": {data: bunnySTL, mediaType: "model/stl"},
}

// handleFetchAsset parses a Foxglove fetchAsset binary request (op=0x07) and
// replies with a fetchAssetResponse (op=0x08) containing the requested asset.
//
// Binary request layout:
//
//	[0x07][uint32LE requestId][uint32LE uriLen][uri bytes]
func (b *Bridge) handleFetchAsset(client *clientState, raw []byte) {
	if len(raw) < 9 {
		return
	}
	requestID := binary.LittleEndian.Uint32(raw[1:5])
	uriLen := binary.LittleEndian.Uint32(raw[5:9])
	if uint32(len(raw)) < 9+uriLen {
		return
	}
	uri := string(raw[9 : 9+uriLen])
	log.Printf("[foxglove-bridge] fetchAsset: %s", uri)

	asset, ok := assetMap[uri]
	if !ok {
		resp := buildFetchAssetResponse(requestID, 1, "asset not found: "+uri, nil)
		client.writeMessage(websocket.BinaryMessage, resp) //nolint:errcheck
		return
	}
	resp := buildFetchAssetResponse(requestID, 0, "", asset.data)
	if err := client.writeMessage(websocket.BinaryMessage, resp); err != nil {
		log.Printf("[foxglove-bridge] fetchAsset write error: %v", err)
	}
}

// buildFetchAssetResponse builds the binary fetchAssetResponse frame:
//
//	[0x08][uint32 requestId][uint8 status][uint32 errLen][err][uint32 dataLen][data]
func buildFetchAssetResponse(requestID uint32, status uint8, errMsg string, data []byte) []byte {
	errBytes := []byte(errMsg)
	buf := make([]byte, 1+4+1+4+len(errBytes)+4+len(data))
	off := 0
	buf[off] = 0x08
	off++
	binary.LittleEndian.PutUint32(buf[off:], requestID)
	off += 4
	buf[off] = status
	off++
	binary.LittleEndian.PutUint32(buf[off:], uint32(len(errBytes)))
	off += 4
	copy(buf[off:], errBytes)
	off += len(errBytes)
	binary.LittleEndian.PutUint32(buf[off:], uint32(len(data)))
	off += 4
	copy(buf[off:], data)
	return buf
}
