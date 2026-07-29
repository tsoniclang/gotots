package runtime

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	channelruntime "github.com/tsoniclang/gotots/internal/emit/runtime/channel"
	schedulerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/scheduler"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildChannel(
	factory tsgo.Factory,
	symbols []api.RuntimeSymbol,
) ([]Definition, error) {
	names, err := channelRuntimeNames()
	if err != nil {
		return nil, err
	}
	seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	definitions := make([]Definition, 0, len(symbols))
	for _, symbol := range symbols {
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		if contract.Module() != api.RuntimeModuleChannel {
			return nil, &AssemblyError{
				Module: api.RuntimeModuleChannel,
				Symbol: symbol,
				Reason: "channel output module received a foreign symbol",
			}
		}
		if _, duplicate := seen[symbol]; duplicate {
			return nil, &AssemblyError{
				Module: api.RuntimeModuleChannel,
				Symbol: symbol,
				Reason: "channel output module symbol is duplicated",
			}
		}
		seen[symbol] = struct{}{}
		statement, err := buildChannelSymbol(factory, symbol, names)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(symbol, statement)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

func channelRuntimeNames() (map[api.RuntimeSymbol]string, error) {
	names := make(map[api.RuntimeSymbol]string, 9)
	for _, symbol := range []api.RuntimeSymbol{
		api.RuntimeChannel,
		api.RuntimeReceiveChannel,
		api.RuntimeSendChannel,
		api.RuntimeSelectCase,
		api.RuntimeSelect,
		api.RuntimeScheduler,
		api.RuntimeSelectReady,
		api.RuntimeSelectAttempt,
		api.RuntimePanic,
	} {
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		names[symbol] = contract.ExportedName()
	}
	return names, nil
}

func buildChannelSymbol(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	names map[api.RuntimeSymbol]string,
) (tsgo.Statement, error) {
	if symbol == api.RuntimeScheduler {
		return schedulerruntime.Build(
			factory,
			symbol,
			names[api.RuntimeScheduler],
			names[api.RuntimePanic],
		)
	}
	return channelruntime.Build(
		factory,
		symbol,
		names[api.RuntimeChannel],
		names[api.RuntimeReceiveChannel],
		names[api.RuntimeSendChannel],
		names[api.RuntimeSelectCase],
		names[api.RuntimeSelect],
		names[api.RuntimeSelectReady],
		names[api.RuntimeSelectAttempt],
		names[api.RuntimePanic],
	)
}
