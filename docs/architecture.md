# 系统架构文档

## 系统架构图

```plantuml
@startuml
!theme plain
skinparam componentStyle rectangle
skinparam backgroundColor #F5F5F5
skinparam defaultFontName Microsoft YaHei
skinparam defaultFontSize 14
skinparam component {
    BackgroundColor #FFFFFF
    BorderColor #333333
    ArrowColor #666666
}

skinparam package {
    BackgroundColor #F0F0F0
    BorderColor #333333
    FontStyle bold
}

package "外部系统" {
    together {
        component "用户平台" #LightBlue {
            [训练平台] as TrainingPlatform #LightBlue
            [推理平台] as InferencePlatform #LightBlue
            [Notebook平台] as NotebookPlatform #LightBlue
            [数据蒸馏平台] as DistillationPlatform #LightBlue
        }
        
        component "外部服务" #LightYellow {
            [配额服务] as QuotaService #LightYellow
            [资源服务] as ResourceService #LightYellow
            [认证服务] as AuthService #LightYellow
        }
        
        component "基础设施" #LightGreen {
            [K8S集群] as K8S #LightGreen
            [数据库] as DB #LightGreen
        }
    }
}

package "任务管理系统" {
    component "API层" #Pink {
        [控制器] as Controllers #Pink
    }
    
    component "服务层" #LightPink {
        [任务服务] as TaskService #LightPink
        [实例服务] as InstanceService #LightPink
        [队列服务] as QueueService #LightPink
    }
    
    component "内部组件" #LightCyan {
        [任务管理] as TaskManager #LightCyan
        [流程控制] as FlowControl #LightCyan
        [监控系统] as Monitor #LightCyan
        [工具类] as Utils #LightCyan
    }
    
    component "数据访问层" #LightGray {
        [DAO] as DAO #LightGray
    }
}

TrainingPlatform -[#666666]-> Controllers
InferencePlatform -[#666666]-> Controllers
NotebookPlatform -[#666666]-> Controllers
DistillationPlatform -[#666666]-> Controllers

Controllers -[#666666]-> TaskService
Controllers -[#666666]-> InstanceService
Controllers -[#666666]-> QueueService

TaskService -[#666666]-> TaskManager
InstanceService -[#666666]-> TaskManager
QueueService -[#666666]-> FlowControl

TaskManager -[#666666]-> DAO
FlowControl -[#666666]-> DAO
Monitor -[#666666]-> DAO

DAO -[#666666]-> DB
TaskManager -[#666666]-> K8S
Monitor -[#666666]-> K8S

TaskManager -[#666666]-> QuotaService
TaskManager -[#666666]-> ResourceService
Controllers -[#666666]-> AuthService

legend right
    | 颜色 | 含义 |
    | <#LightBlue> | 用户平台 |
    | <#LightGreen> | 基础设施 |
    | <#LightYellow> | 外部服务 |
    | <#Pink> | API层 |
    | <#LightPink> | 服务层 |
    | <#LightCyan> | 内部组件 |
    | <#LightGray> | 数据访问层 |
endlegend

@enduml
```

## 系统组件说明

### 1. 外部系统
#### 1.1 用户平台
- **训练平台**：用于提交和管理训练任务
- **推理平台**：用于部署和管理推理服务
- **Notebook平台**：用于创建和管理开发环境
- **数据蒸馏平台**：用于执行和管理蒸馏任务

#### 1.2 基础设施
- **K8S集群**：提供计算资源，运行各类任务
- **数据库**：存储系统数据，包括任务信息、用户信息等

#### 1.3 外部服务
- **配额服务**：管理用户和项目的资源配额
- **资源服务**：管理集群资源池，提供资源分配和释放
- **认证服务**：处理用户认证和权限管理

### 2. API层（Controllers）
- 处理所有HTTP请求
- 实现RESTful API接口
- 请求参数验证
- 响应数据格式化
- 与认证服务集成

### 3. 服务层（Service）
- 实现业务逻辑
- 任务生命周期管理
- 实例状态管理
- 队列调度管理
- 与配额服务集成

