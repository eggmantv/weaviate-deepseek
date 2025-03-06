package weaviatelib

import (
	"context"
	"encoding/json"
	"fmt"
	"go-weaviate-deepseek/ext"

	"github.com/tidwall/gjson"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/data"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/data/replication"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

// Clear 删除数据，必须有条件才能删除
func Clear(clsName, key, value string) error {
	clsName = GetClsName(clsName)
	client := GetClient()

	where := filters.Where().
		WithOperator(filters.Equal).
		WithPath([]string{key}).
		WithValueString(value)
	rsp, err := client.Batch().
		ObjectsBatchDeleter().
		WithClassName(clsName).
		WithOutput("minimal").
		WithWhere(where).
		WithConsistencyLevel(replication.ConsistencyLevel.ALL).
		Do(context.Background())
	if err != nil {
		return err
	}
	L.Println("rsp results:", rsp.Results)
	return nil
}

func Create(clsName string, id string, attrs map[string]interface{}, vector []float32) (*data.ObjectWrapper, error) {
	clsName = GetClsName(clsName)
	client := GetClient()
	created, err := client.Data().Creator().
		WithClassName(clsName).
		WithID(id).
		WithProperties(attrs).
		WithVector(vector).
		Do(context.Background())

	if err != nil {
		return nil, err
	}
	return created, nil
}

func BatchImport(jsonB []byte) {
	client := GetClient()
	objects := make([]*models.Object, 0)
	err := json.Unmarshal(jsonB, &objects)
	if err != nil {
		L.Errorf(err.Error())
	}
	L.Println("importing, objects size:", len(objects))
	// set vector before import
	for _, o := range objects {
		b, _ := json.Marshal(o.Properties)
		text := gjson.ParseBytes(b).Get("captions").String()
		vec, err := VectorizerFunc(text)
		if err != nil {
			L.Warn("cal openai vector err:", err)
			continue
		}
		o.Vector = vec
		L.Printf("text: %s, vector size: %v", text, len(o.Vector))
	}

	res, err := client.Batch().ObjectsBatcher().WithObjects(objects...).
		WithConsistencyLevel(replication.ConsistencyLevel.ALL).
		Do(context.Background())
	if err != nil {
		L.Errorf(err.Error())
	}
	for _, r := range res {
		err, _ := r.Result.Errors.MarshalBinary()
		L.Warnf("err: %s", string(err))
	}
}

func Scan(clsName string) (ext.M, error) {
	clsName = GetClsName(clsName)
	client := GetClient()
	// meta := graphql.Field{
	// 	Name: "title",
	// }
	result, err := client.Data().ObjectsGetter().
		WithClassName(clsName).
		WithLimit(25). // default 25
		Do(context.Background())
	if err != nil {
		return nil, err
	}
	rows := make([]string, 0)
	for _, v := range result {
		b, _ := v.MarshalBinary()
		rows = append(rows, string(b))
		L.Printf("object: %s, property: %s", string(b), v.Properties)
	}
	res := ext.M{
		"size": len(result),
		"rows": rows,
	}
	return res, nil
}

// Query
// opts[0]: distance, range: 0-2, 越小越匹配, https://weaviate.io/developers/weaviate/config-refs/distances#distance-fields-in-the-apis
func Query(clsName string, phase string, opts ...float32) ([]byte, error) {
	clsName = GetClsName(clsName)
	client := GetClient()

	// field1 := graphql.Field{Name: "id"}
	_additional := graphql.Field{
		Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
			{Name: "certainty"}, // only supported if distance==cosine
			{Name: "distance"},  // always supported
		},
	}
	fields := make([]graphql.Field, 0)
	if clsName == ClsRubyGPT {
		fields = []graphql.Field{
			{Name: "title"},
			{Name: "captions"},
			_additional,
		}
	} else {
		fields = []graphql.Field{
			{Name: "title"},
			{Name: "url"},
			{Name: "media_type"},
			{Name: "captions"},
			_additional,
		}
	}

	L.Println("calculate vector for:", phase)
	textVector, err := VectorizerFunc(phase)
	if err != nil {
		return nil, err
	}
	L.Println("vector size:", len(textVector))

	var distanceFloat float32 = 0.5
	if len(opts) > 0 {
		distanceFloat = opts[0]
	}

	L.Println("distanceFloat:", distanceFloat)
	nearVector := client.GraphQL().NearVectorArgBuilder().
		WithVector(textVector).WithDistance(distanceFloat)

	// where := filters.Where().
	// 	WithPath([]string{"title"}).
	// 	WithOperator(filters.Equal).
	// 	WithValueString("体育")
	rsp, err := client.GraphQL().Get().
		WithClassName(clsName).
		WithFields(fields...).
		// WithWhere(where).
		WithNearVector(nearVector).
		WithLimit(3).
		// WithSort()
		// WithOffset()
		Do(context.Background())
	if err != nil {
		fmt.Printf("111, result: %s", err)
		return nil, err
	}

	// 应该只有一组key/value
	res := make([]byte, 0)
	for k, v := range rsp.Data {
		res, _ = json.Marshal(v)
		size := len(gjson.ParseBytes(res).Get(clsName).Array())
		L.Printf("db query, key: %s, size: %d", k, size)
	}
	return res, nil
}

