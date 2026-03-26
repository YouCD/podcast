

* 查询所有的节点
```dql
{
  all_data(func: has(name)) {
    uid
    name
    aliases
    dgraph.type
    expand(_all_) {
      uid
      name
      dgraph.type
      expand(_all_)
    }
  }
}

```
* 查询所有的 类型

```dql

{
  q(func: has(dgraph.type)) @groupby(dgraph.type) {
    count(uid)
  }
}

```
* 查询所有的 AI模型
```dql
{
  models(func: type("AI模型")) {
    uid
    name
    aliases
  }
}
```

* 查询所有的 `企业` 和相关的关系
```dql
{
  models(func: type("企业")) {
    uid
    name
    aliases
    <研发> {
      name
      dgraph.type
      <基于_时间>
      <基于>{
		    name
 		    dgraph.type
      }
    }
	<投资>{name}
	<收购>{name}
	<竞争>{name}
	<合作>{name}
	<雇佣>{name}
	<裁员>{name}
	<掌管>{name}
	<所属行业>{name}
  }
}

```


* 查询所有的 `AI模型` 和相关的关系

```dql

{
  models(func: type("AI模型")) {
    uid
    name
    aliases
    ~<研发> {
      name
      dgraph.type
      <基于_时间>
      <基于>
      <投资>
      <收购>
      <竞争>
      <合作>
      <雇佣>
      <裁员>
      <掌管>
      <所属行业>
    }
  }
}

```


* 查询指定企业及相关的关系
```dql
{
	q(func: eq(name, "openai")) {
		uid
		name
		aliases
		<研发>{name}
		<投资>{name}
		<收购>{name}
		<竞争>{name}
		<合作>{name}
		<雇佣>{name}
		<裁员>{name}
		<掌管>{name}
		<所属行业>{name}
	}
}

```

* 检查现有 Schema
```dql
schema {
  predicate
  type
  index
}

```


* 所有企业研发的技术
```dql

# 1. 找到所有“企业”类型的节点
{
  var(func: type("企业")) {
    u as uid
  }
  # 2. 找到所有“技术”类型的节点
  var(func: type("技术")) {
    t as uid
  }
  # 3. 查询这些节点之间的“研发”关系
  q(func: uid(u)) {
    uid
    name
    dgraph.type
    <研发> @filter(uid(t)) {
       uid
       name
       dgraph.type
    }
  }
}

```

* 展开指定企业的所有模型
```dql

{
    var(func: eq(name, "小米")) {
    u as uid
  }
} 


{
  var (func: type("AI模型")) {
      m as uid
  }
}


{
  q(func: uid(u)) {
    uid
    name
    dgraph.type
	
    <研发> @filter(uid(m)) {
       uid
       name
       dgraph.type
    }
  }
}

```
* 展开指定企业的所有节点

```sql

{
  var(func: eq(name, "小米")) {
    u as uid
  }

  q(func: uid(u)) {
    uid
    name
    aliases

    <研发> {
      name
      dgraph.type
      <基于_时间>
      <基于> {
        name
        dgraph.type
      }
    }

    <投资> { name }
    <收购> { name }
    <竞争> { name }
    <合作> { name }
    <雇佣> { name }
    <裁员> { name }
    <掌管> { name }
    <所属行业> { name }
  }
}


```

* 展开多个企业的所有节点

```sql

{
 var(func: eq(name, "小米")) { a as uid }
 var(func: eq(name, "华为")) { b as uid }


  q(func: uid(a,b)) {
    uid
    name
    aliases

    <研发> {
      name
      dgraph.type
      <基于_时间>
      <基于> {
        name
        dgraph.type
      }
    }

    <投资> { name }
    <收购> { name }
    <竞争> { name }
    <合作> { name }
    <雇佣> { name }
    <裁员> { name }
    <掌管> { name }
    <所属行业> { name }
  }
}


```

* 获取指定节点的详细信息
* 
```dql
{
  q(func: uid(0x507)) {
    uid
    name
    aliases

    <研发> {
      name
      dgraph.type
      <基于_时间>
      <基于> {
        name
        dgraph.type
      }
    }

    <投资> { name }
    <收购> { name }
    <竞争> { name }
    <合作> { name }
    <雇佣> { name }
    <裁员> { name }
    <掌管> { name }
    <所属行业> { name }
  }
}


```

## 节点合并操作
1. 查看两个节点的当前状态
```dql
{
  # 查询源节点 0x8d6
  source as var(func: uid(0x8d6))
  
  # 查询目标节点 0x493
  target as var(func: uid(0x493))
  
  # 获取详细信息（修复后）
  merge_check(func: uid(source, target)) {
    uid
    # 删掉手动写的 name、aliases、dgraph.type，expand 会自动包含
    expand(_all_) {
      uid
      name
    }
  }
} 

```


2. 检查两个节点的关系引用（入边/出边）
```dql
{
  # 检查谁引用了这两个节点（入边）
  incoming(func: uid(0x8d6, 0x493)) {
    uid
    name
    ~* {  # 所有入边
      uid
      name
      predicate
    }
  }
  
  # 检查这两个节点引用了谁（出边）—— 已修复
  outgoing(func: uid(0x8d6, 0x493)) {
    uid
    # 删掉 name、aliases 等手动字段，只保留 expand
    expand(_all_) {
      uid
      name
    }
  }
} 



```

3. 统计别名数量（用于对比）

```dql
{
  alias_count(func: uid(0x493)) {
    uid
    name
    alias_count: count(aliases)
    aliases
  }
  
  source_alias_count(func: uid(0x8d6)) {
    uid
    name
    alias_count: count(aliases)
    aliases
  }
}

```
```
"uid": "0x493",
        "name": "OpenClaw",
        "alias_count": 37,

     "name": "虾宝",
        "alias_count": 1,
```

二、执行合并操作

步骤 1：将 0x8d6 的别名迁移到 0x493

```sql
{
  "set": [
    {
      "uid": "0x493",
      "aliases": "虾宝"
    },
    {
      "uid": "0x493",
      "aliases": "OpenClawAgent"
    },
    {
      "uid": "0x493",
      "aliases": "虾宝AI"
    }
  ]
}

```