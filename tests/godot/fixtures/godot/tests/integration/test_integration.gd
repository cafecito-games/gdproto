extends VestTest

const IntegrationProto = preload("res://generated/integration.pb.gd")
const SharedProto = preload("res://generated/shared.pb.gd")
const TimestampProto = preload("res://generated/google/protobuf/timestamp.pb.gd")
const Core = preload("res://generated/proto_core_utils.gd")

func _populate(msg: IntegrationProto.KitchenSink) -> void:
	msg.set_name("kitchen")
	msg.set_score(42)

	var s1 := msg.add_stats()
	s1.set_key("hp")
	s1.set_value(100)
	var s2 := msg.add_stats()
	s2.set_key("mp")
	s2.set_value(50)

	msg.add_counters("a", 1)
	msg.add_counters("b", 2)

	msg.set_accent(SharedProto.Hue.BLUE)

	var ts := msg.new_created_at()
	ts.set_seconds(1700000000)
	ts.set_nanos(7)

	var extra := msg.new_extra()
	extra.set_data("payload-data")

	# oneof source: choose the cross-file enum branch.
	msg.set_kind(SharedProto.SourceKind.USER)

func _expect_populated(msg: IntegrationProto.KitchenSink) -> void:
	expect_equal(msg.get_name(), "kitchen")
	expect_equal(msg.get_score(), 42)

	var stats := msg.get_stats()
	expect_equal(stats.size(), 2)
	expect_equal(stats[0].get_key(), "hp")
	expect_equal(stats[0].get_value(), 100)
	expect_equal(stats[1].get_key(), "mp")
	expect_equal(stats[1].get_value(), 50)

	var counters := msg.get_counters()
	expect_equal(counters["a"], 1)
	expect_equal(counters["b"], 2)

	expect_equal(msg.get_accent(), SharedProto.Hue.BLUE)
	expect_equal(msg.get_created_at().get_seconds(), 1700000000)
	expect_equal(msg.get_created_at().get_nanos(), 7)
	expect_equal(msg.get_extra().get_data(), "payload-data")

	expect_equal(msg.get_source_case(), IntegrationProto.KitchenSink.SourceOneOf.KIND)
	expect_equal(msg.get_kind(), SharedProto.SourceKind.USER)

func test_kitchen_sink_round_trip_binary():
	var original := IntegrationProto.KitchenSink.new()
	_populate(original)

	var bytes := original.to_bytes()
	var decoded := IntegrationProto.KitchenSink.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	_expect_populated(decoded)

func test_kitchen_sink_round_trip_text():
	var original := IntegrationProto.KitchenSink.new()
	_populate(original)

	var text := original.to_text()
	var decoded := IntegrationProto.KitchenSink.new()
	expect_equal(decoded.from_text(text), Core.ProtobufError.NO_ERRORS)
	_expect_populated(decoded)

func test_kitchen_sink_oneof_message_branch_round_trip():
	var msg := IntegrationProto.KitchenSink.new()
	msg.set_name("origin-only")
	var origin := IntegrationProto.KitchenSink.Stat.new()
	origin.set_key("source")
	origin.set_value(123)
	msg.set_origin(origin)
	expect_equal(msg.get_source_case(), IntegrationProto.KitchenSink.SourceOneOf.ORIGIN)

	var decoded := IntegrationProto.KitchenSink.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_source_case(), IntegrationProto.KitchenSink.SourceOneOf.ORIGIN)
	expect_equal(decoded.get_origin().get_key(), "source")
	expect_equal(decoded.get_origin().get_value(), 123)
