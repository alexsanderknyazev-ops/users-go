#!/bin/bash

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_step() { echo -e "${BLUE}▶${NC} $1"; }
print_success() { echo -e "${GREEN}✓${NC} $1"; }
print_error() { echo -e "${RED}✗${NC} $1"; }
print_info() { echo -e "${YELLOW}ℹ${NC} $1"; }

echo "========================================"
echo "🚀 USER SERVICE DEPLOYMENT (CLEAN UPDATE)"
echo "========================================"

# Параметры сервиса
SERVICE_NAME="user-service"
APP_PORT="8072"
IMAGE_NAME="user-service"
HEALTH_PATH="/health"  # Измените если другой путь

# 1. Проверка Minikube
print_step "Checking Minikube..."
if ! minikube status | grep -q "Running"; then
    print_error "Minikube is not running."
    exit 1
fi

# 2. Настройка Docker
print_step "Setting up Docker..."
eval $(minikube docker-env)
print_success "Docker configured"

# 3. Переходим в директорию сервиса (если нужно)
if [ -d "$SERVICE_NAME" ]; then
    cd "$SERVICE_NAME"
    print_info "Changed to $SERVICE_NAME directory"
fi

# 4. Сборка образа
print_step "Building Docker image..."
if docker build -t ${IMAGE_NAME}:latest .; then
    print_success "Image built: ${IMAGE_NAME}:latest"
else
    print_error "Build failed"
    exit 1
fi

# 5. Проверка namespace
print_step "Checking namespace..."
if ! kubectl get namespace market >/dev/null 2>&1; then
    print_error "Namespace 'market' not found"
    exit 1
fi

# 6. Удаляем ВСЕ deployment с именем user-service
print_step "Cleaning up OLD user-service deployments..."
USER_DEPLOYMENTS=$(kubectl get deployments -n market --no-headers 2>/dev/null | awk '{print $1}' | grep "^user-service")

if [ -n "$USER_DEPLOYMENTS" ]; then
    echo "Found user-service deployments to delete:"
    for DEPLOY in $USER_DEPLOYMENTS; do
        echo "  - $DEPLOY"
        kubectl delete deployment -n market "$DEPLOY" --ignore-not-found
    done
    print_success "Old user-service deployments deleted"
    
    # Ждем пока старые поды удалятся
    echo "Waiting for old pods to terminate..."
    sleep 5
    
    # Проверяем что старые поды удалены
    OLD_PODS=$(kubectl get pods -n market --no-headers 2>/dev/null | grep -E "(user-service)" | wc -l)
    if [ "$OLD_PODS" -gt 0 ]; then
        echo "Force deleting remaining old pods..."
        kubectl delete pods -n market -l 'app in (user, user-service)' --ignore-not-found
        sleep 2
    fi
else
    print_success "No old user-service deployments found"
fi

# 7. Создаём новый deployment с версией
TIMESTAMP=$(date +%Y%m%d%H%M%S)
NEW_DEPLOYMENT_NAME="${SERVICE_NAME}-v${TIMESTAMP}"

print_step "Creating NEW deployment: ${NEW_DEPLOYMENT_NAME}..."
cat <<YAML | kubectl apply -n market -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${NEW_DEPLOYMENT_NAME}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${SERVICE_NAME}
      version: "v${TIMESTAMP}"
  template:
    metadata:
      labels:
        app: ${SERVICE_NAME}
        version: "v${TIMESTAMP}"
    spec:
      containers:
      - name: ${SERVICE_NAME}
        image: ${IMAGE_NAME}:latest
        imagePullPolicy: Never
        ports:
        - containerPort: ${APP_PORT}
        env:
        - name: APP_PORT
          value: "${APP_PORT}"
        - name: DB_HOST
          value: "postgres"
        - name: DB_PORT
          value: "5432"
        - name: DB_USER
          value: "admin"
        - name: DB_PASSWORD
          value: "admin123"
        - name: DB_NAME
          value: "marketdb"  # Измените на вашу БД пользователей
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        readinessProbe:
          httpGet:
            path: /users/health 
            port: 8072
          initialDelaySeconds: 15
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /users/health 
            port: 8072
          initialDelaySeconds: 30
          periodSeconds: 15
YAML
print_success "Deployment ${NEW_DEPLOYMENT_NAME} created"

# 8. Service (создаем или обновляем)
print_step "Creating/Updating service..."
cat <<YAML | kubectl apply -n market -f -
apiVersion: v1
kind: Service
metadata:
  name: ${SERVICE_NAME}
spec:
  selector:
    app: ${SERVICE_NAME}
  ports:
  - name: http
    port: ${APP_PORT}
    targetPort: ${APP_PORT}
  type: ClusterIP
YAML
print_success "Service ${SERVICE_NAME} ready"