### 4. 内部组件（Internal）
#### 4.1 任务管理（TaskManager）
- 任务引擎注册
- 任务创建和销毁
- 任务状态转换
- 超时处理
- 与资源服务集成

#### 4.2 流程控制（FlowControl）
- 任务队列管理
- 周期性任务处理
- 任务状态监控
- 资源调度
- 与K8S集群交互

#### 4.3 监控系统（Monitor）
- 资源使用监控
- 性能指标收集
- 告警处理
- 白名单管理
- 与K8S集群集成

#### 4.4 工具类（Utils）
- 配置管理
- 日志处理
- 认证授权
- K8S操作
- 通用工具函数

### 5. 数据访问层（DAO）
- 数据库操作封装
- 数据模型定义
- 事务管理
- 缓存处理
- 数据持久化

## 系统交互流程

### 1. 用户认证流程
```plantuml
@startuml
actor User
participant "Controllers" as Controllers
participant "AuthService" as Auth
participant "TaskService" as Service

User -> Controllers: 1. 发送请求
Controllers -> Auth: 2. 验证Token
Auth --> Controllers: 3. 返回认证结果
alt 认证成功
    Controllers -> Service: 4a. 处理请求
    Service --> Controllers: 5a. 返回结果
else 认证失败
    Controllers --> User: 4b. 返回错误
end
@enduml
```

### 2. 资源分配流程
```plantuml
@startuml
participant "TaskManager" as Manager
participant "QuotaService" as Quota
participant "ResourceService" as Resource
participant "K8S" as K8S

Manager -> Quota: 1. 检查配额
Quota --> Manager: 2. 返回配额状态
alt 配额充足
    Manager -> Resource: 3a. 申请资源
    Resource --> Manager: 4a. 返回资源
    Manager -> K8S: 5a. 创建资源
    K8S --> Manager: 6a. 确认创建
else 配额不足
    Manager -> Quota: 3b. 申请额外配额
    Quota --> Manager: 4b. 返回申请结果
end
@enduml
```

### 3. 任务调度流程
```plantuml
@startuml
participant "QueueService" as Queue
participant "TaskManager" as Manager
participant "ResourceService" as Resource
participant "K8S" as K8S

Queue -> Manager: 1. 获取待调度任务
Manager -> Resource: 2. 检查资源
Resource --> Manager: 3. 返回资源状态
alt 资源充足
    Manager -> K8S: 4a. 创建任务
    K8S --> Manager: 5a. 返回状态
    Manager -> Queue: 6a. 更新队列
else 资源不足
    Manager -> Queue: 4b. 保持等待
end
@enduml
```

## 平台集成说明

### 1. 训练平台集成
训练平台使用任务管理系统主要实现训练任务的提交和管理，具体包括：
- 提交训练任务到任务队列
- 管理训练任务的资源分配
- 监控训练任务的执行状态
- 获取训练任务的日志和结果
- 管理训练任务的优先级和调度

```plantuml
@startuml
package "训练平台" {
    [训练任务提交] as TrainingSubmit
    [任务状态监控] as TrainingMonitor
    [资源管理] as TrainingResource
}

package "任务管理系统" {
    [TaskManager] as TaskManager
    [ResourceManager] as Resource
    [K8S] as K8S
}

TrainingSubmit -> TaskManager: 1. 提交训练任务
TaskManager -> Resource: 2. 分配GPU资源
Resource --> TaskManager: 3. 返回资源
TaskManager -> K8S: 4. 创建训练Pod
K8S --> TaskManager: 5. 返回状态
TaskManager --> TrainingSubmit: 6. 返回任务ID

TrainingMonitor -> TaskManager: 7. 查询任务状态
TaskManager --> TrainingMonitor: 8. 返回状态信息

TrainingResource -> Resource: 9. 查询资源使用
Resource --> TrainingResource: 10. 返回资源状态
@enduml
```

### 2. 推理平台集成
推理平台使用任务管理系统主要实现推理服务的部署和管理，具体包括：
- 部署推理服务到K8S集群
- 管理推理服务的资源分配
- 监控推理服务的运行状态
- 管理推理服务的扩缩容
- 获取推理服务的性能指标

