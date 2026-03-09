

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