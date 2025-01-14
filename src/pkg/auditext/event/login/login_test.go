package login

import (
	"context"
	"testing"

	"github.com/goharbor/harbor/src/controller/event/metadata/commonevent"
	"github.com/goharbor/harbor/src/controller/event/model"
	"github.com/goharbor/harbor/src/pkg/notifier/event"
)

func Test_resolver_Resolve(t *testing.T) {
	type args struct {
		ce    *commonevent.Metadata
		event *event.Event
	}
	tests := []struct {
		name                     string
		l                        *resolver
		args                     args
		wantErr                  bool
		wantUsername             string
		wantOperation            string
		wantOperationDescription string
		wantOperationResult      bool
	}{

		{"test normal", &resolver{}, args{
			ce: &commonevent.Metadata{
				Username:      "test",
				RequestURL:    "/c/login",
				RequestMethod: "POST",
				Payload:       "principal=test&password=123456",
				ResponseCode:  200,
			}, event: &event.Event{}}, false, "test", "login", "login", true},
		{"test fail", &resolver{}, args{
			ce: &commonevent.Metadata{
				Username:      "test",
				RequestURL:    "/c/login",
				RequestMethod: "POST",
				Payload:       "principal=test&password=123456",
				ResponseCode:  401,
			}, event: &event.Event{}}, false, "test", "login", "login", false},
		{"test logout", &resolver{}, args{
			ce: &commonevent.Metadata{
				Username:      "test",
				RequestURL:    "/c/log_out",
				RequestMethod: "GET",
				Payload:       "",
				ResponseCode:  200,
			}, event: &event.Event{}}, false, "test", "logout", "logout", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &resolver{}
			if err := l.Resolve(tt.args.ce, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("resolver.Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.args.event.Data.(*model.CommonEvent).Operator != tt.wantUsername {
				t.Errorf("resolver.Resolve() got = %v, want %v", tt.args.event.Data.(*model.CommonEvent).Operator, tt.wantUsername)
			}
			if tt.args.event.Data.(*model.CommonEvent).Operation != tt.wantOperation {
				t.Errorf("resolver.Resolve() got = %v, want %v", tt.args.event.Data.(*model.CommonEvent).Operation, tt.wantOperation)
			}
			if tt.args.event.Data.(*model.CommonEvent).OperationDescription != tt.wantOperationDescription {
				t.Errorf("resolver.Resolve() got = %v, want %v", tt.args.event.Data.(*model.CommonEvent).OperationDescription, tt.wantOperationDescription)
			}
			if tt.args.event.Data.(*model.CommonEvent).OperationResult != tt.wantOperationResult {
				t.Errorf("resolver.Resolve() got = %v, want %v", tt.args.event.Data.(*model.CommonEvent).OperationResult, tt.wantOperationResult)
			}
		})
	}
}

func Test_resolver_PreCheck(t *testing.T) {
	type args struct {
		ctx    context.Context
		url    string
		method string
	}
	tests := []struct {
		name             string
		e                *resolver
		args             args
		wantMatched      bool
		wantResourceName string
	}{
		{"test normal", &resolver{}, args{context.Background(), "/c/login", "POST"}, true, ""},
		{"test fail", &resolver{}, args{context.Background(), "/c/logout", "GET"}, true, ""},
		{"test fail method", &resolver{}, args{context.Background(), "/c/login", "PUT"}, false, ""},
		{"test fail wrong url", &resolver{}, args{context.Background(), "/c/logout", "DELETE"}, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &resolver{}
			got, got1 := e.PreCheck(tt.args.ctx, tt.args.url, tt.args.method)
			if got != tt.wantMatched {
				t.Errorf("resolver.PreCheck() got = %v, want %v", got, tt.wantMatched)
			}
			if got1 != tt.wantResourceName {
				t.Errorf("resolver.PreCheck() got1 = %v, want %v", got1, tt.wantResourceName)
			}
		})
	}
}
