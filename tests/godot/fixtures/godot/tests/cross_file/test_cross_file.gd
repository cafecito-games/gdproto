extends VestTest

# Regression tests for the bugs that motivated this fixture:
#   - cross-file enum field (helper match arms must qualify by wrapper class)
#   - cross-file message field (from_text must instantiate the qualified type)
#   - well-known type import (transitive imports must be generated)
#   - oneof carrying a cross-file enum

func test_cross_file_enum_round_trip_binary():
	var event := EventEvent.new()
	event.set_color(SharedHue.Hue.RED)
	var bytes := event.to_bytes()

	var decoded := EventEvent.new()
	var err := decoded.from_bytes(bytes)
	expect_equal(err, ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_color(), SharedHue.Hue.RED)

func test_cross_file_message_round_trip_binary():
	var event := EventEvent.new()
	var payload := event.new_payload()
	payload.set_data("hello")
	var bytes := event.to_bytes()

	var decoded := EventEvent.new()
	var err := decoded.from_bytes(bytes)
	expect_equal(err, ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_payload().get_data(), "hello")

func test_well_known_type_round_trip_binary():
	var event := EventEvent.new()
	var ts := event.new_occurred_at()
	ts.set_seconds(1700000000)
	ts.set_nanos(123)
	var bytes := event.to_bytes()

	var decoded := EventEvent.new()
	var err := decoded.from_bytes(bytes)
	expect_equal(err, ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_occurred_at().get_seconds(), 1700000000)
	expect_equal(decoded.get_occurred_at().get_nanos(), 123)

func test_oneof_with_cross_file_enum_round_trip_binary():
	var event := EventEvent.new()
	event.set_kind(SharedSourceKind.SourceKind.USER)
	var bytes := event.to_bytes()

	var decoded := EventEvent.new()
	var err := decoded.from_bytes(bytes)
	expect_equal(err, ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_kind(), SharedSourceKind.SourceKind.USER)

func test_text_format_round_trip_with_cross_file_types():
	# This is the exact path that used to emit unqualified `Timestamp.new()`
	# inside from_text and unqualified enum match arms.
	var event := EventEvent.new()
	event.set_color(SharedHue.Hue.BLUE)
	var ts := event.new_occurred_at()
	ts.set_seconds(42)
	var payload := event.new_payload()
	payload.set_data("text")
	event.set_kind(SharedSourceKind.SourceKind.SYSTEM)

	var text := event.to_text()
	var decoded := EventEvent.new()
	var err := decoded.from_text(text)
	expect_equal(err, ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_color(), SharedHue.Hue.BLUE)
	expect_equal(decoded.get_occurred_at().get_seconds(), 42)
	expect_equal(decoded.get_payload().get_data(), "text")
	expect_equal(decoded.get_kind(), SharedSourceKind.SourceKind.SYSTEM)

func test_cross_file_message_field_from_integration():
	# integration.proto imports shared.proto and uses fixture.Payload as a
	# message-typed field. Round-tripping proves the IntegrationKitchenSink
	# wrapper can resolve SharedPayload across generated files via class_name
	# globals without any preload statements.
	var msg := IntegrationKitchenSink.new()
	msg.set_name("cross-file-message")
	var extra := msg.new_extra()
	extra.set_data("from-shared-proto")

	var bytes := msg.to_bytes()
	var decoded := IntegrationKitchenSink.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_name(), "cross-file-message")
	expect_equal(decoded.get_extra().get_data(), "from-shared-proto")