# 9. Ждём запуска новой поды
print_step "Waiting for NEW pod..."
MAX_WAIT=90
POD_READY=false
for i in $(seq 1 $MAX_WAIT); do
    POD_NAME=$(kubectl get pods -n market -l version=v${TIMESTAMP} -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -n "$POD_NAME" ]; then
        POD_STATUS=$(kubectl get pod -n market "$POD_NAME" -o jsonpath='{.status.phase}' 2>/dev/null)
        POD_READY_STATE=$(kubectl get pod -n market "$POD_NAME" -o jsonpath='{.status.containerStatuses[0].ready}' 2>/dev/null)
        
        if [[ "$POD_STATUS" == "Running" ]] && [[ "$POD_READY_STATE" == "true" ]]; then
            print_success "✅ New pod $POD_NAME is running and ready!"
            POD_READY=true
            break
        elif [[ "$POD_STATUS" == "Error" ]] || [[ "$POD_STATUS" == "CrashLoopBackOff" ]]; then
            print_error "❌ Pod in error state: $POD_STATUS"
            echo "Checking logs..."
            kubectl logs -n market "$POD_NAME" --tail=20
            exit 1
        fi
    fi
    
    if [ $i -eq $MAX_WAIT ]; then
        print_error "❌ Timeout waiting for pod"
        echo "Current pods:"
        kubectl get pods -n market
        echo ""
        echo "Checking deployment status:"
        kubectl describe deployment -n market ${NEW_DEPLOYMENT_NAME} | tail -30
        echo ""
        echo "Checking pod logs:"
        if [ -n "$POD_NAME" ]; then
            kubectl logs -n market "$POD_NAME" --tail=20
        fi
        exit 1
    fi
    
    if [ $((i % 10)) -eq 0 ]; then
        echo -n "$i "
    else
        echo -n "."
    fi
    sleep 1
done
echo ""

# 10. Проверка статуса
print_step "Checking status..."
echo ""
echo "📊 CURRENT PODS:"
kubectl get pods -n market -o wide | grep -E "(NAME|user)"
echo ""
echo "📊 CURRENT SERVICES:"
kubectl get svc -n market | grep -E "(NAME|user)"

# 11. Тест приложения
print_step "Testing application..."
kubectl port-forward -n market svc/${SERVICE_NAME} ${APP_PORT}:${APP_PORT} > /dev/null 2>&1 &
PF_PID=$!
sleep 3

echo ""
echo "🧪 Testing endpoints:"
echo "  1. Health check..."

if curl -s --max-time 5 http://localhost:${APP_PORT}${HEALTH_PATH} > /dev/null 2>&1; then
    print_success "   ✅ Health check passed"
    echo "     Health response:"
    curl -s http://localhost:${APP_PORT}${HEALTH_PATH} | head -c 100
    echo ""
else
    print_error "   ❌ Health check failed"
    echo "     Trying different health endpoints..."
    
    # Пробуем другие возможные пути
    for path in "/" "/health" "/api/health" "/actuator/health"; do
        if curl -s --max-time 3 http://localhost:${APP_PORT}${path} > /dev/null 2>&1; then
            print_success "     Found working endpoint: ${path}"
            break
        fi
    done
    
    echo "     Checking logs..."
    kubectl logs -n market -l version=v${TIMESTAMP} --tail=15
fi

echo "  2. Main endpoint test..."
if curl -s --max-time 5 http://localhost:${APP_PORT}/users > /dev/null 2>&1; then
    print_success "   ✅ Main endpoint responding"
else
    print_info "   ℹ Main endpoint may require specific path"
fi

kill $PF_PID 2>/dev/null

# 12. Удаляем старые deployment (кроме текущего)
print_step "Final cleanup of other ${SERVICE_NAME} deployments..."
OTHER_DEPLOYMENTS=$(kubectl get deployments -n market --no-headers 2>/dev/null | awk '{print $1}' | grep "^${SERVICE_NAME}" | grep -v "^${NEW_DEPLOYMENT_NAME}$")

if [ -n "$OTHER_DEPLOYMENTS" ]; then
    echo "Found other ${SERVICE_NAME} deployments to clean up:"
    for DEPLOY in $OTHER_DEPLOYMENTS; do
        echo "  - $DEPLOY"
        kubectl delete deployment -n market "$DEPLOY" --ignore-not-found
    done
    print_success "Other ${SERVICE_NAME} deployments cleaned"
else
    print_success "No other ${SERVICE_NAME} deployments found"
fi

# 13. Показываем доступные команды
echo ""
echo "========================================"
echo "✅ ${SERVICE_NAME^^} DEPLOYMENT COMPLETE!"
echo "========================================"
echo ""
echo "📌 Summary:"
echo "   • Service: ${SERVICE_NAME}"
echo "   • Deployment: ${NEW_DEPLOYMENT_NAME}"
echo "   • Version: v${TIMESTAMP}"
echo "   • Image: ${IMAGE_NAME}:latest"
echo "   • Port: ${APP_PORT}"
echo ""
echo "🌐 Access from Postman:"
echo "   1. kubectl port-forward -n market svc/${SERVICE_NAME} ${APP_PORT}:${APP_PORT}"
echo "   2. Use: http://localhost:${APP_PORT}"
echo ""
echo "📊 Monitoring commands:"
echo "   kubectl get pods -n market | grep user"
echo "   kubectl logs -n market -l app=${SERVICE_NAME} -f"
echo "   kubectl describe pod -n market -l app=${SERVICE_NAME}"
echo ""
echo "🔄 To update again:"
echo "   ./$(basename $0)"
echo ""
echo "🔗 Connect with market-service:"
echo "   Inside cluster: http://${SERVICE_NAME}.market:${APP_PORT}"
echo ""

# Вернуться в исходную директорию
if [ "$PWD" != "$OLDPWD" ]; then
    cd - > /dev/null
fi