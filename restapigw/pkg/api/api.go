// Package api - ADMIN API 기능을 제공하는 패키지
package api

import (
	"net/http"
	"reflect"

	"github.com/cloud-barista/cb-apigw/restapigw/pkg/config"
	"github.com/cloud-barista/cb-apigw/restapigw/pkg/errors"
)

// ===== [ Constants and Variables ] =====

const (
	// RemovedOperation - 설정 제거 작업
	RemovedOperation ConfigurationOperation = iota
	// UpdatedOperation - 설정 변경 작업
	UpdatedOperation
	// AddedOperation - 설정 등록 작업
	AddedOperation
	// RemovedSourceOperation - 소스 제거 작업
	RemovedSourceOperation
	// AddedSourceOperation - 소스 추가 작업
	AddedSourceOperation
	// ApplySourcesOperation - 설정 변경사항 모두 저장 (File or ETCD, ...)
	ApplySourcesOperation
)

var (
	// ErrAPIDefinitionNotFound - 레파지토리에 API 정의가 존재하지 않는 경우 오류
	ErrAPIDefinitionNotFound = errors.NewWithCode(http.StatusNotFound, "api definition not found")
	// ErrAPIsNotChanged - 레파지토리에 저장할 API 정의 변경 사항이 존재하지 않는 경우 오류
	ErrAPIsNotChanged = errors.NewWithCode(http.StatusNotModified, "api definitions are not changed")
	// ErrAPINameExists - 레파지토리에 동일한 이름의 API 정의가 존재하는 경우 오류
	ErrAPINameExists = errors.NewWithCode(http.StatusConflict, "api name is already registered")
	// ErrAPIListenPathExists - 레파지토리에 동일한 수신 경로의 API 정의가 존재하는 경우 오류
	ErrAPIListenPathExists = errors.NewWithCode(http.StatusConflict, "api listen path is already registered")
	// ErrSourceExists - 리파지토리에 동일한 이름의 소스가 존재하는 경우 오류
	ErrSourceExists = errors.NewWithCode(http.StatusConflict, "api source is already registered")
	// ErrSourceNotExists - 리파지토리에 동일한 이름의 소스가 존재하지 않는 경우 오류
	ErrSourceNotExists = errors.NewWithCode(http.StatusNotFound, "api source not found")

	// TODO: ETCD, Database 관련 오류들
	// ErrDBContextNotSet is used when the database request context is not set
	// ErrDBContextNotSet = errors.NewWithCode(http.StatusInternalServerError, "DB context was not set for this request")
)

// ===== [ Types ] =====

type (
	// ConfigurationOperation - Configuration 변경에 연계되는 Operation 형식
	ConfigurationOperation int

	// ConfigModel - 클라이언트와 통신에 사용할 정보 구조
	ConfigModel struct {
		Source     string                 `json:"source"`
		Definition *config.EndpointConfig `json:"definition"`
	}

	// Configuration - API Definitions 관리 구조
	Configuration struct {
		status         string
		DefinitionMaps []*DefinitionMap
	}

	// ConfigurationChanged - Configuration 변경이 발생했을 떄 전송되는 메시지 처리 형식
	ConfigurationChanged struct {
		Configurations *Configuration
	}

	// ChangeMessage - Configuration 변경시 사용할 메시지 형식
	ChangeMessage struct {
		Operation  ConfigurationOperation
		Source     string
		Definition *config.EndpointConfig
	}
)

// ===== [ Implementations ] =====

// EqualsTo - 현재 동작 중인 설정과 지정된 설정이 동일한지 여부 검증
func (c *Configuration) EqualsTo(tc *Configuration) bool {
	return reflect.DeepEqual(c, tc)
}

// GetAllDefinitions - 관리하고 있는 API Definition들 반환
func (c *Configuration) GetAllDefinitions() []*config.EndpointConfig {
	defs := make([]*config.EndpointConfig, 0)
	for _, dm := range c.DefinitionMaps {
		if dm.State != "REMOVED" {
			for _, def := range dm.Definitions {
				defs = append(defs, def)
			}
		}
	}
	return defs
}

// GetDefinitionMaps - 관리하고 있는 API Definition Map들 반환
func (c *Configuration) GetDefinitionMaps() []*DefinitionMap {
	maps := make([]*DefinitionMap, 0)
	for _, dm := range c.DefinitionMaps {
		if dm.State != "REMOVED" {
			maps = append(maps, dm)
		}
	}

	return maps
}