```plantuml
@startuml
package "推理平台" {
    [服务部署] as ServiceDeploy
    [服务监控] as ServiceMonitor
    [资源管理] as ServiceResource
}

package "任务管理系统" {
    [TaskManager] as TaskManager
    [ResourceManager] as Resource
    [K8S] as K8S
}

ServiceDeploy -> TaskManager: 1. 部署推理服务
TaskManager -> Resource: 2. 分配推理资源
Resource --> TaskManager: 3. 返回资源
TaskManager -> K8S: 4. 创建推理服务
K8S --> TaskManager: 5. 返回服务地址
TaskManager --> ServiceDeploy: 6. 返回服务信息

ServiceMonitor -> TaskManager: 7. 监控服务状态
TaskManager --> ServiceMonitor: 8. 返回状态信息

ServiceResource -> Resource: 9. 管理服务资源
Resource --> ServiceResource: 10. 返回资源状态
@enduml
```

### 3. Notebook平台集成
```plantuml
@startuml
package "Notebook平台" {
    [环境管理] as EnvManager
    [数据管理] as DataManager
    [协作管理] as CollabManager
}

package "任务管理系统" {
    [TaskManager] as TaskManager
    [ResourceManager] as Resource
    [K8S] as K8S
}

EnvManager -> TaskManager: 1. 创建Notebook环境
TaskManager -> Resource: 2. 分配资源
Resource --> TaskManager: 3. 返回资源
TaskManager -> K8S: 4. 创建Notebook实例
K8S --> TaskManager: 5. 返回实例信息
TaskManager --> EnvManager: 6. 返回访问地址

EnvManager -> DataManager: 7. 挂载数据卷
DataManager --> EnvManager: 8. 确认挂载

EnvManager -> CollabManager: 9. 设置协作权限
CollabManager --> EnvManager: 10. 确认设置
@enduml
```

### 4. 数据蒸馏平台集成
```plantuml
@startuml
package "数据蒸馏平台" {
    [蒸馏任务管理] as DistillationManager
    [数据管理] as DataManager
    [模型管理] as ModelManager
}

package "任务管理系统" {
    [TaskManager] as TaskManager
    [ResourceManager] as Resource
    [K8S] as K8S
}

DistillationManager -> TaskManager: 1. 创建蒸馏任务
TaskManager -> Resource: 2. 分配资源
Resource --> TaskManager: 3. 返回资源
TaskManager -> K8S: 4. 创建蒸馏任务
K8S --> TaskManager: 5. 返回任务状态
TaskManager --> DistillationManager: 6. 返回任务ID

DistillationManager -> DataManager: 7. 获取源数据
DataManager --> DistillationManager: 8. 返回数据路径

DistillationManager -> ModelManager: 9. 获取教师模型
ModelManager --> DistillationManager: 10. 返回模型路径
@enduml
```

## 平台集成说明

### 1. 训练平台集成
训练平台主要用于模型训练任务，通过调用任务管理系统的接口实现以下功能：
- 创建和管理训练任务
- 分配和管理训练资源
- 监控训练进度
- 保存和管理训练模型
- 管理训练数据

### 2. 推理平台集成
推理平台主要用于模型推理服务，通过调用任务管理系统的接口实现以下功能：
- 创建和管理推理服务
- 部署和更新模型
- 管理服务流量
- 监控服务性能
- 扩缩容管理

### 3. Notebook平台集成
Notebook平台主要用于交互式开发环境，通过调用任务管理系统的接口实现以下功能：
- 创建和管理开发环境
- 管理数据卷
- 设置协作权限
- 监控资源使用
- 环境隔离管理

### 4. 数据蒸馏平台集成
数据蒸馏平台主要用于模型蒸馏任务，通过调用任务管理系统的接口实现以下功能：
- 创建和管理蒸馏任务
- 管理源数据和教师模型
- 配置蒸馏参数
- 监控蒸馏进度
- 管理蒸馏结果

## 扩展性设计

系统采用插件化架构，支持：
1. 自定义任务模板
2. 自定义监控指标
3. 自定义调度策略
4. 自定义存储后端 