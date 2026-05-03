// =============================================================================
// Jenkinsfile — CI для users-go (лабораторный контур: Jenkins + Sonar + registry + Minikube)
// =============================================================================
// Поток стадий:
//   1. Checkout — код из SCM.
//   2. Go test + coverage — ставим Go в агенте, go test, coverage.out (без docker run -v workspace: демон не видит файлы).
//   3. SonarQube — sonar-scanner + Node (сенсоры), токен из credentials.
//   4. Docker build and push — docker build локальный тег; Skopeo push в HTTP registry (без insecure-registry на демоне).
//   5. Deploy to Minikube (опционально) — kubectl apply k8s/user-service-registry.yaml; см. комментарии внутри stage.
//
// Параметры ниже задают registry, имя образа, включение деплоя, БД и т.д. (подробности — в description каждого параметра).
// =============================================================================

pipeline {
  agent any

  parameters {
    // --- SonarQube: доп. аргументы sonar-scanner (переопределение properties из UI) ---
    string(
      name: 'SONAR_EXTRA_OPTS',
      defaultValue: '',
      description: 'Доп. аргументы sonar-scanner (переопределяют properties), напр. -Dsonar.projectKey=КЛЮЧ_ИЗ_SONAR_UI'
    )
    // --- Docker registry / имя образа (Skopeo push после docker build) ---
    string(
      name: 'DOCKER_REGISTRY',
      defaultValue: 'host.docker.internal:5050',
      description: 'Docker registry host:port. Jenkins в Docker → host.docker.internal:5050; агент на хосте → localhost:5050'
    )
    string(
      name: 'DOCKER_IMAGE',
      defaultValue: 'users-go',
      description: 'Имя образа в реестре (без registry-префикса)'
    )
    // --- Kubernetes / Minikube: включение деплоя, контейнер minikube, registry для pull-манифеста ---
    booleanParam(
      name: 'DEPLOY_MINIKUBE',
      defaultValue: false,
      description: 'После push: kubectl apply в Minikube. Либо ~/.kube в Jenkins, либо (fallback) docker exec в контейнер minikube при общем docker.sock'
    )
    string(
      name: 'MINIKUBE_CONTAINER',
      defaultValue: 'minikube',
      description: 'Имя контейнера Minikube (docker ps), для kubectl через docker exec если нет JENKINS_HOME/.kube/config'
    )
    string(
      name: 'K8S_PULL_REGISTRY',
      defaultValue: 'host.minikube.internal:5050',
      description: 'Registry host:port для образа в манифесте, если K8S_CTR_IMPORT_IMAGE=false'
    )
    // Локальная загрузка образа в dockerd ноды minikube (см. комментарий в stage Deploy).
    booleanParam(
      name: 'K8S_CTR_IMPORT_IMAGE',
      defaultValue: true,
      description: 'Только при docker exec minikube: docker save | docker load внутри ноды (cri-dockerd); imagePullPolicy Never. Не использовать ctr — kubelet не видит k8s.io import'
    )
    // --- БД для подов user-service (плейсхолдеры в k8s/user-service-registry.yaml) ---
    string(
      name: 'K8S_DB_HOST',
      defaultValue: 'postgres',
      description: 'DB host: Service postgres в market (k8s/postgres-market.yaml) или host.minikube.internal для Postgres на Mac'
    )
    string(
      name: 'K8S_DB_PORT',
      defaultValue: '5432',
      description: 'Порт Postgres: 5432 in-cluster; 5433 если БД на хосте (property.yaml)'
    )
  }

  // Переменные окружения для стадий Sonar/Go (версии инструментов; SONAR_HOST_URL — к Sonar в Docker на хосте).
  environment {
    SONAR_HOST_URL = 'http://host.docker.internal:9000'
    // Совпадает с `toolchain` в go.mod (users-go).
    GO_VERSION = '1.24.11'
    SONAR_SCANNER_VERSION = '8.0.1.6346'
    // Для сенсоров JS/TS/CSS Sonar нужен Node.js в PATH.
    NODE_JS_VERSION = '20.18.1'
  }

  stages {
    // --- Клонирование репозитория ---
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    // --- Тесты Go и покрытие (артефакт coverage.out для Sonar) ---
    stage('Go test + coverage') {
      steps {
        // GOTOOLCHAIN=local до любого вызова go: иначе под auto может подтянуться другой toolchain, чем бинарь в /usr/local/go.
        // Проверяем только /usr/local/go/bin/go — не «go» из PATH с другим поведением.
        sh """#!/bin/bash
set -eux
GO_VER='${env.GO_VERSION ?: '1.24.11'}'
export GOTOOLCHAIN=local
export PATH="/usr/local/go/bin:\${PATH}"

ARCH="\$(uname -m)"
case "\$ARCH" in
  aarch64|arm64) GOARCH=arm64 ;;
  x86_64) GOARCH=amd64 ;;
  *) echo "unsupported arch: \$ARCH"; exit 1 ;;
esac

if [ -x /usr/local/go/bin/go ] && /usr/local/go/bin/go version 2>/dev/null | grep -qF "go\${GO_VER}"; then
  echo "Go already at \${GO_VER} under /usr/local/go"
else
  GOURL="https://go.dev/dl/go\${GO_VER}.linux-\${GOARCH}.tar.gz"
  # Повреждённый .tar.gz (обрыв сети/прокси) даёт «gzip: invalid compressed data» — проверяем gzip до rm /usr/local/go.
  for attempt in 1 2 3; do
    echo "Downloading Go \${GO_VER} (\${GOARCH}), attempt \${attempt}"
    curl -fSL --connect-timeout 30 --max-time 600 --retry 5 --retry-delay 2 "\${GOURL}" -o /tmp/go.tgz
    if gzip -t /tmp/go.tgz 2>/dev/null; then break; fi
    echo "go.tgz is not valid gzip, retrying"
    rm -f /tmp/go.tgz
    if [ "\${attempt}" -eq 3 ]; then echo "Giving up after 3 attempts"; exit 1; fi
  done
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tgz
fi

export GOROOT=/usr/local/go
export GOTOOLCHAIN=local

go version
cd "\${WORKSPACE}"
go test ./... -coverprofile=coverage.out -covermode=atomic
"""
      }
    }

    // --- Статический анализ SonarQube (sonar-project.properties в корне); при FAILED Quality Gate сканер падает с кодом 3 ---
    stage('SonarQube analysis') {
      environment {
        SONAR_TOKEN = credentials('sonarqube-token-user-go')
      }
      steps {
        sh """#!/bin/bash
set -eux
if ! command -v curl >/dev/null 2>&1 || ! command -v unzip >/dev/null 2>&1 || ! command -v xz >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq curl ca-certificates unzip xz-utils
fi

ARCH="\$(uname -m)"
case "\$ARCH" in
  aarch64|arm64) ZIP_ARCH=aarch64; NODE_DIST_ARCH=arm64 ;;
  x86_64) ZIP_ARCH=x64; NODE_DIST_ARCH=x64 ;;
  *) echo "unsupported arch: \$ARCH"; exit 1 ;;
esac

NODE_VER='${env.NODE_JS_VERSION ?: '20.18.1'}'
NODE_BASE="node-v\${NODE_VER}-linux-\${NODE_DIST_ARCH}"
NODE_ROOT="/usr/local/\${NODE_BASE}"
if [ ! -x "\${NODE_ROOT}/bin/node" ]; then
  curl -fsSL "https://nodejs.org/dist/v\${NODE_VER}/\${NODE_BASE}.tar.xz" -o /tmp/node.txz
  tar -C /usr/local -xJf /tmp/node.txz
fi
export PATH="\${NODE_ROOT}/bin:\${PATH}"
node -v

ZIP="sonar-scanner-cli-${env.SONAR_SCANNER_VERSION}-linux-\${ZIP_ARCH}.zip"
URL="https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/\${ZIP}"
curl -fsSL "\$URL" -o "/tmp/\${ZIP}"
# Родитель нельзя называть sonar-scanner-* — find совпадёт с ним раньше, чем с каталогом из zip.
SCANNER_ROOT=/tmp/ss-unpack
rm -rf "\${SCANNER_ROOT}"
mkdir -p "\${SCANNER_ROOT}"
unzip -q -o "/tmp/\${ZIP}" -d "\${SCANNER_ROOT}"
SCANNER_HOME="\$(find "\${SCANNER_ROOT}" -maxdepth 1 -mindepth 1 -type d -name 'sonar-scanner-*' | head -1)"
test -x "\${SCANNER_HOME}/bin/sonar-scanner"

cd "\${WORKSPACE}"
"\${SCANNER_HOME}/bin/sonar-scanner" \\
  -Dsonar.host.url="${env.SONAR_HOST_URL}" \\
  -Dsonar.token="\${SONAR_TOKEN}" ${params.SONAR_EXTRA_OPTS?.trim() ?: ''}
"""
      }
    }

    // --- Сборка образа на Docker-демоне агента и публикация в registry (Skopeo, теги BUILD_NUMBER и latest) ---
    stage('Docker build and push') {
      steps {
        sh """#!/bin/bash
set -eux
command -v docker
# Сборка классическим builder; push в HTTP registry:2 через skopeo (--dest-tls-verify=false), без insecure-registries на демоне.
export DOCKER_BUILDKIT=0
export BUILDKIT_PROGRESS=plain

cd "\${WORKSPACE}"
REG='${params.DOCKER_REGISTRY}'
NAME='${params.DOCKER_IMAGE}'
TAG='${env.BUILD_NUMBER}'
FULL="\${REG}/\${NAME}:\${TAG}"
LATEST="\${REG}/\${NAME}:latest"
LOCAL_TAG="jenkins-\${TAG}"

docker build -t "\${NAME}:\${LOCAL_TAG}" .

SKOPEO_IMG='quay.io/skopeo/stable:latest'
docker pull -q "\${SKOPEO_IMG}" || docker pull "\${SKOPEO_IMG}"

# Skopeo в контейнере: на Linux-агенте добавляем host.docker.internal → host-gateway (доступ к registry на хосте).
HOST_ARGS=()
if docker run --help 2>&1 | grep -qF 'host-gateway'; then
  HOST_ARGS=(--add-host=host.docker.internal:host-gateway)
fi

# Копия образа с docker-daemon (локальный тег) в HTTP registry (--dest-tls-verify=false).
run_skopeo_copy() {
  local dest="\$1"
  docker run --rm \\
    "\${HOST_ARGS[@]}" \\
    -v /var/run/docker.sock:/var/run/docker.sock \\
    "\${SKOPEO_IMG}" \\
    copy --dest-tls-verify=false \\
    "docker-daemon:\${NAME}:\${LOCAL_TAG}" \\
    "docker://\${dest}"
}

run_skopeo_copy "\${FULL}"
run_skopeo_copy "\${LATEST}"
"""
      }
    }

    // --- Деплой в Minikube: kubectl apply из k8s/user-service-registry.yaml (только если DEPLOY_MINIKUBE=true) ---
    stage('Deploy to Minikube') {
      when {
        expression { return params.DEPLOY_MINIKUBE }
      }
      steps {
        sh """#!/bin/bash
set -eux
cd "\${WORKSPACE}"
command -v docker

# --- Контекст: kubeconfig в Jenkins или доступ к контейнеру minikube по docker.sock ---
JHOME="\${JENKINS_HOME:-\$HOME}"
MK='${params.MINIKUBE_CONTAINER}'
USE_DOCKER_EXEC=0
KUBECTL=""

# Ветка A: есть ~/.kube/config в агенте — kubectl скачивается в /tmp, kubeconfig правится (127.0.0.1 → host.docker.internal для API из контейнера).
if [ -f "\$JHOME/.kube/config" ]; then
  ARCH="\$(uname -m)"
  case "\$ARCH" in aarch64|arm64) KARCH=arm64 ;; x86_64) KARCH=amd64 ;; *) echo "unsupported arch: \$ARCH"; exit 1 ;; esac
  KVER="\$(curl -fsSL https://dl.k8s.io/release/stable.txt)"
  KUBECTL="/tmp/kubectl-\${KVER}"
  if [ ! -x "\$KUBECTL" ]; then
    curl -fSL "https://dl.k8s.io/release/\${KVER}/bin/linux/\${KARCH}/kubectl" -o "\$KUBECTL"
    chmod +x "\$KUBECTL"
  fi
  KCFG="/tmp/kubeconfig-jenkins-\${BUILD_NUMBER}"
  cp "\$JHOME/.kube/config" "\$KCFG"
  export KUBECONFIG="\$KCFG"
  sed -i.bak 's|127.0.0.1|host.docker.internal|g' "\$KCFG"
  CLUSTER="\$( "\$KUBECTL" config view --minify -o jsonpath='{.clusters[0].name}' 2>/dev/null || echo minikube )"
  if [ -f "\$JHOME/.minikube/ca.crt" ]; then
    SERVER="\$( "\$KUBECTL" config view --minify -o jsonpath='{.clusters[0].cluster.server}' )"
    "\$KUBECTL" config set-cluster "\$CLUSTER" --server="\$SERVER" --certificate-authority="\$JHOME/.minikube/ca.crt" --embed-certs=false --kubeconfig="\$KCFG"
  else
    echo "WARN: нет \$JHOME/.minikube/ca.crt — TLS verify off (лаборатория)."
    "\$KUBECTL" config set-cluster "\$CLUSTER" --insecure-skip-tls-verify=true --kubeconfig="\$KCFG"
  fi
  "\$KUBECTL" cluster-info
else
  # Ветка B: kubeconfig нет — kubectl выполняется внутри контейнера minikube (имя из MINIKUBE_CONTAINER).
  if docker inspect "\$MK" >/dev/null 2>&1; then
    USE_DOCKER_EXEC=1
    echo "kubeconfig в Jenkins нет — kubectl через docker exec \$MK"
  else
    echo "Нет \$JHOME/.kube/config и Docker-контейнер '\$MK' не найден."
    echo "Смонтируйте хостовый ~/.kube в Jenkins (-v ~/.kube:/var/jenkins_home/.kube) или задайте MINIKUBE_CONTAINER."
    exit 1
  fi
fi

# --- Подготовка kubectl для ветки B: бинарь в образе minikube или скачанный в /tmp внутри контейнера ---
# KUBECONFIG на ноде: /etc/kubernetes/admin.conf (или запасной путь minikube).
MK_KUBECTL=""
MK_KUBECONFIG="/etc/kubernetes/admin.conf"
if [ "\$USE_DOCKER_EXEC" = 1 ]; then
  MK_KUBECTL="\$(docker exec "\$MK" sh -c 'command -v kubectl 2>/dev/null || find /var/lib/minikube -type f -name kubectl 2>/dev/null | head -n1')"
  if [ -z "\$MK_KUBECTL" ] || ! docker exec "\$MK" test -x "\$MK_KUBECTL" 2>/dev/null; then
    echo "kubectl в образе \$MK не найден — качаем бинарь и копируем в контейнер"
    ARCH="\$(docker exec "\$MK" uname -m)"
    case "\$ARCH" in aarch64|arm64) KARCH=arm64 ;; x86_64) KARCH=amd64 ;; *) echo "unsupported minikube arch: \$ARCH"; exit 1 ;; esac
    KVER="\$(curl -fsSL https://dl.k8s.io/release/stable.txt)"
    TMPK="/tmp/kubectl-mk-\${BUILD_NUMBER}-\${KVER}"
    curl -fSL "https://dl.k8s.io/release/\${KVER}/bin/linux/\${KARCH}/kubectl" -o "\$TMPK"
    chmod +x "\$TMPK"
    docker cp "\$TMPK" "\$MK:/tmp/kubectl-from-jenkins"
    docker exec "\$MK" chmod +x /tmp/kubectl-from-jenkins
    MK_KUBECTL=/tmp/kubectl-from-jenkins
  fi
  echo "kubectl in \$MK: \$MK_KUBECTL"
  if ! docker exec "\$MK" test -f "\$MK_KUBECONFIG" 2>/dev/null; then
    MK_KUBECONFIG=/var/lib/minikube/kubeconfig
  fi
  docker exec -e KUBECONFIG="\$MK_KUBECONFIG" "\$MK" "\$MK_KUBECTL" cluster-info
fi

# --- Имя образа на демоне Jenkins и тег билда; IMG по умолчанию — pull из registry (если не включён локальный импорт) ---
NAME='${params.DOCKER_IMAGE}'
TAG='${env.BUILD_NUMBER}'
LOCAL_REF="\${NAME}:jenkins-\${TAG}"
CTR_IMP='${params.K8S_CTR_IMPORT_IMAGE}'
K8S_CTR_IMPORT=0
IMG='${params.K8S_PULL_REGISTRY}/${params.DOCKER_IMAGE}:${env.BUILD_NUMBER}'

# Локальный импорт в minikube (K8S_CTR_IMPORT_IMAGE=true + ветка docker exec): kubelet идёт через cri-dockerd → dockerd,
# поэтому «docker save | docker load» в контейнер minikube; в манифесте — image users-go:jenkins-N и imagePullPolicy Never.
if [ "\$USE_DOCKER_EXEC" = 1 ] && [ "\$CTR_IMP" = "true" ]; then
  docker image inspect "\$LOCAL_REF" >/dev/null
  # Minikube kic: kubelet → cri-dockerd → dockerd. Образы должны быть в «docker images» внутри ноды, не только в ctr -n k8s.io.
  if ! docker exec "\$MK" sh -c 'command -v docker >/dev/null 2>&1'; then
    echo "В \$MK нет docker — отключите K8S_CTR_IMPORT_IMAGE или обновите minikube" >&2
    exit 1
  fi
  echo "Загрузка \$LOCAL_REF в docker внутри \$MK (docker load для cri-dockerd)…"
  docker save "\$LOCAL_REF" | docker exec -i "\$MK" docker load
  IMG="\$LOCAL_REF"
  K8S_CTR_IMPORT=1
  echo "Деплой с локальным образом \$IMG и imagePullPolicy Never"
  echo "Образы docker в minikube (фрагмент):"
  docker exec "\$MK" docker images 2>/dev/null | grep -F "jenkins-\${TAG}" | head -8 || true
fi

# --- Подстановка БД в YAML: плейсхолдеры __K8S_DB_HOST__ / __K8S_DB_PORT__ (параметры job) ---
K8S_DB_HOST='${params.K8S_DB_HOST}'
K8S_DB_PORT='${params.K8S_DB_PORT}'

# Читает k8s/user-service-registry.yaml построчно: подставляет БД, image (IMG), при K8S_CTR_IMPORT=1 меняет Always→Never.
render_manifest() {
  while IFS= read -r line || [[ -n "\$line" ]]; do
    line="\${line//__K8S_DB_HOST__/\$K8S_DB_HOST}"
    line="\${line//__K8S_DB_PORT__/\$K8S_DB_PORT}"
    if [[ "\$line" =~ ^([[:space:]]*)image:[[:space:]].* ]]; then
      echo "\${BASH_REMATCH[1]}image: \${IMG}"
    elif [[ "\${K8S_CTR_IMPORT}" = 1 ]] && [[ "\$line" =~ imagePullPolicy:[[:space:]]*Always ]]; then
      echo "\${line/Always/Never}"
    else
      echo "\$line"
    fi
  done < k8s/user-service-registry.yaml
}

# Проверка доступности HTTP registry с ноды minikube (только если pull из registry, без docker load).
REG_PULL='${params.K8S_PULL_REGISTRY}'
registry_probe() {
  if [ "\$USE_DOCKER_EXEC" != 1 ] || [ "\${K8S_CTR_IMPORT:-0}" = 1 ]; then return 0; fi
  if ! docker exec "\$MK" sh -c 'command -v curl >/dev/null 2>&1'; then
    echo "WARN: в minikube нет curl — проверку http://\${REG_PULL}/v2/ пропускаем"
    return 0
  fi
  if docker exec "\$MK" sh -ec "curl -sf --connect-timeout 5 http://\${REG_PULL}/v2/ >/dev/null"; then
    echo "OK: с ноды minikube отвечает http://\${REG_PULL}/v2/ (если всё же ImagePullBackOff — добавьте insecure-registry для HTTP registry:2)"
    return 0
  fi
  echo "=== ОШИБКА: с ноды minikube НЕ открывается http://\${REG_PULL}/v2/ (kubelet не сможет скачать образ) ===" >&2
  echo "Запустите registry:2 на хосте :5050 и пересоздайте minikube, например:" >&2
  echo "  minikube stop && minikube start --insecure-registry \"\${REG_PULL}\"" >&2
  echo "Проверка вручную: minikube ssh -- curl -sI http://\${REG_PULL}/v2/" >&2
  return 1
}

# --- apply манифеста и ожидание готовности Deployment user-service в namespace market ---
if [ "\$USE_DOCKER_EXEC" = 1 ]; then
  registry_probe
  render_manifest | docker exec -i -e KUBECONFIG="\$MK_KUBECONFIG" "\$MK" "\$MK_KUBECTL" apply -f -
  docker exec -e KUBECONFIG="\$MK_KUBECONFIG" "\$MK" "\$MK_KUBECTL" -n market rollout status deployment/user-service --timeout=180s
else
  render_manifest | "\$KUBECTL" apply -f -
  "\$KUBECTL" -n market rollout status deployment/user-service --timeout=180s
fi
"""
      }
    }
  }
}
