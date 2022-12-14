# 单元测试
.PHONY: ut
ut:
	@go test -race ./...

# e2e 测试
.PHONY: e2e
e2e:
	sh ./script/integrate_test.sh

.PHONY: e2e_up
e2e_up:
	docker compose -f script/emicro_test_compose.yml up -d

.PHONY: e2e_down
e2e_down:
	docker compose -f script/emicro_test_compose.yml down