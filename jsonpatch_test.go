package jsonpatch

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	jp "github.com/evanphx/json-patch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var simpleA = `{"a":100, "b":200, "c":"hello"}`
var simpleB = `{"a":100, "b":200, "c":"goodbye"}`
var simpleC = `{"a":100, "b":100, "c":"hello"}`
var simpleD = `{"a":100, "b":200, "c":"hello", "d":"foo"}`
var simpleE = `{"a":100, "b":200}`
var simplef = `{"a":100, "b":100, "d":"foo"}`
var simpleG = `{"a":100, "b":null, "d":"foo"}`
var empty = `{}`

var arraySrc = `
{
  "spec": {
    "loadBalancerSourceRanges": [
      "192.101.0.0/16",
      "192.0.0.0/24"
    ]
  }
}
`

var arrayDst = `
{
  "spec": {
    "loadBalancerSourceRanges": [
      "192.101.0.0/24"
    ]
  }
}
`

var complexBase = `{"a":100, "b":[{"c1":"hello", "d1":"foo"},{"c2":"hello2", "d2":"foo2"} ], "e":{"f":200, "g":"h", "i":"j"}}`
var complexA = `{"a":100, "b":[{"c1":"goodbye", "d1":"foo"},{"c2":"hello2", "d2":"foo2"} ], "e":{"f":200, "g":"h", "i":"j"}}`
var complexB = `{"a":100, "b":[{"c1":"hello", "d1":"foo"},{"c2":"hello2", "d2":"foo2"} ], "e":{"f":100, "g":"h", "i":"j"}}`
var complexC = `{"a":100, "b":[{"c1":"hello", "d1":"foo"},{"c2":"hello2", "d2":"foo2"} ], "e":{"f":200, "g":"h", "i":"j"}, "k":[{"l":"m"}, {"l":"o"}]}`

var point = `{"type":"Point", "coordinates":[0.0, 1.0]}`
var lineString = `{"type":"LineString", "coordinates":[[0.0, 1.0], [2.0, 3.0]]}`

//go:embed testdata/hyper_complex_base.json
var hyperComplexBase string

//go:embed testdata/hyper_complex_a.json
var hyperComplexA string

//go:embed testdata/super_complex_base.json
var superComplexBase string

//go:embed testdata/super_complex_a.json
var superComplexA string

var (
	oldDeployment = `{
  "apiVersion": "apps/v1beta1",
  "kind": "Deployment",
  "metadata": {
    "annotations": {
      "k8s.io/app": "busy-dep"
    }
  }
}`

	newDeployment = `{
  "apiVersion": "apps/v1beta1",
  "kind": "Deployment",
  "metadata": {
    "annotations": {
      "k8s.io/app": "busy-dep",
      "docker.com/commit": "github.com/myrepo#xyz"
    }
  }
}`
)

var (
	oldNestedObj = `{
  "apiVersion": "kubedb.com/v1alpha1",
  "kind": "Elasticsearch",
  "metadata": {
    "name": "quick-elasticsearch",
    "namespace": "demo"
  },
  "spec": {
    "doNotPause": true,
    "version": "5.6"
  }
}`

	newNestedObj = `{
  "apiVersion": "kubedb.com/v1alpha1",
  "kind": "Elasticsearch",
  "metadata": {
    "name": "quick-elasticsearch",
    "namespace": "demo"
  },
  "spec": {
    "doNotPause": true,
    "version": "5.6",
    "storageType": "Durable",
    "updateStrategy": {
      "type": "RollingUpdate"
    },
    "terminationPolicy": "Pause"
  }
}`
)

var (
	oldArray = `{
  "apiVersion": "kubedb.com/v1alpha1",
  "kind": "Elasticsearch",
  "metadata": {
    "name": "quick-elasticsearch",
    "namespace": "demo"
  },
  "spec": {
    "tolerations": [
      {
          "key": "node.kubernetes.io/key1",
          "operator": "Equal",
          "value": "value1",
          "effect": "NoSchedule"
      },
      {
          "key": "node.kubernetes.io/key2",
          "operator": "Equal",
          "value": "value2",
          "effect": "NoSchedule"
      },
      {
          "key": "node.kubernetes.io/not-ready",
          "operator": "Exists",
          "effect": "NoExecute",
          "tolerationSeconds": 300
      },
      {
          "key": "node.kubernetes.io/unreachable",
          "operator": "Exists",
          "effect": "NoExecute",
          "tolerationSeconds": 300
      }
    ]
  }
}`

	newArray = `{
  "apiVersion": "kubedb.com/v1alpha1",
  "kind": "Elasticsearch",
  "metadata": {
    "name": "quick-elasticsearch",
    "namespace": "demo"
  },
  "spec": {
    "tolerations": [
      {
          "key": "node.kubernetes.io/key2",
          "operator": "Equal",
          "value": "value2",
          "effect": "NoSchedule"
      },
      {
          "key": "node.kubernetes.io/key1",
          "operator": "Equal",
          "value": "value1",
          "effect": "NoSchedule"
      }
    ]
  }
}`
)

