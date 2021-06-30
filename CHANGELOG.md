
# v0.4.0 (Cafe Mocha) (2021.06.30.)

### Feature

- 기본값 설정 정리, timeout 등 승계 처리 및 코드 재 정리 ([#45](https://github.com/cloud-barista/cb-apigw/pull/45))
- Backend Error with Response (bypass, return_error_detils) 반영 ([#47](https://github.com/cloud-barista/cb-apigw/pull/47))


### Bug Fix

- Api definition file 수정시 Process Kill 발생 오류 수정 및 기타 조정 ([#43](https://github.com/cloud-barista/cb-apigw/pull/43))
- 컨테이너로 실행 시 "no such file or directory" 에러 발생 수정 ([#49](https://github.com/cloud-barista/cb-apigw/pull/49))
- conf/apis/*.yaml 파일 수정시 restapigw 강제 종료됨. 처리 적용 ([#54](https://github.com/cloud-barista/cb-apigw/pull/54))


# v0.3.5

[conf/apis/*.yaml 파일 수정시 restapigw 강제 종료됨. 처리 적용 #54]

- conf/apis/*.yaml 파일 수정 시 restapigw 강제 종료되는 이슈 처리
  - ubuntu 에서 파일 수정 시 변경 이벤트 (Filesystem Watcher: github.com/fsnotify/fsnotify) 2번 발생
  - 변경 이벤트 처리 시에 파일의 내용이 없는 경우는 Skip 로직 추가
- grafana 폴더/파일 퍼미션 관련
  - README.md 파일에 관련된 내용 기술
- yaml.v2 관련
  - yaml.v3 로 관련 소스 수정

# v0.3.4

[컨테이너로 실행 시 "no such file or directory" 에러 발생 수정 (#49)]

- Dockerfile 및 docker-compose.yaml 파일 수정
- InfluxDB 및 Grafana 관련 수정

# v0.3.3

[#47: Backend Error with Response (bypass, return_error_detils) 반영]

> **참고**
> 오류이면서 Response가 존재하는 경우는 기존의 Bypass 처리 (API G/W 는 전달/반환만 하고 가공을 하지 않는) 로 한정해서 그대로 반환하도록 처리하여 작업 기준을 단순화 했습니다.
> 
> 따라서 이번에 작업할 대상을 전체가 아닌 Bypass 인 경우로 한정하여 처리합니다. (이 방식이면 전용과 범용을 모두 만족시킬 수 있을 것 같습니다)
> 
> 테스트는 deploy 폴더의 docker-compose를 실행해서 테스트 가능 (apis/dev-test.yaml API 들 대상)
> 
> **단, Backend를 호출하지도 못하는 경우 (ex. Host / URL 잘못 지정 또는 서버 다운 상태 등) 는 기존 오류 상태로 처리 됩니다.**

- Backend 호출 오류 상태인데 Response가 존재하는 경우 반영 기준
  - Bypass 인 경우는 Backend에서 받은 그대로 반환 (단일 Backend)
  - Bypass 가 아닌 경우는 "return_error_details" 라는 미들웨어 옵션을 사용해서 오류 발생 정보를 Group 처리하여 다른 정상 Backend 결과와 Merge 가 가능하도록 처리 
  - Bypass도 아니고 미들웨어 옵션도 사용하지 않으면 기존처럼 오류 처리하고 Header로 오류 정보 반환
- 기타
   - yoda condition 코드 변경
   - Metrics 수집처리할 떄 "pipe" 라는 Layer 명칭을 "proxy" 로 통일
   - apis/dev-test.yaml 에 테스트용 API 추가

# v0.3.2

[#43 : Api definition file 수정시 Process Kill 발생 오류 수정 및 기타 조정]
- API Definition file 수정시 Process Kill 발생 문제
  - 원인 : time.Duration 설정시 Unit 이 생략된 경우 Yaml 파싱 오류 발생을 캐치하였으나 다음 단계로 진행되는 것을 방지하는 코드 누락
  - 해결 : `오류 캐치 후 다음 단계로 진행 방지하고 로그에 파싱 오류로 무시된다고 출력`
- 기타 조정
  - api/cb-restapigw-apis.yaml 파일에서 개발용 테스트 api 부분을 `apis/dev-test.yaml` 로 분리
  - HMAC 서버의 echo 버전 Upgrade to v4.2.1
  - 배열 요소 복제 loop 단순화
  - InfluxDB 버전 고정 v1.8.4

---

[#45 : 기본값 설정 정리, timeout 등 승계 처리 및 코드 재 정리]
- 기본값 설정 관련 코드 정리 및 API Definition 파일 정리
   -  설정 파일 (Conf)에 미 지정된 항목들에 대한 기본 값 설정 처리 조정
   - API Definition 파일들의 기본 설정들 제거하고 변경되는 항목만 설정 (cb-restapigw.yaml, apis/cb-restapigw.apis.yaml, apis/dev-test.yaml, ...)

- Timout 정보 등의 미 지정시 승계 처리
   - Timeout 
     - 서비스 설정에만 지정 시 Endpoint, Backend로 승계
     - Endpoint에만 지정 시 Backend로 승계
   - OutputEncoding
     - 서비스 설정에만 지정 시 Endpoint로 승계
   - CacheTTL
     - 서비스 설정에만 지정 시 Endpoint로 승계
   - Host
     - Endpoint에만 지정 시 Backend로 승계 
   - Method
     - Endpoint에만 지정 시 Backend로 승계
  - 모든 항목들은 미 지정시 승계 또는 기본 값 사용 (Struct 에 Tag Option으로 기본 값 지정)

- Grafana Upgrade 관련 조정
  - v7.4.x 버전
  - Provisioning 관련 Dashboard 설정, Datasource 설정 변경

- Go Lint 기반 코드 정리 (vscode go extension - staticcheck)
  - Modify : yoda conditions, variable declaration with assignment, ...
  - Remove : unused variables, functions, redundant statements
  - Remove : unnecessary use of functions or expressions
  - Remove : SSL v3.0 deprecated since go 1.13

# v0.3.0-espresso (2020.12.10.)

### Feature & Bug Fix
- Add Docker build test & push logic (#10)
- 2020년 10월 변경사항 반영 (#12)
  - 구조 개선
  - 설정 분리 (System, APIs)
  - Repository 추가
  - cb-store 연계 (System 설정과 store.conf 파일로 NutsDB/ETCD 활용)
  - 동적 변경 반영 추가
  - Admin API 추가
  - Admin Web 추가
  - Load Balancer 기능 추가
- cb-restapigw Metric 수집 검증 및 소스 정리 (#13)
  - Router 단위에서 Metric 수집이 누락되는 부분 검증 및 정리
  - Metrics Structure 구현 method 단일화 (Router & Proxy Metrics)
  - logging 출력시 각 영역별 Prefix 문자 추가
  - log_conf.yaml 내에 $CBLOG_ROOT 환경변수 명 수정 ($CBSTORE_ROOT to $CBLOG_ROOT)
- cb-restapigw 환경 및 모호한 이름 변경, 파싱 오류 반환 추가 (#14)
  - go version upgrade to 1.15
  - yaml.v2, yaml.v3 혼용되는 부분이 발견되어 yaml.v3 로 통일
  - Definition 파싱할 떄 발생할 수 있는 오류 검증 및 반환 구조로 변경
  - Admin Web에서 time.Duration 관련 수정 반영


# v0.2.0-cappuccino (2020.06.02.)

### Feature
- JSON array 관련 ([e62b3c1](https://github.com/cloud-barista/cb-apigw/commit/e62b3c19b8ee9051573f376601d893dba455fa92))
    -  `is_collection: true` 인 경우
        - `wrap_collection_to_json: true` 인 경우: 응답을 `"collection"` 이라는 필드의 객체 형식으로 반환
        - `wrap_collection_to_json: false` 인 경우: 응답을 Array 형태로 반환
- Bypass 기능 추가 ([4573e84](https://github.com/cloud-barista/cb-apigw/commit/4573e8492a7fa22026fb6be4183cdc770eb80778))
- Query param, HTTP header 의 전달 정책을 whitelist 에서 blacklist 로 변경
- Rate Limit 기능 추가 ([1d9911b](https://github.com/cloud-barista/cb-apigw/commit/1d9911ba83057e3d708fba0731f2d33aec555729))

### Bug Fix
- API call 의 Query param 을 forward 하지 않는 오류 수정 ([0dc7753](https://github.com/cloud-barista/cb-apigw/commit/0dc775362cd5011adf851d598f83a10763b70f32))

# v0.1.0-americano (2019.12.23.)

### Feature
- cb-restapigw 공개
