package fixture

import (
	"bytes"
	"encoding/binary"
	"unicode/utf16"
)

const (
	androidNS = "http://schemas.android.com/apk/res/android"

	attrVersionCode   = 0x0101021b
	attrVersionName   = 0x0101021c
	attrMinSdkVersion = 0x0101020c
	attrHasCode       = 0x0101000c
)

// BinaryManifest は APK に入るバイナリ XML 形式の AndroidManifest.xml を返す.
//
//	<manifest xmlns:android="..." android:versionCode="1" android:versionName="1.0" package="{packageName}"/>
func BinaryManifest(packageName string) []byte {
	// リソースマップは先頭の文字列から順に対応するので, 属性名を先頭に置く.
	strs := []string{"versionCode", "versionName", "package", "manifest", "android", androidNS, "1.0", packageName}
	const (
		sVersionCode = iota
		sVersionName
		sPackage
		sManifest
		sAndroid
		sAndroidNS
		sOneDotZero
		sPackageName
	)
	const none = 0xffffffff

	var body bytes.Buffer
	body.Write(stringPoolChunk(strs))

	// RES_XML_RESOURCE_MAP_TYPE
	writeChunk(&body, 0x0180, 8, u32s(attrVersionCode, attrVersionName))

	// RES_XML_START_NAMESPACE_TYPE
	writeChunk(&body, 0x0100, 16, u32s(1, none, sAndroid, sAndroidNS))

	// RES_XML_START_ELEMENT_TYPE
	var elem bytes.Buffer
	elem.Write(u32s(1, none, none, sManifest))
	binary.Write(&elem, binary.LittleEndian, []uint16{20, 20, 3, 0, 0, 0})
	elem.Write(attr(sAndroidNS, sVersionCode, none, 0x10, 1))
	elem.Write(attr(sAndroidNS, sVersionName, sOneDotZero, 0x03, sOneDotZero))
	elem.Write(attr(none, sPackage, sPackageName, 0x03, sPackageName))
	writeChunk(&body, 0x0102, 16, elem.Bytes())

	// RES_XML_END_ELEMENT_TYPE
	writeChunk(&body, 0x0103, 16, u32s(1, none, none, sManifest))

	// RES_XML_END_NAMESPACE_TYPE
	writeChunk(&body, 0x0101, 16, u32s(1, none, sAndroid, sAndroidNS))

	// RES_XML_TYPE
	var out bytes.Buffer
	writeChunk(&out, 0x0003, 8, body.Bytes())
	return out.Bytes()
}

// emptyResourceTable は package を持たない resources.arsc を返す.
func emptyResourceTable() []byte {
	pool := stringPoolChunk(nil)
	var out bytes.Buffer
	// ヘッダは type, headerSize, size, packageCount の 12 バイト.
	binary.Write(&out, binary.LittleEndian, []uint16{0x0002, 12})
	binary.Write(&out, binary.LittleEndian, []uint32{uint32(12 + len(pool)), 0})
	out.Write(pool)
	return out.Bytes()
}

// stringPoolChunk は UTF-16 の RES_STRING_POOL_TYPE チャンクを返す.
func stringPoolChunk(strs []string) []byte {
	var data bytes.Buffer
	offsets := make([]uint32, len(strs))
	for i, s := range strs {
		offsets[i] = uint32(data.Len())
		u := utf16.Encode([]rune(s))
		binary.Write(&data, binary.LittleEndian, uint16(len(u)))
		binary.Write(&data, binary.LittleEndian, u)
		binary.Write(&data, binary.LittleEndian, uint16(0))
	}
	for data.Len()%4 != 0 {
		data.WriteByte(0)
	}

	const headerSize = 28
	stringsStart := uint32(0)
	if len(strs) > 0 {
		stringsStart = uint32(headerSize + 4*len(strs))
	}
	var body bytes.Buffer
	binary.Write(&body, binary.LittleEndian, []uint32{uint32(len(strs)), 0, 0, stringsStart, 0})
	binary.Write(&body, binary.LittleEndian, offsets)
	body.Write(data.Bytes())

	var out bytes.Buffer
	writeChunk(&out, 0x0001, headerSize, body.Bytes())
	return out.Bytes()
}