var (
	nullKeyA = `{
  "apiVersion": "cert-manager.io/v1",
  "kind": "CertificateRequest",
  "metadata": {
    "creationTimestamp": null,
    "name": "test-cr",
    "namespace": "default-unit-test-ns"
  },
  "spec": {
    "issuerRef": {
      "name": ""
    },
    "request": null
  },
  "status": {}
}`
	nullKeyB = `{
  "apiVersion": "cert-manager.io/v1",
  "kind": "CertificateRequest",
  "metadata": {
    "creationTimestamp": null,
    "name": "test-cr",
    "namespace": "default-unit-test-ns"
  },
  "spec": {
    "issuerRef": {
      "name": ""
    },
    "request": "bXV0YXRpb24gY2FsbGVk"
  },
  "status": {}
}`
)

func TestCreatePatch(t *testing.T) {
	cases := []struct {
		name string
		src  string
		dst  string
	}{
		// simple
		{"Simple:OneNullReplace", simplef, simpleG},
		{"Simple:Same", simpleA, simpleA},
		{"Simple:OneStringReplace", simpleA, simpleB},
		{"Simple:OneIntReplace", simpleA, simpleC},
		{"Simple:OneAdd", simpleA, simpleD},
		{"Simple:OneRemove", simpleA, simpleE},
		{"Simple:VsEmpty", simpleA, empty},
		// array types
		{"Array:Same", arraySrc, arraySrc},
		{"Array:BoolReplace", arraySrc, arrayDst},
		{"Array:AlmostSame", `{"Lines":[1,2,3,4,5,6,7,8,9,10]}`, `{"Lines":[2,3,4,5,6,7,8,9,10,11]}`},
		{"Array:Remove", `{"x":["A", "B", "C"]}`, `{"x":["D"]}`},
		{"Array:EditDistance", `{"letters":["A","B","C","D","E","F","G","H","I","J","K"]}`, `{"letters":["L","M","N"]}`},
		// complex types
		{"Complex:Same", complexBase, complexBase},
		{"Complex:OneStringReplaceInArray", complexBase, complexA},
		{"Complex:OneIntReplace", complexBase, complexB},
		{"Complex:OneAdd", complexBase, complexC},
		{"Complex:OneAddToArray", complexBase, complexC},
		{"Complex:VsEmpty", complexBase, empty},
		// geojson
		{"GeoJson:PointLineStringReplace", point, lineString},
		{"GeoJson:LineStringPointReplace", lineString, point},
		// HyperComplex
		{"HyperComplex:Same", hyperComplexBase, hyperComplexBase},
		{"HyperComplex:BoolReplace", hyperComplexBase, hyperComplexA},
		// SuperComplex
		{"SuperComplex:Same", superComplexBase, superComplexBase},
		{"SuperComplex:BoolReplace", superComplexBase, superComplexA},
		// map
		{"Kubernetes:Annotations", oldDeployment, newDeployment},
		// crd with nested object
		{"Nested Member Object", oldNestedObj, newNestedObj},
		// array with different order
		{"Different Array", oldArray, newArray},
		{"Array at root", `[{"asdf":"qwerty"}]`, `[{"asdf":"bla"},{"asdf":"zzz"}]`},
		{"Empty array at root", `[]`, `[{"asdf":"bla"},{"asdf":"zzz"}]`},
		{"Null Key uses replace operation", nullKeyA, nullKeyB},
	}

	for _, c := range cases {
		t.Run(c.name+"[src->dst]", func(t *testing.T) {
			check(t, c.src, c.dst)
		})
		t.Run(c.name+"[dst->src]", func(t *testing.T) {
			check(t, c.dst, c.src)
		})
	}
}

