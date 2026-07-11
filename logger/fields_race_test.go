package logger

import (
	"context"
	"sync"
	"testing"
)

// TestWithFields_NoSharedMapRace проверяет, что деривации одного контекста
// не делят одну map полей: горутины, порождённые из общего контекста, могут
// конкурентно звать WithField и логировать (getLogrusEntry итерирует map) без
// гонки. До фикса это падало с `fatal error: concurrent map iteration and map
// write`. Запускать под `-race`.
func TestWithFields_NoSharedMapRace(t *testing.T) {
	l, _ := testLogger()

	// Общий родитель с уже прикреплёнными полями — как хендлер, который делает
	// WithFields, а затем спавнит горутины.
	parent := l.WithFields(context.Background(), Fields{"user_id": "u1", "req": "r1"})

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			ctx := l.WithField(parent, "worker", n)
			for j := 0; j < 200; j++ {
				ctx = l.WithField(ctx, "iter", j)
				// getLogrusEntry итерирует map полей — конкурентно с записями выше.
				l.Info(ctx, "tick")
			}
		}(i)
	}
	wg.Wait()
}

// TestWithFields_ChildDoesNotMutateParent — дочерний контекст не должен менять
// поля родителя (следствие копирования map).
func TestWithFields_ChildDoesNotMutateParent(t *testing.T) {
	l, _ := testLogger()
	adapter := l.(*logrusAdapter)

	parent := l.WithField(context.Background(), "k", "parent")
	_ = l.WithField(parent, "k", "child") // дериват с перезаписью ключа

	got := adapter.getFieldsFromContext(parent)["k"]
	if got != "parent" {
		t.Fatalf("родительское поле мутировано дочерним контекстом: k=%v, ожидалось parent", got)
	}
}