// writeChunk は type, headerSize, size のチャンクヘッダに続けて body を書く.
// body にはヘッダの残り (headerSize-8 バイト) も含める.
func writeChunk(w *bytes.Buffer, typ, headerSize uint16, body []byte) {
	binary.Write(w, binary.LittleEndian, []uint16{typ, headerSize})
	binary.Write(w, binary.LittleEndian, uint32(8+len(body)))
	w.Write(body)
}

func attr(ns, name, raw uint32, dataType uint8, data uint32) []byte {
	var b bytes.Buffer
	b.Write(u32s(ns, name, raw))
	binary.Write(&b, binary.LittleEndian, uint16(8))
	b.Write([]byte{0, dataType})
	binary.Write(&b, binary.LittleEndian, data)
	return b.Bytes()
}

func u32s(vs ...uint32) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, vs)
	return b.Bytes()
}

// protoManifest は App Bundle 用の protobuf 形式 (aapt.pb.XmlNode) の AndroidManifest.xml を返す.
func protoManifest(packageName string) []byte {
	intItem := func(v int32) []byte { // Item{prim: Primitive{int_decimal_value}}
		return pbBytes(7, pbVarint(6, uint64(uint32(v))))
	}
	strItem := func(s string) []byte { // Item{str: String{value}}
		return pbBytes(2, pbString(1, s))
	}
	boolItem := func(v bool) []byte { // Item{prim: Primitive{boolean_value}}
		n := uint64(0)
		if v {
			n = 1
		}
		return pbBytes(7, pbVarint(8, n))
	}
	androidAttr := func(name, value string, resID uint32, item []byte) []byte {
		return cat(pbString(1, androidNS), pbString(2, name), pbString(3, value), pbVarint(5, uint64(resID)), pbBytes(6, item))
	}
	element := func(name string, attrs [][]byte, children ...[]byte) []byte {
		e := pbString(3, name)
		for _, a := range attrs {
			e = cat(e, pbBytes(4, a))
		}
		for _, c := range children {
			e = cat(e, pbBytes(5, c))
		}
		return e
	}
	node := func(elem []byte) []byte { return pbBytes(1, elem) }

	usesSdk := node(element("uses-sdk", [][]byte{
		androidAttr("minSdkVersion", "21", attrMinSdkVersion, intItem(21)),
	}))
	application := node(element("application", [][]byte{
		androidAttr("hasCode", "false", attrHasCode, boolItem(false)),
	}))
	manifest := element("manifest", [][]byte{
		cat(pbString(2, "package"), pbString(3, packageName)),
		androidAttr("versionCode", "1", attrVersionCode, intItem(1)),
		androidAttr("versionName", "1.0", attrVersionName, strItem("1.0")),
	}, usesSdk, application)
	// namespace_declaration
	manifest = cat(pbBytes(1, cat(pbString(1, "android"), pbString(2, androidNS))), manifest)
	return node(manifest)
}

// bundleConfigProto は BundleConfig{bundletool: Bundletool{version}} を返す.
func bundleConfigProto(version string) []byte {
	return pbBytes(1, pbString(2, version))
}

func pbVarint(field int, v uint64) []byte {
	b := binary.AppendUvarint(nil, uint64(field<<3)) // wire type 0
	return binary.AppendUvarint(b, v)
}

func pbBytes(field int, data []byte) []byte {
	b := binary.AppendUvarint(nil, uint64(field<<3|2))
	b = binary.AppendUvarint(b, uint64(len(data)))
	return append(b, data...)
}

func pbString(field int, s string) []byte { return pbBytes(field, []byte(s)) }

func cat(bs ...[]byte) []byte {
	var out []byte
	for _, b := range bs {
		out = append(out, b...)
	}
	return out
}