func TestJSONPatchCreate(t *testing.T) {
	cases := []struct {
		name     string
		a        string
		b        string
		expected string
	}{
		{
			"object",
			`{"asdf":"qwerty"}`,
			`{"asdf":"zzz"}`,
			`[{"op":"replace","path":"/asdf","value":"zzz"}]`,
		},
		{
			"object with array",
			`{"items":[{"asdf":"qwerty"}]}`,
			`{"items":[{"asdf":"bla"},{"asdf":"zzz"}]}`,
			`[{"op":"add","path":"/items/1","value":{"asdf":"zzz"}},{"op":"replace","path":"/items/0/asdf","value":"bla"}]`,
		},
		{
			"array",
			`[{"asdf":"qwerty"}]`,
			`[{"asdf":"bla"},{"asdf":"zzz"}]`,
			`[{"op":"add","path":"/1","value":{"asdf":"zzz"}},{"op":"replace","path":"/0/asdf","value":"bla"}]`,
		},
		{
			"from empty array",
			`[]`,
			`[{"asdf":"bla"},{"asdf":"zzz"}]`,
			`[{"op":"add","path":"/0","value":{"asdf":"zzz"}},{"op":"add","path":"/0","value":{"asdf":"bla"}}]`,
		},
		{
			"to empty array",
			`[{"asdf":"bla"},{"asdf":"zzz"}]`,
			`[]`,
			`[{"op":"remove","path":"/1"},{"op":"remove","path":"/0"}]`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			patch, err := CreatePatchFromBytes([]byte(tc.a), []byte(tc.b))
			require.NoError(t, err)

			actual, err := json.Marshal(patch)
			require.NoError(t, err)

			require.Equal(t, tc.expected, string(actual))
		})
	}
}

func check(t *testing.T, src, dst string) {
	patch, err := CreatePatchFromBytes([]byte(src), []byte(dst))
	assert.Nil(t, err)

	data, err := json.Marshal(patch)
	assert.Nil(t, err)

	p2, err := jp.DecodePatch(data)
	assert.Nil(t, err)

	d2, err := p2.Apply([]byte(src))
	assert.Nil(t, err)

	assert.JSONEq(t, dst, string(d2))
}

var (
	arrayBase = `{
  "persons": [{"name":"Ed"},{}]
}`

	arrayUpdated = `{
  "persons": [{"name":"Ed"},{},{}]
}`
)

func TestArrayAddMultipleEmptyObjects(t *testing.T) {
	patch, e := CreatePatchFromBytes([]byte(arrayBase), []byte(arrayUpdated))
	require.NoError(t, e)
	t.Log("Patch:", patch)
	require.Equal(t, 1, len(patch), "they should be equal")
	sort.Sort(ByPath(patch))

	change := patch[0]
	require.Equal(t, "add", change.Operation, "they should be equal")
	require.Equal(t, "/persons/2", change.Path, "they should be equal")
	require.Equal(t, map[string]any{}, change.Value, "they should be equal")
}

func TestArrayRemoveMultipleEmptyObjects(t *testing.T) {
	patch, e := CreatePatchFromBytes([]byte(arrayUpdated), []byte(arrayBase))
	require.NoError(t, e)
	t.Log("Patch:", patch)
	require.Equal(t, 1, len(patch), "they should be equal")
	sort.Sort(ByPath(patch))

	change := patch[0]
	require.Equal(t, "remove", change.Operation, "they should be equal")
	require.Equal(t, "/persons/2", change.Path, "they should be equal")
	require.Equal(t, nil, change.Value, "they should be equal")
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)

	return string(b)
}

func TestMarshalNullableValue(t *testing.T) {
	p1 := Operation{
		Operation: "replace",
		Path:      "/a1",
		Value:     nil,
	}
	require.JSONEq(t, `{"op":"replace", "path":"/a1","value":null}`, toJSON(&p1))

	fmt.Println(toJSON(p1))

	p2 := Operation{
		Operation: "replace",
		Path:      "/a2",
		Value:     "v2",
	}
	require.JSONEq(t, `{"op":"replace", "path":"/a2", "value":"v2"}`, toJSON(&p2))
}

func TestMarshalNonNullableValue(t *testing.T) {
	p1 := Operation{
		Operation: "remove",
		Path:      "/a1",
	}
	require.JSONEq(t, `{"op":"remove", "path":"/a1"}`, toJSON(p1))

}
