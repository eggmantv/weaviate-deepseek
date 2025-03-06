# DeepSeek本地知识库开发实战

本专题介绍如何基于Golang语言构建本地化智能知识库系统，通过阿里云百炼平台的DeepSeek大模型API与Weaviate向量数据库的结合，实现高效的知识存储与智能问答能力。该项目可部署在私有化环境中，提供安全可靠的智能知识服务。

本专题分为两个项目：

1. go-weaviate-deepseek：基于Golang的项目，用于与Weaviate向量数据库和DeepSeek大模型API进行交互。
2. gwd-app：基于React和Vite的Web应用程序，使用Websocket与go-weaviate-deepseek项目进行交互，实现知识问答功能。

## go-weaviate-deepseek

### 安装Weaviate

Weaviate 使用Docker安装，docker compose文件在 `/go-weaviate-deepseek/weaviate-docker-compose.yml`中。

### 项目配置

将阿里云百炼平台DeepSeek API Key 进行配置，在 `/go-weaviate-deepseek/conf/conf.go` 文件的 `AliDeepSeekAPIKey` 中。

### Weaviate 操作接口

#### 创建集合

``` shell
curl --location 'http://localhost:5012/weaviate/create_db' \
--header 'X_KEY: xxxxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "cls_name": "GoWeaviateDeepseek",
    "desp": "desp",
    "schema": "[{\"name\":\"title\",\"dataType\":[\"string\"]},{\"name\":\"captions\",\"dataType\":[\"text\"]},{\"name\":\"url\",\"dataType\":[\"string\"]},{\"name\":\"media_type\",\"dataType\":[\"string\"]}]"
}'
```

#### 删除集合

``` shell
curl --location 'http://localhost:5012/weaviate/remove_db' \
--header 'X_KEY: xxxxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "cls_name": "Aa5c363a7b30d48df96e21033c8716903"
}'
```

#### 获取集合信息

``` shell
curl --location 'http://localhost:5012/weaviate/db_info' \
--header 'X_KEY: xxxxxxx'
```

#### 插入数据

``` shell
curl --location 'http://localhost:5012/weaviate/create' \
--header 'X_KEY: xxxxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "cls_name": "GoWeaviateDeepseek",
    "type": "text",
    "data": "你好，我是智能AI助手EE，有什么问题都可以随时向我提问，有什么可以帮到你？"
}'
```

#### 删除数据

``` shell
curl --location 'http://localhost:5012/weaviate/delete' \
--header 'X_KEY: xxxxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "cls_name": "GoWeaviateDeepseek",
    "id": "160ee368-41dc-4b99-90c5-1f001c2ed787"
}'
```

#### 查看具体集合数据

``` shell
curl --location 'http://localhost:5012/weaviate/scan' \
--header 'X_KEY: xxxxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "cls_name": "GoWeaviateDeepseek"
}'
```

#### 搜索

``` shell
curl --location 'http://localhost:5012/weaviate/search' \
--header 'X_KEY: xxxxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "cls_name": "GoWeaviateDeepseek",
    "distance": 2,
    "prompt": "蛋人网"
}'
```

### websocket 连接

``` shell
ws://localhost:5012/ds-ws
```

## gwd-app

安装依赖：
``` shell
yarn install
```

启动项目：
``` shell
yarn dev
```

## 效果
![图1](https://imgs.eggman.tv/e2eda2dd8b1f435d84ffac61b70de2e1_QQ20250306-132617.png)

## 在线体验

[https://eggman.tv/rubychat](https://eggman.tv/rubychat)

## 具体代码讲解

[https://eggman.tv/c/s-weaviate-deepseek](https://eggman.tv/c/s-weaviate-deepseek)
