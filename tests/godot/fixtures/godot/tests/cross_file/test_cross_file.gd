extends VestTest

# Regression tests for the bugs that motivated this fixture:
#   - cross-file enum field (helper match arms must qualify by wrapper class)
#   - cross-file message field (from_text must instantiate the qualified type)
#   - well-known type import (transitive imports must be generated)
#   - oneof carrying a cross-file enum

const SharedProto = preload("res://generated/shared.pb.gd")
const EventProto = preload("res://generated/event.pb.gd")
const TimestampProto = preload("res://generated/google/protobuf/timestamp.pb.gd")
const Core = preload("res://generated/proto_core_utils.gd")

func test_cross_file_enum_round_trip_binary():
	var event := EventProto.Event.new()
	event.set_color(SharedProto.Hue.RED)
	var bytes := event.to_bytes()

	var decoded := EventProto.Event.new()
	var err := decoded.from_bytes(bytes)
	expect_equal(err, Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_color(), SharedProto.Hue.RED)

func test_cross_file_message_round_trip_binary():
	var event := EventProto.Event.new()
	var payload := event.new_payload()
	payload.set_data("hello")
	var bytes := event.to_bytes()

	var decoded := EventProto.Event.new()
	var err := decoded.from_bytes(bytes)
	expect_equal(err, Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_payload().get_data(), "hello")

func test_well_known_type_round_trip_binary():
	var event := EventProto.Event.new()
	var ts := event.new_occurred_at()
	ts.set_seconds(1700000000)
	ts.set_nanos(123)
	var bytes := event.to_bytes()

	var decoded := EventProto.Event.new()
	var err := decoded.from_bytes(bytes)
	expect_equal(err, Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_occurred_at().get_seconds(), 1700000000)
	expect_equal(decoded.get_occurred_at().get_nanos(), 123)

func test_oneof_with_cross_file_enum_round_trip_binary():
	var event := EventProto.Event.new()
	event.set_kind(SharedProto.SourceKind.USER)
	var bytes := event.to_bytes()

	var decoded := EventProto.Event.new()
	var err := decoded.from_bytes(bytes)
	expect_equal(err, Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_kind(), SharedProto.SourceKind.USER)

func test_text_format_round_trip_with_cross_file_types():
	# This is the exact path that used to emit unqualified `Timestamp.new()`
	# inside from_text and unqualified enum match arms.
	var event := EventProto.Event.new()
	event.set_color(SharedProto.Hue.BLUE)
	var ts := event.new_occurred_at()
	ts.set_seconds(42)
	var payload := event.new_payload()
	payload.set_data("text")
	event.set_kind(SharedProto.SourceKind.SYSTEM)

	var text := event.to_text()
	var decoded := EventProto.Event.new()
	var err := decoded.from_text(text)
	expect_equal(err, Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_color(), SharedProto.Hue.BLUE)
	expect_equal(decoded.get_occurred_at().get_seconds(), 42)
	expect_equal(decoded.get_payload().get_data(), "text")
	expect_equal(decoded.get_kind(), SharedProto.SourceKind.SYSTEM)