// ExistSource - 지정한 Source가 존재하는지 검증
func (c *Configuration) ExistSource(source string) bool {
	for _, dm := range c.DefinitionMaps {
		if dm.Source == source {
			return true
		}
	}
	return false
}

// Exists - 지정한 Definition이 존재하는지 검증
func (c *Configuration) Exists(source string, ec *config.EndpointConfig) (bool, error) {
	for _, dm := range c.DefinitionMaps {
		if dm.Source == source {
			for _, def := range dm.Definitions {
				if def.Name == ec.Name {
					return true, ErrAPINameExists
				}

				if def.Endpoint == ec.Endpoint {
					return true, ErrAPIListenPathExists
				}
			}
		} else {
			return false, nil
		}
	}

	return false, nil
}

// FindByName - 지정한 이름의 Endpoint Definition이 존재하는지 검증 (동일 소스 대상)
func (c *Configuration) FindByName(source, name string) *config.EndpointConfig {
	for _, dm := range c.DefinitionMaps {
		if dm.Source == source {
			for _, def := range dm.Definitions {
				if def.Name == name {
					return def
				}
			}
		}
	}

	return nil
}

// FindByListenPath - 지정한 Path의 Endpoint Definition이 존재하는 검증 (전체 대상)
func (c *Configuration) FindByListenPath(listenPath string) *config.EndpointConfig {
	for _, dm := range c.DefinitionMaps {
		for _, def := range dm.Definitions {
			if def.Endpoint == listenPath {
				return def
			}
		}
	}

	return nil
}

// AddDefinition - 지정한 정보를 기준으로 관리 중인 API Defintion 추가
func (c *Configuration) AddDefinition(source string, ec *config.EndpointConfig) error {
	for _, dm := range c.DefinitionMaps {
		if dm.Source == source {
			dm.Definitions = append(dm.Definitions, ec)
			dm.State = "CHANGED"
			return nil
		}
	}
	return errors.New("Specific source path not exist [" + source + "]")
}

// UpdateDefinition - 지정한 정보를 기준으로 관리 중인 API Definition 갱신
func (c *Configuration) UpdateDefinition(source string, ec *config.EndpointConfig) error {
	for _, dm := range c.DefinitionMaps {
		if dm.Source == source {
			for i, def := range dm.Definitions {
				if def.Name == ec.Name {
					dm.Definitions[i] = ec
					dm.State = "CHANGED"
					return nil
				}
			}
		}
	}

	return errors.New("No definition to update in source path [" + source + "]")
}

// RemoveDefinition - 지정한 정보를 기준으로 관리 중인 API Definition 삭제
func (c *Configuration) RemoveDefinition(source string, ec *config.EndpointConfig) error {
	for _, dm := range c.DefinitionMaps {
		if dm.Source == source {
			for i, def := range dm.Definitions {
				if def.Name == ec.Name {
					copy(dm.Definitions[1:], dm.Definitions[i+1:])
					dm.Definitions = dm.Definitions[:len(dm.Definitions)-1]
					dm.State = "CHANGED"
					return nil
				}
			}
		}
	}
	return errors.New("No defintion to remove in source path [" + source + "]")
}

// AddSource - 지정한 정보를 기준으로 API Source 생성
func (c *Configuration) AddSource(source string) error {
	c.DefinitionMaps = append(c.DefinitionMaps, &DefinitionMap{Source: source, State: "ADD", Definitions: make([]*config.EndpointConfig, 0)})
	return nil
}

// RemoveSource - 지정한 정보를 기준으로 API Source 삭제
func (c *Configuration) RemoveSource(source string) error {
	for _, dm := range c.DefinitionMaps {
		if dm.Source == source {
			dm.State = "REMOVED"
			return nil
		}
	}
	return errors.New("No source to remove in sources [" + source + "]")
}

// ClearRemoved - 현재 관리 중인 API Defintion Soruce들 중에서 삭제된 내용을 제거
func (c *Configuration) ClearRemoved() {
	for i, dm := range c.DefinitionMaps {
		if dm.State == "REMOVED" {
			copy(c.DefinitionMaps[1:], c.DefinitionMaps[i+1:])
			c.DefinitionMaps = c.DefinitionMaps[:len(c.DefinitionMaps)-1]
		}
	}
}

// ===== [ Private Functions ] =====
// ===== [ Public Functions ] =====

// NewDefinition - 기본 값으로 설정된 API Routing Endpoint 인스턴스 생성
func NewDefinition() *config.EndpointConfig {
	return &config.EndpointConfig{}
}
