package emicro

import (
	"context"
	"emicro/internal/errs"
	"encoding/json"
	"reflect"
)

func InitClientProxy(address string, srv Service) error {
	c, err := NewClient(address)
	if err != nil {
		return err
	}
	err = setFuncField(srv, c)
	if err != nil {
		return err
	}
	return nil
}

// 这个单独的拆出来，就是为了测试，可以考虑传入一个 mock proxy
func setFuncField(srv Service, p Proxy) error {
	val := reflect.ValueOf(srv)
	valElem := val.Elem()
	typElem := valElem.Type()
	if typElem.Kind() != reflect.Struct {
		return errs.ServiceTypError
	}
	numField := typElem.NumField()
	for i := 0; i < numField; i++ {
		fieldTyp := typElem.Field(i)
		fieldVal := valElem.Field(i)
		if !fieldVal.CanSet() {
			continue
		}
		fn := func(args []reflect.Value) (results []reflect.Value) {
			ctx := args[0].Interface().(context.Context)
			in := args[1].Interface()
			// reflect.Interface() -> 将 reflect.Value 转换成定义在 args,results 的类型
			out := reflect.New(fieldTyp.Type.Out(0).Elem()).Interface()
			inData, err := json.Marshal(in)
			if err != nil {
				return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
			}
			req := &Request{
				ServiceName: srv.ServiceName(),
				Method:      fieldTyp.Name,
				Data:        inData,
			}
			// 要在下面考虑发过去
			resp, err := p.Invoke(ctx, req)
			if err != nil {
				return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
			}
			err = json.Unmarshal(resp.Data, out)
			if err != nil {
				return []reflect.Value{reflect.ValueOf(out), reflect.ValueOf(err)}
			}
			// 在闭包中不能返回不带类型的 nil 值，
			// 例如: 当 err 为 nil 时 reflect.Zero(err) 就是不带类型的 nil 值
			// 只能 reflect.Zero(reflect.TypeOf(new(目标类型))).Elem()
			//  reflect.Zero(reflect.ValueOf(new(目标类型))) 行不行 ???
			nilErr := reflect.Zero(reflect.TypeOf(new(error)).Elem())
			return []reflect.Value{reflect.ValueOf(out), nilErr}
		}
		fieldVal.Set(reflect.MakeFunc(fieldTyp.Type, fn))
	}
	return nil
}