func FindByID(clsName string, id string) (*models.Object, error) {
	clsName = GetClsName(clsName)
	client := GetClient()
	data, err := client.Data().ObjectsGetter().
		WithClassName(clsName).
		WithID(id).
		WithVector().
		WithConsistencyLevel(replication.ConsistencyLevel.ONE). // default QUORUM
		Do(context.Background())

	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		return data[0], nil
	}
	return nil, fmt.Errorf("not found with id: %s", id)
}

func DeleteByID(clsName string, id string) error {
	clsName = GetClsName(clsName)
	client := GetClient()
	err := client.Data().Deleter().
		WithClassName(clsName).
		WithID(id).
		WithConsistencyLevel(replication.ConsistencyLevel.ALL). // default QUORUM
		Do(context.Background())

	if err != nil {
		return err
	}
	return nil
}

func IsExists(clsName string, id string) bool {
	clsName = GetClsName(clsName)
	client := GetClient()
	exists, err := client.Data().Checker().
		WithClassName(clsName).
		WithID(id).
		Do(context.Background())

	if err != nil {
		L.Errorf(err.Error())
	}
	return exists
}

// UpdateByID
//
//	updates := map[string]interface{}{
//	    "name": "J. Kantor",
//	}
func UpdateByID(clsName string, id string, updates map[string]interface{}) error {
	clsName = GetClsName(clsName)
	client := GetClient()

	err := client.Data().Updater().
		WithID(id).
		WithClassName(clsName).
		WithProperties(updates).
		WithConsistencyLevel(replication.ConsistencyLevel.ALL). // default QUORUM
		Do(context.Background())

	if err != nil {
		return err
	}
	return nil
}

func Count(clsName string) (int64, error) {
	clsName = GetClsName(clsName)
	client := GetClient()

	// meta这个属性是内置的，可以获取系统的一些数据，和上面的_additional一样
	meta := graphql.Field{
		Name: "meta", Fields: []graphql.Field{
			{Name: "count"},
		},
	}

	res, err := client.GraphQL().Aggregate().
		WithClassName(clsName).
		WithFields(meta).
		Do(context.Background())
	if err != nil {
		return 0, err
	}
	b, err := json.Marshal(res.Data["Aggregate"])
	if err != nil {
		return 0, err
	}

	data := gjson.ParseBytes(b).Get(clsName).Array()
	if len(data) > 0 {
		return data[0].Get("meta.count").Int(), nil
	}
	return -1, nil
}
