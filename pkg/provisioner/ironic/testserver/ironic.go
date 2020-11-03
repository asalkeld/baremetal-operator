package testserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/baremetal/v1/nodes"
)

// CreatedNode holds the body of the request to create the node and
// the details generated by the server and included in the response.
type CreatedNode struct {
	Body string
	UUID string
}

// IronicMock is a test server that implements Ironic's semantics
type IronicMock struct {
	*MockServer
	CreatedNodes []CreatedNode
}

// NewIronic builds an ironic mock server
func NewIronic(t *testing.T) *IronicMock {

	return &IronicMock{
		MockServer:   New(t, "ironic"),
		CreatedNodes: nil,
	}
}

// WithDefaultResponses sets a valid answer for all the API calls
func (m *IronicMock) WithDefaultResponses() *IronicMock {
	m.AddDefaultResponseJSON("/v1/nodes/{id}", "", http.StatusOK, nodes.Node{
		UUID: "{id}",
	})
	m.AddDefaultResponse("/v1/nodes/{id}/states/provision", "", http.StatusAccepted, "{}")
	m.AddDefaultResponse("/v1/nodes/{id}/states/power", "", http.StatusAccepted, "{}")
	m.AddDefaultResponse("/v1/nodes/{id}/validate", "", http.StatusOK, "{}")
	m.Ready()

	return m
}

// Endpoint returns the URL for accessing the server
func (m *IronicMock) Endpoint() string {
	if m == nil {
		return "https://ironic.test/v1/"
	}
	return m.MockServer.Endpoint()
}

// Ready configures the server with a valid response for /v1
func (m *IronicMock) Ready() *IronicMock {
	m.Response("/v1", "{}")
	return m
}

// NotReady configures the server with an error response for /v1
func (m *IronicMock) NotReady(errorCode int) *IronicMock {
	m.ErrorResponse("/v1", errorCode)
	return m
}

// WithDrivers configures the server so /v1/drivers returns a valid value
func (m *IronicMock) WithDrivers() *IronicMock {
	m.Response("/v1/drivers", `
	{
		"drivers": [{
			"hosts": [
			  "master-2.ostest.test.metalkube.org"
			],
			"links": [
			  {
				"href": "http://[fd00:1101::3]:6385/v1/drivers/fake-hardware",
				"rel": "self"
			  },
			  {
				"href": "http://[fd00:1101::3]:6385/drivers/fake-hardware",
				"rel": "bookmark"
			  }
			],
			"name": "fake-hardware"
		}]
	}
	`)
	return m
}

func (m *IronicMock) buildURL(url string, method string) string {
	return fmt.Sprintf("%s:%s", url, method)
}

func (m *IronicMock) withNode(node nodes.Node, method string) *IronicMock {

	if node.UUID != "" {
		m.ResponseJSON(m.buildURL("/v1/nodes/"+node.UUID, method), node)
	}
	if node.Name != "" {
		m.ResponseJSON(m.buildURL("/v1/nodes/"+node.Name, method), node)
	}
	return m
}

// WithNode configures the server with a valid response for [GET] /v1/nodes
func (m *IronicMock) WithNode(node nodes.Node) *IronicMock {
	return m.withNode(node, http.MethodGet)
}

// WithNodeUpdate configures the server with a valid response for [PATCH] /v1/nodes
func (m *IronicMock) WithNodeUpdate(node nodes.Node) *IronicMock {
	return m.withNode(node, http.MethodPatch)
}

func (m *IronicMock) withNodeStatesProvision(nodeUUID string, method string) *IronicMock {
	m.ResponseWithCode(m.buildURL("/v1/nodes/"+nodeUUID+"/states/provision", method), "{}", http.StatusAccepted)
	return m
}

// WithNodeStatesProvision configures the server with a valid response for [GET] /v1/nodes/<node>/states/provision
func (m *IronicMock) WithNodeStatesProvision(nodeUUID string) *IronicMock {
	return m.withNodeStatesProvision(nodeUUID, http.MethodGet)
}

// WithNodeStatesProvision configures the server with a valid response for [PATCH] /v1/nodes/<node>/states/provision
func (m *IronicMock) WithNodeStatesProvisionUpdate(nodeUUID string) *IronicMock {
	return m.withNodeStatesProvision(nodeUUID, http.MethodPut)
}

// NoNode configures the server so /v1/nodes/name returns a 404
func (m *IronicMock) NoNode(name string) *IronicMock {
	return m.NodeError(name, http.StatusNotFound)
}

// NodeError configures the server to return the specified error code for /v1/nodes/name
func (m *IronicMock) NodeError(name string, errorCode int) *IronicMock {
	m.ErrorResponse(fmt.Sprintf("/v1/nodes/%s", name), errorCode)
	return m
}

// CreateNodes configures the server so POSTing to /v1/nodes saves the data
func (m *IronicMock) CreateNodes() *IronicMock {
	m.Handler("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		bodyRaw, err := ioutil.ReadAll(r.Body)
		if err != nil {
			m.logRequest(r, fmt.Sprintf("ERROR: %s", err))
			http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
			return
		}

		body := string(bodyRaw)
		m.t.Logf("%s: create nodes request %v", m.name, body)

		// The UUID value doesn't actually have to be a UUID, so we
		// just make a new string based on the count of nodes already
		// created.
		uuid := fmt.Sprintf("node-%d", len(m.CreatedNodes))
		m.t.Logf("%s: uuid %s", m.name, uuid)

		// Record what we have so the test can examine it later
		m.CreatedNodes = append(m.CreatedNodes, CreatedNode{
			Body: body,
			UUID: uuid,
		})

		// hackily add uuid to the json response by inserting it to the front of the string
		response := fmt.Sprintf("{\"uuid\": \"%s\", %s", uuid, strings.TrimLeft(body, "{"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, response)
		m.logRequest(r, response)
	})
	return m
}

func (m *IronicMock) withNodeStatesPower(nodeUUID string, code int, method string) *IronicMock {
	m.ResponseWithCode(m.buildURL("/v1/nodes/"+nodeUUID+"/states/power", method), "{}", code)
	return m
}

// WithNodeStatesPower configures the server with a valid response for [GET] /v1/nodes/<node>/states/power
func (m *IronicMock) WithNodeStatesPower(nodeUUID string, code int) *IronicMock {
	return m.withNodeStatesPower(nodeUUID, code, http.MethodGet)
}

// WithNodeStatesPowerUpdate configures the server with a valid response for [PUT] /v1/nodes/<node>/states/power
func (m *IronicMock) WithNodeStatesPowerUpdate(nodeUUID string, code int) *IronicMock {
	return m.withNodeStatesPower(nodeUUID, code, http.MethodPut)
}

// WithNodeValidate configures the server with a valid response for /v1/nodes/<node>/validate
func (m *IronicMock) WithNodeValidate(nodeUUID string) *IronicMock {
	m.ResponseWithCode("/v1/nodes/"+nodeUUID+"/validate", "{}", http.StatusOK)
	return m
}
